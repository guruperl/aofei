package cache

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/spreadcache"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type contextBlockingRedisClient struct {
	radix.Client
}

func (contextBlockingRedisClient) Do(ctx context.Context, _ radix.Action) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestValidateMode(t *testing.T) {
	for _, mode := range []string{ModeRedis, ModeSpread, ModeAll, ModeRoutes} {
		if err := ValidateMode(mode); err != nil {
			t.Fatalf("ValidateMode(%q) = %v", mode, err)
		}
	}
	if err := ValidateMode("bad"); err == nil {
		t.Fatal("ValidateMode(bad) = nil, want error")
	}
}

type recordingSpreadPublisher struct {
	messages []*nats.Msg
	flushed  bool
	failAt   int
}

func (p *recordingSpreadPublisher) PublishMsg(message *nats.Msg) error {
	if p.failAt > 0 && len(p.messages)+1 == p.failAt {
		return errors.New("publish failed")
	}
	copyMessage := &nats.Msg{Subject: message.Subject, Data: append([]byte(nil), message.Data...)}
	if message.Header != nil {
		copyMessage.Header = make(nats.Header, len(message.Header))
		for key, values := range message.Header {
			copyMessage.Header[key] = append([]string(nil), values...)
		}
	}
	p.messages = append(p.messages, copyMessage)
	return nil
}

func (p *recordingSpreadPublisher) FlushWithContext(context.Context) error {
	p.flushed = true
	return nil
}

func TestPublishSpreadMessagesUsesOrderedGenerationProtocol(t *testing.T) {
	publisher := &recordingSpreadPublisher{}
	messages := []spreadcache.Message{
		{Subject: "creative:7", Data: []byte("creative")},
		{Subject: "slot:1:2", Data: []byte("slot")},
	}
	if err := publishSpreadMessages(context.Background(), publisher, 9, messages); err != nil {
		t.Fatal(err)
	}
	if !publisher.flushed {
		t.Fatal("spread generation was not flushed")
	}
	if len(publisher.messages) != 4 {
		t.Fatalf("published messages = %d, want 4", len(publisher.messages))
	}
	if publisher.messages[0].Subject != spreadcache.BeginSubject || publisher.messages[3].Subject != spreadcache.CommitSubject {
		t.Fatalf("control order = %q ... %q", publisher.messages[0].Subject, publisher.messages[3].Subject)
	}
	manifest, err := spreadcache.ParseManifest(publisher.messages[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Sequence != 9 || manifest.EntryCount != len(messages) {
		t.Fatalf("manifest = %#v", manifest)
	}
	for i, message := range publisher.messages[1:3] {
		if message.Subject != spreadcache.DataSubject || message.Header.Get(spreadcache.GenerationHeader) != "9" || message.Header.Get(spreadcache.OriginalSubjectHeader) != messages[i].Subject {
			t.Fatalf("data message %d = %#v", i, message)
		}
	}
}

func TestPublishSpreadMessagesFailureCannotEmitCommit(t *testing.T) {
	publisher := &recordingSpreadPublisher{failAt: 3}
	err := publishSpreadMessages(context.Background(), publisher, 9, []spreadcache.Message{
		{Subject: "creative:7", Data: []byte("creative")},
		{Subject: "slot:1:2", Data: []byte("slot")},
	})
	if err == nil || !strings.Contains(err.Error(), "publish failed") {
		t.Fatalf("publish error = %v", err)
	}
	if publisher.flushed {
		t.Fatal("failed spread generation was flushed")
	}
	for _, message := range publisher.messages {
		if message.Subject == spreadcache.CommitSubject {
			t.Fatal("failed spread generation emitted commit")
		}
	}
}

func TestPublicationActivatesOnlyConfiguredServingBackend(t *testing.T) {
	tests := []struct {
		name   string
		local  bool
		mode   string
		active bool
	}{
		{"remote redis", false, ModeRedis, true},
		{"remote spread", false, ModeSpread, false},
		{"local redis", true, ModeRedis, false},
		{"local spread", true, ModeSpread, true},
		{"remote all", false, ModeAll, true},
		{"local all", true, ModeAll, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := publicationActivatesManagementAPI(test.local, test.mode); got != test.active {
				t.Fatalf("publicationActivatesManagementAPI(%t, %q)=%t want %t", test.local, test.mode, got, test.active)
			}
		})
	}
}

func TestWriteMiddlemanActivationManifestEnforcesStagesAndHidesCredentialValues(t *testing.T) {
	config := middlemanActivationConfig()
	desired := middlemanActivationCache("Fallback")
	published := middlemanActivationCache("Fallback")
	var out bytes.Buffer
	validatorCalls := 0
	validator := func(ref string) error {
		validatorCalls++
		if ref != "D03_TEST_HEADERS" {
			return errors.New("unexpected credential ref")
		}
		return nil
	}
	if err := WriteMiddlemanActivationManifest(&out, config, desired, published, MiddlemanStageFallback, validator); err != nil {
		t.Fatal(err)
	}
	if validatorCalls != 1 {
		t.Fatalf("credential validator calls = %d, want 1", validatorCalls)
	}
	for _, required := range []string{
		"middleman_activation_ready stage=fallback",
		"entries=1 groups=1 bidders=1 credential_refs=1",
		"fallback_entries=1 always_entries=0",
	} {
		if !strings.Contains(out.String(), required) {
			t.Fatalf("activation manifest missing %q: %s", required, out.String())
		}
	}
	for _, forbidden := range []string{"D03_TEST_HEADERS", "Bearer", "Authorization"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("activation manifest exposed %q: %s", forbidden, out.String())
		}
	}

	config.MiddlemanAlwaysEnabled = true
	if err := WriteMiddlemanActivationManifest(&bytes.Buffer{}, config, middlemanActivationCache("Always"), middlemanActivationCache("Always"), MiddlemanStageAlways, validator); err != nil {
		t.Fatalf("Always activation: %v", err)
	}
}

func TestWriteMiddlemanActivationManifestRejectsStalePublicationAndUnsafeGates(t *testing.T) {
	config := middlemanActivationConfig()
	desired := middlemanActivationCache("Fallback")
	published := middlemanActivationCache("Fallback")
	published.Metadata.RouteDBHighWater = "2026-07-31T23:59:59Z"
	if err := WriteMiddlemanActivationManifest(&bytes.Buffer{}, config, desired, published, MiddlemanStageFallback, func(string) error { return nil }); err == nil || !strings.Contains(err.Error(), "republish") {
		t.Fatalf("stale publication error = %v", err)
	}

	config.MiddlemanEnabled = false
	if err := WriteMiddlemanActivationManifest(&bytes.Buffer{}, config, desired, desired, MiddlemanStageFallback, func(string) error { return nil }); err == nil || !strings.Contains(err.Error(), "fallback activation requires") {
		t.Fatalf("unsafe fallback gate error = %v", err)
	}
	if err := WriteMiddlemanActivationManifest(&bytes.Buffer{}, config, desired, desired, MiddlemanStagePreflight, func(string) error { return errors.New("missing secret") }); err == nil || !strings.Contains(err.Error(), "missing secret") {
		t.Fatalf("credential failure error = %v", err)
	}
}

func middlemanActivationConfig() *dsp.Config {
	return &dsp.Config{
		ConnectArray:                []string{"mysql", "test-dsn"},
		Redis:                       &dsp.Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		TrackingSecret:              "d03-test-signing-secret",
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          7776000,
		DeliveryCacheMaxAgeSeconds:  900,
		DeliveryReservationSeconds:  86700,
		DeliveryStateTTLSeconds:     172800,
		MiddlemanEnabled:            true,
		MiddlemanTimeoutMS:          100,
		MiddlemanMaxBiddersPerImp:   5,
		MiddlemanRouteCacheTTLMS:    5000,
		MiddlemanExchangeDomain:     "exchange.example",
		MiddlemanCallbackTTLSeconds: 86700,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanCallbackBaseURL:    "https://exchange.example",
		PrivacyContextualMiddleman:  true,
	}
}

func middlemanActivationCache(trigger string) *match.MiddlemanRouteCache {
	cache := &match.MiddlemanRouteCache{
		Version: match.MiddlemanRouteCacheVersion,
		Entries: []match.MiddlemanRouteEntry{{
			TargetID: 1, GroupID: 2, TriggerMode: trigger, RouteBidderID: 3,
			BidderID: 4, AdvID: 5, GroupTimeoutMS: 100, BidderTimeoutMS: 100,
			EndpointURL: "https://bidder.example/openrtb", OpenRTBVersion: "2.5",
			CredentialRef: "D03_TEST_HEADERS", SyntheticCampaignID: 6,
			SyntheticItemID: 7, SyntheticCreativeID: 8,
		}},
	}
	cache.Metadata = &match.MiddlemanRouteCacheMetadata{
		GeneratedAt:      time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		EntryCount:       len(cache.Entries),
		Source:           "mysql",
		RouteDBHighWater: "2026-08-01T11:59:00Z",
		Checksum:         cache.RouteChecksum(),
	}
	return cache
}

func TestRunRejectsInvalidUpdateInterval(t *testing.T) {
	err := Run(context.TODO(), nil, nil, nil, nil, Options{Mode: ModeRedis})
	if err == nil {
		t.Fatal("Run with zero update interval = nil, want error")
	}
}

func TestWritePublisherInventoryManifestIsDeterministicAndCredentialFree(t *testing.T) {
	t.Setenv("P03_MANIFEST_TOKEN_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	config := &dsp.Config{
		DirectSSPTokens: dsp.DirectSSPTokenConfig{
			Enabled: true, LegacyReadMode: "allow",
			Current: dsp.DirectSSPTokenKeyConfig{KeyID: "primary", Epoch: 3, KeyEnv: "P03_MANIFEST_TOKEN_KEY"},
		},
	}
	config.DirectSSPAuth.Enabled = true
	issuer, err := dsp.NewDirectSSPTokenIssuer(config)
	if err != nil {
		t.Fatal(err)
	}
	pubmap := acl.PubMap{
		"pub.example": {
			PubID:      7,
			Active:     true,
			Sites:      map[string]uint32{"example.com": 11},
			SiteTypes:  map[uint32]acl.SiteType{11: acl.SiteTypeWeb},
			Slots:      map[uint32]map[string]uint32{11: {"leaderboard": 13}},
			SlotSizes:  map[uint32]map[uint32]uint32{11: {13: match.SizeID2To1(300, 250)}},
			SlotFloors: map[uint32]map[uint32]float64{11: {13: 1.25}},
			Seller: acl.SellerMetadata{
				ID: "seller-7", Type: "Publisher", ASI: "w8m.com",
				Name: "Example Media", Domain: "pub.example", Authorized: true,
			},
			SiteSupply: map[uint32]acl.SiteSupplyMetadata{11: {
				Environment: "Web", CanonicalIdentity: "example.com",
				StoreURL: "https://example.com/ads", IntegrationMode: "BrowserTag",
			}},
			SlotSupply: map[uint32]map[uint32]acl.SlotSupplyMetadata{11: {13: {
				MediaIntent: "Banner", Placement: "AboveFold", RenderContext: "WebPage",
				RefreshMode: "Timed", RefreshSeconds: 60, AdDensity: "Low",
				TrafficQuality: "Reviewed", SourceQuality: "OwnedOperated",
				ManagementControl: "Publisher",
			}}},
		},
	}
	var out bytes.Buffer
	if err := WritePublisherInventoryManifest(&out, pubmap, issuer); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, required := range []string{
		`direct_ssp_integration token_version=v2 token_key_id="primary" token_epoch=3 legacy_read_mode=allow request_authentication=required credential_refresh_seconds=30 credential_max_age_seconds=120 rotation_max_overlap_seconds=86400`,
		`publisher_ready pub_id=7 domain="pub.example" seller_id="seller-7" seller_type=Publisher seller_asi="w8m.com" seller_name="Example Media" seller_domain="pub.example" seller_authorized=true`,
		`site_ready pub_id=7 site_id=11 type=Web identity="example.com" environment=Web canonical_identity="example.com" store_url="https://example.com/ads" integration_mode=BrowserTag token_version=v2 site_token=pz2.site.primary.3.`,
		`slot_ready pub_id=7 site_id=11 slot_id=13 name="leaderboard" size=300x250 floor_usd_cpm=1.250000 media_intent=Banner placement=AboveFold render_context=WebPage refresh_mode=Timed refresh_seconds=60 ad_density=Low traffic_quality=Reviewed source_quality=OwnedOperated management_control=Publisher token_version=v2 slot_token=pz2.slot.primary.3.`,
		`publisher_inventory_ready publishers=1 sites=1 slots=1`,
	} {
		if !bytes.Contains(out.Bytes(), []byte(required)) {
			t.Fatalf("manifest is missing %q:\n%s", required, text)
		}
	}
	for _, forbidden := range []string{"password", "passwd", "cookie", "secret", "account_number"} {
		if bytes.Contains(bytes.ToLower(out.Bytes()), []byte(forbidden)) {
			t.Fatalf("manifest contains forbidden field %q", forbidden)
		}
	}
}

var errManifestOutput = errors.New("manifest output failed")

type failingManifestWriter struct{}

func (failingManifestWriter) Write([]byte) (int, error) {
	return 0, errManifestOutput
}

func TestWritePublisherInventoryManifestReturnsOutputFailure(t *testing.T) {
	issuer, err := dsp.NewDirectSSPTokenIssuer(&dsp.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pubmap := acl.PubMap{
		"pub.example": {
			PubID:      7,
			Active:     true,
			Sites:      map[string]uint32{"example.com": 11},
			SiteTypes:  map[uint32]acl.SiteType{11: acl.SiteTypeWeb},
			Slots:      map[uint32]map[string]uint32{11: {"leaderboard": 13}},
			SlotSizes:  map[uint32]map[uint32]uint32{11: {13: match.SizeID2To1(300, 250)}},
			SlotFloors: map[uint32]map[uint32]float64{11: {13: 1.25}},
		},
	}
	if err := WritePublisherInventoryManifest(failingManifestWriter{}, pubmap, issuer); !errors.Is(err, errManifestOutput) {
		t.Fatalf("WritePublisherInventoryManifest error = %v", err)
	}
}

func TestSwapRedisStaticCachesReplacesOneCompleteGeneration(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	liveHash := func(key string) {
		t.Helper()
		if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "value", "old")); err != nil {
			t.Fatal(err)
		}
	}
	shadowHash := func(key string) {
		t.Helper()
		if err := client.Do(ctx, radix.Cmd(nil, "HSET", key+redisShadowSuffix, "value", "new")); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience, match.HashNameCreative} {
		liveHash(key)
	}
	for _, key := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience} {
		shadowHash(key)
	}
	for _, key := range []string{match.HashNameMiddlemanRoutes, match.HashNameMiddlemanRoutesV2} {
		if err := client.Do(ctx, radix.Cmd(nil, "SET", key, "old")); err != nil {
			t.Fatal(err)
		}
		if err := client.Do(ctx, radix.Cmd(nil, "SET", key+redisShadowSuffix, "new")); err != nil {
			t.Fatal(err)
		}
	}
	newSlot := match.HashNameRAdvs(1)
	oldSlot := match.HashNameRAdvs(2)
	liveHash(newSlot)
	liveHash(oldSlot)
	shadowHash(newSlot)

	if err := swapRedisStaticCaches(ctx, client, []uint32{1}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience, newSlot} {
		var got string
		if err := client.Do(ctx, radix.Cmd(&got, "HGET", key, "value")); err != nil {
			t.Fatal(err)
		}
		if got != "new" {
			t.Fatalf("%s value = %q, want new", key, got)
		}
	}
	for _, key := range []string{match.HashNameMiddlemanRoutes, match.HashNameMiddlemanRoutesV2} {
		var got string
		if err := client.Do(ctx, radix.Cmd(&got, "GET", key)); err != nil {
			t.Fatal(err)
		}
		if got != "new" {
			t.Fatalf("%s value = %q, want new", key, got)
		}
	}
	for _, key := range []string{match.HashNameCreative, oldSlot} {
		var exists int
		if err := client.Do(ctx, radix.Cmd(&exists, "EXISTS", key)); err != nil {
			t.Fatal(err)
		}
		if exists != 0 {
			t.Fatalf("%s still exists after empty/obsolete swap", key)
		}
	}
}

func TestPublishRedisGenerationBuildFailureLeavesLiveGeneration(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Do(ctx, radix.Cmd(nil, "HSET", acl.HashNamePubmap, "value", "old")); err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT t.slot_id").WillReturnError(errors.New("build failed"))
	if err := PublishRedisGeneration(ctx, client, db, nil, []uint32{1}); err == nil {
		t.Fatal("PublishRedisGeneration error = nil, want build failure")
	}
	var got string
	if err := client.Do(ctx, radix.Cmd(&got, "HGET", acl.HashNamePubmap, "value")); err != nil {
		t.Fatal(err)
	}
	if got != "old" {
		t.Fatalf("live pubmap value = %q, want old generation", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRedisShadowFailureCleanupIsBounded(t *testing.T) {
	started := time.Now()
	err := cleanupRedisShadowCachesWithTimeout(contextBlockingRedisClient{}, 20*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cleanup error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("cleanup took %s, want bounded exit", elapsed)
	}
}

func TestAttributeLogScannerAcceptsLargeLines(t *testing.T) {
	line := bytes.Repeat([]byte("x"), maxAttributeLogLineBytes)
	scanner := newAttributeLogScanner(bytes.NewReader(append(line, '\n')))
	if !scanner.Scan() {
		t.Fatalf("Scan = false: %v", scanner.Err())
	}
	if got := len(scanner.Bytes()); got != len(line) {
		t.Fatalf("line length = %d, want %d", got, len(line))
	}
}

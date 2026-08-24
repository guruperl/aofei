package cache

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/managementapi"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

const (
	redisShadowSuffix        = ":next"
	maxAttributeLogLineBytes = 8 * 1024 * 1024
)

const (
	ModeRedis  = "redis"
	ModeSpread = "spread"
	ModeAll    = "all"
	ModeRoutes = "routes"

	MiddlemanStagePreflight = "preflight"
	MiddlemanStageFallback  = "fallback"
	MiddlemanStageAlways    = "always"
)

type Options struct {
	Mode           string
	UpdatePubMap   bool
	UpdateInterval int
	UpdateStamp    int
}

func ValidateMode(mode string) error {
	switch mode {
	case ModeRedis, ModeSpread, ModeAll, ModeRoutes:
		return nil
	default:
		return fmt.Errorf("unknown cache mode %q; expected redis, spread, all, or routes", mode)
	}
}

func Run(ctx context.Context, c *dsp.Config, redis radix.Client, db *sql.DB, nc *nats.Conn, opts Options) error {
	if err := ValidateMode(opts.Mode); err != nil {
		return err
	}
	if opts.UpdateInterval <= 0 {
		return fmt.Errorf("cache update interval must be positive")
	}
	if opts.Mode == ModeRoutes {
		return match.DBGetMiddlemanRoutesToRedis(ctx, redis, db)
	}
	if c == nil {
		return fmt.Errorf("cache configuration is nil")
	}
	if opts.Mode == ModeSpread || opts.Mode == ModeAll {
		if nc == nil {
			return fmt.Errorf("cache mode %q requires NATS", opts.Mode)
		}
		if c.Spread == "" {
			return fmt.Errorf("cache mode %q requires spread directory", opts.Mode)
		}
		if err := os.MkdirAll(c.Spread, 0755); err != nil {
			return err
		}
	}
	activatesManagementAPI := publicationActivatesManagementAPI(c.IsLocal, opts.Mode)
	var publicationToken []byte
	if activatesManagementAPI {
		var err error
		publicationToken, err = managementapi.PrepareOperationsPublication(ctx, db)
		if err != nil {
			return err
		}
	}

	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
		return err
	}

	pubmap, err := acl.DBGetPubMap(db)
	if err != nil {
		return err
	}
	if err := acl.ValidateCommercialPubMap(pubmap); err != nil {
		return fmt.Errorf("publisher inventory readiness: %w", err)
	}

	if opts.UpdatePubMap {
		if err := UpdatePubMap(c, db, pubmap, opts.UpdateInterval, opts.UpdateStamp); err != nil {
			return err
		}
	}

	var publicationErr error
	switch opts.Mode {
	case ModeSpread:
		publicationErr = WriteToSpread(ctx, nc, db, pubmap, sizeIDs)
	case ModeRedis:
		publicationErr = WriteToRedis(ctx, redis, db, pubmap, sizeIDs)
	case ModeAll:
		if err := WriteToSpread(ctx, nc, db, pubmap, sizeIDs); err != nil {
			return err
		}
		publicationErr = WriteToRedis(ctx, redis, db, pubmap, sizeIDs)
	default:
		return ValidateMode(opts.Mode)
	}
	if publicationErr != nil {
		return publicationErr
	}
	if !activatesManagementAPI {
		return nil
	}
	return managementapi.MarkOperationsActive(ctx, db, opts.Mode, publicationToken, time.Now())
}

func publicationActivatesManagementAPI(isLocal bool, mode string) bool {
	if mode == ModeAll {
		return true
	}
	if isLocal {
		return mode == ModeSpread
	}
	return mode == ModeRedis
}

func Read(ctx context.Context, out io.Writer, c *dsp.Config, redis radix.Client, db *sql.DB, mode string) error {
	if err := ValidateMode(mode); err != nil {
		return err
	}
	if mode == ModeRoutes {
		return RedisRoutesRead(ctx, out, redis)
	}
	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
		return err
	}
	switch mode {
	case ModeSpread:
		return SpreadRead(out, c.Spread, sizeIDs)
	case ModeRedis:
		return RedisRead(ctx, out, redis, sizeIDs)
	case ModeAll:
		if err := SpreadRead(out, c.Spread, sizeIDs); err != nil {
			return err
		}
		return RedisRead(ctx, out, redis, sizeIDs)
	case ModeRoutes:
		return RedisRoutesRead(ctx, out, redis)
	default:
		return ValidateMode(mode)
	}
}

// ValidatePublisherInventory reads the current database inventory and emits a
// deterministic, secret-free activation manifest without mutating a cache or
// database row. Public locators come from the same configured issuer as /pz.
func ValidatePublisherInventory(out io.Writer, db *sql.DB, issuer *dsp.DirectSSPTokenIssuer) error {
	if db == nil {
		return fmt.Errorf("publisher inventory database is nil")
	}
	pubmap, err := acl.DBGetPubMap(db)
	if err != nil {
		return err
	}
	return WritePublisherInventoryManifest(out, pubmap, issuer)
}

// ValidateMiddlemanActivation compares the current active MySQL route model to
// the published v2 Redis generation, resolves every credential reference
// without printing its value, and enforces the requested rollout gate. It is a
// read-only activation check and performs no partner request.
func ValidateMiddlemanActivation(ctx context.Context, out io.Writer, c *dsp.Config, redis radix.Client, db *sql.DB, stage string) error {
	if db == nil {
		return fmt.Errorf("middleman activation database is nil")
	}
	if redis == nil {
		return fmt.Errorf("middleman activation redis client is nil")
	}
	if err := match.DBValidateMiddlemanActivation(ctx, db); err != nil {
		return err
	}
	desired, err := match.DBGetMiddlemanRouteCache(ctx, db)
	if err != nil {
		return err
	}
	published, err := match.MiddlemanRouteCacheFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	return WriteMiddlemanActivationManifest(out, c, desired, published, stage, dsp.ValidateMiddlemanCredentialRef)
}

// WriteMiddlemanActivationManifest is the pure validation/reporting boundary
// used by the command and activation tests.
func WriteMiddlemanActivationManifest(out io.Writer, c *dsp.Config, desired, published *match.MiddlemanRouteCache, stage string, validateCredential func(string) error) error {
	if out == nil {
		return fmt.Errorf("middleman activation output is nil")
	}
	if c == nil {
		return fmt.Errorf("middleman activation config is nil")
	}
	switch stage {
	case MiddlemanStagePreflight, MiddlemanStageFallback, MiddlemanStageAlways:
	default:
		return fmt.Errorf("middleman activation stage %q is invalid; expected preflight, fallback, or always", stage)
	}
	activationConfig := *c
	activationConfig.MiddlemanEnabled = true
	if err := activationConfig.Validate(dsp.ConfigModeCache); err != nil {
		return fmt.Errorf("middleman activation config: %w", err)
	}
	if activationConfig.MiddlemanTimeoutMS <= 0 || activationConfig.MiddlemanTimeoutMS > 5000 {
		return fmt.Errorf("middleman activation config: middleman_timeout_ms must be between 1 and 5000")
	}
	if activationConfig.MiddlemanMaxBiddersPerImp <= 0 || activationConfig.MiddlemanMaxBiddersPerImp > 100 {
		return fmt.Errorf("middleman activation config: middleman_max_bidders_per_imp must be between 1 and 100")
	}
	if strings.TrimSpace(activationConfig.MiddlemanExchangeDomain) != activationConfig.MiddlemanExchangeDomain || activationConfig.MiddlemanExchangeDomain == "" || len(activationConfig.MiddlemanExchangeDomain) > 253 {
		return fmt.Errorf("middleman activation config: middleman_exchange_domain must be a nonempty bounded domain")
	}
	if c.MiddlemanAlwaysEnabled && !c.MiddlemanEnabled {
		return fmt.Errorf("middleman activation config: middleman_always_enabled requires middleman_enabled")
	}
	if err := validateMiddlemanRouteGeneration("database", desired, false); err != nil {
		return err
	}
	if err := validateMiddlemanRouteGeneration("redis", published, true); err != nil {
		return err
	}
	if desired.RouteChecksum() != published.RouteChecksum() || desired.Metadata.RouteDBHighWater != published.Metadata.RouteDBHighWater {
		return fmt.Errorf("published middleman routes do not match current database state; republish route cache")
	}
	if validateCredential == nil {
		return fmt.Errorf("middleman credential validator is nil")
	}
	bidders := make(map[uint32]struct{})
	groups := make(map[uint32]struct{})
	credentials := make(map[string]struct{})
	fallbackEntries, alwaysEntries := 0, 0
	for _, entry := range desired.Entries {
		bidders[entry.BidderID] = struct{}{}
		groups[entry.GroupID] = struct{}{}
		credentials[entry.CredentialRef] = struct{}{}
		switch entry.TriggerMode {
		case "", "Fallback":
			fallbackEntries++
		case "Always":
			alwaysEntries++
		default:
			return fmt.Errorf("middleman route trigger mode %q is invalid", entry.TriggerMode)
		}
	}
	for ref := range credentials {
		if err := validateCredential(ref); err != nil {
			return fmt.Errorf("middleman credential %q is not ready: %w", ref, err)
		}
	}
	switch stage {
	case MiddlemanStageFallback:
		if !c.MiddlemanEnabled || !c.PrivacyContextualMiddleman || c.MiddlemanAlwaysEnabled {
			return fmt.Errorf("fallback activation requires middleman_enabled and privacy_contextual_middleman_enabled true with middleman_always_enabled false")
		}
		if fallbackEntries == 0 {
			return fmt.Errorf("fallback activation has no active Fallback route entries")
		}
	case MiddlemanStageAlways:
		if !c.MiddlemanEnabled || !c.PrivacyContextualMiddleman || !c.MiddlemanAlwaysEnabled {
			return fmt.Errorf("Always activation requires all middleman and privacy gates enabled")
		}
		if alwaysEntries == 0 {
			return fmt.Errorf("Always activation has no active Always route entries")
		}
	}
	_, err := fmt.Fprintf(out,
		"middleman_activation_ready stage=%s entries=%d groups=%d bidders=%d credential_refs=%d fallback_entries=%d always_entries=%d middleman_enabled=%t privacy_contextual_middleman_enabled=%t middleman_always_enabled=%t route_db_high_water=%q checksum=%s\n",
		stage, len(desired.Entries), len(groups), len(bidders), len(credentials), fallbackEntries, alwaysEntries,
		c.MiddlemanEnabled, c.PrivacyContextualMiddleman, c.MiddlemanAlwaysEnabled,
		published.Metadata.RouteDBHighWater, published.Metadata.Checksum)
	return err
}

func validateMiddlemanRouteGeneration(source string, cache *match.MiddlemanRouteCache, requireCurrent bool) error {
	if cache == nil || len(cache.Entries) == 0 {
		return fmt.Errorf("%s middleman route generation has no active entries", source)
	}
	if requireCurrent && cache.Version != match.MiddlemanRouteCacheVersion {
		return fmt.Errorf("%s middleman route generation version is %d, want %d", source, cache.Version, match.MiddlemanRouteCacheVersion)
	}
	if cache.Metadata == nil {
		return fmt.Errorf("%s middleman route generation metadata is missing", source)
	}
	if cache.Metadata.Source != "mysql" {
		return fmt.Errorf("%s middleman route source is %q, want mysql", source, cache.Metadata.Source)
	}
	if cache.Metadata.EntryCount != len(cache.Entries) {
		return fmt.Errorf("%s middleman route entry count is %d, want %d", source, cache.Metadata.EntryCount, len(cache.Entries))
	}
	if cache.Metadata.Checksum == "" || cache.Metadata.Checksum != cache.RouteChecksum() {
		return fmt.Errorf("%s middleman route checksum is invalid", source)
	}
	if _, err := time.Parse(time.RFC3339, cache.Metadata.GeneratedAt); err != nil {
		return fmt.Errorf("%s middleman route generated_at is invalid: %w", source, err)
	}
	for _, entry := range cache.Entries {
		if err := entry.ValidatePartnerProfile(); err != nil {
			return fmt.Errorf("%s middleman bidder %d profile: %w", source, entry.BidderID, err)
		}
	}
	return nil
}

func WritePublisherInventoryManifest(out io.Writer, pubmap acl.PubMap, issuer *dsp.DirectSSPTokenIssuer) error {
	if out == nil {
		return fmt.Errorf("publisher inventory output is nil")
	}
	if issuer == nil {
		return fmt.Errorf("publisher inventory token issuer is nil")
	}
	if err := acl.ValidateCommercialPubMap(pubmap); err != nil {
		return err
	}
	domains := make([]string, 0, len(pubmap))
	for domain, pub := range pubmap {
		if pub != nil && pub.Active && (pub.LimitImps == 0 || pub.CurrentImps < pub.LimitImps) {
			domains = append(domains, domain)
		}
	}
	sort.Strings(domains)
	if len(domains) == 0 {
		return fmt.Errorf("no active approved publisher inventory is publishable")
	}
	writef := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(out, format, args...)
		return err
	}
	metadata := issuer.Metadata()
	if err := writef("direct_ssp_integration token_version=%s token_key_id=%q token_epoch=%d legacy_read_mode=%s request_authentication=%s credential_refresh_seconds=%d credential_max_age_seconds=%d rotation_max_overlap_seconds=%d\n",
		metadata.TokenVersion, metadata.TokenKeyID, metadata.TokenEpoch,
		metadata.LegacyReadMode, metadata.RequestAuthentication,
		metadata.CredentialRefreshSeconds, metadata.CredentialMaxAgeSeconds,
		metadata.RotationMaxOverlapSeconds); err != nil {
		return err
	}
	siteCount, slotCount := 0, 0
	for _, domain := range domains {
		pub := pubmap[domain]
		seller := pub.Seller
		if seller.Type == "" {
			seller.Type = "Publisher"
		}
		if err := writef("publisher_ready pub_id=%d domain=%q seller_id=%q seller_type=%s seller_asi=%q seller_name=%q seller_domain=%q seller_authorized=%t\n",
			pub.PubID, domain, seller.ID, seller.Type, seller.ASI,
			seller.Name, seller.Domain, seller.Authorized); err != nil {
			return err
		}
		siteIdentities := make([]string, 0, len(pub.Sites))
		for identity := range pub.Sites {
			siteIdentities = append(siteIdentities, identity)
		}
		sort.Strings(siteIdentities)
		for _, identity := range siteIdentities {
			siteID := pub.Sites[identity]
			supply := pub.SupplyFor(siteID, 0).Site
			siteToken, err := issuer.PackSite(pub.PubID, siteID)
			if err != nil {
				return err
			}
			if err := writef("site_ready pub_id=%d site_id=%d type=%s identity=%q environment=%s canonical_identity=%q store_url=%q integration_mode=%s token_version=%s site_token=%s\n",
				pub.PubID, siteID, commercialSiteType(pub.SiteTypes[siteID]), identity,
				supply.Environment, supply.CanonicalIdentity, supply.StoreURL,
				supply.IntegrationMode, metadata.TokenVersion, siteToken); err != nil {
				return err
			}
			siteCount++
			slotNames := make([]string, 0, len(pub.Slots[siteID]))
			for name := range pub.Slots[siteID] {
				slotNames = append(slotNames, name)
			}
			sort.Strings(slotNames)
			for _, name := range slotNames {
				slotID := pub.Slots[siteID][name]
				supply := pub.SupplyFor(siteID, slotID).Slot
				sizeID := pub.SlotSizes[siteID][slotID]
				width, height := match.SizeID1To2(sizeID)
				slotToken, err := issuer.PackSlot(pub.PubID, siteID, slotID, sizeID)
				if err != nil {
					return err
				}
				if err := writef("slot_ready pub_id=%d site_id=%d slot_id=%d name=%q size=%dx%d floor_usd_cpm=%.6f media_intent=%s placement=%s render_context=%s refresh_mode=%s refresh_seconds=%d ad_density=%s traffic_quality=%s source_quality=%s management_control=%s token_version=%s slot_token=%s\n",
					pub.PubID, siteID, slotID, name, width, height,
					pub.SlotFloors[siteID][slotID], supply.MediaIntent, supply.Placement,
					supply.RenderContext, supply.RefreshMode, supply.RefreshSeconds,
					supply.AdDensity, supply.TrafficQuality, supply.SourceQuality,
					supply.ManagementControl, metadata.TokenVersion, slotToken); err != nil {
					return err
				}
				slotCount++
			}
		}
	}
	return writef("publisher_inventory_ready publishers=%d sites=%d slots=%d\n", len(domains), siteCount, slotCount)
}

func commercialSiteType(siteType acl.SiteType) string {
	switch siteType {
	case acl.SiteTypeWeb:
		return "Web"
	case acl.SiteTypeAPP:
		return "App"
	default:
		return "Unknown"
	}
}

func RedisRoutesRead(ctx context.Context, out io.Writer, redis radix.Client) error {
	middlemanRoutes, err := match.MiddlemanRouteCacheFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(middlemanRoutes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Middleman routes:\n%s\n", bs)
	return nil
}

func RedisRead(ctx context.Context, out io.Writer, redis radix.Client, sizeIDs []uint32) error {
	pubmap, err := acl.PubMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "pubmap:\n%s\n", bs)
	direct, err := acl.DirectPubMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(direct, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\npubmap:by-id:\n%s\n", bs)

	for _, sizeID := range sizeIDs {
		width, height := match.SizeID1To2(sizeID)
		hash, err := match.RAdvsFromRedisBySizeID(ctx, redis, sizeID)
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "\nRAdvs for sizeID %d x %d:\n%s\n", width, height, bs)
	}

	audiences, err := match.AudienceMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nAudiences:\n%s\n", bs)

	creatives, err := match.CreativeMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nCreatives:\n%s\n", bs)

	middlemanRoutes, err := match.MiddlemanRouteCacheFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(middlemanRoutes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nMiddleman routes:\n%s\n", bs)
	return nil
}

func WriteToRedis(ctx context.Context, redis radix.Client, db *sql.DB, pubmap acl.PubMap, sizeIDs []uint32) error {
	if err := cleanupRedisShadowCaches(ctx, redis); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = cleanupRedisShadowCaches(context.Background(), redis)
		}
	}()

	if err := pubmap.ToRedisKeys(ctx, redis, acl.HashNamePubmap+redisShadowSuffix, acl.HashNamePubByID+redisShadowSuffix); err != nil {
		return err
	}

	sink := match.RedisCacheSink{Client: redis, KeySuffix: redisShadowSuffix}
	for _, sizeID := range sizeIDs {
		if err := match.DBGetRAdvsToRedisSpreadBySizeID(ctx, sink, db, sizeID); err != nil {
			return err
		}
	}

	if err := match.DBGetAudiencesToCache(ctx, sink, db); err != nil {
		return err
	}

	if err := match.DBGetCreativesToRedisSpread(ctx, sink, db); err != nil {
		return err
	}

	if err := match.DBGetMiddlemanRoutesToRedisKeys(ctx, redis, db, match.HashNameMiddlemanRoutes+redisShadowSuffix, match.HashNameMiddlemanRoutesV2+redisShadowSuffix); err != nil {
		return err
	}
	if err := swapRedisStaticCaches(ctx, redis, sizeIDs); err != nil {
		return err
	}
	committed = true
	return cleanupRedisShadowCaches(ctx, redis)
}

func cleanupRedisShadowCaches(ctx context.Context, redis radix.Client) error {
	keys := []string{
		acl.HashNamePubmap + redisShadowSuffix,
		acl.HashNamePubByID + redisShadowSuffix,
		match.HashNameAudience + redisShadowSuffix,
		match.HashNameCreative + redisShadowSuffix,
		match.HashNameMiddlemanRoutes + redisShadowSuffix,
		match.HashNameMiddlemanRoutesV2 + redisShadowSuffix,
	}
	if err := redis.Do(ctx, radix.Cmd(nil, "DEL", keys...)); err != nil {
		return err
	}
	return DeleteRedisKeysByPattern(ctx, redis, match.HashNameSlot+":*"+redisShadowSuffix)
}

func swapRedisStaticCaches(ctx context.Context, redis radix.Client, sizeIDs []uint32) error {
	type keyPair struct {
		live   string
		shadow string
		exists bool
	}
	pairs := []keyPair{
		{live: acl.HashNamePubmap, shadow: acl.HashNamePubmap + redisShadowSuffix},
		{live: acl.HashNamePubByID, shadow: acl.HashNamePubByID + redisShadowSuffix},
		{live: match.HashNameAudience, shadow: match.HashNameAudience + redisShadowSuffix},
		{live: match.HashNameCreative, shadow: match.HashNameCreative + redisShadowSuffix},
		{live: match.HashNameMiddlemanRoutes, shadow: match.HashNameMiddlemanRoutes + redisShadowSuffix},
		{live: match.HashNameMiddlemanRoutesV2, shadow: match.HashNameMiddlemanRoutesV2 + redisShadowSuffix},
	}
	newSlots := make(map[string]struct{})
	for _, sizeID := range sizeIDs {
		live := match.HashNameRAdvs(sizeID)
		if _, ok := newSlots[live]; ok {
			continue
		}
		newSlots[live] = struct{}{}
		pairs = append(pairs, keyPair{live: live, shadow: live + redisShadowSuffix})
	}
	for i := range pairs {
		var exists int
		if err := redis.Do(ctx, radix.Cmd(&exists, "EXISTS", pairs[i].shadow)); err != nil {
			return err
		}
		pairs[i].exists = exists != 0
	}
	liveSlots, err := redisLiveSlotKeys(ctx, redis)
	if err != nil {
		return err
	}
	return redis.Do(ctx, radix.WithConn("", func(ctx context.Context, conn radix.Conn) (err error) {
		if err = conn.Do(ctx, radix.Cmd(nil, "MULTI")); err != nil {
			return err
		}
		inMulti := true
		defer func() {
			if err != nil && inMulti {
				_ = conn.Do(ctx, radix.Cmd(nil, "DISCARD"))
			}
		}()
		for _, pair := range pairs {
			if pair.exists {
				err = conn.Do(ctx, radix.Cmd(nil, "RENAME", pair.shadow, pair.live))
			} else {
				err = conn.Do(ctx, radix.Cmd(nil, "DEL", pair.live))
			}
			if err != nil {
				return err
			}
		}
		for _, key := range liveSlots {
			if _, ok := newSlots[key]; ok {
				continue
			}
			if err = conn.Do(ctx, radix.Cmd(nil, "DEL", key)); err != nil {
				return err
			}
		}
		err = conn.Do(ctx, radix.Cmd(nil, "EXEC"))
		inMulti = false
		return err
	}))
}

func redisLiveSlotKeys(ctx context.Context, redis radix.Client) ([]string, error) {
	var keys []string
	scanner := (radix.ScannerConfig{Command: "SCAN", Pattern: match.HashNameSlot + ":*", Count: 256}).New(redis)
	for {
		var key string
		if !scanner.Next(ctx, &key) {
			break
		}
		suffix := strings.TrimPrefix(key, match.HashNameSlot+":")
		if _, err := strconv.ParseUint(suffix, 10, 32); err == nil {
			keys = append(keys, key)
		}
	}
	if err := scanner.Close(); err != nil {
		return nil, err
	}
	sort.Strings(keys)
	return keys, nil
}

func SpreadRead(out io.Writer, top string, sizeIDs []uint32) error {
	pubmap, err := acl.PubMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "pubmap:\n%s\n", bs)
	direct := acl.DirectPubMapFromPubMap(pubmap)
	bs, err = json.MarshalIndent(direct, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nderived pubmap:by-id:\n%s\n", bs)

	for _, sizeID := range sizeIDs {
		width, height := match.SizeID1To2(sizeID)
		hash, err := match.RAdvsFromIOBySizeID(top, sizeID)
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "\nRAdvs for sizeID %d x %d:\n%s\n", width, height, bs)
	}

	audiences, err := match.AudienceMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nAudiences:\n%s\n", bs)

	creatives, err := match.CreativeMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nCreatives:\n%s\n", bs)
	return nil
}

func WriteToSpread(ctx context.Context, nc *nats.Conn, db *sql.DB, pubmap acl.PubMap, sizeIDs []uint32) error {
	if nc == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	for _, family := range []string{acl.HashNamePubmap, match.HashNameAudience, match.HashNameCreative, match.HashNameSlot} {
		if err := PublishSpreadReset(nc, family); err != nil {
			return err
		}
	}

	if err := pubmap.ToSpread(nc); err != nil {
		return err
	}

	for _, sizeID := range sizeIDs {
		if err := match.DBGetRAdvsToRedisSpreadBySizeID(ctx, nc, db, sizeID); err != nil {
			return err
		}
	}

	if err := match.DBGetAudiencesToSpread(nc, db); err != nil {
		return err
	}

	return match.DBGetCreativesToRedisSpread(ctx, nc, db)
}

func ResetRedisStaticCaches(ctx context.Context, redis radix.Client) error {
	for _, name := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience, match.HashNameCreative, match.HashNameMiddlemanRoutes, match.HashNameMiddlemanRoutesV2} {
		if err := redis.Do(ctx, radix.Cmd(nil, "DEL", name)); err != nil {
			return err
		}
	}
	return DeleteRedisKeysByPattern(ctx, redis, match.HashNameSlot+":*")
}

func DeleteRedisKeysByPattern(ctx context.Context, redis radix.Client, pattern string) error {
	const chunk = 256
	var keys []string
	scanner := (radix.ScannerConfig{Command: "SCAN", Pattern: pattern, Count: chunk}).New(redis)
	for {
		var key string
		if !scanner.Next(ctx, &key) {
			break
		}
		keys = append(keys, key)
		if len(keys) == chunk {
			if err := redis.Do(ctx, radix.Cmd(nil, "DEL", keys...)); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	if err := scanner.Close(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return redis.Do(ctx, radix.Cmd(nil, "DEL", keys...))
	}
	return nil
}

func PublishSpreadReset(nc *nats.Conn, family string) error {
	return nc.Publish(family+":__reset__", nil)
}

func UpdatePubMap(c *dsp.Config, db *sql.DB, pubmap acl.PubMap, interval, stamp int) error {
	if interval <= 0 {
		return fmt.Errorf("cache update interval must be positive")
	}
	var stampObject *dsp.Stamp
	if stamp > 0 {
		stampObject = dsp.NewStamp(interval, stamp)
	} else {
		stampObject = dsp.NewStamp(interval)
	}
	fn := c.NewLogfileName(dsp.SUBJECTAttribute, stampObject)
	fh, err := os.Open(fn)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer fh.Close()

	scanner := newAttributeLogScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		plus := new(match.AttributePlus)
		if err := json.Unmarshal([]byte(line), plus); err != nil {
			return err
		}
		acl := plus.Attribute.ACL
		siteType := "Web"
		if plus.Attribute.IsApp {
			siteType = "App"
		}
		pub, err := pubmap.DBAddNew(db, acl.PubStr, acl.SiteStr, siteType, acl.SlotStr)
		if err != nil {
			return err
		}
		pubmap[acl.PubStr] = pub
	}
	return scanner.Err()
}

func newAttributeLogScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxAttributeLogLineBytes+1)
	return scanner
}

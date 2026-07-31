package dsp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

func TestControllerOptionsCanDisableNATSAndMaxMindIndependently(t *testing.T) {
	defaults := applyControllerOptions()
	if !defaults.nats || !defaults.maxmind {
		t.Fatalf("defaults = %+v, want both optional services enabled", defaults)
	}

	withoutNATS := applyControllerOptions(WithoutNATS())
	if withoutNATS.nats || !withoutNATS.maxmind {
		t.Fatalf("WithoutNATS = %+v, want nats disabled and maxmind enabled", withoutNATS)
	}

	withoutMaxMind := applyControllerOptions(WithoutMaxMind())
	if !withoutMaxMind.nats || withoutMaxMind.maxmind {
		t.Fatalf("WithoutMaxMind = %+v, want nats enabled and maxmind disabled", withoutMaxMind)
	}

	withoutBoth := applyControllerOptions(WithoutNATS(), WithoutMaxMind())
	if withoutBoth.nats || withoutBoth.maxmind {
		t.Fatalf("without both = %+v, want both disabled", withoutBoth)
	}

	guard := func(context.Context, string) error { return fmt.Errorf("guard") }
	client := &http.Client{}
	logger := zap.NewNop()
	withDeps := applyControllerOptions(WithHTTPClient(client), WithLogger(logger), WithCallbackURLGuard(guard), withMiddlemanCallbackStore(newMemoryMiddlemanCallbackStore()))
	if withDeps.httpClient != client || withDeps.logger != logger || withDeps.callbackURLGuard == nil || withDeps.callbackStore == nil {
		t.Fatalf("injected deps not retained: %+v", withDeps)
	}
}

func TestServeStatusRejectsInvalidTrackingAuctionPrice(t *testing.T) {
	err := (&Controller{}).serveStatus(context.TODO(), StatusTrackImp, time.Now(), url.Values{
		"auction_price": []string{"bad-price"},
	})
	if err == nil {
		t.Fatal("expected invalid auction_price to return an error")
	}
}

func TestPublishBidAuditNoNATSIsNoop(t *testing.T) {
	err := (&Controller{}).publishBidAudit(nil, nil, nil, zeroRAdv(), 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServeStatusRequiresSignatureForCapMutation(t *testing.T) {
	capValue, err := (match.Cap{CapNumber: 1, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	err = (&Controller{C: &Config{TrackingSecret: "test-secret"}}).serveStatus(context.TODO(), StatusTrackImp, time.Now(), url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
		"cap":              []string{capValue},
	})
	if err == nil {
		t.Fatal("expected unsigned cap mutation to fail")
	}
}

func TestServeStatusRequiresSignatureWithoutCap(t *testing.T) {
	err := (&Controller{C: &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600}}).serveStatus(context.TODO(), StatusTrackImp, time.Now(), url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
	})
	if err == nil {
		t.Fatal("expected unsigned tracker without cap to fail")
	}
}

func TestServeStatusUsesConfiguredSignatureTTL(t *testing.T) {
	args := url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
	}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("test-secret", "/imp", args))
	err := (&Controller{C: &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600}}).serveStatus(context.TODO(), StatusTrackImp, time.Now(), args)
	if err == nil {
		t.Fatal("expected configured tracking TTL to reject expired impression")
	}
}

func TestClickRedirectUsesConfiguredSignatureTTL(t *testing.T) {
	args := url.Values{
		"auction_id":     []string{"auction"},
		"auction_bid_id": []string{"0000000000000001user"},
		"auction_imp_id": []string{"imp"},
		"auction_price":  []string{"1.0"},
		"demand":         []string{"demand"},
		"supply":         []string{"supply"},
		"redirect":       []string{"https://advertiser.example/landing"},
	}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("test-secret", "/clk", args))
	_, _, err := (&Controller{C: &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600}}).clickRedirectTarget(args)
	if err == nil {
		t.Fatal("expected configured tracking TTL to reject expired click redirect")
	}
}

func TestServeStatusPublishesCapEventWithoutUserID(t *testing.T) {
	capValue, err := (match.Cap{CapNumber: 1, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	args := url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
		"cap":              []string{capValue},
	}
	addTrackingSignature("test-secret", "/imp", args)
	published := 0
	controller := &Controller{
		C: &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600},
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	if err := controller.serveStatus(context.TODO(), StatusTrackImp, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published = %d, want 1", published)
	}
}

func TestServeStatusSuppressesDuplicateImpAndClickEvents(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	capValue, err := (match.Cap{CapNumber: 2, CapPeriod: 60, ClickNumber: 2, ClickPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	base := url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
		"cap":              []string{capValue},
	}
	published := 0
	controller := &Controller{
		C: &Config{
			TrackingSecret:              "test-secret",
			TrackingSignatureTTLSeconds: 3600,
			CapStateTTLSeconds:          7200,
		},
		Redis: client,
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	for _, status := range []Status{StatusTrackImp, StatusTrackClk} {
		args := make(url.Values, len(base))
		for key, values := range base {
			args[key] = append([]string(nil), values...)
		}
		addTrackingSignature("test-secret", status.path(), args)
		for i := 0; i < 2; i++ {
			if err := controller.serveStatus(ctx, status, time.Now(), args); err != nil {
				t.Fatal(err)
			}
		}
		key, ok := trackingEventKey(status, args)
		if !ok {
			t.Fatal("tracking event key was not generated")
		}
		var ttl int64
		if err := client.Do(ctx, radix.Cmd(&ttl, "TTL", key)); err != nil {
			t.Fatal(err)
		}
		if ttl <= 0 {
			t.Fatalf("tracking replay TTL = %d, want positive", ttl)
		}
	}
	if published != 2 {
		t.Fatalf("published = %d, want one impression and one click", published)
	}
	var data []byte
	if err := client.Do(ctx, radix.Cmd(&data, "HGET", match.HashNameBothCap("user"), "0")); err != nil {
		t.Fatal(err)
	}
	bothcap, err := match.UnpackBothCap(data)
	if err != nil {
		t.Fatal(err)
	}
	if bothcap.Imp.Total != 1 || bothcap.Cli.Total != 1 {
		t.Fatalf("cap totals = imp %d click %d, want 1/1", bothcap.Imp.Total, bothcap.Cli.Total)
	}
}

func TestServeStatusRetriesPublishWithoutRepeatingCapMutation(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	capValue, err := (match.Cap{CapNumber: 2, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	args := url.Values{
		"auction_id":       []string{"auction-retry"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp-retry"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
		"cap":              []string{capValue},
	}
	addTrackingSignature("test-secret", "/imp", args)
	published := 0
	controller := &Controller{
		C: &Config{
			TrackingSecret:              "test-secret",
			TrackingSignatureTTLSeconds: 3600,
			CapStateTTLSeconds:          7200,
		},
		Redis: client,
		publishWinLossFunc: func([]byte) error {
			published++
			if published == 1 {
				return errors.New("publish failed")
			}
			return nil
		},
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err == nil {
		t.Fatal("first publish unexpectedly succeeded")
	}
	key, ok := trackingEventKey(StatusTrackImp, args)
	if !ok {
		t.Fatal("tracking event key was not generated")
	}
	if server.Exists(key) {
		t.Fatal("failed event retained its processing claim")
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatalf("completed duplicate failed: %v", err)
	}
	if published != 2 {
		t.Fatalf("publish attempts = %d, want failed attempt plus one retry", published)
	}
	if got, err := server.Get(key); err != nil || got != "done" {
		t.Fatalf("completed replay key = %q, err %v; want done", got, err)
	}
	var data []byte
	if err := client.Do(ctx, radix.Cmd(&data, "HGET", match.HashNameBothCap("user"), "0")); err != nil {
		t.Fatal(err)
	}
	bothcap, err := match.UnpackBothCap(data)
	if err != nil {
		t.Fatal(err)
	}
	if bothcap.Imp.Total != 1 {
		t.Fatalf("impression cap total = %d, want 1 across publish retry", bothcap.Imp.Total)
	}
}

func TestServeStatusSuppressesConcurrentProcessingClaim(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 2}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	args := url.Values{
		"auction_id":       []string{"auction-concurrent"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp-concurrent"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
	}
	addTrackingSignature("test-secret", "/imp", args)
	started := make(chan struct{})
	release := make(chan struct{})
	var published atomic.Int32
	controller := &Controller{
		C:     &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600},
		Redis: client,
		publishWinLossFunc: func([]byte) error {
			if published.Add(1) == 1 {
				close(started)
			}
			<-release
			return nil
		},
	}
	firstErr := make(chan error, 1)
	go func() {
		firstErr <- controller.serveStatus(ctx, StatusTrackImp, time.Now(), args)
	}()
	<-started
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatalf("concurrent duplicate failed: %v", err)
	}
	close(release)
	if err := <-firstErr; err != nil {
		t.Fatalf("claim owner failed: %v", err)
	}
	if got := published.Load(); got != 1 {
		t.Fatalf("published = %d, want 1", got)
	}
}

func TestServeStatusReplayCheckFailsOpen(t *testing.T) {
	args := url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"0000000000000001user"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
	}
	addTrackingSignature("test-secret", "/imp", args)
	published := 0
	controller := &Controller{
		C: &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600},
		trackingEventOnce: func(context.Context, Status, url.Values, time.Duration) (bool, error) {
			return false, fmt.Errorf("redis unavailable")
		},
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	if err := controller.serveStatus(context.Background(), StatusTrackImp, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published = %d, want fail-open event", published)
	}
}

func TestServeStatusRejectsExpiredSignatureBeforeRedis(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	redis := &interceptRedisClient{Client: client}
	defer redis.Close()
	controller := &Controller{
		C:     &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 60},
		Redis: redis,
		publishWinLossFunc: func([]byte) error {
			return nil
		},
	}
	expired := trackingTestArgs("expired", false)
	expired.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Add(-2*time.Minute).Unix(), 10))
	expired.Set(trackingSignatureParam, signTrackingValues("test-secret", "/imp", expired))
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), expired); err == nil {
		t.Fatal("expired signature unexpectedly succeeded")
	}
	if got := redis.calls.Load(); got != 0 {
		t.Fatalf("expired signature performed %d Redis calls, want 0", got)
	}

	valid := trackingTestArgs("valid", false)
	addTrackingSignature("test-secret", "/imp", valid)
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), valid); err != nil {
		t.Fatalf("valid callback after expired request failed: %v", err)
	}
}

func TestServeStatusCanceledRequestDoesNotPoisonSingleRedisConnection(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	redis := &interceptRedisClient{
		Client:       client,
		firstStarted: make(chan struct{}),
		firstRelease: make(chan struct{}),
	}
	defer redis.Close()
	capValue, err := (match.Cap{CapNumber: 5, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	controller := &Controller{
		C: &Config{
			TrackingSecret:              "test-secret",
			TrackingSignatureTTLSeconds: 3600,
			CapStateTTLSeconds:          7200,
		},
		Redis: redis,
		publishWinLossFunc: func([]byte) error {
			return nil
		},
	}
	first := trackingTestArgs("canceled-first", true)
	first.Set("cap", capValue)
	addTrackingSignature("test-secret", "/imp", first)
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		firstResult <- controller.serveStatus(firstCtx, StatusTrackImp, time.Now(), first)
	}()
	<-redis.firstStarted

	second := trackingTestArgs("waiting-second", true)
	second.Set("cap", capValue)
	addTrackingSignature("test-secret", "/imp", second)
	secondResult := make(chan error, 1)
	go func() {
		secondResult <- controller.serveStatus(ctx, StatusTrackImp, time.Now(), second)
	}()
	cancelFirst()
	close(redis.firstRelease)
	if err := <-firstResult; err != nil {
		t.Fatalf("canceled request's detached Redis work failed: %v", err)
	}
	if err := <-secondResult; err != nil {
		t.Fatalf("later callback failed: %v", err)
	}
	var pong string
	if err := client.Do(ctx, radix.Cmd(&pong, "PING")); err != nil {
		t.Fatalf("connection remained watched or transactional: %v", err)
	}
	if pong != "PONG" {
		t.Fatalf("PING = %q, want PONG", pong)
	}
}

func TestServeStatusCapFailurePublishesAndFinalizesClaim(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	redis := &interceptRedisClient{
		Client: client,
		failAt: map[int32]error{2: errors.New("cap Redis unavailable")},
	}
	defer redis.Close()
	capValue, err := (match.Cap{CapNumber: 2, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	args := trackingTestArgs("cap-failure", true)
	args.Set("cap", capValue)
	addTrackingSignature("test-secret", "/imp", args)
	published := 0
	metricBefore := metricTrackingCapUpdateFailOpen.Value()
	controller := &Controller{
		C: &Config{
			TrackingSecret:              "test-secret",
			TrackingSignatureTTLSeconds: 3600,
			CapStateTTLSeconds:          7200,
		},
		Redis: redis,
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/imp?"+args.Encode(), nil)
		rr := httptest.NewRecorder()
		controller.ServeWinLoss(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("callback %d status = %d body=%s, want 204", i+1, rr.Code, rr.Body.String())
		}
	}
	if published != 1 {
		t.Fatalf("published = %d, want successful event suppressed after finalization", published)
	}
	if got := metricTrackingCapUpdateFailOpen.Value() - metricBefore; got != 1 {
		t.Fatalf("cap fail-open metric delta = %d, want 1", got)
	}
	key, _ := trackingEventKey(StatusTrackImp, args)
	if got, err := server.Get(key); err != nil || got != "done" {
		t.Fatalf("completed replay key = %q, err %v; want done", got, err)
	}
}

func TestServeStatusClaimFailureStillAttemptsIdempotentCapUpdate(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	redis := &interceptRedisClient{
		Client: client,
		failAt: map[int32]error{1: errors.New("claim Redis unavailable")},
	}
	defer redis.Close()
	capValue, err := (match.Cap{CapNumber: 2, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	args := trackingTestArgs("claim-failure", true)
	args.Set("cap", capValue)
	addTrackingSignature("test-secret", "/imp", args)
	published := 0
	controller := &Controller{
		C: &Config{
			TrackingSecret:              "test-secret",
			TrackingSignatureTTLSeconds: 3600,
			CapStateTTLSeconds:          7200,
		},
		Redis: redis,
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published = %d, want 1", published)
	}
	var data []byte
	if err := client.Do(ctx, radix.Cmd(&data, "HGET", match.HashNameBothCap("user"), "0")); err != nil {
		t.Fatal(err)
	}
	capState, err := match.UnpackBothCap(data)
	if err != nil {
		t.Fatal(err)
	}
	if capState.Imp.Total != 1 {
		t.Fatalf("impression cap total = %d, want 1", capState.Imp.Total)
	}
}

func TestServeStatusUnkeyedEventPublishesWithoutCapMutation(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	redis := &interceptRedisClient{Client: client}
	defer redis.Close()
	capValue, err := (match.Cap{CapNumber: 2, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	args := trackingTestArgs("unkeyed", true)
	args.Del("auction_imp_id")
	args.Set("cap", capValue)
	addTrackingSignature("test-secret", "/imp", args)
	published := 0
	controller := &Controller{
		C:     &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600},
		Redis: redis,
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published = %d, want 1", published)
	}
	if got := redis.calls.Load(); got != 0 {
		t.Fatalf("unkeyed event performed %d Redis calls, want 0", got)
	}
}

func TestServeStatusFutureSkewReplayMarkerCoversFullValidity(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	capValue, err := (match.Cap{CapNumber: 2, CapPeriod: 60}).PackString()
	if err != nil {
		t.Fatal(err)
	}
	args := trackingTestArgs("future-skew", true)
	args.Set("cap", capValue)
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Add(4*time.Minute).Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("test-secret", "/imp", args))
	published := 0
	controller := &Controller{
		C: &Config{
			TrackingSecret:              "test-secret",
			TrackingSignatureTTLSeconds: 3600,
			CapStateTTLSeconds:          7200,
		},
		Redis: client,
		publishWinLossFunc: func([]byte) error {
			published++
			return nil
		},
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	key, _ := trackingEventKey(StatusTrackImp, args)
	if ttl := server.TTL(key); ttl < 63*time.Minute {
		t.Fatalf("future-skew replay TTL = %s, want full remaining validity", ttl)
	}
	if ttl := server.TTL(key + ":cap"); ttl < 63*time.Minute {
		t.Fatalf("future-skew cap marker TTL = %s, want full remaining validity", ttl)
	}
	server.FastForward(61 * time.Minute)
	if !server.Exists(key + ":cap") {
		t.Fatal("future-skew cap marker expired before signature validity")
	}
	if err := controller.serveStatus(ctx, StatusTrackImp, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published = %d after 61 minutes, want duplicate suppressed", published)
	}
}

func TestServeClickPublishesNormallyWhenClaimRedisFails(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		t.Run(fmt.Sprintf("redirect=%t", redirect), func(t *testing.T) {
			server := miniredis.RunT(t)
			ctx := context.Background()
			client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
			if err != nil {
				t.Fatal(err)
			}
			redis := &interceptRedisClient{Client: client, failAt: map[int32]error{1: errors.New("claim Redis unavailable")}}
			defer redis.Close()
			args := trackingTestArgs("click-fail-open", false)
			if redirect {
				demand, _ := (match.Demand{AdvID: 1}).PackString()
				supply, _ := (match.RPub{PubID: 1}).PackString()
				args.Set("demand", demand)
				args.Set("supply", supply)
				args.Set("redirect", "https://advertiser.example/landing")
			}
			addTrackingSignature("test-secret", "/clk", args)
			published := 0
			controller := &Controller{
				C:     &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600},
				Redis: redis,
				publishWinLossFunc: func([]byte) error {
					published++
					return nil
				},
			}
			req := httptest.NewRequest(http.MethodGet, "/clk?"+args.Encode(), nil)
			rr := httptest.NewRecorder()
			controller.ServeWinLoss(rr, req)
			wantStatus := http.StatusNoContent
			if redirect {
				wantStatus = http.StatusFound
			}
			if rr.Code != wantStatus {
				t.Fatalf("status = %d body=%s, want %d", rr.Code, rr.Body.String(), wantStatus)
			}
			if published != 1 {
				t.Fatalf("published = %d, want 1", published)
			}
		})
	}
}

func TestServeWinLossRequiresSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/win?auction_bid_id=bid", nil)
	rr := httptest.NewRecorder()
	(&Controller{C: &Config{TrackingSecret: "test-secret"}}).ServeWinLoss(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want bad request", rr.Code)
	}
}

func TestServeWinLossPublishesWinOnce(t *testing.T) {
	winloss := NewWinLoss(
		StatusBid,
		time.Now(),
		match.RPub{PubID: 1, SiteID: 2, SlotID: 3},
		match.RAdv{Demand: match.Demand{AdvID: 4, CampaignID: 5, ItemID: 6, CreativeID: 7}, Cost: 1.25, CostType: 2},
		nil,
		"5",
		"auction",
		"bid",
		"imp",
		"7",
		"https://dsp.example",
	).WithTrackingSecret("test-secret")
	u, err := url.Parse(winloss.NURL())
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("auction_id", "auction")
	q.Set("auction_bid_id", "bid")
	q.Set("auction_imp_id", "imp")
	q.Set("auction_price", "1.25")
	q.Set("auction_currency", "USD")
	u.RawQuery = q.Encode()

	seen := false
	published := 0
	controller := &Controller{
		C: &Config{TrackingSecret: "test-secret"},
		trackingNotifyOnce: func(_ context.Context, _ Status, _ string, _ time.Duration) (bool, error) {
			if seen {
				return false, nil
			}
			seen = true
			return true, nil
		},
		publishWinLossFunc: func(_ []byte) error {
			published++
			return nil
		},
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
		rr := httptest.NewRecorder()
		controller.ServeWinLoss(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
		}
	}
	if published != 1 {
		t.Fatalf("published = %d, want one", published)
	}
}

func TestServeWinLossMarkerUsesRemainingSignatureValidity(t *testing.T) {
	signedAt := time.Now().Add(-30 * time.Minute).Truncate(time.Second)
	args := url.Values{
		"auction_id":       []string{"auction"},
		"auction_bid_id":   []string{"bid"},
		"auction_imp_id":   []string{"imp"},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
	}
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(signedAt.Unix(), 10))
	args.Set(trackingSignatureParam, signTrackingValues("test-secret", "/win", args))
	var markerTTL time.Duration
	controller := &Controller{
		C: &Config{TrackingSecret: "test-secret", TrackingSignatureTTLSeconds: 3600},
		trackingNotifyOnce: func(_ context.Context, _ Status, _ string, ttl time.Duration) (bool, error) {
			markerTTL = ttl
			return true, nil
		},
		publishWinLossFunc: func([]byte) error { return nil },
	}
	if err := controller.serveStatus(context.Background(), StatusWin, time.Now(), args); err != nil {
		t.Fatal(err)
	}
	if markerTTL < 29*time.Minute || markerTTL > 31*time.Minute {
		t.Fatalf("marker TTL = %s, want remaining signature validity", markerTTL)
	}
}

func TestAuditPublisherDropsWhenQueueFull(t *testing.T) {
	publisher := newAuditPublisher(nil, 1)
	defer publisher.Close()

	publisher.Enqueue(SUBJECTRequest, []byte("one"))
	publisher.Enqueue(SUBJECTResponse, []byte("two"))
	if got := publisher.Dropped(); got != 1 {
		t.Fatalf("Dropped = %d, want 1", got)
	}
}

func TestPublishBidAuditsInitializesPublisherOnce(t *testing.T) {
	created := 0
	controller := &Controller{
		Nc: &nats.Conn{},
		auditFactory: func(*nats.Conn, int) *auditPublisher {
			created++
			return newAuditPublisher(nil, defaultAuditQueueSize)
		},
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := controller.publishBidAuditsFor(auditSourceADX, nil, nil, nil); err != nil {
				t.Errorf("publishBidAuditsFor: %v", err)
			}
		}()
	}
	wg.Wait()
	if controller.audit == nil {
		t.Fatal("audit publisher was not initialized")
	}
	if created != 1 {
		t.Fatalf("audit publishers created = %d, want 1", created)
	}
	controller.audit.Close()
}

func TestPublishBidAuditsDoesNotInitializePublisherAfterClose(t *testing.T) {
	created := 0
	controller := &Controller{
		Nc: &nats.Conn{},
		auditFactory: func(*nats.Conn, int) *auditPublisher {
			created++
			return newAuditPublisher(nil, defaultAuditQueueSize)
		},
	}
	controller.Close()
	if err := controller.publishBidAuditsFor(auditSourceADX, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if created != 0 {
		t.Fatalf("audit publishers created after close = %d, want 0", created)
	}
	if controller.audit != nil {
		t.Fatal("audit publisher was initialized after close")
	}
}

func zeroRAdv() match.RAdv {
	return match.RAdv{}
}

func trackingTestArgs(id string, withUser bool) url.Values {
	bidID := "0000000000000001"
	if withUser {
		bidID += "user"
	}
	return url.Values{
		"auction_id":       []string{"auction-" + id},
		"auction_bid_id":   []string{bidID},
		"auction_imp_id":   []string{"imp-" + id},
		"auction_price":    []string{"1.0"},
		"auction_currency": []string{"USD"},
	}
}

type interceptRedisClient struct {
	radix.Client
	calls        atomic.Int32
	failAt       map[int32]error
	firstStarted chan struct{}
	firstRelease chan struct{}
}

func (c *interceptRedisClient) Addr() net.Addr {
	return c.Client.Addr()
}

func (c *interceptRedisClient) Do(ctx context.Context, action radix.Action) error {
	call := c.calls.Add(1)
	if err := c.failAt[call]; err != nil {
		return err
	}
	if call == 1 && c.firstStarted != nil {
		action = blockingRedisAction{
			Action:  action,
			started: c.firstStarted,
			release: c.firstRelease,
		}
	}
	return c.Client.Do(ctx, action)
}

func (c *interceptRedisClient) Close() error {
	return c.Client.Close()
}

type blockingRedisAction struct {
	radix.Action
	started chan struct{}
	release chan struct{}
}

func (a blockingRedisAction) Properties() radix.ActionProperties {
	return a.Action.Properties()
}

func (a blockingRedisAction) Perform(ctx context.Context, conn radix.Conn) error {
	close(a.started)
	<-a.release
	return a.Action.Perform(ctx, conn)
}

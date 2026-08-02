package dsp

import (
	"context"
	"errors"
	"math"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
)

func newDeliveryTestController(t *testing.T) (*Controller, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 8}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return &Controller{
		C: &Config{
			DeliveryCacheMaxAgeSeconds: 900,
			DeliveryReservationSeconds: 86700,
			DeliveryStateTTLSeconds:    172800,
		},
		Redis: client,
	}, server
}

func deliveryTestRAdv(now time.Time) match.RAdv {
	delivery := match.Delivery{
		GeneratedAtUnix: now.Unix(),
		ItemTotal: match.DeliveryBalance{
			ID:         10,
			LimitSpend: 10,
			LimitImp:   10,
		},
	}
	if err := delivery.SetTimezone("UTC"); err != nil {
		panic(err)
	}
	return match.RAdv{
		Demand:   match.Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4},
		Weight:   1,
		CostType: 2,
		Cost:     1,
		Delivery: delivery,
	}
}

func TestAuctionCPMToSpendUsesOneImpressionUnit(t *testing.T) {
	if got := auctionCPMToSpend(2.5); math.Abs(float64(got-0.0025)) > 0.0000001 {
		t.Fatalf("auctionCPMToSpend(2.5) = %.8f, want 0.0025", got)
	}
}

func TestDeliveryReservationBoundsConcurrentSpendAndImpressions(t *testing.T) {
	controller, _ := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	var successes atomic.Int64
	var rejected atomic.Int64
	var tokensMu sync.Mutex
	var tokens []string
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := controller.reserveDelivery(context.Background(), block, now, 1)
			switch {
			case err == nil:
				successes.Add(1)
				tokensMu.Lock()
				tokens = append(tokens, token)
				tokensMu.Unlock()
			case errors.Is(err, errDeliveryLimit):
				rejected.Add(1)
			default:
				t.Errorf("reserve delivery: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 10 || rejected.Load() != 40 {
		t.Fatalf("reservations success/rejected = %d/%d, want 10/40", successes.Load(), rejected.Load())
	}
	if err := controller.releaseDeliveryReservation(context.Background(), tokens[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.reserveDelivery(context.Background(), block, now, 1); err != nil {
		t.Fatalf("reservation after release: %v", err)
	}
}

func TestDeliveryFinalizationKeepsSpendReservedAndClickIsIdempotent(t *testing.T) {
	controller, _ := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal.LimitSpend = 1
	block.Delivery.ItemTotal.LimitImp = 1
	block.Delivery.ItemTotal.LimitClick = 1
	token, err := controller.reserveDelivery(context.Background(), block, now, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.finalizeDeliveryReservation(context.Background(), token, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := controller.releaseDeliveryReservation(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.reserveDelivery(context.Background(), block, now, 1); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("finalized spend reopened: %v", err)
	}
	if err := controller.recordDeliveryClick(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if err := controller.recordDeliveryClick(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	var clicks int
	if err := controller.Redis.Do(context.Background(), radix.Cmd(&clicks, "HGET", deliveryTotalKey(10), "used_click")); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatalf("used_click = %d, want 1", clicks)
	}
}

func TestDeliveryReservationUsesAllFourBudgetScopes(t *testing.T) {
	controller, server := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.CampaignTotal = match.DeliveryBalance{ID: 1, LimitImp: 4, CurrentImp: 3}
	block.Delivery.CampaignDaily = match.DeliveryBalance{ID: 2, LimitImp: 3, CurrentImp: 2}
	block.Delivery.ItemTotal = match.DeliveryBalance{ID: 3, LimitImp: 2, CurrentImp: 1}
	block.Delivery.ItemDaily = match.DeliveryBalance{ID: 4, LimitImp: 1}
	if _, err := controller.reserveDelivery(context.Background(), block, now, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.reserveDelivery(context.Background(), block, now, 1); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("item daily limit did not reject second reservation: %v", err)
	}
	for _, key := range []string{
		deliveryTotalKey(1),
		deliveryDailyKey("2026-08-01", 2),
		deliveryTotalKey(3),
		deliveryDailyKey("2026-08-01", 4),
	} {
		if !controllerRedisKeyExists(t, controller, key) {
			t.Fatalf("budget state key %s was not written", key)
		}
	}
	if ttl := server.TTL(deliveryTotalKey(1)); ttl != 0 {
		t.Fatalf("total budget TTL = %s, want persistent", ttl)
	}
	if ttl := server.TTL(deliveryDailyKey("2026-08-01", 2)); ttl < controller.deliveryStateTTL() || ttl > controller.deliveryStateTTL()+24*time.Hour {
		t.Fatalf("daily budget TTL = %s, want UTC day remainder plus state grace", ttl)
	}
}

func TestDeliveryReservationCoversAcceptedCallbackLifetime(t *testing.T) {
	controller, server := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal.LimitClick = 2
	token, err := controller.reserveDelivery(context.Background(), block, now, 1)
	if err != nil {
		t.Fatal(err)
	}
	server.FastForward(defaultTrackingSignatureTTL)
	if err := controller.recordDeliveryClick(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	var clicks int
	if err := controller.Redis.Do(context.Background(), radix.Cmd(&clicks, "HGET", deliveryTotalKey(10), "used_click")); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatalf("callback after 24 hours did not update delivery state: clicks=%d", clicks)
	}
}

func TestDeliveryEvenPacingIsDeterministicAndNeverExceedsHardLimit(t *testing.T) {
	controller, _ := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal = match.DeliveryBalance{}
	block.Delivery.Item.Pacing = match.DeliveryPacingEven
	block.Delivery.ItemDaily = match.DeliveryBalance{ID: 11, LimitImp: 24}
	for i := 0; i < 12; i++ {
		if _, err := controller.reserveDelivery(context.Background(), block, now, 0); err != nil {
			t.Fatalf("paced reservation %d: %v", i+1, err)
		}
	}
	if _, err := controller.reserveDelivery(context.Background(), block, now, 0); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("thirteenth noon reservation = %v, want pacing rejection", err)
	}
	later := time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)
	if _, err := controller.reserveDelivery(context.Background(), block, later, 0); err != nil {
		t.Fatalf("reservation after pacing allowance advances: %v", err)
	}
}

func TestExpiredReservationDoesNotSilentlyReopenTotalBudget(t *testing.T) {
	controller, server := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal.LimitImp = 1
	block.Delivery.ItemTotal.LimitSpend = 0
	if _, err := controller.reserveDelivery(context.Background(), block, now, 0); err != nil {
		t.Fatal(err)
	}
	server.FastForward(controller.deliveryReservationTTL() + time.Second)
	if _, err := controller.reserveDelivery(context.Background(), block, now, 0); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("expired callback reservation reopened budget: %v", err)
	}
	server.FastForward(controller.deliveryStateTTL() + 24*time.Hour)
	if _, err := controller.reserveDelivery(context.Background(), block, now, 0); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("total budget silently reopened after state grace window: %v", err)
	}
}

func TestReservationReleaseNeverDropsBelowNewerLedgerFloor(t *testing.T) {
	controller, server := newDeliveryTestController(t)
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal = match.DeliveryBalance{ID: 41, LimitImp: 20, CurrentImp: 10}
	first, err := controller.reserveDelivery(context.Background(), block, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	newer := block
	newer.Delivery.ItemTotal.CurrentImp = 12
	second, err := controller.reserveDelivery(context.Background(), newer, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.releaseDeliveryReservation(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := controller.releaseDeliveryReservation(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	key := deliveryTotalKey(41)
	if used, floor := server.HGet(key, "used_imp"), server.HGet(key, "floor_imp"); used != "12" || floor != "12" {
		t.Fatalf("used/floor impressions after releases = %q/%q, want 12/12", used, floor)
	}
	authoritative := block
	authoritative.Delivery.ItemTotal.CurrentImp = 13
	authoritative.Delivery.ItemTotal.LimitImp = 13
	if _, err := controller.reserveDelivery(context.Background(), authoritative, now, 0); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("authoritatively exhausted snapshot = %v, want limit rejection", err)
	}
	if used, floor := server.HGet(key, "used_imp"), server.HGet(key, "floor_imp"); used != "13" || floor != "13" {
		t.Fatalf("rejected snapshot used/floor = %q/%q, want 13/13", used, floor)
	}
	stale := block
	stale.Delivery.ItemTotal.LimitImp = 14
	if _, err := controller.reserveDelivery(context.Background(), stale, now, 0); err != nil {
		t.Fatalf("fourteenth impression should still fit: %v", err)
	}
	if _, err := controller.reserveDelivery(context.Background(), stale, now, 0); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("stale cache reopened authoritative floor: %v", err)
	}
}

func TestBudgetedDeliveryFailsClosedWithoutRedis(t *testing.T) {
	now := time.Now()
	controller := &Controller{C: &Config{}}
	if _, err := controller.reserveDelivery(context.Background(), deliveryTestRAdv(now), now, 1); err == nil {
		t.Fatal("budgeted delivery should fail closed without Redis")
	}
	unlimited := deliveryTestRAdv(now)
	unlimited.Delivery.ItemTotal = match.DeliveryBalance{}
	if token, err := controller.reserveDelivery(context.Background(), unlimited, now, 1); err != nil || token != "" {
		t.Fatalf("unlimited delivery = %q, %v", token, err)
	}
}

func TestBudgetedDeliveryRejectsInvalidCost(t *testing.T) {
	controller, _ := newDeliveryTestController(t)
	now := time.Now()
	for _, cost := range []float32{-1, float32(math.NaN()), float32(math.Inf(1))} {
		if _, err := controller.reserveDelivery(context.Background(), deliveryTestRAdv(now), now, cost); err == nil {
			t.Fatalf("reserveDelivery cost %v succeeded", cost)
		}
	}
}

func TestTrackingCallbacksFinalizeClickAndReleaseDeliveryReservation(t *testing.T) {
	controller, server := newDeliveryTestController(t)
	controller.C.TrackingSecret = "test-secret"
	controller.C.TrackingSignatureTTLSeconds = int(defaultTrackingSignatureTTL.Seconds())
	controller.publishWinLossFunc = func([]byte) error { return nil }
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal.LimitImp = 1
	block.Delivery.ItemTotal.LimitSpend = 0
	block.Delivery.ItemTotal.LimitClick = 1
	token, err := controller.reserveDelivery(context.Background(), block, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	imp := signedDeliveryTrackingArgs("finalize", "/imp", token)
	if err := controller.serveStatus(context.Background(), StatusTrackImp, now, imp); err != nil {
		t.Fatal(err)
	}
	if status := server.HGet(deliveryReservationKey(token), "status"); status != "final" {
		t.Fatalf("reservation status = %q, want final", status)
	}
	click := signedDeliveryTrackingArgs("click", "/clk", token)
	if err := controller.serveStatus(context.Background(), StatusTrackClk, now, click); err != nil {
		t.Fatal(err)
	}
	if clicks := server.HGet(deliveryTotalKey(10), "used_click"); clicks != "1" {
		t.Fatalf("used_click = %q, want 1", clicks)
	}
	if _, err := controller.reserveDelivery(context.Background(), block, now, 0); !errors.Is(err, errDeliveryLimit) {
		t.Fatalf("finalized impression reopened: %v", err)
	}

	lossBlock := deliveryTestRAdv(now)
	lossBlock.Delivery.ItemTotal = match.DeliveryBalance{ID: 20, LimitImp: 1}
	lossToken, err := controller.reserveDelivery(context.Background(), lossBlock, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	loss := signedDeliveryTrackingArgs("loss", "/loss", lossToken)
	if err := controller.serveStatus(context.Background(), StatusLoss, now, loss); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.reserveDelivery(context.Background(), lossBlock, now, 0); err != nil {
		t.Fatalf("loss did not release reservation: %v", err)
	}
}

func TestTrackingPublishRetryDoesNotDoubleReserveDelivery(t *testing.T) {
	controller, server := newDeliveryTestController(t)
	controller.C.TrackingSecret = "test-secret"
	controller.C.TrackingSignatureTTLSeconds = int(defaultTrackingSignatureTTL.Seconds())
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	block := deliveryTestRAdv(now)
	block.Delivery.ItemTotal = match.DeliveryBalance{ID: 30, LimitImp: 2}
	token, err := controller.reserveDelivery(context.Background(), block, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	args := signedDeliveryTrackingArgs("publish-retry", "/imp", token)
	attempt := 0
	controller.publishWinLossFunc = func([]byte) error {
		attempt++
		if attempt == 1 {
			return errors.New("publish failed")
		}
		return nil
	}
	if err := controller.serveStatus(context.Background(), StatusTrackImp, now, args); err == nil {
		t.Fatal("first publication unexpectedly succeeded")
	}
	if status := server.HGet(deliveryReservationKey(token), "status"); status != "active" {
		t.Fatalf("failed publication reservation status = %q, want active", status)
	}
	if err := controller.serveStatus(context.Background(), StatusTrackImp, now, args); err != nil {
		t.Fatal(err)
	}
	if imps := server.HGet(deliveryTotalKey(30), "used_imp"); imps != "1" {
		t.Fatalf("used_imp after retry = %q, want one reservation", imps)
	}
}

func signedDeliveryTrackingArgs(id, path, token string) url.Values {
	args := trackingTestArgs(id, true)
	args.Set("delivery_reservation", token)
	addTrackingSignature("test-secret", path, args)
	return args
}

func controllerRedisKeyExists(t *testing.T, controller *Controller, key string) bool {
	t.Helper()
	var exists int
	if err := controller.Redis.Do(context.Background(), radix.Cmd(&exists, "EXISTS", key)); err != nil {
		t.Fatal(err)
	}
	return exists == 1
}

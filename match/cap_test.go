package match

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func TestCapRejectsNumberWithoutPeriod(t *testing.T) {
	for _, cap := range []Cap{
		{CapNumber: 1},
		{ClickNumber: 1},
	} {
		if err := cap.Validate(); err == nil || !strings.Contains(err.Error(), "requires a positive period") {
			t.Fatalf("Validate(%+v) = %v", cap, err)
		}
		if _, err := cap.Pack(); err == nil {
			t.Fatalf("Pack(%+v) succeeded", cap)
		}
		if cap.CanServe(time.Now(), BothCap{}) {
			t.Fatalf("invalid cap %+v served", cap)
		}
		if _, _, err := (RAdvs{{Demand: Demand{ItemID: 9}, Cap: cap}}).FilterByCaps(context.Background(), nil, time.Now(), "user"); err == nil {
			t.Fatalf("invalid cached cap %+v passed runtime filtering", cap)
		}
	}
}

func TestCapAllowsStandaloneThrottleAndCompleteNumberedWindows(t *testing.T) {
	for _, cap := range []Cap{
		{CapThrottle: 10},
		{CapNumber: 2, CapPeriod: 60},
		{ClickNumber: 2, ClickPeriod: 60},
	} {
		if err := cap.Validate(); err != nil {
			t.Fatalf("Validate(%+v) = %v", cap, err)
		}
	}
}

func TestStandaloneThrottleLoadsStateAndRejectsRapidRepeat(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	state := NewBothCap(now)
	state.Imp.Refresh(now)
	if err := BothCapsToRedis(ctx, client, "person", map[uint32]BothCap{9: state}); err != nil {
		t.Fatal(err)
	}
	radvs := RAdvs{{Demand: Demand{ItemID: 9}, Cap: Cap{CapThrottle: 10}}}
	ids := radvs.capItemIDs()
	if len(ids) != 1 || ids[0] != "9" {
		t.Fatalf("throttle-only cap IDs = %v, want [9]", ids)
	}
	filtered, loaded, err := radvs.FilterByCaps(ctx, client, now.Add(5*time.Minute), "person")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 0 {
		t.Fatalf("filtered demand = %v, want throttle-only demand rejected", filtered)
	}
	if loaded != nil {
		t.Fatalf("fully filtered result exposed cap state %v, want nil", loaded)
	}
	if server.HGet(HashNameBothCap("person"), "9") == "" {
		t.Fatal("throttle-only cap state was incorrectly expired or removed")
	}
}

func TestExpiredImpressionWindowPreservesActiveClickState(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	now := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour)
	state := NewBothCap(start)
	state.Imp.Refresh(start)
	state.Cli.Refresh(start)
	if err := BothCapsToRedis(ctx, client, "person", map[uint32]BothCap{9: state}); err != nil {
		t.Fatal(err)
	}
	radvs := RAdvs{{Demand: Demand{ItemID: 9}, Cap: Cap{
		CapNumber: 1, CapPeriod: 60,
		ClickNumber: 1, ClickPeriod: 180,
	}}}
	filtered, _, err := radvs.FilterByCaps(ctx, client, now, "person")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 0 {
		t.Fatalf("filtered demand = %v, want active click cap to reject it", filtered)
	}
	if server.HGet(HashNameBothCap("person"), "9") == "" {
		t.Fatal("expired impression window deleted active sibling cap state")
	}
}

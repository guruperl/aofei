package match

import (
	"context"
	"testing"
	"time"

	"github.com/mediocregopher/radix/v4"
)

// TestFcap tests the Fcap struct.
func TestFcap(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	fcap := NewFcap(now)
	if fcap.Total != 0 {
		t.Errorf("Total should be 0, but got %d", fcap.Total)
	}
	if fcap.StartYM != 1 {
		t.Errorf("StartYM should be 1, but got %d", fcap.StartYM)
	}
	if fcap.StartDHM != 2048 {
		t.Errorf("StartDHM should be 2048, but got %d", fcap.StartDHM)
	}
	if fcap.Last != 0 {
		t.Errorf("Last should be 0, but got %d", fcap.Last)
	}
}

func TestBothCapRefreshClickPeriod(t *testing.T) {
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	when := start.Add(11 * time.Minute)
	bothcap := NewBothCap(start)
	bothcap.Cli.Refresh(start)

	bothcap.Refresh(when, RAdv{Cap: Cap{CapPeriod: 60, ClickPeriod: 10}}, false, true)

	if bothcap.Cli.Total != 1 {
		t.Fatalf("Cli.Total = %d, want reset count 1", bothcap.Cli.Total)
	}
	if !bothcap.Cli.GetStart().Equal(when) {
		t.Fatalf("Cli.GetStart() = %s, want %s", bothcap.Cli.GetStart(), when)
	}
}

func TestBothCapRefreshExpiredImpAndClick(t *testing.T) {
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	when := start.Add(2 * time.Hour)
	bothcap := NewBothCap(start)
	bothcap.Imp.Refresh(start)
	bothcap.Cli.Refresh(start)

	bothcap.Refresh(when, RAdv{Cap: Cap{CapPeriod: 30, ClickPeriod: 10}}, true, true)

	if bothcap.Imp.Total != 1 {
		t.Fatalf("Imp.Total = %d, want reset count 1", bothcap.Imp.Total)
	}
	if bothcap.Cli.Total != 1 {
		t.Fatalf("Cli.Total = %d, want reset count 1", bothcap.Cli.Total)
	}
	if !bothcap.Imp.GetStart().Equal(when) {
		t.Fatalf("Imp.GetStart() = %s, want %s", bothcap.Imp.GetStart(), when)
	}
	if !bothcap.Cli.GetStart().Equal(when) {
		t.Fatalf("Cli.GetStart() = %s, want %s", bothcap.Cli.GetStart(), when)
	}
}

func TestMustRefreshBothCapConcurrent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := (radix.PoolConfig{Size: 4}).New(ctx, "tcp", "127.0.0.1:6379")
	if err != nil {
		t.Skipf("Redis unavailable for concurrent cap test: %v", err)
	}
	defer client.Close()

	pid := "fcap-concurrent-test"
	itemID := uint32(123)
	key := HashNameBothCap(pid)
	if err := client.Do(ctx, radix.Cmd(nil, "DEL", key)); err != nil {
		t.Fatal(err)
	}

	when := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			errs <- MustRefreshBothCap(ctx, client, when, pid, itemID, Cap{CapPeriod: 60}, true, false)
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	var data []byte
	if err := client.Do(ctx, radix.Cmd(&data, "HGET", key, "123")); err != nil {
		t.Fatal(err)
	}
	bothcap, err := UnpackBothCap(data)
	if err != nil {
		t.Fatal(err)
	}
	if bothcap.Imp.Total != 2 {
		t.Fatalf("Imp.Total = %d, want 2", bothcap.Imp.Total)
	}
}

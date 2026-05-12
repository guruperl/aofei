package match

import (
	"testing"
	"time"
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

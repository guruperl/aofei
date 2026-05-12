package dh

import (
	"testing"
	"time"
)

func TestNewDHFromMinutesUsesOpenRTBOffset(t *testing.T) {
	when := time.Date(2026, time.May, 12, 23, 45, 0, 0, time.UTC)

	_, hour, weekday := NewDHFromMinutes(when, 330).dhw()
	if hour != 6 {
		t.Fatalf("UTC+05:30 hour = %d, want 6", hour)
	}
	if weekday != 3 {
		t.Fatalf("UTC+05:30 weekday = %d, want Wednesday=3", weekday)
	}

	_, hour, weekday = NewDHFromMinutes(when, -480).dhw()
	if hour != 16 {
		t.Fatalf("UTC-08:00 hour = %d, want 16", hour)
	}
	if weekday != 2 {
		t.Fatalf("UTC-08:00 weekday = %d, want Tuesday=2", weekday)
	}
}

func TestDHAudienceOffsetEnumStillOverridesVisitorOffset(t *testing.T) {
	when := time.Date(2026, time.May, 12, 23, 45, 0, 0, time.UTC)

	_, hour, _ := NewDHFromMinutes(when, 330).dhw(1)
	if hour != 24 {
		t.Fatalf("stored UTC enum hour = %d, want 24", hour)
	}
}

package match

import (
	"strings"
	"testing"
	"time"
)

func TestDeliveryWindowWeeklySchedule(t *testing.T) {
	var window DeliveryWindow
	schedule := []byte(strings.Repeat("0", DeliveryHoursPerWeek))
	// Monday 09:00 and Sunday 23:00.
	schedule[9] = '1'
	schedule[6*24+23] = '1'
	if err := window.SetWeeklySchedule(string(schedule)); err != nil {
		t.Fatal(err)
	}
	if got := window.WeeklySchedule(); got != string(schedule) {
		t.Fatalf("weekly schedule round trip mismatch")
	}
	if !window.Allows(time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC), time.UTC) {
		t.Fatal("Monday 09:30 should be allowed")
	}
	if window.Allows(time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC), time.UTC) {
		t.Fatal("Monday 10:00 should be denied")
	}
	if !window.Allows(time.Date(2026, 8, 9, 23, 59, 0, 0, time.UTC), time.UTC) {
		t.Fatal("Sunday 23:59 should be allowed")
	}
}

func TestDeliveryEligibilityEnforcesFreshnessWindowsAndLimits(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	delivery := Delivery{
		GeneratedAtUnix: now.Add(-time.Minute).Unix(),
		Campaign: DeliveryWindow{
			StartUnix: now.Add(-time.Hour).Unix(),
			EndUnix:   now.Add(time.Hour).Unix(),
		},
		ItemTotal: DeliveryBalance{ID: 7, LimitImp: 10, CurrentImp: 9},
	}
	if err := delivery.SetTimezone("UTC"); err != nil {
		t.Fatal(err)
	}
	if eligible, reason := delivery.EligibleAt(now, 5*time.Minute); !eligible {
		t.Fatalf("eligible delivery denied: %s", reason)
	}
	delivery.ItemTotal.CurrentImp = 10
	if eligible, _ := delivery.EligibleAt(now, 5*time.Minute); eligible {
		t.Fatal("exhausted delivery should be denied")
	}
	delivery.ItemTotal.CurrentImp = 0
	delivery.GeneratedAtUnix = now.Add(-6 * time.Minute).Unix()
	if eligible, reason := delivery.EligibleAt(now, 5*time.Minute); eligible || reason != "stale delivery cache" {
		t.Fatalf("stale delivery = %v, %q", eligible, reason)
	}
	delivery.GeneratedAtUnix = now.Unix()
	delivery.Campaign.EndUnix = now.Add(-time.Second).Unix()
	if eligible, reason := delivery.EligibleAt(now, 5*time.Minute); eligible || reason != "campaign schedule" {
		t.Fatalf("ended delivery = %v, %q", eligible, reason)
	}
	delivery.Campaign.EndUnix = 0
	delivery.GeneratedAtUnix = now.Add(6 * time.Minute).Unix()
	if eligible, reason := delivery.EligibleAt(now, 5*time.Minute); eligible || reason != "delivery cache timestamp is in the future" {
		t.Fatalf("future delivery cache = %v, %q", eligible, reason)
	}
}

func TestDeliveryPacingFractionsUseUTCResetAndEffectiveWindow(t *testing.T) {
	when := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	var delivery Delivery
	if err := delivery.SetTimezone("America/New_York"); err != nil {
		t.Fatal(err)
	}
	daily, err := delivery.DailyPacingFraction(when)
	if err != nil {
		t.Fatal(err)
	}
	if daily != 0.5 {
		t.Fatalf("daily pacing fraction = %f, want 0.5 at 12:00 UTC", daily)
	}
	delivery.Campaign.StartUnix = when.Add(-time.Hour).Unix()
	delivery.Campaign.EndUnix = when.Add(3 * time.Hour).Unix()
	if total := delivery.TotalPacingFraction(when); total != 0.25 {
		t.Fatalf("total pacing fraction = %f, want 0.25", total)
	}
}

func TestDeliveryRejectsMalformedScheduleAndTimezone(t *testing.T) {
	var window DeliveryWindow
	if err := window.SetWeeklySchedule("1"); err == nil {
		t.Fatal("short schedule should fail")
	}
	if err := window.SetWeeklySchedule(strings.Repeat("x", DeliveryHoursPerWeek)); err == nil {
		t.Fatal("invalid schedule bit should fail")
	}
	var delivery Delivery
	if err := delivery.SetTimezone("Not/AZone"); err == nil {
		t.Fatal("invalid timezone should fail")
	}
}

func TestDeliveryEligibilityRejectsMalformedPolicy(t *testing.T) {
	now := time.Now().UTC()
	delivery := Delivery{GeneratedAtUnix: now.Unix(), Campaign: DeliveryWindow{Pacing: 99}}
	if eligible, reason := delivery.EligibleAt(now, time.Minute); eligible || reason != "invalid delivery pacing" {
		t.Fatalf("invalid pacing = %v, %q", eligible, reason)
	}
	delivery.Campaign.Pacing = DeliveryPacingFast
	delivery.ItemTotal = DeliveryBalance{ID: 7, CurrentSpend: -1}
	if eligible, reason := delivery.EligibleAt(now, time.Minute); eligible || !strings.Contains(reason, "invalid current spend") {
		t.Fatalf("invalid balance = %v, %q", eligible, reason)
	}
}

func TestDeliveryBalanceRejectsInvalidFacts(t *testing.T) {
	for _, balance := range []DeliveryBalance{
		{ID: 1, LimitSpend: -1},
		{ID: 1, CurrentSpend: -1},
		{LimitImp: 1},
	} {
		if err := balance.Validate(); err == nil {
			t.Fatalf("DeliveryBalance.Validate(%#v) succeeded", balance)
		}
	}
}

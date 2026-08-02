package trafficquality

import (
	"strings"
	"testing"
	"time"
)

func TestWindowDerivesReplaySequenceAndRateFixtures(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	window := Window{
		Scope: Scope{Type: ScopePublisher, ID: 7}, WindowKey: "window-1",
		StartedAt: start, EndedAt: start.Add(10 * time.Second),
		Requests: 120, UniqueEvents: 100, Impressions: 10, Clicks: 15,
		Actions: 17, InvalidOriginApp: 3, MalformedIdentity: 4,
		AutomationMatches: 5, PartnerPolicyViolations: 6,
		Evidence: EvidenceComplete,
	}
	observations, err := window.Observations()
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 8 {
		t.Fatalf("observations=%d", len(observations))
	}
	values := make(map[Signal]float64)
	for _, observation := range observations {
		values[observation.Signal] = observation.ObservedValue
	}
	if values[SignalReplay] != 20 || values[SignalImpossibleSequence] != 7 || values[SignalAbnormalRate] != 12 || values[SignalAbnormalCTR] != 15_000 {
		t.Fatalf("derived values=%#v", values)
	}
}

func TestWindowCannotRepresentInfrastructureFailureAsIVT(t *testing.T) {
	start := time.Now().UTC()
	window := Window{
		Scope: Scope{Type: ScopePublisher, ID: 7}, WindowKey: "window-2",
		StartedAt: start, EndedAt: start.Add(time.Minute), Evidence: EvidenceMissing,
	}
	observations, err := window.Observations()
	if err != nil {
		t.Fatal(err)
	}
	for _, observation := range observations {
		if observation.Signal == "DependencyError" || observation.Signal == "Timeout" {
			t.Fatalf("infrastructure outcome escaped into taxonomy: %#v", observation)
		}
	}
}

func TestWindowRejectsOverflowingCountersAndDerivedKeys(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	window := Window{
		Scope: Scope{Type: ScopePublisher, ID: 7}, WindowKey: "window-3",
		StartedAt: start, EndedAt: start.Add(time.Minute),
		Requests: 1_000_000_001, Evidence: EvidenceComplete,
	}
	if _, err := window.Observations(); err == nil {
		t.Fatal("unbounded counter accepted")
	}
	window.Requests = 0
	window.WindowKey = strings.Repeat("a", 250)
	if _, err := window.Observations(); err == nil {
		t.Fatal("window key that overflows derived identity accepted")
	}
}

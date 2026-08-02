package trafficquality

import (
	"bytes"
	"testing"
	"time"
)

func TestEnforcementSnapshotIsScopedDeterministicAndExpires(t *testing.T) {
	engine, err := NewEngine(bytes.Repeat([]byte{0x51}, 32))
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{engine: engine}
	now := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	scope := Scope{Type: ScopePublisher, ID: 7}
	snapshot := &Snapshot{LoadedAt: now, entries: map[Scope][]Enforcement{
		scope: {
			{ID: 2, Scope: scope, Action: ActionReject, State: "Active", ExpiresAt: now.Add(time.Hour)},
			{ID: 1, Scope: scope, Action: ActionQuarantine, State: "Canary", CanaryBasisPoints: 10_000, ExpiresAt: now.Add(time.Hour)},
		},
	}}
	action, id, err := service.EnforcementAction(snapshot, scope, "event-1", now, 2*time.Minute)
	if err != nil || action != ActionQuarantine || id != 1 {
		t.Fatalf("action=%s id=%d err=%v", action, id, err)
	}
	action, id, err = service.EnforcementAction(snapshot, Scope{Type: ScopePublisher, ID: 8}, "event-1", now, 2*time.Minute)
	if err != nil || action != ActionObserve || id != 0 {
		t.Fatalf("cross-scope action=%s id=%d err=%v", action, id, err)
	}
	action, _, err = service.EnforcementAction(snapshot, scope, "event-1", now.Add(3*time.Minute), 2*time.Minute)
	if err != nil || action != ActionObserve {
		t.Fatalf("stale snapshot action=%s err=%v", action, err)
	}
	action, _, err = service.EnforcementAction(snapshot, scope, "event-1", now.Add(2*time.Hour), 3*time.Hour)
	if err != nil || action != ActionObserve {
		t.Fatalf("expired row action=%s err=%v", action, err)
	}
}

func TestCanarySelectionIsStableAndBounded(t *testing.T) {
	enforcement := Enforcement{ID: 99, CanaryBasisPoints: 5_000}
	digest := [32]byte{1, 2, 3}
	first := enforcementCanarySelected(enforcement, digest)
	for i := 0; i < 10; i++ {
		if enforcementCanarySelected(enforcement, digest) != first {
			t.Fatal("enforcement canary selection changed")
		}
	}
	enforcement.CanaryBasisPoints = 0
	if enforcementCanarySelected(enforcement, digest) {
		t.Fatal("zero-size enforcement canary selected")
	}
	enforcement.CanaryBasisPoints = 10_000
	if !enforcementCanarySelected(enforcement, digest) {
		t.Fatal("full enforcement canary did not select")
	}
}

func TestEnforcementSnapshotRejectsAmbiguousRolloutState(t *testing.T) {
	now := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	scope := Scope{Type: ScopePublisher, ID: 7}
	for _, item := range []Enforcement{
		{ID: 1, Scope: scope, Action: ActionReject, State: "Canary", CanaryBasisPoints: 10_000, ExpiresAt: now.Add(time.Hour)},
		{ID: 2, Scope: scope, Action: ActionReject, State: "Active", CanaryBasisPoints: 500, ExpiresAt: now.Add(time.Hour)},
	} {
		if _, err := NewEnforcementSnapshot(now, []Enforcement{item}); err == nil {
			t.Fatalf("ambiguous enforcement rollout accepted: %#v", item)
		}
	}
}

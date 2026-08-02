package trafficquality

import (
	"bytes"
	"testing"
	"time"
)

func testRule(action Action) Rule {
	return Rule{
		ID: 7, Key: "replay.window", Version: 3, Signal: SignalReplay,
		Action: action, Mode: ModeActive, Scope: Scope{Type: ScopePublisher, ID: 11},
		Threshold: 2, WindowSeconds: 60, ReasonCode: "replay.threshold",
		EvidenceRetentionHrs: 24, AggregateRetentionDays: 400,
		FalsePositiveLimitBPS: 100, CreatedBy: "admin:42",
	}
}

func testObservation() Observation {
	return Observation{
		Signal: SignalReplay, Scope: Scope{Type: ScopePublisher, ID: 11},
		EventKey: "auction-event-1", PartnerKey: "partner-1", ObservedValue: 3,
		Evidence: EvidenceComplete, ObservedAt: time.Date(2026, 8, 1, 1, 2, 3, 0, time.FixedZone("offset", 3600)),
		SafeSummary: "replay count exceeded the configured threshold",
	}
}

func TestSignalTaxonomyIsClosedAndValidated(t *testing.T) {
	for _, signal := range []Signal{
		SignalReplay, SignalImpossibleSequence, SignalInvalidOriginApp,
		SignalMalformedIdentity, SignalAbnormalRate, SignalAbnormalCTR,
		SignalAutomation, SignalPartnerPolicy,
	} {
		rule := testRule(ActionFlag)
		rule.Signal = signal
		if err := rule.Validate(); err != nil {
			t.Errorf("signal %s: %v", signal, err)
		}
	}
	rule := testRule(ActionFlag)
	rule.Signal = "database-timeout"
	if err := rule.Validate(); err == nil {
		t.Fatal("infrastructure outcome was accepted as a traffic-quality signal")
	}
}

func TestEveryBlockingActionRequiresCompleteEvidence(t *testing.T) {
	engine, err := NewEngine(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []Action{ActionThrottle, ActionReject, ActionQuarantine} {
		for _, evidence := range []EvidenceState{EvidencePartial, EvidenceMissing} {
			rule := testRule(action)
			observation := testObservation()
			observation.Evidence = evidence
			decision, evaluated, err := engine.Evaluate(rule, observation)
			if err != nil || !evaluated {
				t.Fatalf("%s/%s: evaluated=%t err=%v", action, evidence, evaluated, err)
			}
			if decision.AppliedAction != ActionObserve || decision.BillingDisposition != BillingObserve {
				t.Fatalf("%s/%s enforced: %#v", action, evidence, decision)
			}
		}
	}
}

func TestRuleModesAreExplainableAndDeterministic(t *testing.T) {
	engine, err := NewEngine(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatal(err)
	}
	observation := testObservation()
	for _, mode := range []RuleMode{ModeDraft, ModeDisabled} {
		rule := testRule(ActionReject)
		rule.Mode = mode
		decision, evaluated, err := engine.Evaluate(rule, observation)
		if err != nil || evaluated || decision != (Decision{}) {
			t.Fatalf("mode %s: decision=%#v evaluated=%t err=%v", mode, decision, evaluated, err)
		}
	}
	rule := testRule(ActionReject)
	rule.Mode = ModeObserve
	decision, evaluated, err := engine.Evaluate(rule, observation)
	if err != nil || !evaluated || !decision.Matched || decision.AppliedAction != ActionObserve {
		t.Fatalf("observe decision=%#v evaluated=%t err=%v", decision, evaluated, err)
	}
	rule.Mode = ModeCanary
	rule.CanaryBasisPoints = 5_000
	first, evaluated, err := engine.Evaluate(rule, observation)
	if err != nil || !evaluated {
		t.Fatal(err)
	}
	second, _, err := engine.Evaluate(rule, observation)
	if err != nil || first.CanarySelected != second.CanarySelected || first.EventDigest != second.EventDigest {
		t.Fatalf("canary decision is not deterministic: %#v %#v err=%v", first, second, err)
	}
	if first.AppliedAction != ActionObserve && first.AppliedAction != ActionReject {
		t.Fatalf("canary action=%s", first.AppliedAction)
	}
}

func TestRuleScopeAndDigestsDoNotLeakRawIdentity(t *testing.T) {
	engine, err := NewEngine(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatal(err)
	}
	rule := testRule(ActionFlag)
	wrong := testObservation()
	wrong.Scope.ID++
	if decision, evaluated, err := engine.Evaluate(rule, wrong); err != nil || evaluated || decision != (Decision{}) {
		t.Fatalf("wrong scope: decision=%#v evaluated=%t err=%v", decision, evaluated, err)
	}
	decision, evaluated, err := engine.Evaluate(rule, testObservation())
	if err != nil || !evaluated {
		t.Fatal(err)
	}
	if decision.EventDigest == ([32]byte{}) || decision.PartnerDigest == ([32]byte{}) {
		t.Fatalf("missing keyed digests: %#v", decision)
	}
	if !decision.ObservedAt.Equal(time.Date(2026, 8, 1, 0, 2, 3, 0, time.UTC)) {
		t.Fatalf("observed time=%s", decision.ObservedAt)
	}
}

func TestInvalidObservationAndRuleBoundaries(t *testing.T) {
	engine, _ := NewEngine(bytes.Repeat([]byte{0x42}, 32))
	rule := testRule(ActionFlag)
	observation := testObservation()
	observation.SafeSummary = "unsafe\nsummary"
	if _, _, err := engine.Evaluate(rule, observation); err == nil {
		t.Fatal("multiline evidence summary accepted")
	}
	rule = testRule(ActionFlag)
	rule.Mode = ModeCanary
	rule.CanaryBasisPoints = 0
	if _, _, err := engine.Evaluate(rule, testObservation()); err == nil {
		t.Fatal("zero-size canary accepted")
	}
	if _, err := NewEngine([]byte("short")); err == nil {
		t.Fatal("short digest key accepted")
	}
}

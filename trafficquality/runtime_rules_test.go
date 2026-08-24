package trafficquality

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRuntimeRulesSelectHighestVersionPerRolloutMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	created := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	columns := []string{
		"rule_id", "rule_key", "rule_version", "signal_type", "rule_action",
		"rollout_mode", "scope_type", "scope_id", "threshold_value",
		"window_seconds", "canary_basis_points", "reason_code",
		"evidence_retention_hours", "aggregate_retention_days",
		"false_positive_limit_bps", "created_by", "created_at",
	}
	rows := sqlmock.NewRows(columns).
		AddRow(50, "replay.window", 5, SignalReplay, ActionReject, ModeObserve, ScopePublisher, 11, "2", 60, 0, "observe.v5", 24, 400, 100, "admin:1", created).
		AddRow(40, "replay.window", 4, SignalReplay, ActionReject, ModeCanary, ScopePublisher, 11, "2", 60, 5000, "canary.v4", 24, 400, 100, "admin:1", created).
		AddRow(20, "replay.window", 2, SignalReplay, ActionThrottle, ModeActive, ScopePublisher, 11, "2", 60, 0, "active.v2", 24, 400, 100, "admin:1", created).
		AddRow(30, "replay.window", 3, SignalReplay, ActionReject, ModeActive, ScopePublisher, 11, "2", 60, 0, "active.v3", 24, 400, 100, "admin:1", created).
		AddRow(10, "replay.window", 1, SignalReplay, ActionFlag, ModeObserve, ScopePublisher, 11, "2", 60, 0, "observe.v1", 24, 400, 100, "admin:1", created)
	mock.ExpectQuery(`FROM quality_rule[\s\S]+rollout_mode IN \('Observe','Canary','Active'\)`).
		WithArgs(SignalReplay, ScopePublisher, uint64(11)).
		WillReturnRows(rows)

	service := &Service{DB: db}
	rules, err := service.runtimeRules(context.Background(), Observation{
		Signal: SignalReplay, Scope: Scope{Type: ScopePublisher, ID: 11},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("selected rules = %#v, want one Active, Canary, and Observe version", rules)
	}
	wantModes := []RuleMode{ModeActive, ModeCanary, ModeObserve}
	wantVersions := []uint32{3, 4, 5}
	for i := range rules {
		if rules[i].Mode != wantModes[i] || rules[i].Version != wantVersions[i] {
			t.Fatalf("rule[%d] = %s/v%d, want %s/v%d", i, rules[i].Mode, rules[i].Version, wantModes[i], wantVersions[i])
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMultipleRuntimeModesKeepIncompleteEvidenceObserveOnly(t *testing.T) {
	candidates := []Rule{
		runtimeRuleFixture(50, 5, ModeObserve, ActionReject),
		runtimeRuleFixture(40, 4, ModeCanary, ActionReject),
		runtimeRuleFixture(30, 3, ModeActive, ActionReject),
		runtimeRuleFixture(20, 2, ModeActive, ActionThrottle),
		runtimeRuleFixture(60, 6, ModeDisabled, ActionReject),
	}
	rules := selectRuntimeRules(candidates)
	if len(rules) != 3 || rules[0].Mode != ModeActive || rules[0].Version != 3 || rules[1].Mode != ModeCanary || rules[2].Mode != ModeObserve {
		t.Fatalf("runtime precedence = %#v", rules)
	}
	engine, err := NewEngine(bytes.Repeat([]byte{0x73}, 32))
	if err != nil {
		t.Fatal(err)
	}
	observation := testObservation()
	observation.Evidence = EvidencePartial
	for _, rule := range rules {
		decision, evaluated, err := engine.Evaluate(rule, observation)
		if err != nil || !evaluated {
			t.Fatalf("%s/v%d evaluated=%t error=%v", rule.Mode, rule.Version, evaluated, err)
		}
		if decision.AppliedAction != ActionObserve || decision.BillingDisposition != BillingObserve {
			t.Fatalf("incomplete %s/v%d enforced: %#v", rule.Mode, rule.Version, decision)
		}
	}

	observation.Evidence = EvidenceComplete
	wantActions := []Action{ActionReject, ActionReject, ActionObserve}
	for i, rule := range rules {
		decision, evaluated, err := engine.Evaluate(rule, observation)
		if err != nil || !evaluated {
			t.Fatalf("complete %s/v%d evaluated=%t error=%v", rule.Mode, rule.Version, evaluated, err)
		}
		if decision.AppliedAction != wantActions[i] {
			t.Fatalf("complete %s/v%d action=%s, want %s", rule.Mode, rule.Version, decision.AppliedAction, wantActions[i])
		}
	}
}

func runtimeRuleFixture(id uint64, version uint32, mode RuleMode, action Action) Rule {
	rule := testRule(action)
	rule.ID = id
	rule.Version = version
	rule.Mode = mode
	rule.ReasonCode = "runtime.selection"
	if mode == ModeCanary {
		rule.CanaryBasisPoints = 10_000
	} else {
		rule.CanaryBasisPoints = 0
	}
	return rule
}

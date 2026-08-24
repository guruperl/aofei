package trafficquality

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestMySQLTrafficQualityLifecycle(t *testing.T) {
	dsn := os.Getenv("AOFEI_QUALITY_MYSQL_DSN")
	if dsn == "" {
		t.Skip("AOFEI_QUALITY_MYSQL_DSN is unset")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	service, err := NewServiceWithKey(db, bytes.Repeat([]byte{0x72}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	admin := integrationActor("42")
	checker := integrationActor("43")
	rule, err := service.CreateRule(ctx, admin, Rule{
		Key: "integration.replay", Signal: SignalReplay, Action: ActionQuarantine,
		Scope: Scope{Type: ScopeGlobal}, Threshold: 2, WindowSeconds: 60,
		ReasonCode: "replay.threshold", EvidenceRetentionHrs: 24,
		AggregateRetentionDays: 400, FalsePositiveLimitBPS: 100,
	}, "S03 disposable integration rule")
	if err != nil {
		t.Fatal(err)
	}
	if rule.Version == 0 || rule.Mode != ModeDraft {
		t.Fatalf("created rule=%#v", rule)
	}
	if _, err := db.ExecContext(ctx, `UPDATE quality_rule SET threshold_value=99 WHERE rule_id=?`, rule.ID); err == nil {
		t.Fatal("rule behavior was rewritten without a new version")
	}
	if err := service.SetRuleMode(ctx, admin, rule.ID, ModeDraft, ModeObserve, 0, "observe before enforcement"); err != nil {
		t.Fatal(err)
	}
	scope := Scope{Type: ScopePublisher, ID: 3000000001}
	observation := Observation{
		Signal: SignalReplay, Scope: scope, EventKey: "s03-integration-event-1",
		PartnerKey: "s03-partner", ObservedValue: 3, Evidence: EvidenceComplete,
		ObservedAt: now.Add(-25 * time.Hour), SafeSummary: "bounded disposable replay evidence",
	}
	decisions, err := service.Assess(ctx, observation)
	if err != nil || len(decisions) != 1 || decisions[0].AppliedAction != ActionObserve {
		t.Fatalf("observe decisions=%#v err=%v", decisions, err)
	}
	replay, err := service.Assess(ctx, observation)
	if err != nil || len(replay) != 1 || replay[0].ID != decisions[0].ID {
		t.Fatalf("idempotent decision=%#v err=%v", replay, err)
	}
	var decisionCount, counterCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM quality_decision WHERE rule_id=?`, rule.ID).Scan(&decisionCount); err != nil || decisionCount != 1 {
		t.Fatalf("decision count=%d err=%v", decisionCount, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT SUM(decisions) FROM quality_counter WHERE rule_id=?`, rule.ID).Scan(&counterCount); err != nil || counterCount != 1 {
		t.Fatalf("counter count=%d err=%v", counterCount, err)
	}
	observeCases, err := service.ListCases(ctx, admin, scope, 20)
	if err != nil || len(observeCases) != 1 {
		t.Fatalf("observe cases=%#v err=%v", observeCases, err)
	}
	if err := service.ResolveCase(ctx, admin, observeCases[0].ID, observeCases[0].Version, ReviewInvalidTraffic, "observe evidence reviewed before canary"); err != nil {
		t.Fatal(err)
	}
	if err := service.SetRuleMode(ctx, admin, rule.ID, ModeObserve, ModeCanary, 10_000, "start full deterministic canary after observe mode"); err != nil {
		t.Fatal(err)
	}
	observation.EventKey = "s03-integration-event-2"
	observation.ObservedAt = now
	decisions, err = service.Assess(ctx, observation)
	if err != nil || len(decisions) != 1 || decisions[0].AppliedAction != ActionQuarantine || decisions[0].BillingDisposition != BillingHold {
		t.Fatalf("active decisions=%#v err=%v", decisions, err)
	}
	if err := service.SetRuleMode(ctx, checker, rule.ID, ModeCanary, ModeActive, 0, "must not use an older observe review for active rollout"); !errors.Is(err, ErrConflict) {
		t.Fatalf("active rollout without reviewed canary err=%v", err)
	}
	cases, err := service.ListCases(ctx, admin, scope, 20)
	if err != nil || len(cases) != 2 {
		t.Fatalf("cases=%#v err=%v", cases, err)
	}
	activeCase := cases[0]
	if err := service.ResolveCase(ctx, admin, activeCase.ID, activeCase.Version, ReviewInvalidTraffic, "complete evidence reviewed as invalid"); err != nil {
		t.Fatal(err)
	}
	if err := service.SetRuleMode(ctx, checker, rule.ID, ModeCanary, ModeActive, 0, "activate after reviewed canary stayed within false-positive limit"); err != nil {
		t.Fatal(err)
	}
	pub := Actor{
		Role: "pub", ID: "3000000001", Scope: scope,
		Permissions: map[string]bool{PermissionEvidenceRead: true, PermissionAppealSubmit: true},
	}
	if _, err := service.ListCases(ctx, pub, Scope{Type: ScopePublisher, ID: scope.ID + 1}, 20); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-account list err=%v", err)
	}
	cases, err = service.ListCases(ctx, pub, scope, 20)
	if err != nil {
		t.Fatal(err)
	}
	activeCase = cases[0]
	if err := service.AppealCase(ctx, pub, activeCase.ID, activeCase.Version, "publisher supplied a bounded appeal reason"); err != nil {
		t.Fatal(err)
	}
	cases, err = service.ListCases(ctx, admin, scope, 20)
	if err != nil {
		t.Fatal(err)
	}
	activeCase = cases[0]
	if err := service.ResolveAppeal(ctx, checker, activeCase.ID, activeCase.Version, false, "independent appeal review confirmed evidence"); err != nil {
		t.Fatal(err)
	}
	// A fresh reviewed case is required for enforcement activation.
	observation.EventKey = "s03-integration-event-3"
	decisions, err = service.Assess(ctx, observation)
	if err != nil || len(decisions) != 1 {
		t.Fatal(err)
	}
	cases, err = service.ListCases(ctx, admin, scope, 20)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ResolveCase(ctx, admin, cases[0].ID, cases[0].Version, ReviewInvalidTraffic, "second complete evidence review"); err != nil {
		t.Fatal(err)
	}
	enforcementID, err := service.ActivateEnforcement(ctx, checker, decisions[0].ID, ActionQuarantine, 10_000, time.Hour, "reviewed quarantine canary")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE quality_enforcement SET scope_id=scope_id+1 WHERE enforcement_id=?`, enforcementID); err == nil {
		t.Fatal("protected enforcement scope update succeeded")
	}
	snapshot, err := service.LoadEnforcementSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	action, selectedID, err := service.EnforcementAction(snapshot, scope, "serving-event", now, 2*time.Minute)
	if err != nil || action != ActionQuarantine || selectedID != enforcementID {
		t.Fatalf("enforcement action=%s id=%d err=%v", action, selectedID, err)
	}
	if err := service.RollbackEnforcement(ctx, admin, enforcementID, "false-positive rollback drill"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE quality_enforcement SET state='Active', rolled_back_by=NULL,
 rollback_reason=NULL, rolled_back_at=NULL WHERE enforcement_id=?`, enforcementID); err == nil {
		t.Fatal("terminal enforcement rewrite succeeded")
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO acct_statement
 (request_key, party_type, party_id, cadence, period_start, period_end,
 currency, source_amount, adjustment_amount, total_amount, status, created_by,
 created_at, updated_at)
VALUES ('s03-integration-statement','publisher',3000000001,'daily',CURRENT_DATE,CURRENT_DATE,
 'USD',1,0,1,'Draft','integration',UTC_TIMESTAMP(),UTC_TIMESTAMP())`); err != nil {
		t.Fatal(err)
	}
	var statementID uint64
	if err := db.QueryRowContext(ctx, `SELECT statement_id FROM acct_statement WHERE request_key='s03-integration-statement'`).Scan(&statementID); err != nil {
		t.Fatal(err)
	}
	billingID, err := service.RecommendBilling(ctx, admin, decisions[0].ID, statementID,
		"billable-s03-integration", BillingHold, "reviewed invalid billable identity")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE quality_billing SET disposition='Exclude' WHERE billing_id=?`, billingID); err == nil {
		t.Fatal("protected billing disposition update succeeded")
	}
	if _, err := db.ExecContext(ctx, `
UPDATE quality_billing SET state='Rejected', approved_by=recommended_by,
 updated_at=UTC_TIMESTAMP(6) WHERE billing_id=?`, billingID); err == nil {
		t.Fatal("same-actor billing review succeeded")
	}
	cases, err = service.ListCases(ctx, pub, scope, 20)
	if err != nil {
		t.Fatal(err)
	}
	activeCase = cases[0]
	if err := service.AppealCase(ctx, pub, activeCase.ID, activeCase.Version, "second bounded appeal pauses billing approval"); err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveBilling(ctx, checker, billingID, true, "must not approve while the case is appealed"); !errors.Is(err, ErrConflict) {
		t.Fatalf("billing approval during appeal err=%v", err)
	}
	cases, err = service.ListCases(ctx, checker, scope, 20)
	if err != nil {
		t.Fatal(err)
	}
	activeCase = cases[0]
	if err := service.ResolveAppeal(ctx, checker, activeCase.ID, activeCase.Version, false, "independent review denied the second appeal"); err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveBilling(ctx, checker, billingID, true, "independent checker applied statement hold"); err != nil {
		t.Fatal(err)
	}
	var statementStatus, billingStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM acct_statement WHERE statement_id=?`, statementID).Scan(&statementStatus); err != nil || statementStatus != "Held" {
		t.Fatalf("statement status=%s err=%v", statementStatus, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT state FROM quality_billing WHERE billing_id=?`, billingID).Scan(&billingStatus); err != nil || billingStatus != "Applied" {
		t.Fatalf("billing status=%s err=%v", billingStatus, err)
	}
	if _, err := db.ExecContext(ctx, `
UPDATE quality_billing SET approved_by='admin:tampered' WHERE billing_id=?`, billingID); err == nil {
		t.Fatal("terminal billing approval rewrite succeeded")
	}
	if _, err := db.ExecContext(ctx, `UPDATE quality_decision SET reason_code='changed' WHERE decision_id=?`, decisions[0].ID); err == nil {
		t.Fatal("immutable decision update succeeded")
	}
	maintenance := Actor{Role: "admin", ID: "unix-uid:1001", Scope: Scope{Type: ScopeGlobal}, Permissions: map[string]bool{PermissionRetentionPrune: true}}
	deleted, err := service.PruneEvidence(ctx, maintenance, 100, "scheduled disposable evidence retention")
	if err != nil || deleted < 1 {
		t.Fatalf("pruned=%d err=%v", deleted, err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var bypass int
	if err := conn.QueryRowContext(ctx, `SELECT COALESCE(@aofei_quality_retention,0)`).Scan(&bypass); err != nil || bypass != 0 {
		t.Fatalf("retention bypass=%d err=%v", bypass, err)
	}
}

func integrationActor(id string) Actor {
	return Actor{
		Role: "admin", ID: id, Scope: Scope{Type: ScopeGlobal}, RecentMFA: true,
		Permissions: map[string]bool{"*": true},
	}
}

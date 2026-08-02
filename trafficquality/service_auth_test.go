package trafficquality

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestActorScopeAndSensitivePermissionBoundary(t *testing.T) {
	pubScope := Scope{Type: ScopePublisher, ID: 7}
	actor := Actor{
		Role: "pub", ID: "7", Scope: pubScope,
		Permissions: map[string]bool{PermissionEvidenceRead: true, PermissionAppealSubmit: true},
	}
	if err := actor.Validate(); err != nil || !actor.CanRead(pubScope) {
		t.Fatalf("publisher actor invalid/read denied: %v", err)
	}
	if actor.CanRead(Scope{Type: ScopePublisher, ID: 8}) {
		t.Fatal("publisher actor can read another publisher")
	}
	if err := requireActor(actor, PermissionEnforcementActivate, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("enforcement permission err=%v", err)
	}
	admin := Actor{
		Role: "admin", ID: "42", Scope: Scope{Type: ScopeGlobal}, RecentMFA: true,
		Permissions: map[string]bool{PermissionEvidenceRead: true, PermissionEnforcementActivate: true},
	}
	if !admin.CanRead(pubScope) {
		t.Fatal("authorized administrator cannot read scoped evidence")
	}
	if err := requireActor(admin, PermissionEnforcementActivate, true); err != nil {
		t.Fatal(err)
	}
}

func TestScopedActorCannotListCrossAccountRuleDefinitions(t *testing.T) {
	actor := Actor{
		Role: "pub", ID: "7", Scope: Scope{Type: ScopePublisher, ID: 7},
		Permissions: map[string]bool{PermissionEvidenceRead: true},
	}
	if _, err := (&Service{}).ListRules(context.Background(), actor); !errors.Is(err, ErrForbidden) {
		t.Fatalf("scoped rule list err=%v", err)
	}
}

func TestIncompleteEvidenceCannotRecommendBillingChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service, err := NewServiceWithKey(db, bytes.Repeat([]byte{0x66}, 32))
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{
		Role: "admin", ID: "42", Scope: Scope{Type: ScopeGlobal}, RecentMFA: true,
		Permissions: map[string]bool{"*": true},
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT d.scope_type").WithArgs(uint64(9)).WillReturnRows(
		sqlmock.NewRows([]string{"scope_type", "scope_id", "evidence_state", "status"}).
			AddRow("Publisher", 7, "Partial", "InvalidTraffic"))
	mock.ExpectRollback()
	if _, err := service.RecommendBilling(context.Background(), actor, 9, 11, "billable-1", BillingHold, "incomplete evidence must remain observe only"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("incomplete evidence billing err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFutureObservationIsRejectedBeforeDatabaseWork(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service, err := NewServiceWithKey(db, bytes.Repeat([]byte{0x67}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	observation := testObservation()
	observation.ObservedAt = now.Add(time.Minute + time.Nanosecond)
	if _, err := service.Assess(context.Background(), observation); err == nil {
		t.Fatal("far-future observation accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMachineServiceRoleCannotBePresentedAsHumanActor(t *testing.T) {
	actor := Actor{Role: "service", ID: "rule-engine", Scope: Scope{Type: ScopeGlobal}, Permissions: map[string]bool{"*": true}, RecentMFA: true}
	if err := actor.Validate(); err == nil {
		t.Fatal("machine service identity accepted as an S02 human actor")
	}
}

func TestFalsePositiveLimitRecommendsRollbackOnlyForEnforcingModes(t *testing.T) {
	health := RuleHealth{
		Mode: ModeCanary, ValidTraffic: 2, InvalidTraffic: 8,
		FalsePositiveLimitBPS: 1_000,
	}
	finalizeRuleHealth(&health)
	if health.FalsePositiveBasisPoints != 2_000 || !health.RollbackRecommended {
		t.Fatalf("health=%#v", health)
	}
	health.Mode = ModeObserve
	health.RollbackRecommended = false
	finalizeRuleHealth(&health)
	if health.RollbackRecommended {
		t.Fatalf("observe-only rule recommended serving rollback: %#v", health)
	}
}

func TestRuleRolloutRequiresObserveThenCanaryBeforeActive(t *testing.T) {
	allowed := map[[2]RuleMode]bool{
		{ModeDraft, ModeObserve}: true, {ModeObserve, ModeCanary}: true,
		{ModeCanary, ModeActive}: true, {ModeCanary, ModeObserve}: true,
		{ModeActive, ModeObserve}: true, {ModeActive, ModeDisabled}: true,
		{ModeDisabled, ModeObserve}: true,
	}
	for _, pair := range [][2]RuleMode{
		{ModeDraft, ModeActive}, {ModeObserve, ModeActive}, {ModeDraft, ModeCanary},
		{ModeActive, ModeCanary}, {ModeDisabled, ModeActive},
	} {
		if ruleModeTransitionAllowed(pair[0], pair[1]) {
			t.Errorf("unsafe direct rollout allowed: %s -> %s", pair[0], pair[1])
		}
	}
	for pair := range allowed {
		if !ruleModeTransitionAllowed(pair[0], pair[1]) {
			t.Errorf("documented rollout denied: %s -> %s", pair[0], pair[1])
		}
	}
}

package reporting

import (
	"context"
	"encoding/hex"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func testExperiment() Experiment {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	return Experiment{
		ID: 7, OwnerType: "Operator", Name: "checkout-copy", Version: 2,
		AssignmentAlgorithmVersion: AssignmentAlgorithmV2,
		Status:                     "Running", AssignmentSalt: "00112233445566778899aabbccddeeff",
		PrimaryMetric: "actions", GuardrailMetric: "spend", RetentionHours: 2160, StartsAt: start,
		Variants: []Variant{{Key: "control", AllocationBasisPts: 5000}, {Key: "treatment", AllocationBasisPts: 5000}},
	}
}

func TestAssignV1RetainsLegacyGoldenHash(t *testing.T) {
	experiment := testExperiment()
	experiment.AssignmentAlgorithmVersion = AssignmentAlgorithmV1
	assignment, err := Assign(experiment, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", experiment.StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(assignment.SubjectHash[:]); got != "b0e93894e33027b140179b180397e96fd734abcfa6d7f8f6915884edd4b53944" {
		t.Fatalf("v1 subject hash = %s, want legacy golden", got)
	}
	if assignment.AlgorithmVersion != AssignmentAlgorithmV1 {
		t.Fatalf("algorithm version = %d, want v1", assignment.AlgorithmVersion)
	}
}

func TestAssignV2SeparatesExperimentIdentityEvenWithSameSalt(t *testing.T) {
	first := testExperiment()
	second := first
	second.ID++
	subject := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	firstAssignment, err := Assign(first, subject, first.StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	secondAssignment, err := Assign(second, subject, second.StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	if firstAssignment.SubjectHash == secondAssignment.SubjectHash {
		t.Fatal("v2 subject hash is linkable across experiment identities")
	}
}

func TestAssignIsDeterministicAndStoresOnlyHash(t *testing.T) {
	experiment := testExperiment()
	at := experiment.StartsAt.Add(time.Hour)
	subject := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	first, err := Assign(experiment, subject, at)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Assign(experiment, subject, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if first.VariantKey != second.VariantKey || first.SubjectHash != second.SubjectHash {
		t.Fatalf("assignment changed: %#v %#v", first, second)
	}
	if string(first.SubjectHash[:]) == subject {
		t.Fatal("raw experiment pseudonym was retained")
	}
}

func TestAllocationModuloSkewStaysWithinV2AcceptanceBound(t *testing.T) {
	if allocationModuloRelativeSkew > maxAllocationModuloRelativeSkew {
		t.Fatalf("relative modulo skew %.18g exceeds bound %.18g", allocationModuloRelativeSkew, maxAllocationModuloRelativeSkew)
	}
	if allocationModuloRelativeSkew <= 0 {
		t.Fatal("allocation skew measurement must remain explicit")
	}
}

func TestAssignRejectsInactiveOrMalformedSubjects(t *testing.T) {
	experiment := testExperiment()
	experiment.Status = "Draft"
	if _, err := Assign(experiment, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", experiment.StartsAt); err == nil {
		t.Fatal("draft experiment accepted an assignment")
	}
	experiment.Status = "Running"
	if _, err := Assign(experiment, "raw-user-id", experiment.StartsAt); err == nil {
		t.Fatal("raw subject identifier was accepted")
	}
}

func TestRecordExposureIsIdempotentAndDetectsConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	assignment, err := Assign(testExperiment(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", testExperiment().StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT e.owner_type, e.adv_id, e.assignment_algorithm_version, e.status,")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.VariantKey).
		WillReturnRows(sqlmock.NewRows([]string{
			"owner_type", "adv_id", "assignment_algorithm_version", "status", "assignment_salt", "retention_hours", "starts_at", "ends_at",
		}).AddRow("Operator", nil, AssignmentAlgorithmV2, "Running", testExperiment().AssignmentSalt, testExperiment().RetentionHours, testExperiment().StartsAt, nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO report_exposure")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:], assignment.VariantKey, assignment.AssignedAt, assignment.ExpiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT exposure_id, variant_key, exposed_at, expires_at")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"exposure_id", "variant_key", "exposed_at", "expires_at"}).
			AddRow(1, assignment.VariantKey, assignment.AssignedAt, assignment.ExpiresAt))
	mock.ExpectCommit()
	if err := RecordExposure(context.Background(), db, assignment); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordExposureRejectsCallerBuiltAssignment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	assignment := Assignment{
		ExperimentID: 7, Version: 2, AlgorithmVersion: AssignmentAlgorithmV2,
		OwnerType: "Operator", VariantKey: "control", SubjectHash: [32]byte{1},
		AssignedAt: testExperiment().StartsAt,
		ExpiresAt:  testExperiment().StartsAt.Add(2160 * time.Hour),
	}
	if err := RecordExposure(context.Background(), db, assignment); err == nil {
		t.Fatal("caller-built assignment without internal proof was accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAssignmentContractRejectsScopeRetentionAndProofMismatch(t *testing.T) {
	experiment := testExperiment()
	assignment, err := Assign(experiment, "abababababababababababababababababababababababababababababababab", experiment.StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	contract := assignmentContract{
		ownerType: "Operator", algorithm: AssignmentAlgorithmV2,
		salt: experiment.AssignmentSalt, retentionHours: experiment.RetentionHours,
		startsAt: experiment.StartsAt,
	}
	for _, mutate := range []func(*Assignment){
		func(value *Assignment) { value.OwnerID = 7 },
		func(value *Assignment) { value.ExpiresAt = value.ExpiresAt.Add(time.Hour) },
		func(value *Assignment) { value.proof = [32]byte{1} },
	} {
		candidate := assignment
		mutate(&candidate)
		if err := contract.validate(candidate); err == nil {
			t.Fatalf("mismatched assignment accepted: %#v", candidate)
		}
	}
}

func TestRecordOutcomeIsScopedToExposureDeclaredMetricAndIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	assignment, err := Assign(testExperiment(), "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", testExperiment().StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := NewOutcome(assignment, "actions", "1.000000", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", assignment.AssignedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT x.exposure_id, x.variant_key, x.exposed_at, x.expires_at,")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{
			"exposure_id", "variant_key", "exposed_at", "expires_at",
			"owner_type", "adv_id", "assignment_algorithm_version", "status", "assignment_salt", "retention_hours", "starts_at", "ends_at",
			"primary_metric", "guardrail_metric",
		}).AddRow(41, assignment.VariantKey, assignment.AssignedAt, assignment.ExpiresAt,
			"Operator", nil, AssignmentAlgorithmV2, "Running", testExperiment().AssignmentSalt, testExperiment().RetentionHours, testExperiment().StartsAt, nil,
			"actions", "spend"))
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO report_experiment_outcome")).
		WithArgs(uint64(41), "actions", "1.000000", outcome.IdempotencyKey[:], outcome.OccurredAt).
		WillReturnResult(sqlmock.NewResult(9, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT metric_name, metric_value, occurred_at")).
		WithArgs(uint64(41), outcome.IdempotencyKey[:]).
		WillReturnRows(sqlmock.NewRows([]string{"metric_name", "metric_value", "occurred_at"}).
			AddRow("actions", "1.000000", outcome.OccurredAt))
	mock.ExpectCommit()
	if err := RecordOutcome(context.Background(), db, outcome); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNewOutcomeRejectsRawOrAmbiguousValues(t *testing.T) {
	assignment, err := Assign(testExperiment(), "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", testExperiment().StartsAt)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, metric, value, key string
		at                       time.Time
	}{
		{name: "raw key", metric: "actions", value: "1.000000", key: "customer-17", at: assignment.AssignedAt},
		{name: "floating ambiguity", metric: "actions", value: "1.0", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "nan", metric: "actions", value: "NaN", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "infinity", metric: "actions", value: "+Inf", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "negative zero", metric: "spend", value: "-0.000000", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "fractional count", metric: "actions", value: "1.500000", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "negative count", metric: "actions", value: "-1.000000", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "ctr above one", metric: "ctr", value: "1.000001", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "cvr below zero", metric: "cvr", value: "-0.000001", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "roi below minus one", metric: "roi", value: "-1.000001", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "negative money", metric: "spend", value: "-0.000001", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "unknown metric", metric: "secret-score", value: "1.000000", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt},
		{name: "before exposure", metric: "actions", value: "1.000000", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.AssignedAt.Add(-time.Microsecond)},
		{name: "after expiry", metric: "actions", value: "1.000000", key: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", at: assignment.ExpiresAt},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewOutcome(assignment, test.metric, test.value, test.key, test.at); err == nil {
				t.Fatal("invalid experiment outcome was accepted")
			}
		})
	}
}

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
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO report_exposure")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:], assignment.VariantKey, assignment.AssignedAt, assignment.ExpiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT variant_key FROM report_exposure")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"variant_key"}).AddRow(assignment.VariantKey))
	if err := RecordExposure(context.Background(), db, assignment); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
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
	mock.ExpectQuery(regexp.QuoteMeta("SELECT x.exposure_id, x.variant_key, e.primary_metric, e.guardrail_metric")).
		WithArgs(assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:]).
		WillReturnRows(sqlmock.NewRows([]string{"exposure_id", "variant_key", "primary_metric", "guardrail_metric"}).
			AddRow(41, assignment.VariantKey, "actions", "spend"))
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

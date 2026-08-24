package reporting

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestExperimentAlgorithmCompatibilityMySQL(t *testing.T) {
	dsn := os.Getenv("AOFEI_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("AOFEI_MYSQL_TEST_DSN is not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	result, err := tx.ExecContext(ctx, `
INSERT INTO report_experiment
  (owner_type, experiment_name, experiment_version, status, assignment_salt,
   primary_metric, guardrail_metric, retention_hours, starts_at, created_by_uid)
VALUES ('Operator','r03-legacy-compatibility',1,'Draft',?,'actions','spend',2160,?,1)`,
		"00112233445566778899aabbccddeeff", start)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var algorithm uint16
	if err := tx.QueryRowContext(ctx, `SELECT assignment_algorithm_version FROM report_experiment WHERE experiment_id=?`, id).Scan(&algorithm); err != nil {
		t.Fatal(err)
	}
	if algorithm != AssignmentAlgorithmV1 {
		t.Fatalf("legacy default algorithm = %d, want v1", algorithm)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET assignment_algorithm_version=2 WHERE experiment_id=?`, id); err == nil {
		t.Fatal("assignment algorithm rewrite succeeded")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO report_experiment_variant (experiment_id, experiment_version, variant_key, allocation_basis_points) VALUES (?,?,?,?)`, id, 1, "control", 5000); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment_variant SET allocation_basis_points=4000 WHERE experiment_id=? AND experiment_version=1 AND variant_key='control'`, id); err == nil {
		t.Fatal("variant allocation rewrite succeeded")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM report_experiment_variant WHERE experiment_id=? AND experiment_version=1 AND variant_key='control'`, id); err == nil {
		t.Fatal("variant deletion succeeded")
	}
}

func TestExperimentFactBindingMySQL(t *testing.T) {
	dsn := os.Getenv("AOFEI_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("AOFEI_MYSQL_TEST_DSN is not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	start := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	experiment := Experiment{
		OwnerType: "Operator", Name: "r03-fact-binding-" + time.Now().UTC().Format("150405.000000"), Version: 1,
		Status: "Draft", PrimaryMetric: "actions", GuardrailMetric: "spend",
		RetentionHours: 24, StartsAt: start,
		Variants: []Variant{{Key: "control", AllocationBasisPts: 5000}, {Key: "treatment", AllocationBasisPts: 5000}},
	}
	id, err := CreateExperiment(ctx, db, experiment, 1, "R03 disposable integration")
	if err != nil {
		t.Fatal(err)
	}
	if err := TransitionExperiment(ctx, db, id, "Running", 1, "R03 disposable start"); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadExperiment(ctx, db, id)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := Assign(loaded, "abababababababababababababababababababababababababababababababab", start.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	forged := assignment
	forged.OwnerID = 9
	if err := RecordExposure(ctx, db, forged); err == nil {
		t.Fatal("wrong-scope assignment was stored")
	}
	if err := RecordExposure(ctx, db, assignment); err != nil {
		t.Fatal(err)
	}
	if err := RecordExposure(ctx, db, assignment); err != nil {
		t.Fatalf("idempotent exposure retry: %v", err)
	}
	outcome, err := NewOutcome(assignment, "actions", "1.000000", "cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd", assignment.AssignedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := RecordOutcome(ctx, db, outcome); err != nil {
		t.Fatal(err)
	}
	if err := RecordOutcome(ctx, db, outcome); err != nil {
		t.Fatalf("idempotent outcome retry: %v", err)
	}
	var exposureID uint64
	if err := db.QueryRowContext(ctx, `
SELECT exposure_id FROM report_exposure
WHERE experiment_id=? AND experiment_version=? AND subject_hash=?`, id, assignment.Version, assignment.SubjectHash[:]).Scan(&exposureID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO report_experiment_outcome
  (exposure_id, metric_name, metric_value, idempotency_key, occurred_at)
VALUES (?,?,?,?,?)`, exposureID, "actions", "-1.000000", []byte("01234567890123456789012345678901"), assignment.AssignedAt.Add(2*time.Minute)); err == nil {
		t.Fatal("database accepted an out-of-domain experiment outcome")
	}
	newAssignment, err := Assign(loaded, "efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef", start.Add(31*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := TransitionExperiment(ctx, db, id, "Stopped", 1, "R03 disposable stop"); err != nil {
		t.Fatal(err)
	}
	if err := RecordExposure(ctx, db, assignment); err != nil {
		t.Fatalf("stopped-state idempotent retry: %v", err)
	}
	if err := RecordExposure(ctx, db, newAssignment); err == nil {
		t.Fatal("new exposure was accepted after stop")
	}
}

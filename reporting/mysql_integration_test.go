package reporting

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
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
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET retention_hours=48 WHERE experiment_id=?`, id); err == nil {
		t.Fatal("experiment version contract rewrite succeeded")
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
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET status='Running' WHERE experiment_id=?`, id); err == nil {
		t.Fatal("incomplete experiment allocation started through direct SQL")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO report_experiment_variant (experiment_id, experiment_version, variant_key, allocation_basis_points) VALUES (?,?,?,?)`, id, 1, "treatment", 5000); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET status='Running' WHERE experiment_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO report_experiment_variant (experiment_id, experiment_version, variant_key, allocation_basis_points) VALUES (?,?,?,?)`, id, 1, "late", 1); err == nil {
		t.Fatal("variant insertion succeeded after experiment start")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET status='Draft' WHERE experiment_id=?`, id); err == nil {
		t.Fatal("backward experiment status transition succeeded")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET stop_reason='rewritten' WHERE experiment_id=?`, id); err == nil {
		t.Fatal("experiment stop reason changed without a status transition")
	}
}

func TestExperimentVariantInsertSerializesWithStartMySQL(t *testing.T) {
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
	start := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	experiment := Experiment{
		OwnerType: "Operator", Name: "r03-variant-start-race-" + time.Now().UTC().Format("150405.000000"), Version: 1,
		Status: "Draft", PrimaryMetric: "actions", GuardrailMetric: "spend",
		RetentionHours: 24, StartsAt: start,
		Variants: []Variant{{Key: "control", AllocationBasisPts: 5000}, {Key: "treatment", AllocationBasisPts: 5000}},
	}
	id, err := CreateExperiment(ctx, db, experiment, 1, "R03 disposable variant race")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE report_experiment SET status='Running' WHERE experiment_id=?`, id); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		_, insertErr := db.ExecContext(ctx, `
INSERT INTO report_experiment_variant
  (experiment_id, experiment_version, variant_key, allocation_basis_points)
VALUES (?,?,?,?)`, id, 1, "late", 1)
		done <- insertErr
	}()
	<-started
	select {
	case insertErr := <-done:
		t.Fatalf("late variant insert did not serialize with start: %v", insertErr)
	case <-time.After(50 * time.Millisecond):
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case insertErr := <-done:
		if insertErr == nil {
			t.Fatal("late variant insert succeeded after concurrent start")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("late variant insert did not finish after start committed")
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
	runConcurrentExposureWrites(t, db, assignment, 8)
	concurrentFirst, err := Assign(loaded, "3434343434343434343434343434343434343434343434343434343434343434", start.Add(33*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	runConcurrentExposureWrites(t, db, concurrentFirst, 2)
	var concurrentCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_exposure WHERE experiment_id=? AND experiment_version=? AND subject_hash=?`,
		id, concurrentFirst.Version, concurrentFirst.SubjectHash[:]).Scan(&concurrentCount); err != nil {
		t.Fatal(err)
	}
	if concurrentCount != 1 {
		t.Fatalf("concurrent first exposure count = %d, want 1", concurrentCount)
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
	adjacentAssignment, err := Assign(loaded, "efefefefefefefefefefefefefefefefefefefefefefefefefefefefefefefef", start.Add(31*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := RecordExposure(ctx, db, adjacentAssignment); err != nil {
		t.Fatal(err)
	}
	newAssignment, err := Assign(loaded, "1212121212121212121212121212121212121212121212121212121212121212", start.Add(32*time.Minute))
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
	assignmentHash := hex.EncodeToString(assignment.SubjectHash[:])
	if deleted, err := DeleteSubject(ctx, db, id, assignment.Version, assignmentHash, 1, "R03 verified disposable erasure"); err != nil || !deleted {
		t.Fatalf("exact subject deletion deleted=%t err=%v", deleted, err)
	}
	for name, test := range map[string]struct {
		hash []byte
		want int
	}{
		"deleted subject":  {assignment.SubjectHash[:], 0},
		"adjacent subject": {adjacentAssignment.SubjectHash[:], 1},
	} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_exposure WHERE experiment_id=? AND experiment_version=? AND subject_hash=?`, id, assignment.Version, test.hash).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != test.want {
			t.Fatalf("%s exposure count = %d, want %d", name, count, test.want)
		}
	}
	var event, auditReason string
	if err := db.QueryRowContext(ctx, `
SELECT event, reason FROM report_experiment_audit
WHERE experiment_id=? AND experiment_version=?
ORDER BY audit_id DESC LIMIT 1`, id, assignment.Version).Scan(&event, &auditReason); err != nil {
		t.Fatal(err)
	}
	if event != "SubjectErased" || auditReason != "R03 verified disposable erasure" || auditReason == assignmentHash {
		t.Fatalf("privacy audit event=%q reason=%q", event, auditReason)
	}

	expiredHash := bytes.Repeat([]byte{0xee}, 32)
	expiredAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	result, err := db.ExecContext(ctx, `
INSERT INTO report_exposure
  (experiment_id, experiment_version, subject_hash, variant_key, exposed_at, expires_at)
VALUES (?,?,?,?,?,?)`, id, assignment.Version, expiredHash, adjacentAssignment.VariantKey, expiredAt.Add(-time.Hour), expiredAt)
	if err != nil {
		t.Fatal(err)
	}
	expiredExposureID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO report_experiment_outcome
  (exposure_id, metric_name, metric_value, idempotency_key, occurred_at)
VALUES (?,?,?,?,?)`, expiredExposureID, "actions", "1.000000", bytes.Repeat([]byte{0xdd}, 32), expiredAt.Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if deleted, err := PruneExpired(ctx, db, 100); err != nil || deleted < 1 {
		t.Fatalf("retention prune deleted=%d err=%v", deleted, err)
	}
	var remaining int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_exposure WHERE exposure_id=?`, expiredExposureID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatal("expired experiment subject survived retention prune")
	}
}

func runConcurrentExposureWrites(t *testing.T, db *sql.DB, assignment Assignment, workers int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, workers)
	for range workers {
		go func() {
			<-start
			results <- RecordExposure(ctx, db, assignment)
		}()
	}
	close(start)
	for range workers {
		select {
		case err := <-results:
			if err != nil {
				t.Fatalf("concurrent exposure write: %v", err)
			}
		case <-ctx.Done():
			t.Fatalf("concurrent exposure writes did not finish: %v", ctx.Err())
		}
	}
}

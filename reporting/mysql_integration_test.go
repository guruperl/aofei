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

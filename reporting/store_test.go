package reporting

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateExperimentWritesDraftVariantsAndAuditAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	experiment := testExperiment()
	experiment.ID = 0
	experiment.Status = "Draft"
	experiment.AssignmentSalt = ""
	experiment.AssignmentAlgorithmVersion = 0
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO report_experiment")).
		WithArgs("Operator", nil, experiment.Name, experiment.Version, AssignmentAlgorithmV2, "Draft", sqlmock.AnyArg(), experiment.PrimaryMetric, experiment.GuardrailMetric, experiment.RetentionHours, experiment.StartsAt, nil, uint64(99)).
		WillReturnResult(sqlmock.NewResult(7, 1))
	for _, variant := range []Variant{{Key: "control", AllocationBasisPts: 5000}, {Key: "treatment", AllocationBasisPts: 5000}} {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO report_experiment_variant")).
			WithArgs(int64(7), experiment.Version, variant.Key, variant.AllocationBasisPts).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO report_experiment_audit")).
		WithArgs(uint64(7), experiment.Version, uint64(99), "Created", "reviewed test").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	id, err := CreateExperiment(context.Background(), db, experiment, 99, "reviewed test")
	if err != nil {
		t.Fatal(err)
	}
	if id != 7 {
		t.Fatalf("experiment id = %d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateExperimentRejectsCallerAssignmentNamespace(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	experiment := testExperiment()
	experiment.ID = 0
	experiment.Status = "Draft"
	if _, err := CreateExperiment(context.Background(), db, experiment, 99, "reviewed test"); err == nil {
		t.Fatal("caller-supplied assignment namespace was accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadExperimentReturnsValidatedRuntimeContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	experiment := testExperiment()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT experiment_id, owner_type, adv_id, experiment_name, experiment_version,")).
		WithArgs(experiment.ID).
		WillReturnRows(sqlmock.NewRows([]string{
			"experiment_id", "owner_type", "adv_id", "experiment_name", "experiment_version", "assignment_algorithm_version",
			"status", "assignment_salt", "primary_metric", "guardrail_metric", "retention_hours", "starts_at", "ends_at",
		}).AddRow(experiment.ID, experiment.OwnerType, nil, experiment.Name, experiment.Version,
			experiment.AssignmentAlgorithmVersion, experiment.Status, experiment.AssignmentSalt, experiment.PrimaryMetric, experiment.GuardrailMetric, experiment.RetentionHours, experiment.StartsAt, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT variant_key, allocation_basis_points")).
		WithArgs(experiment.ID, experiment.Version).
		WillReturnRows(sqlmock.NewRows([]string{"variant_key", "allocation_basis_points"}).
			AddRow("control", 5000).AddRow("treatment", 5000))
	loaded, err := LoadExperiment(context.Background(), db, experiment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AssignmentSalt != experiment.AssignmentSalt || len(loaded.Variants) != 2 || !loaded.StartsAt.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("loaded experiment = %#v", loaded)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListExperimentsIncludesAlgorithmWithoutSalt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT experiment_id, owner_type, adv_id, experiment_name, experiment_version, assignment_algorithm_version,")).
		WillReturnRows(sqlmock.NewRows([]string{
			"experiment_id", "owner_type", "adv_id", "experiment_name", "experiment_version", "assignment_algorithm_version",
			"status", "primary_metric", "guardrail_metric", "retention_hours", "starts_at", "ends_at",
		}).AddRow(7, "Operator", nil, "copy", 2, AssignmentAlgorithmV2, "Running", "actions", "spend", 2160, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), nil))
	items, err := ListExperiments(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].AssignmentAlgorithmVersion != AssignmentAlgorithmV2 {
		t.Fatalf("experiment summaries = %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionExperimentValidatesAllocationBeforeStart(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT experiment_version, status FROM report_experiment")).
		WithArgs(uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"experiment_version", "status"}).AddRow(2, "Draft"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*), COALESCE(SUM(allocation_basis_points),0)")).
		WithArgs(uint64(7), uint32(2)).WillReturnRows(sqlmock.NewRows([]string{"count", "allocation"}).AddRow(2, 9000))
	mock.ExpectRollback()
	if err := TransitionExperiment(context.Background(), db, 7, "Running", 99, "start"); err == nil {
		t.Fatal("incomplete experiment allocation was started")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPruneExpiredDeletesOutcomesThenExposuresTransactionally(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT exposure_id FROM report_exposure")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"exposure_id"}).AddRow(7).AddRow(9))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM report_experiment_outcome WHERE exposure_id IN (?,?)")).
		WithArgs(uint64(7), uint64(9)).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM report_exposure WHERE exposure_id IN (?,?)")).
		WithArgs(uint64(7), uint64(9)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	deleted, err := PruneExpired(context.Background(), db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d", deleted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteSubjectIsExactAndAuditedWithoutHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	hashText := "abababababababababababababababababababababababababababababababab"
	hash := make([]byte, 32)
	for index := range hash {
		hash[index] = 0xab
	}
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT exposure_id FROM report_exposure")).
		WithArgs(uint64(7), uint32(2), hash).
		WillReturnRows(sqlmock.NewRows([]string{"exposure_id"}).AddRow(41))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM report_experiment_outcome WHERE exposure_id=?")).
		WithArgs(uint64(41)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM report_exposure WHERE exposure_id=?")).
		WithArgs(uint64(41)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO report_experiment_audit")).
		WithArgs(uint64(7), uint32(2), uint64(99), "SubjectErased", "verified privacy request").
		WillReturnResult(sqlmock.NewResult(5, 1))
	mock.ExpectCommit()
	deleted, err := DeleteSubject(context.Background(), db, 7, 2, hashText, 99, "verified privacy request")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("existing experiment subject was not deleted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

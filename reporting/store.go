package reporting

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

func validateMetricName(name string) bool {
	for _, contract := range MetricContracts() {
		if contract.Name == name {
			return true
		}
	}
	return false
}

// CreateExperiment stores one Draft and its complete allocation atomically.
// Starting it is a separate reviewed transition.
func CreateExperiment(ctx context.Context, db *sql.DB, experiment Experiment, actorUID uint64, reason string) (uint64, error) {
	if db == nil {
		return 0, fmt.Errorf("experiment database is nil")
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return 0, fmt.Errorf("experiment creator and bounded reason are required")
	}
	if experiment.ID != 0 || experiment.Status != "Draft" {
		return 0, fmt.Errorf("new experiment must be an unassigned Draft")
	}
	if experiment.AssignmentSalt == "" {
		var err error
		experiment.AssignmentSalt, err = NewAssignmentSalt()
		if err != nil {
			return 0, err
		}
	}
	validation := experiment
	validation.ID = 1
	if err := validation.Validate(); err != nil {
		return 0, err
	}
	if !validateMetricName(experiment.PrimaryMetric) || !validateMetricName(experiment.GuardrailMetric) {
		return 0, fmt.Errorf("experiment metrics must use the R02 metric registry")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if experiment.AdvID != nil {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM adv WHERE adv_id=? AND active='Yes'`, *experiment.AdvID).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("experiment advertiser is not active")
			}
			return 0, err
		}
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO report_experiment
  (owner_type, adv_id, experiment_name, experiment_version, status,
   assignment_salt, primary_metric, guardrail_metric, retention_hours, starts_at, ends_at,
   created_by_uid, created_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,NOW(6))`,
		experiment.OwnerType, nullableUint32(experiment.AdvID), experiment.Name,
		experiment.Version, experiment.Status, experiment.AssignmentSalt,
		experiment.PrimaryMetric, experiment.GuardrailMetric, experiment.RetentionHours,
		experiment.StartsAt.UTC(), nullableTime(experiment.EndsAt), actorUID)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	variants := append([]Variant(nil), experiment.Variants...)
	sort.Slice(variants, func(i, j int) bool { return variants[i].Key < variants[j].Key })
	for _, variant := range variants {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO report_experiment_variant
  (experiment_id, experiment_version, variant_key, allocation_basis_points)
VALUES (?,?,?,?)`, id, experiment.Version, variant.Key, variant.AllocationBasisPts); err != nil {
			return 0, err
		}
	}
	if err := insertExperimentAudit(ctx, tx, uint64(id), experiment.Version, actorUID, "Created", reason); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(id), nil
}

// TransitionExperiment performs the explicit operator state transition. It
// never changes campaign bids, weights, budgets, or delivery policy.
func TransitionExperiment(ctx context.Context, db *sql.DB, experimentID uint64, target string, actorUID uint64, reason string) error {
	if db == nil {
		return fmt.Errorf("experiment database is nil")
	}
	if experimentID == 0 || strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return fmt.Errorf("experiment, actor, and bounded reason are required")
	}
	event := ""
	switch target {
	case "Running":
		event = "Started"
	case "Stopped":
		event = "Stopped"
	case "Completed":
		event = "Completed"
	default:
		return fmt.Errorf("experiment target status %q is invalid", target)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version uint32
	var current string
	if err := tx.QueryRowContext(ctx, `
SELECT experiment_version, status FROM report_experiment
WHERE experiment_id=? FOR UPDATE`, experimentID).Scan(&version, &current); err != nil {
		return err
	}
	allowed := (current == "Draft" && target == "Running") ||
		(current == "Running" && (target == "Stopped" || target == "Completed")) ||
		(current == "Stopped" && target == "Completed")
	if !allowed {
		return fmt.Errorf("experiment transition %s to %s is not allowed", current, target)
	}
	if target == "Running" {
		var count, allocation int
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*), COALESCE(SUM(allocation_basis_points),0)
FROM report_experiment_variant
WHERE experiment_id=? AND experiment_version=?`, experimentID, version).Scan(&count, &allocation); err != nil {
			return err
		}
		if count < 2 || count > 20 || allocation != allocationBasisPoints {
			return fmt.Errorf("experiment allocation is incomplete")
		}
	}
	stopReason := interface{}(nil)
	if target == "Stopped" || target == "Completed" {
		stopReason = reason
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE report_experiment SET status=?, stop_reason=? WHERE experiment_id=?`, target, stopReason, experimentID); err != nil {
		return err
	}
	if err := insertExperimentAudit(ctx, tx, experimentID, version, actorUID, event, reason); err != nil {
		return err
	}
	return tx.Commit()
}

type ExperimentSummary struct {
	ID              uint64
	OwnerType       string
	AdvID           *uint32
	Name            string
	Version         uint32
	Status          string
	PrimaryMetric   string
	GuardrailMetric string
	RetentionHours  uint32
	StartsAt        time.Time
	EndsAt          *time.Time
}

// LoadExperiment returns the complete runtime assignment contract for one
// version. Unlike ListExperiments, this privileged runtime path includes the
// assignment salt needed to compute a domain-separated subject hash.
func LoadExperiment(ctx context.Context, db *sql.DB, experimentID uint64) (Experiment, error) {
	if db == nil {
		return Experiment{}, fmt.Errorf("experiment database is nil")
	}
	if experimentID == 0 {
		return Experiment{}, fmt.Errorf("experiment id is required")
	}
	var out Experiment
	var advID sql.NullInt64
	var endsAt sql.NullTime
	if err := db.QueryRowContext(ctx, `
SELECT experiment_id, owner_type, adv_id, experiment_name, experiment_version,
       status, assignment_salt, primary_metric, guardrail_metric, retention_hours, starts_at, ends_at
FROM report_experiment
WHERE experiment_id=?`, experimentID).Scan(
		&out.ID, &out.OwnerType, &advID, &out.Name, &out.Version, &out.Status,
		&out.AssignmentSalt, &out.PrimaryMetric, &out.GuardrailMetric, &out.RetentionHours, &out.StartsAt, &endsAt); err != nil {
		return Experiment{}, err
	}
	if advID.Valid {
		value := uint32(advID.Int64)
		out.AdvID = &value
	}
	if endsAt.Valid {
		value := endsAt.Time.UTC()
		out.EndsAt = &value
	}
	out.StartsAt = out.StartsAt.UTC()
	rows, err := db.QueryContext(ctx, `
SELECT variant_key, allocation_basis_points
FROM report_experiment_variant
WHERE experiment_id=? AND experiment_version=?
ORDER BY variant_key`, out.ID, out.Version)
	if err != nil {
		return Experiment{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var variant Variant
		if err := rows.Scan(&variant.Key, &variant.AllocationBasisPts); err != nil {
			return Experiment{}, err
		}
		out.Variants = append(out.Variants, variant)
	}
	if err := rows.Err(); err != nil {
		return Experiment{}, err
	}
	if err := out.Validate(); err != nil {
		return Experiment{}, fmt.Errorf("stored experiment is invalid: %w", err)
	}
	if !validateMetricName(out.PrimaryMetric) || !validateMetricName(out.GuardrailMetric) {
		return Experiment{}, fmt.Errorf("stored experiment metrics are not in the R02 registry")
	}
	return out, nil
}

// ListExperiments returns bounded configuration metadata only. Assignment
// salts, subject hashes, exposures, and audit reasons are deliberately absent.
func ListExperiments(ctx context.Context, db *sql.DB) ([]ExperimentSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("experiment database is nil")
	}
	rows, err := db.QueryContext(ctx, `
SELECT experiment_id, owner_type, adv_id, experiment_name, experiment_version,
       status, primary_metric, guardrail_metric, retention_hours, starts_at, ends_at
FROM report_experiment
ORDER BY experiment_id, experiment_version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ExperimentSummary, 0)
	for rows.Next() {
		var item ExperimentSummary
		var advID sql.NullInt64
		var endsAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.OwnerType, &advID, &item.Name, &item.Version, &item.Status, &item.PrimaryMetric, &item.GuardrailMetric, &item.RetentionHours, &item.StartsAt, &endsAt); err != nil {
			return nil, err
		}
		if advID.Valid {
			value := uint32(advID.Int64)
			item.AdvID = &value
		}
		if endsAt.Valid {
			value := endsAt.Time.UTC()
			item.EndsAt = &value
		}
		item.StartsAt = item.StartsAt.UTC()
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// PruneExpired removes a bounded batch of expired pseudonymous outcomes and
// exposures in one transaction. Experiment definitions and non-subject audit
// records remain intact.
func PruneExpired(ctx context.Context, db *sql.DB, limit int) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("experiment database is nil")
	}
	if limit < 1 || limit > 10_000 {
		return 0, fmt.Errorf("experiment prune limit must be between 1 and 10000")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
SELECT exposure_id FROM report_exposure
WHERE expires_at<=UTC_TIMESTAMP(6)
ORDER BY exposure_id LIMIT ? FOR UPDATE`, limit)
	if err != nil {
		return 0, err
	}
	ids := make([]uint64, 0, limit)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	arguments := make([]interface{}, len(ids))
	for index, id := range ids {
		arguments[index] = id
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM report_experiment_outcome WHERE exposure_id IN (`+placeholders+`)`, arguments...); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM report_exposure WHERE exposure_id IN (`+placeholders+`)`, arguments...)
	if err != nil {
		return 0, err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return deleted, nil
}

// DeleteSubject removes one exact experiment-version subject hash and its
// outcomes for an authorized privacy request, then records a non-identifying
// audit event. It never scans or returns neighboring subjects.
func DeleteSubject(ctx context.Context, db *sql.DB, experimentID uint64, version uint32, subjectHash string, actorUID uint64, reason string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("experiment database is nil")
	}
	if experimentID == 0 || version == 0 || len(subjectHash) != 64 || strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return false, fmt.Errorf("experiment, version, subject hash, and bounded reason are required")
	}
	hash, err := hex.DecodeString(subjectHash)
	if err != nil {
		return false, fmt.Errorf("experiment subject hash is invalid")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var exposureID uint64
	err = tx.QueryRowContext(ctx, `
SELECT exposure_id FROM report_exposure
WHERE experiment_id=? AND experiment_version=? AND subject_hash=?
FOR UPDATE`, experimentID, version, hash).Scan(&exposureID)
	if err == sql.ErrNoRows {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM report_experiment_outcome WHERE exposure_id=?`, exposureID); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM report_exposure WHERE exposure_id=?`, exposureID); err != nil {
		return false, err
	}
	if err := insertExperimentAudit(ctx, tx, experimentID, version, actorUID, "SubjectErased", reason); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func insertExperimentAudit(ctx context.Context, tx *sql.Tx, experimentID uint64, version uint32, actorUID uint64, event, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO report_experiment_audit
  (experiment_id, experiment_version, actor_uid, event, reason, created_at)
VALUES (?,?,?,?,?,NOW(6))`, experimentID, version, actorUID, event, reason)
	return err
}

func nullableUint32(value *uint32) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return value.UTC()
}

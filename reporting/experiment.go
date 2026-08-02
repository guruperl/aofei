package reporting

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const allocationBasisPoints = 10_000

const (
	minExperimentRetentionHours = 24
	maxExperimentRetentionHours = 400 * 24
)

var outcomeDecimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,13})\.[0-9]{6}$`)

type Variant struct {
	Key                string
	AllocationBasisPts uint16
}

type Experiment struct {
	ID              uint64
	OwnerType       string
	AdvID           *uint32
	Name            string
	Version         uint32
	Status          string
	AssignmentSalt  string
	PrimaryMetric   string
	GuardrailMetric string
	RetentionHours  uint32
	StartsAt        time.Time
	EndsAt          *time.Time
	Variants        []Variant
}

type Assignment struct {
	ExperimentID uint64
	Version      uint32
	VariantKey   string
	SubjectHash  [32]byte
	AssignedAt   time.Time
	ExpiresAt    time.Time
}

// Outcome is one immutable observed metric value for an exposed subject. It
// carries only the exposure's domain-separated hash and a caller-supplied
// idempotency digest; raw subject and event identifiers are never retained.
type Outcome struct {
	Assignment     Assignment
	MetricName     string
	MetricValue    string
	IdempotencyKey [32]byte
	OccurredAt     time.Time
}

// NewAssignmentSalt returns a non-secret random namespace used to prevent a
// pseudonym from receiving correlated buckets across experiments.
func NewAssignmentSalt() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func (e Experiment) Validate() error {
	if e.ID == 0 || e.Version == 0 || strings.TrimSpace(e.Name) == "" || len(e.Name) > 128 || strings.IndexFunc(e.Name, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return fmt.Errorf("experiment identity, version, and bounded name are required")
	}
	switch e.OwnerType {
	case "Operator":
		if e.AdvID != nil {
			return fmt.Errorf("operator experiment cannot have an advertiser owner")
		}
	case "Advertiser":
		if e.AdvID == nil || *e.AdvID == 0 {
			return fmt.Errorf("advertiser experiment requires an advertiser owner")
		}
	default:
		return fmt.Errorf("experiment owner type %q is invalid", e.OwnerType)
	}
	switch e.Status {
	case "Draft", "Running", "Stopped", "Completed":
	default:
		return fmt.Errorf("experiment status %q is invalid", e.Status)
	}
	if len(e.AssignmentSalt) != 32 {
		return fmt.Errorf("experiment assignment salt must be 16-byte hexadecimal")
	}
	if _, err := hex.DecodeString(e.AssignmentSalt); err != nil {
		return fmt.Errorf("experiment assignment salt is invalid")
	}
	if e.PrimaryMetric == "" || e.GuardrailMetric == "" {
		return fmt.Errorf("experiment primary and guardrail metrics are required")
	}
	if e.RetentionHours < minExperimentRetentionHours || e.RetentionHours > maxExperimentRetentionHours {
		return fmt.Errorf("experiment retention must be between 24 and 9600 hours")
	}
	if e.StartsAt.IsZero() {
		return fmt.Errorf("experiment start is required")
	}
	if e.EndsAt != nil && !e.EndsAt.After(e.StartsAt) {
		return fmt.Errorf("experiment end must be after its start")
	}
	if len(e.Variants) < 2 || len(e.Variants) > 20 {
		return fmt.Errorf("experiment must have between 2 and 20 variants")
	}
	keys := make(map[string]struct{}, len(e.Variants))
	total := 0
	for _, variant := range e.Variants {
		if !validVariantKey(variant.Key) || variant.AllocationBasisPts == 0 {
			return fmt.Errorf("experiment variant %q is invalid", variant.Key)
		}
		if _, exists := keys[variant.Key]; exists {
			return fmt.Errorf("experiment variant %q is duplicated", variant.Key)
		}
		keys[variant.Key] = struct{}{}
		total += int(variant.AllocationBasisPts)
	}
	if total != allocationBasisPoints {
		return fmt.Errorf("experiment variant allocation is %d, want %d basis points", total, allocationBasisPoints)
	}
	return nil
}

func validVariantKey(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for index, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (index > 0 && r >= '0' && r <= '9') || (index > 0 && (r == '_' || r == '-')) {
			continue
		}
		return false
	}
	return true
}

// Assign deterministically chooses a variant for one 32-byte hexadecimal
// pseudonym. The raw pseudonym is never returned or stored.
func Assign(e Experiment, subjectPseudonym string, at time.Time) (Assignment, error) {
	if err := e.Validate(); err != nil {
		return Assignment{}, err
	}
	if e.Status != "Running" || at.Before(e.StartsAt) || (e.EndsAt != nil && !at.Before(*e.EndsAt)) {
		return Assignment{}, fmt.Errorf("experiment is not accepting exposures")
	}
	if len(subjectPseudonym) != 64 {
		return Assignment{}, fmt.Errorf("experiment subject must be a 32-byte hexadecimal pseudonym")
	}
	subject, err := hex.DecodeString(subjectPseudonym)
	if err != nil {
		return Assignment{}, fmt.Errorf("experiment subject pseudonym is invalid")
	}
	hashInput := make([]byte, 0, len(e.AssignmentSalt)+len(subject)+24)
	hashInput = append(hashInput, e.AssignmentSalt...)
	hashInput = append(hashInput, 0)
	hashInput = append(hashInput, subject...)
	var version [4]byte
	binary.BigEndian.PutUint32(version[:], e.Version)
	hashInput = append(hashInput, version[:]...)
	subjectHash := sha256.Sum256(hashInput)
	bucket := int(binary.BigEndian.Uint64(subjectHash[:8]) % allocationBasisPoints)
	cumulative := 0
	for _, variant := range e.Variants {
		cumulative += int(variant.AllocationBasisPts)
		if bucket < cumulative {
			assignedAt := at.UTC().Truncate(time.Microsecond)
			return Assignment{
				ExperimentID: e.ID, Version: e.Version, VariantKey: variant.Key,
				SubjectHash: subjectHash, AssignedAt: assignedAt,
				ExpiresAt: assignedAt.Add(time.Duration(e.RetentionHours) * time.Hour),
			}, nil
		}
	}
	return Assignment{}, fmt.Errorf("experiment allocation did not cover bucket")
}

// RecordExposure inserts one immutable exposure per experiment version and
// subject hash. Repeated calls return the existing variant and reject a
// conflicting assignment.
func RecordExposure(ctx context.Context, db *sql.DB, assignment Assignment) error {
	if db == nil {
		return fmt.Errorf("experiment database is nil")
	}
	if assignment.ExperimentID == 0 || assignment.Version == 0 || !validVariantKey(assignment.VariantKey) || assignment.AssignedAt.IsZero() || !assignment.ExpiresAt.After(assignment.AssignedAt) {
		return fmt.Errorf("experiment assignment is incomplete")
	}
	if _, err := db.ExecContext(ctx, `
INSERT IGNORE INTO report_exposure
  (experiment_id, experiment_version, subject_hash, variant_key, exposed_at, expires_at)
VALUES (?,?,?,?,?,?)`, assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:], assignment.VariantKey, assignment.AssignedAt.UTC(), assignment.ExpiresAt.UTC()); err != nil {
		return err
	}
	var stored string
	if err := db.QueryRowContext(ctx, `
SELECT variant_key FROM report_exposure
WHERE experiment_id=? AND experiment_version=? AND subject_hash=?`, assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:]).Scan(&stored); err != nil {
		return err
	}
	if stored != assignment.VariantKey {
		return fmt.Errorf("experiment exposure conflicts with existing variant")
	}
	return nil
}

// NewOutcome validates and normalizes an observational experiment result. The
// fixed six-decimal string avoids binary floating-point ambiguity at the SQL
// DECIMAL(20,6) boundary.
func NewOutcome(assignment Assignment, metricName, metricValue, idempotencyDigest string, occurredAt time.Time) (Outcome, error) {
	if assignment.ExperimentID == 0 || assignment.Version == 0 || !validVariantKey(assignment.VariantKey) || assignment.AssignedAt.IsZero() || !assignment.ExpiresAt.After(assignment.AssignedAt) {
		return Outcome{}, fmt.Errorf("experiment assignment is incomplete")
	}
	if !validateMetricName(metricName) {
		return Outcome{}, fmt.Errorf("experiment outcome metric is not in the R02 registry")
	}
	if !outcomeDecimalPattern.MatchString(metricValue) {
		return Outcome{}, fmt.Errorf("experiment outcome value must be a signed DECIMAL(20,6)")
	}
	if len(idempotencyDigest) != 64 {
		return Outcome{}, fmt.Errorf("experiment outcome idempotency key must be a 32-byte hexadecimal digest")
	}
	decoded, err := hex.DecodeString(idempotencyDigest)
	if err != nil {
		return Outcome{}, fmt.Errorf("experiment outcome idempotency key is invalid")
	}
	if occurredAt.IsZero() || occurredAt.Before(assignment.AssignedAt) || !occurredAt.Before(assignment.ExpiresAt) {
		return Outcome{}, fmt.Errorf("experiment outcome must occur after exposure and before expiry")
	}
	var key [32]byte
	copy(key[:], decoded)
	return Outcome{
		Assignment: assignment, MetricName: metricName, MetricValue: metricValue,
		IdempotencyKey: key, OccurredAt: occurredAt.UTC().Truncate(time.Microsecond),
	}, nil
}

// RecordOutcome attaches one immutable, idempotent metric observation to an
// existing exposure. Only the experiment's declared primary or guardrail
// metric is accepted. This function never changes serving or accounting state.
func RecordOutcome(ctx context.Context, db *sql.DB, outcome Outcome) error {
	if db == nil {
		return fmt.Errorf("experiment database is nil")
	}
	if outcome.Assignment.ExperimentID == 0 || outcome.Assignment.Version == 0 ||
		!validVariantKey(outcome.Assignment.VariantKey) || outcome.Assignment.AssignedAt.IsZero() ||
		!outcome.Assignment.ExpiresAt.After(outcome.Assignment.AssignedAt) ||
		!validateMetricName(outcome.MetricName) || !outcomeDecimalPattern.MatchString(outcome.MetricValue) ||
		outcome.OccurredAt.IsZero() || outcome.OccurredAt.Before(outcome.Assignment.AssignedAt) ||
		!outcome.OccurredAt.Before(outcome.Assignment.ExpiresAt) {
		return fmt.Errorf("experiment outcome is incomplete")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exposureID uint64
	var storedVariant, primaryMetric, guardrailMetric string
	if err := tx.QueryRowContext(ctx, `
SELECT x.exposure_id, x.variant_key, e.primary_metric, e.guardrail_metric
FROM report_exposure x
INNER JOIN report_experiment e
  ON (e.experiment_id=x.experiment_id AND e.experiment_version=x.experiment_version)
WHERE x.experiment_id=? AND x.experiment_version=? AND x.subject_hash=?
FOR UPDATE`, outcome.Assignment.ExperimentID, outcome.Assignment.Version, outcome.Assignment.SubjectHash[:]).Scan(
		&exposureID, &storedVariant, &primaryMetric, &guardrailMetric); err != nil {
		return err
	}
	if storedVariant != outcome.Assignment.VariantKey {
		return fmt.Errorf("experiment outcome conflicts with stored exposure variant")
	}
	if outcome.MetricName != primaryMetric && outcome.MetricName != guardrailMetric {
		return fmt.Errorf("experiment outcome metric is not declared by the experiment")
	}
	if _, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO report_experiment_outcome
  (exposure_id, metric_name, metric_value, idempotency_key, occurred_at)
VALUES (?,?,?,?,?)`, exposureID, outcome.MetricName, outcome.MetricValue, outcome.IdempotencyKey[:], outcome.OccurredAt.UTC()); err != nil {
		return err
	}
	var storedMetric, storedValue string
	var storedAt time.Time
	if err := tx.QueryRowContext(ctx, `
SELECT metric_name, metric_value, occurred_at
FROM report_experiment_outcome
WHERE exposure_id=? AND idempotency_key=?`, exposureID, outcome.IdempotencyKey[:]).Scan(&storedMetric, &storedValue, &storedAt); err != nil {
		return err
	}
	if storedMetric != outcome.MetricName || storedValue != outcome.MetricValue || !storedAt.UTC().Equal(outcome.OccurredAt.UTC()) {
		return fmt.Errorf("experiment outcome conflicts with existing idempotency key")
	}
	return tx.Commit()
}

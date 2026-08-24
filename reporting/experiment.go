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
	AssignmentAlgorithmV1 uint16 = 1
	AssignmentAlgorithmV2 uint16 = 2

	currentAssignmentAlgorithm = AssignmentAlgorithmV2
	assignmentDomainV2         = "aofei/reporting/experiment-assignment/v2\x00"
)

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
	ID                         uint64
	OwnerType                  string
	AdvID                      *uint32
	Name                       string
	Version                    uint32
	AssignmentAlgorithmVersion uint16
	Status                     string
	AssignmentSalt             string
	PrimaryMetric              string
	GuardrailMetric            string
	RetentionHours             uint32
	StartsAt                   time.Time
	EndsAt                     *time.Time
	Variants                   []Variant
}

type Assignment struct {
	ExperimentID     uint64
	Version          uint32
	AlgorithmVersion uint16
	OwnerType        string
	OwnerID          uint32
	VariantKey       string
	SubjectHash      [32]byte
	AssignedAt       time.Time
	ExpiresAt        time.Time
	proof            [32]byte
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
	if e.AssignmentAlgorithmVersion != AssignmentAlgorithmV1 && e.AssignmentAlgorithmVersion != AssignmentAlgorithmV2 {
		return fmt.Errorf("experiment assignment algorithm version %d is invalid", e.AssignmentAlgorithmVersion)
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
	hashInput, err := assignmentHashInput(e, subject)
	if err != nil {
		return Assignment{}, err
	}
	subjectHash := sha256.Sum256(hashInput)
	bucket := int(binary.BigEndian.Uint64(subjectHash[:8]) % allocationBasisPoints)
	cumulative := 0
	for _, variant := range e.Variants {
		cumulative += int(variant.AllocationBasisPts)
		if bucket < cumulative {
			assignedAt := at.UTC().Truncate(time.Microsecond)
			assignment := Assignment{
				ExperimentID: e.ID, Version: e.Version, AlgorithmVersion: e.AssignmentAlgorithmVersion, VariantKey: variant.Key,
				SubjectHash: subjectHash, AssignedAt: assignedAt,
				ExpiresAt: assignedAt.Add(time.Duration(e.RetentionHours) * time.Hour),
				OwnerType: e.OwnerType,
			}
			if e.AdvID != nil {
				assignment.OwnerID = *e.AdvID
			}
			assignment.proof = assignmentProof(e.AssignmentSalt, assignment)
			return assignment, nil
		}
	}
	return Assignment{}, fmt.Errorf("experiment allocation did not cover bucket")
}

func assignmentProof(saltText string, assignment Assignment) [32]byte {
	salt, _ := hex.DecodeString(saltText)
	input := make([]byte, 0, 160)
	input = append(input, "aofei/reporting/assignment-proof/v1\x00"...)
	var numbers [34]byte
	binary.BigEndian.PutUint64(numbers[0:8], assignment.ExperimentID)
	binary.BigEndian.PutUint32(numbers[8:12], assignment.Version)
	binary.BigEndian.PutUint16(numbers[12:14], assignment.AlgorithmVersion)
	binary.BigEndian.PutUint32(numbers[14:18], assignment.OwnerID)
	binary.BigEndian.PutUint64(numbers[18:26], uint64(assignment.AssignedAt.UTC().UnixMicro()))
	binary.BigEndian.PutUint64(numbers[26:34], uint64(assignment.ExpiresAt.UTC().UnixMicro()))
	input = append(input, numbers[:]...)
	input = append(input, assignment.SubjectHash[:]...)
	input = append(input, 0)
	input = append(input, assignment.OwnerType...)
	input = append(input, 0)
	input = append(input, assignment.VariantKey...)
	input = append(input, 0)
	input = append(input, salt...)
	return sha256.Sum256(input)
}

func assignmentHashInput(e Experiment, subject []byte) ([]byte, error) {
	if e.AssignmentAlgorithmVersion == AssignmentAlgorithmV1 {
		input := make([]byte, 0, len(e.AssignmentSalt)+len(subject)+5)
		input = append(input, e.AssignmentSalt...)
		input = append(input, 0)
		input = append(input, subject...)
		var version [4]byte
		binary.BigEndian.PutUint32(version[:], e.Version)
		return append(input, version[:]...), nil
	}
	salt, err := hex.DecodeString(e.AssignmentSalt)
	if err != nil {
		return nil, fmt.Errorf("experiment assignment salt is invalid")
	}
	input := make([]byte, 0, len(assignmentDomainV2)+8+2+4+len(salt)+len(subject))
	input = append(input, assignmentDomainV2...)
	var identity [14]byte
	binary.BigEndian.PutUint64(identity[0:8], e.ID)
	binary.BigEndian.PutUint16(identity[8:10], e.AssignmentAlgorithmVersion)
	binary.BigEndian.PutUint32(identity[10:14], e.Version)
	input = append(input, identity[:]...)
	input = append(input, salt...)
	return append(input, subject...), nil
}

// RecordExposure inserts one immutable exposure per experiment version and
// subject hash. Repeated calls return the existing variant and reject a
// conflicting assignment.
func RecordExposure(ctx context.Context, db *sql.DB, assignment Assignment) error {
	if db == nil {
		return fmt.Errorf("experiment database is nil")
	}
	if !validAssignment(assignment) {
		return fmt.Errorf("experiment assignment is incomplete")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	contract, err := loadAssignmentContract(ctx, tx, assignment)
	if err != nil {
		return err
	}
	if err := contract.validate(assignment); err != nil {
		return err
	}
	if contract.status == "Running" {
		if _, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO report_exposure
  (experiment_id, experiment_version, subject_hash, variant_key, exposed_at, expires_at)
VALUES (?,?,?,?,?,?)`, assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:], assignment.VariantKey, assignment.AssignedAt.UTC(), assignment.ExpiresAt.UTC()); err != nil {
			return err
		}
	}
	var exposureID uint64
	var storedVariant string
	var storedAt, storedExpires time.Time
	err = tx.QueryRowContext(ctx, `
SELECT exposure_id, variant_key, exposed_at, expires_at
FROM report_exposure
WHERE experiment_id=? AND experiment_version=? AND subject_hash=?
FOR UPDATE`, assignment.ExperimentID, assignment.Version, assignment.SubjectHash[:]).Scan(&exposureID, &storedVariant, &storedAt, &storedExpires)
	if err == sql.ErrNoRows {
		return fmt.Errorf("experiment is not accepting exposures")
	}
	if err != nil {
		return err
	}
	if storedVariant != assignment.VariantKey || !storedAt.UTC().Equal(assignment.AssignedAt.UTC()) || !storedExpires.UTC().Equal(assignment.ExpiresAt.UTC()) {
		return fmt.Errorf("experiment exposure conflicts with existing assignment")
	}
	return tx.Commit()
}

type assignmentContract struct {
	ownerType      string
	ownerID        uint32
	algorithm      uint16
	status         string
	salt           string
	retentionHours uint32
	startsAt       time.Time
	endsAt         sql.NullTime
}

func loadAssignmentContract(ctx context.Context, tx *sql.Tx, assignment Assignment) (assignmentContract, error) {
	var contract assignmentContract
	var advID sql.NullInt64
	err := tx.QueryRowContext(ctx, `
SELECT e.owner_type, e.adv_id, e.assignment_algorithm_version, e.status,
       e.assignment_salt, e.retention_hours, e.starts_at, e.ends_at
FROM report_experiment e
INNER JOIN report_experiment_variant v
  ON (v.experiment_id=e.experiment_id AND v.experiment_version=e.experiment_version)
WHERE e.experiment_id=? AND e.experiment_version=? AND v.variant_key=?
FOR SHARE`, assignment.ExperimentID, assignment.Version, assignment.VariantKey).Scan(
		&contract.ownerType, &advID, &contract.algorithm, &contract.status,
		&contract.salt, &contract.retentionHours, &contract.startsAt, &contract.endsAt)
	if err != nil {
		return assignmentContract{}, err
	}
	if advID.Valid {
		contract.ownerID = uint32(advID.Int64)
	}
	return contract, nil
}

func (contract assignmentContract) validate(assignment Assignment) error {
	if assignment.OwnerType != contract.ownerType || assignment.OwnerID != contract.ownerID {
		return fmt.Errorf("experiment assignment owner scope does not match")
	}
	if assignment.AlgorithmVersion != contract.algorithm {
		return fmt.Errorf("experiment assignment algorithm does not match")
	}
	if assignment.AssignedAt.Before(contract.startsAt.UTC()) || (contract.endsAt.Valid && !assignment.AssignedAt.Before(contract.endsAt.Time.UTC())) {
		return fmt.Errorf("experiment assignment is outside its version window")
	}
	expires := assignment.AssignedAt.UTC().Add(time.Duration(contract.retentionHours) * time.Hour)
	if !assignment.ExpiresAt.UTC().Equal(expires) {
		return fmt.Errorf("experiment assignment retention does not match")
	}
	if assignment.proof != assignmentProof(contract.salt, assignment) {
		return fmt.Errorf("experiment assignment proof does not match")
	}
	return nil
}

func validAssignment(assignment Assignment) bool {
	return assignment.ExperimentID != 0 && assignment.Version != 0 &&
		validAssignmentAlgorithm(assignment.AlgorithmVersion) &&
		(assignment.OwnerType == "Operator" || (assignment.OwnerType == "Advertiser" && assignment.OwnerID != 0)) &&
		assignment.SubjectHash != [32]byte{} && assignment.proof != [32]byte{} &&
		validVariantKey(assignment.VariantKey) && !assignment.AssignedAt.IsZero() &&
		assignment.ExpiresAt.After(assignment.AssignedAt)
}

// NewOutcome validates and normalizes an observational experiment result. The
// fixed six-decimal string avoids binary floating-point ambiguity at the SQL
// DECIMAL(20,6) boundary.
func NewOutcome(assignment Assignment, metricName, metricValue, idempotencyDigest string, occurredAt time.Time) (Outcome, error) {
	if !validAssignment(assignment) {
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
	if !validAssignment(outcome.Assignment) ||
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
	var storedAt, storedExpires time.Time
	var contract assignmentContract
	var advID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
SELECT x.exposure_id, x.variant_key, x.exposed_at, x.expires_at,
       e.owner_type, e.adv_id, e.assignment_algorithm_version, e.status,
       e.assignment_salt, e.retention_hours, e.starts_at, e.ends_at,
       e.primary_metric, e.guardrail_metric
FROM report_exposure x
INNER JOIN report_experiment e
  ON (e.experiment_id=x.experiment_id AND e.experiment_version=x.experiment_version)
WHERE x.experiment_id=? AND x.experiment_version=? AND x.subject_hash=?
FOR UPDATE`, outcome.Assignment.ExperimentID, outcome.Assignment.Version, outcome.Assignment.SubjectHash[:]).Scan(
		&exposureID, &storedVariant, &storedAt, &storedExpires,
		&contract.ownerType, &advID, &contract.algorithm, &contract.status,
		&contract.salt, &contract.retentionHours, &contract.startsAt, &contract.endsAt,
		&primaryMetric, &guardrailMetric); err != nil {
		return err
	}
	if advID.Valid {
		contract.ownerID = uint32(advID.Int64)
	}
	if err := contract.validate(outcome.Assignment); err != nil {
		return err
	}
	if storedVariant != outcome.Assignment.VariantKey || !storedAt.UTC().Equal(outcome.Assignment.AssignedAt.UTC()) || !storedExpires.UTC().Equal(outcome.Assignment.ExpiresAt.UTC()) {
		return fmt.Errorf("experiment outcome conflicts with stored exposure assignment")
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
	var storedOutcomeAt time.Time
	if err := tx.QueryRowContext(ctx, `
SELECT metric_name, metric_value, occurred_at
FROM report_experiment_outcome
WHERE exposure_id=? AND idempotency_key=?`, exposureID, outcome.IdempotencyKey[:]).Scan(&storedMetric, &storedValue, &storedOutcomeAt); err != nil {
		return err
	}
	if storedMetric != outcome.MetricName || storedValue != outcome.MetricValue || !storedOutcomeAt.UTC().Equal(outcome.OccurredAt.UTC()) {
		return fmt.Errorf("experiment outcome conflicts with existing idempotency key")
	}
	return tx.Commit()
}

func validAssignmentAlgorithm(version uint16) bool {
	return version == AssignmentAlgorithmV1 || version == AssignmentAlgorithmV2
}

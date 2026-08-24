package trafficquality

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"
)

const retentionCleanupTimeout = 2 * time.Second

const (
	PermissionEvidenceRead        = "quality.evidence.read"
	PermissionRuleDraft           = "quality.rule.draft"
	PermissionRuleActivate        = "quality.rule.activate"
	PermissionReviewResolve       = "quality.review.resolve"
	PermissionAppealSubmit        = "quality.appeal.submit"
	PermissionAppealResolve       = "quality.appeal.resolve"
	PermissionEnforcementActivate = "quality.enforcement.activate"
	PermissionEnforcementRollback = "quality.enforcement.rollback"
	PermissionBillingRecommend    = "quality.billing.recommend"
	PermissionBillingApprove      = "quality.billing.approve"
	PermissionRetentionPrune      = "quality.retention.prune"
)

type Service struct {
	DB     *sql.DB
	engine *Engine
	now    func() time.Time
}

type Case struct {
	ID                uint64
	DecisionID        uint64
	Scope             Scope
	Status            ReviewStatus
	OpenedBy          string
	ResolvedBy        sql.NullString
	Version           uint32
	RuleKey           string
	RuleVersion       uint32
	Signal            Signal
	ConfiguredAction  Action
	AppliedAction     Action
	Evidence          EvidenceState
	ReasonCode        string
	ObservedValue     float64
	Threshold         float64
	ObservedAt        time.Time
	EvidenceExpiresAt time.Time
	SafeSummary       sql.NullString
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Enforcement struct {
	ID                uint64
	RuleID            uint64
	DecisionID        uint64
	Scope             Scope
	Action            Action
	State             string
	CanaryBasisPoints uint16
	CreatedBy         string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

type BillingRecommendation struct {
	ID                uint64
	DecisionID        uint64
	StatementID       uint64
	Disposition       BillingDisposition
	State             string
	RecommendedBy     string
	ApprovedBy        sql.NullString
	AccountingVersion string
}

type RuleHealth struct {
	RuleID                   uint64
	RuleKey                  string
	RuleVersion              uint32
	Mode                     RuleMode
	Decisions                uint64
	Matched                  uint64
	Enforced                 uint64
	ValidTraffic             uint64
	InvalidTraffic           uint64
	Appeals                  uint64
	AppealsUpheld            uint64
	FalsePositiveBasisPoints uint16
	FalsePositiveLimitBPS    uint16
	RollbackRecommended      bool
}

func NewServiceWithKey(db *sql.DB, key []byte) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("traffic-quality database is nil")
	}
	engine, err := NewEngine(key)
	if err != nil {
		return nil, err
	}
	return &Service{DB: db, engine: engine, now: time.Now}, nil
}

func (s *Service) CreateRule(ctx context.Context, actor Actor, rule Rule, reason string) (Rule, error) {
	if err := requireActor(actor, PermissionRuleDraft, false); err != nil {
		return Rule{}, err
	}
	if actor.Role != "admin" {
		return Rule{}, ErrForbidden
	}
	if !validAuditReason(reason) {
		return Rule{}, fmt.Errorf("a bounded single-line reason is required")
	}
	rule.ID = 0
	rule.Version = 1
	rule.Mode = ModeDraft
	rule.CanaryBasisPoints = 0
	rule.CreatedBy = actor.Name()
	rule.CreatedAt = s.currentTime()
	if err := rule.Validate(); err != nil {
		return Rule{}, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return Rule{}, err
	}
	defer tx.Rollback()
	var priorID sql.NullInt64
	var version uint32
	err = tx.QueryRowContext(ctx, `
SELECT rule_id, rule_version FROM quality_rule
WHERE rule_key=? ORDER BY rule_version DESC LIMIT 1 FOR UPDATE`, rule.Key).Scan(&priorID, &version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Rule{}, err
	}
	if priorID.Valid {
		rule.Version = version + 1
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO quality_rule
 (rule_key, rule_version, signal_type, rule_action, rollout_mode, scope_type, scope_id,
  threshold_value, window_seconds, canary_basis_points, reason_code,
  evidence_retention_hours, aggregate_retention_days,
  false_positive_limit_bps, supersedes_rule_id, created_by, created_at, updated_at)
VALUES (?,?,?,?,'Draft',?,?,?,?,0,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`,
		rule.Key, rule.Version, rule.Signal, rule.Action, rule.Scope.Type, rule.Scope.ID,
		rule.Threshold, rule.WindowSeconds, rule.ReasonCode, rule.EvidenceRetentionHrs,
		rule.AggregateRetentionDays, rule.FalsePositiveLimitBPS, nullableInt64(priorID),
		actor.Name())
	if err != nil {
		return Rule{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Rule{}, err
	}
	rule.ID = uint64(id)
	if err := insertAudit(ctx, tx, actor, "RuleCreated", "Rule", rule.ID, rule.Scope, "Absent", string(ModeDraft), reason); err != nil {
		return Rule{}, err
	}
	if err := tx.Commit(); err != nil {
		return Rule{}, err
	}
	return rule, nil
}

func (s *Service) SetRuleMode(ctx context.Context, actor Actor, ruleID uint64, expected, next RuleMode, canaryBPS uint16, reason string) error {
	if err := requireActor(actor, PermissionRuleActivate, true); err != nil {
		return err
	}
	if actor.Role != "admin" || ruleID == 0 || !allModes[expected] || !allModes[next] || !validAuditReason(reason) {
		return ErrForbidden
	}
	if next == ModeDraft || (next == ModeCanary && canaryBPS == 0) || canaryBPS > 10_000 {
		return fmt.Errorf("invalid traffic-quality rollout target")
	}
	if next != ModeCanary {
		canaryBPS = 0
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var scope Scope
	var ruleAction Action
	var falsePositiveLimit uint16
	if err := tx.QueryRowContext(ctx, `SELECT scope_type, scope_id, rule_action, false_positive_limit_bps FROM quality_rule WHERE rule_id=? FOR UPDATE`, ruleID).Scan(&scope.Type, &scope.ID, &ruleAction, &falsePositiveLimit); err != nil {
		return normalizeNotFound(err)
	}
	if !ruleModeTransitionAllowed(expected, next) {
		return fmt.Errorf("%w: rule mode %s cannot transition directly to %s", ErrConflict, expected, next)
	}
	if expected == ModeCanary && next == ModeActive && ruleAction != ActionObserve {
		var selected, reviewed, falsePositives uint64
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*),
 COALESCE(SUM(c.status IN ('ValidTraffic','InvalidTraffic','AppealUpheld','AppealDenied')),0),
 COALESCE(SUM(c.status IN ('ValidTraffic','AppealUpheld')),0)
FROM quality_decision d LEFT JOIN quality_case c USING (decision_id)
WHERE d.rule_id=? AND d.rule_mode='Canary' AND d.canary_selected='Yes'
 AND d.applied_action<>'Observe' AND d.evidence_state='Complete'`, ruleID).
			Scan(&selected, &reviewed, &falsePositives); err != nil {
			return err
		}
		falsePositiveBPS := uint64(0)
		if reviewed != 0 {
			falsePositiveBPS = falsePositives * 10_000 / reviewed
		}
		if selected == 0 || reviewed == 0 || falsePositiveBPS > uint64(falsePositiveLimit) {
			return fmt.Errorf("%w: active rollout requires reviewed canary evidence within the false-positive limit", ErrConflict)
		}
	}
	result, err := tx.ExecContext(ctx, `
UPDATE quality_rule SET rollout_mode=?, canary_basis_points=?, activated_by=?,
 activated_at=IF(? IN ('Canary','Active'),UTC_TIMESTAMP(6),activated_at), updated_at=UTC_TIMESTAMP(6)
WHERE rule_id=? AND rollout_mode=?`, next, canaryBPS, actor.Name(), next, ruleID, expected)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, actor, "RuleModeChanged", "Rule", ruleID, scope, string(expected), string(next), reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ListRules(ctx context.Context, actor Actor) ([]Rule, error) {
	if err := requireActor(actor, PermissionEvidenceRead, false); err != nil || actor.Role != "admin" {
		return nil, ErrForbidden
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT rule_id, rule_key, rule_version, signal_type, rule_action, rollout_mode, scope_type,
 scope_id, CAST(threshold_value AS CHAR), window_seconds,
 canary_basis_points, reason_code, evidence_retention_hours,
 aggregate_retention_days, false_positive_limit_bps, created_by, created_at,
 activated_at
FROM quality_rule ORDER BY rule_key, rule_version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		var rule Rule
		var threshold string
		var activated sql.NullTime
		if err := rows.Scan(&rule.ID, &rule.Key, &rule.Version, &rule.Signal, &rule.Action,
			&rule.Mode, &rule.Scope.Type, &rule.Scope.ID, &threshold, &rule.WindowSeconds,
			&rule.CanaryBasisPoints, &rule.ReasonCode, &rule.EvidenceRetentionHrs,
			&rule.AggregateRetentionDays, &rule.FalsePositiveLimitBPS, &rule.CreatedBy,
			&rule.CreatedAt, &activated); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscan(threshold, &rule.Threshold); err != nil {
			return nil, err
		}
		if activated.Valid {
			at := activated.Time.UTC()
			rule.ActivatedAt = &at
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (s *Service) RuleHealth(ctx context.Context, actor Actor, since time.Time) ([]RuleHealth, error) {
	if err := requireActor(actor, PermissionEvidenceRead, false); err != nil || actor.Role != "admin" {
		return nil, ErrForbidden
	}
	now := s.currentTime()
	if since.IsZero() || since.After(now) || now.Sub(since) > 400*24*time.Hour {
		return nil, fmt.Errorf("rule-health start must be within the last 400 days")
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT r.rule_id, r.rule_key, r.rule_version, r.rollout_mode,
 COALESCE(SUM(c.decisions),0), COALESCE(SUM(c.matched),0),
 COALESCE(SUM(c.enforced),0), COALESCE(SUM(c.valid_traffic),0),
 COALESCE(SUM(c.invalid_traffic),0), COALESCE(SUM(c.appeals),0),
 COALESCE(SUM(c.appeals_upheld),0), r.false_positive_limit_bps
FROM quality_rule r LEFT JOIN quality_counter c
 ON c.rule_id=r.rule_id AND c.hour_start>=?
GROUP BY r.rule_id, r.rule_key, r.rule_version, r.rollout_mode,
 r.false_positive_limit_bps
ORDER BY r.rule_key, r.rule_version DESC`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RuleHealth
	for rows.Next() {
		var health RuleHealth
		if err := rows.Scan(&health.RuleID, &health.RuleKey, &health.RuleVersion,
			&health.Mode, &health.Decisions, &health.Matched, &health.Enforced,
			&health.ValidTraffic, &health.InvalidTraffic, &health.Appeals,
			&health.AppealsUpheld, &health.FalsePositiveLimitBPS); err != nil {
			return nil, err
		}
		finalizeRuleHealth(&health)
		out = append(out, health)
	}
	return out, rows.Err()
}

func finalizeRuleHealth(health *RuleHealth) {
	if health == nil {
		return
	}
	resolved := health.ValidTraffic + health.InvalidTraffic
	if resolved != 0 {
		falsePositives := health.ValidTraffic + health.AppealsUpheld
		if falsePositives > resolved {
			falsePositives = resolved
		}
		health.FalsePositiveBasisPoints = uint16(float64(falsePositives) * 10_000 / float64(resolved))
	}
	health.RollbackRecommended = (health.Mode == ModeCanary || health.Mode == ModeActive) && resolved != 0 && health.FalsePositiveBasisPoints > health.FalsePositiveLimitBPS
}

// Assess evaluates and durably records every compatible runtime rule. A
// duplicate rule/event digest is returned as the existing decision without
// incrementing counters or opening another case.
func (s *Service) Assess(ctx context.Context, observation Observation) ([]Decision, error) {
	if s == nil || s.DB == nil || s.engine == nil {
		return nil, fmt.Errorf("traffic-quality service is unavailable")
	}
	if err := observation.Validate(); err != nil {
		return nil, err
	}
	if observation.ObservedAt.After(s.currentTime().Add(time.Minute)) {
		return nil, fmt.Errorf("traffic-quality observation is too far in the future")
	}
	rules, err := s.runtimeRules(ctx, observation)
	if err != nil {
		metricDependencyError.Add(1)
		return nil, err
	}
	decisions := make([]Decision, 0, len(rules))
	for _, rule := range rules {
		decision, evaluated, err := s.engine.Evaluate(rule, observation)
		if err != nil {
			return nil, err
		}
		if !evaluated {
			continue
		}
		stored, err := s.storeDecision(ctx, decision, observation.SafeSummary)
		if err != nil {
			metricDependencyError.Add(1)
			return nil, err
		}
		decisions = append(decisions, stored)
	}
	return decisions, nil
}

func (s *Service) runtimeRules(ctx context.Context, observation Observation) ([]Rule, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT rule_id, rule_key, rule_version, signal_type, rule_action, rollout_mode, scope_type,
 scope_id, CAST(threshold_value AS CHAR), window_seconds,
 canary_basis_points, reason_code, evidence_retention_hours,
 aggregate_retention_days, false_positive_limit_bps, created_by, created_at
FROM quality_rule
WHERE signal_type=? AND rollout_mode IN ('Observe','Canary','Active')
 AND ((scope_type='Global' AND scope_id=0) OR (scope_type=? AND scope_id=?))
ORDER BY rule_key, rule_version DESC`, observation.Signal, observation.Scope.Type, observation.Scope.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := make(map[string]bool)
	var rules []Rule
	for rows.Next() {
		var rule Rule
		var threshold string
		if err := rows.Scan(&rule.ID, &rule.Key, &rule.Version, &rule.Signal,
			&rule.Action, &rule.Mode, &rule.Scope.Type, &rule.Scope.ID, &threshold,
			&rule.WindowSeconds, &rule.CanaryBasisPoints, &rule.ReasonCode,
			&rule.EvidenceRetentionHrs, &rule.AggregateRetentionDays,
			&rule.FalsePositiveLimitBPS, &rule.CreatedBy, &rule.CreatedAt); err != nil {
			return nil, err
		}
		if seen[rule.Key] {
			continue
		}
		seen[rule.Key] = true
		if _, err := fmt.Sscan(threshold, &rule.Threshold); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Service) storeDecision(ctx context.Context, decision Decision, summary string) (Decision, error) {
	tx, err := s.begin(ctx)
	if err != nil {
		return Decision{}, err
	}
	defer tx.Rollback()
	var partner any
	if decision.PartnerDigest != ([32]byte{}) {
		partner = decision.PartnerDigest[:]
	}
	result, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO quality_decision
 (rule_id, rule_key, rule_version, signal_type, configured_action, applied_action,
  rule_mode, scope_type, scope_id, event_digest, partner_digest,
  observed_value, threshold_value, matched, canary_selected, evidence_state,
  reason_code, billing_disposition, observed_at, evidence_expires_at, created_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`,
		decision.RuleID, decision.RuleKey, decision.RuleVersion, decision.Signal,
		decision.ConfiguredAction, decision.AppliedAction, decision.Mode,
		decision.Scope.Type, decision.Scope.ID, decision.EventDigest[:], partner,
		decision.ObservedValue, decision.Threshold, yesNo(decision.Matched),
		yesNo(decision.CanarySelected), decision.Evidence, decision.ReasonCode,
		decision.BillingDisposition, decision.ObservedAt, decision.EvidenceExpiresAt)
	if err != nil {
		return Decision{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Decision{}, err
	}
	if affected == 0 {
		if err := tx.QueryRowContext(ctx, `
SELECT decision_id, applied_action, matched, canary_selected,
 billing_disposition, observed_at, evidence_expires_at
FROM quality_decision WHERE rule_id=? AND event_digest=?`,
			decision.RuleID, decision.EventDigest[:]).Scan(&decision.ID,
			&decision.AppliedAction, newYesNoScanner(&decision.Matched),
			newYesNoScanner(&decision.CanarySelected), &decision.BillingDisposition,
			&decision.ObservedAt, &decision.EvidenceExpiresAt); err != nil {
			return Decision{}, err
		}
		if err := tx.Commit(); err != nil {
			return Decision{}, err
		}
		return decision, nil
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Decision{}, err
	}
	decision.ID = uint64(id)
	if summary != "" && decision.Evidence != EvidenceMissing {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO quality_evidence
 (decision_id, event_digest, safe_summary, expires_at, created_at)
VALUES (?,?,?,?,UTC_TIMESTAMP(6))`, decision.ID, decision.EventDigest[:], summary,
			decision.EvidenceExpiresAt); err != nil {
			return Decision{}, err
		}
	}
	if decision.Matched && (decision.ConfiguredAction != ActionObserve || decision.AppliedAction != ActionObserve) {
		result, err := tx.ExecContext(ctx, `
INSERT INTO quality_case
 (decision_id, scope_type, scope_id, status, opened_by, version, created_at, updated_at)
VALUES (?,?,?,'Open','service:rule-engine',1,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`,
			decision.ID, decision.Scope.Type, decision.Scope.ID)
		if err != nil {
			return Decision{}, err
		}
		caseID, err := result.LastInsertId()
		if err != nil {
			return Decision{}, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO quality_case_event
 (case_id, event_name, actor_role, actor_id, prior_status, new_status, reason, created_at)
VALUES (?,'Opened','service','rule-engine',NULL,'Open',?,UTC_TIMESTAMP(6))`,
			caseID, decision.ReasonCode); err != nil {
			return Decision{}, err
		}
	}
	hour := decision.ObservedAt.UTC().Truncate(time.Hour)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO quality_counter
 (rule_id, hour_start, decisions, matched, enforced)
VALUES (?,?,1,?,?)
ON DUPLICATE KEY UPDATE decisions=decisions+1, matched=matched+VALUES(matched),
 enforced=enforced+VALUES(enforced)`, decision.RuleID, hour,
		boolInt(decision.Matched), boolInt(decision.AppliedAction != ActionObserve)); err != nil {
		return Decision{}, err
	}
	if err := tx.Commit(); err != nil {
		return Decision{}, err
	}
	recordDecisionMetric(decision)
	return decision, nil
}

func (s *Service) ListCases(ctx context.Context, actor Actor, scope Scope, limit int) ([]Case, error) {
	if err := actor.Validate(); err != nil || !actor.CanRead(scope) {
		return nil, ErrForbidden
	}
	if err := scope.Validate(); err != nil || scope.Type == ScopeGlobal {
		return nil, ErrForbidden
	}
	if limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("case limit must be between 1 and 1000")
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT c.case_id, c.decision_id, c.scope_type, c.scope_id, c.status,
 c.opened_by, c.resolved_by, c.version, d.rule_key, d.rule_version, d.signal_type,
 d.configured_action, d.applied_action, d.evidence_state, d.reason_code,
 CAST(d.observed_value AS CHAR), CAST(d.threshold_value AS CHAR),
 d.observed_at, d.evidence_expires_at, e.safe_summary, c.created_at, c.updated_at
FROM quality_case c INNER JOIN quality_decision d USING (decision_id)
LEFT JOIN quality_evidence e ON e.decision_id=d.decision_id AND e.expires_at>UTC_TIMESTAMP(6)
WHERE c.scope_type=? AND c.scope_id=?
ORDER BY c.case_id DESC LIMIT ?`, scope.Type, scope.ID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Case
	for rows.Next() {
		var item Case
		var observed, threshold string
		if err := rows.Scan(&item.ID, &item.DecisionID, &item.Scope.Type,
			&item.Scope.ID, &item.Status, &item.OpenedBy, &item.ResolvedBy,
			&item.Version, &item.RuleKey, &item.RuleVersion, &item.Signal,
			&item.ConfiguredAction, &item.AppliedAction, &item.Evidence,
			&item.ReasonCode, &observed, &threshold, &item.ObservedAt,
			&item.EvidenceExpiresAt, &item.SafeSummary, &item.CreatedAt,
			&item.UpdatedAt); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscan(observed, &item.ObservedValue); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscan(threshold, &item.Threshold); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) ListEnforcements(ctx context.Context, actor Actor, scope Scope, limit int) ([]Enforcement, error) {
	if err := actor.Validate(); err != nil || !actor.CanRead(scope) {
		return nil, ErrForbidden
	}
	if err := scope.Validate(); err != nil || scope.Type == ScopeGlobal || limit < 1 || limit > 1000 {
		return nil, ErrForbidden
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT enforcement_id, rule_id, decision_id, scope_type, scope_id, enforcement_action,
 state, canary_basis_points, created_by, created_at, expires_at
FROM quality_enforcement WHERE scope_type=? AND scope_id=?
ORDER BY enforcement_id DESC LIMIT ?`, scope.Type, scope.ID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Enforcement
	for rows.Next() {
		var item Enforcement
		if err := rows.Scan(&item.ID, &item.RuleID, &item.DecisionID,
			&item.Scope.Type, &item.Scope.ID, &item.Action, &item.State,
			&item.CanaryBasisPoints, &item.CreatedBy, &item.CreatedAt,
			&item.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) ResolveCase(ctx context.Context, actor Actor, caseID uint64, expectedVersion uint32, status ReviewStatus, reason string) error {
	if err := requireActor(actor, PermissionReviewResolve, true); err != nil {
		return err
	}
	if actor.Role != "admin" && actor.Role != "agent" || caseID == 0 || expectedVersion == 0 || !validAuditReason(reason) {
		return ErrForbidden
	}
	if status != ReviewValidTraffic && status != ReviewInvalidTraffic {
		return fmt.Errorf("case resolution must be ValidTraffic or InvalidTraffic")
	}
	return s.transitionCase(ctx, actor, caseID, expectedVersion, ReviewOpen, status, reason)
}

func (s *Service) AppealCase(ctx context.Context, actor Actor, caseID uint64, expectedVersion uint32, reason string) error {
	if err := requireActor(actor, PermissionAppealSubmit, false); err != nil {
		return err
	}
	if (actor.Role != "adv" && actor.Role != "pub") || caseID == 0 || expectedVersion == 0 || !validAuditReason(reason) {
		return ErrForbidden
	}
	return s.transitionCase(ctx, actor, caseID, expectedVersion, ReviewInvalidTraffic, ReviewAppealed, reason)
}

func (s *Service) ResolveAppeal(ctx context.Context, actor Actor, caseID uint64, expectedVersion uint32, upheld bool, reason string) error {
	if err := requireActor(actor, PermissionAppealResolve, true); err != nil {
		return err
	}
	if actor.Role != "admin" || caseID == 0 || expectedVersion == 0 || !validAuditReason(reason) {
		return ErrForbidden
	}
	next := ReviewAppealDenied
	if upheld {
		next = ReviewAppealUpheld
	}
	return s.transitionCase(ctx, actor, caseID, expectedVersion, ReviewAppealed, next, reason)
}

func (s *Service) transitionCase(ctx context.Context, actor Actor, caseID uint64, version uint32, from, to ReviewStatus, reason string) error {
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var scope Scope
	var ruleID uint64
	var current ReviewStatus
	if err := tx.QueryRowContext(ctx, `
SELECT c.scope_type, c.scope_id, d.rule_id, c.status
FROM quality_case c INNER JOIN quality_decision d USING (decision_id)
WHERE c.case_id=? FOR UPDATE`, caseID).Scan(&scope.Type, &scope.ID, &ruleID, &current); err != nil {
		return normalizeNotFound(err)
	}
	if !actor.CanRead(scope) || current != from {
		return ErrForbidden
	}
	result, err := tx.ExecContext(ctx, `
UPDATE quality_case SET status=?, resolved_by=?, version=version+1,
 updated_at=UTC_TIMESTAMP(6) WHERE case_id=? AND status=? AND version=?`,
		to, actor.Name(), caseID, from, version)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return ErrConflict
	}
	eventName := map[ReviewStatus]string{
		ReviewValidTraffic: "ResolvedValid", ReviewInvalidTraffic: "ResolvedInvalid",
		ReviewAppealed: "Appealed", ReviewAppealUpheld: "AppealUpheld",
		ReviewAppealDenied: "AppealDenied",
	}[to]
	if _, err := tx.ExecContext(ctx, `
INSERT INTO quality_case_event
 (case_id, event_name, actor_role, actor_id, prior_status, new_status, reason, created_at)
VALUES (?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, caseID, eventName, actor.Role,
		actor.ID, from, to, reason); err != nil {
		return err
	}
	counter := ""
	switch to {
	case ReviewValidTraffic:
		counter = "valid_traffic"
	case ReviewInvalidTraffic:
		counter = "invalid_traffic"
	case ReviewAppealed:
		counter = "appeals"
	case ReviewAppealUpheld:
		counter = "appeals_upheld"
	}
	if counter != "" {
		query := "INSERT INTO quality_counter (rule_id,hour_start," + counter + ") VALUES (?,UTC_TIMESTAMP()-INTERVAL MINUTE(UTC_TIMESTAMP()) MINUTE-INTERVAL SECOND(UTC_TIMESTAMP()) SECOND,1) ON DUPLICATE KEY UPDATE " + counter + "=" + counter + "+1"
		if _, err := tx.ExecContext(ctx, query, ruleID); err != nil {
			return err
		}
	}
	if err := insertAudit(ctx, tx, actor, "CaseStatusChanged", "Case", caseID, scope, string(from), string(to), reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ActivateEnforcement(ctx context.Context, actor Actor, decisionID uint64, action Action, canaryBPS uint16, ttl time.Duration, reason string) (uint64, error) {
	if err := requireActor(actor, PermissionEnforcementActivate, true); err != nil {
		return 0, err
	}
	if actor.Role != "admin" || decisionID == 0 || !blockingAction(action) || canaryBPS > 10_000 || ttl < time.Minute || ttl > 30*24*time.Hour || !validAuditReason(reason) {
		return 0, ErrForbidden
	}
	if action == ActionThrottle && (canaryBPS == 0 || canaryBPS >= 10_000) {
		return 0, fmt.Errorf("throttle enforcement requires a bounded canary sample")
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var ruleID uint64
	var scope Scope
	var evidence EvidenceState
	var decisionAction Action
	var rolloutMode RuleMode
	var review ReviewStatus
	if err := tx.QueryRowContext(ctx, `
SELECT d.rule_id, d.scope_type, d.scope_id, d.evidence_state,
 d.applied_action, r.rollout_mode, c.status
FROM quality_decision d INNER JOIN quality_case c USING (decision_id)
INNER JOIN quality_rule r ON r.rule_id=d.rule_id
WHERE d.decision_id=? FOR UPDATE`, decisionID).Scan(&ruleID, &scope.Type, &scope.ID,
		&evidence, &decisionAction, &rolloutMode, &review); err != nil {
		return 0, normalizeNotFound(err)
	}
	if evidence != EvidenceComplete || !reviewConfirmsInvalid(review) || rolloutMode != ModeActive || decisionAction != action {
		return 0, fmt.Errorf("%w: enforcement requires complete evidence and an InvalidTraffic review", ErrConflict)
	}
	state := "Active"
	if canaryBPS != 0 && canaryBPS < 10_000 {
		state = "Canary"
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO quality_enforcement
 (rule_id, decision_id, scope_type, scope_id, enforcement_action, state,
  canary_basis_points, created_by, created_at, expires_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),?,UTC_TIMESTAMP(6))`, ruleID,
		decisionID, scope.Type, scope.ID, action, state, canaryBPS, actor.Name(),
		s.currentTime().Add(ttl))
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, actor, "EnforcementActivated", "Enforcement", uint64(id), scope, "Absent", state, reason); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(id), nil
}

func (s *Service) RollbackEnforcement(ctx context.Context, actor Actor, enforcementID uint64, reason string) error {
	if err := requireActor(actor, PermissionEnforcementRollback, true); err != nil {
		return err
	}
	if actor.Role != "admin" || enforcementID == 0 || !validAuditReason(reason) {
		return ErrForbidden
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var scope Scope
	var state string
	if err := tx.QueryRowContext(ctx, `
SELECT scope_type, scope_id, state FROM quality_enforcement
WHERE enforcement_id=? FOR UPDATE`, enforcementID).Scan(&scope.Type, &scope.ID, &state); err != nil {
		return normalizeNotFound(err)
	}
	if state != "Active" && state != "Canary" {
		return ErrConflict
	}
	result, err := tx.ExecContext(ctx, `
UPDATE quality_enforcement SET state='RolledBack', rolled_back_by=?,
 rollback_reason=?, rolled_back_at=UTC_TIMESTAMP(6), updated_at=UTC_TIMESTAMP(6)
WHERE enforcement_id=? AND state=?`, actor.Name(), reason, enforcementID, state)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, actor, "EnforcementRolledBack", "Enforcement", enforcementID, scope, state, "RolledBack", reason); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	metricRollback.Add(1)
	return nil
}

func (s *Service) RecommendBilling(ctx context.Context, actor Actor, decisionID, statementID uint64, billableKey string, disposition BillingDisposition, reason string) (uint64, error) {
	if err := requireActor(actor, PermissionBillingRecommend, true); err != nil {
		return 0, err
	}
	if (actor.Role != "admin" && actor.Role != "agent") || decisionID == 0 || statementID == 0 || !safeEventKey.MatchString(billableKey) || !validAuditReason(reason) {
		return 0, ErrForbidden
	}
	if disposition != BillingExclude && disposition != BillingHold && disposition != BillingReverse {
		return 0, fmt.Errorf("billing recommendation must be Exclude, Hold, or Reverse")
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var scope Scope
	var review ReviewStatus
	var evidence EvidenceState
	if err := tx.QueryRowContext(ctx, `
SELECT d.scope_type, d.scope_id, d.evidence_state, c.status
FROM quality_decision d INNER JOIN quality_case c USING (decision_id)
WHERE d.decision_id=? FOR UPDATE`, decisionID).Scan(&scope.Type, &scope.ID, &evidence, &review); err != nil {
		return 0, normalizeNotFound(err)
	}
	if evidence != EvidenceComplete || !reviewConfirmsInvalid(review) || !actor.CanRead(scope) {
		return 0, ErrForbidden
	}
	var party string
	var partyID uint64
	if err := tx.QueryRowContext(ctx, `SELECT party_type, party_id FROM acct_statement WHERE statement_id=? FOR UPDATE`, statementID).Scan(&party, &partyID); err != nil {
		return 0, normalizeNotFound(err)
	}
	if (scope.Type == ScopeAdvertiser && party != "advertiser") || (scope.Type == ScopePublisher && party != "publisher") || scope.ID != partyID {
		return 0, ErrForbidden
	}
	var accountingVersion string
	if err := tx.QueryRowContext(ctx, `SELECT unit_version FROM acct_contract WHERE contract_id=1`).Scan(&accountingVersion); err != nil {
		return 0, err
	}
	digest := s.engine.digest("billable", billableKey)
	result, err := tx.ExecContext(ctx, `
INSERT INTO quality_billing
 (decision_id, statement_id, billable_digest, accounting_version,
  disposition, state, recommended_by, reason, created_at, updated_at)
VALUES (?,?,?,?,?,'Recommended',?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`,
		decisionID, statementID, digest[:], accountingVersion, disposition,
		actor.Name(), reason)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, actor, "BillingRecommended", "Billing", uint64(id), scope, "Absent", "Recommended", reason); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(id), nil
}

func (s *Service) ApproveBilling(ctx context.Context, actor Actor, billingID uint64, approve bool, reason string) error {
	if err := requireActor(actor, PermissionBillingApprove, true); err != nil {
		return err
	}
	if actor.Role != "admin" || billingID == 0 || !validAuditReason(reason) {
		return ErrForbidden
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var item BillingRecommendation
	var scope Scope
	var review ReviewStatus
	if err := tx.QueryRowContext(ctx, `
SELECT b.billing_id, b.decision_id, b.statement_id, b.disposition, b.state,
 b.recommended_by, b.approved_by, b.accounting_version, d.scope_type, d.scope_id,
 c.status
FROM quality_billing b INNER JOIN quality_decision d USING (decision_id)
INNER JOIN quality_case c USING (decision_id)
WHERE b.billing_id=? FOR UPDATE`, billingID).Scan(&item.ID, &item.DecisionID,
		&item.StatementID, &item.Disposition, &item.State, &item.RecommendedBy,
		&item.ApprovedBy, &item.AccountingVersion, &scope.Type, &scope.ID, &review); err != nil {
		return normalizeNotFound(err)
	}
	if item.State != "Recommended" || item.RecommendedBy == actor.Name() || !reviewConfirmsInvalid(review) {
		return ErrConflict
	}
	next := "Rejected"
	if approve {
		next = "Approved"
		if item.Disposition == BillingHold {
			var from string
			if err := tx.QueryRowContext(ctx, `SELECT status FROM acct_statement WHERE statement_id=? FOR UPDATE`, item.StatementID).Scan(&from); err != nil {
				return err
			}
			if from != "Draft" && from != "Confirmed" {
				return fmt.Errorf("%w: quality hold requires a Draft or Confirmed statement", ErrConflict)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE acct_statement SET status='Held', updated_at=UTC_TIMESTAMP() WHERE statement_id=? AND status=?`, item.StatementID, from); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO acct_audit
 (statement_id, actor, event, status_from, status_to, reason, created_at)
VALUES (?,?,'quality_hold',?,'Held',?,UTC_TIMESTAMP())`, item.StatementID,
				actor.Name(), from, reason); err != nil {
				return err
			}
			next = "Applied"
		}
	}
	result, err := tx.ExecContext(ctx, `
UPDATE quality_billing SET state=?, approved_by=?, updated_at=UTC_TIMESTAMP(6)
WHERE billing_id=? AND state='Recommended'`, next, actor.Name(), billingID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, actor, "BillingReviewed", "Billing", billingID, scope, "Recommended", next, reason); err != nil {
		return err
	}
	return tx.Commit()
}

// PruneEvidence removes only expired short-lived summaries. Decisions,
// aggregate counters, case events, and audit records remain intact.
func (s *Service) PruneEvidence(ctx context.Context, actor Actor, limit int, reason string) (deleted int64, err error) {
	if err := requireMaintenanceActor(actor, PermissionRetentionPrune); err != nil {
		return 0, err
	}
	if actor.Role != "admin" || limit < 1 || limit > 10_000 || !validAuditReason(reason) {
		return 0, ErrForbidden
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return 0, err
	}
	clean := false
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), retentionCleanupTimeout)
		defer cancel()
		if _, cleanupErr := conn.ExecContext(cleanupCtx, `SET @aofei_quality_retention=0`); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("clear traffic-quality retention mode: %w", cleanupErr), driver.ErrBadConn)
		} else {
			clean = true
		}
		if !clean {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
		err = errors.Join(err, conn.Close())
	}()
	if _, err := conn.ExecContext(ctx, `SET @aofei_quality_retention=1`); err != nil {
		return 0, err
	}
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM quality_evidence WHERE expires_at<=UTC_TIMESTAMP(6) ORDER BY expires_at LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	deleted, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, actor, "EvidencePruned", "Retention", 0, Scope{Type: ScopeGlobal}, "Expired", fmt.Sprintf("Deleted:%d", deleted), reason); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *Service) begin(ctx context.Context) (*sql.Tx, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("traffic-quality database is nil")
	}
	return s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

func (s *Service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func requireActor(actor Actor, permission string, recentMFA bool) error {
	if err := actor.Validate(); err != nil {
		return ErrForbidden
	}
	if strings.HasPrefix(actor.ID, "unix-uid:") {
		if permission != PermissionEvidenceRead || !actor.isUnixMaintenance(permission) {
			return ErrForbidden
		}
		return nil
	}
	if !actor.Can(permission) || recentMFA && !actor.RecentMFA {
		return ErrForbidden
	}
	return nil
}

func requireMaintenanceActor(actor Actor, permission string) error {
	if err := actor.Validate(); err != nil || !actor.isUnixMaintenance(permission) {
		return ErrForbidden
	}
	return nil
}

func ruleModeTransitionAllowed(from, to RuleMode) bool {
	switch from {
	case ModeDraft:
		return to == ModeObserve
	case ModeObserve:
		return to == ModeCanary || to == ModeDisabled
	case ModeCanary:
		return to == ModeActive || to == ModeObserve || to == ModeDisabled
	case ModeActive:
		return to == ModeObserve || to == ModeDisabled
	case ModeDisabled:
		return to == ModeObserve
	default:
		return false
	}
}

func reviewConfirmsInvalid(status ReviewStatus) bool {
	return status == ReviewInvalidTraffic || status == ReviewAppealDenied
}

func insertAudit(ctx context.Context, tx *sql.Tx, actor Actor, event, object string, objectID uint64, scope Scope, prior, next, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO quality_audit
 (event_name, actor_role, actor_id, object_type, object_id, scope_type,
  scope_id, prior_state, new_state, reason, created_at)
VALUES (?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, event, actor.Role, actor.ID,
		object, objectID, scope.Type, scope.ID, prior, next, reason)
	return err
}

func nullableInt64(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func normalizeNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type yesNoScanner struct{ target *bool }

func newYesNoScanner(target *bool) *yesNoScanner { return &yesNoScanner{target: target} }

func (s *yesNoScanner) Scan(src any) error {
	var value string
	switch typed := src.(type) {
	case string:
		value = typed
	case []byte:
		value = string(typed)
	default:
		return fmt.Errorf("scan Yes/No from %T", src)
	}
	switch value {
	case "Yes":
		*s.target = true
	case "No":
		*s.target = false
	default:
		return fmt.Errorf("invalid Yes/No value %q", value)
	}
	return nil
}

func safeSingleLine(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
}

package accounting

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PartyType string
type Cadence string
type Status string

const (
	PartyAdvertiser PartyType = "advertiser"
	PartyPublisher  PartyType = "publisher"

	CadenceDaily   Cadence = "daily"
	CadenceWeekly  Cadence = "weekly"
	CadenceMonthly Cadence = "monthly"

	StatusDraft     Status = "Draft"
	StatusHeld      Status = "Held"
	StatusConfirmed Status = "Confirmed"
	StatusSettled   Status = "Settled"
	StatusCorrected Status = "Corrected"
)

var (
	ErrConflict          = errors.New("accounting state conflict")
	ErrSourceDiscrepancy = errors.New("accounting source discrepancy")
	safeRequestKey       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	safeActor            = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9@._:-]{0,127}$`)
	safeReference        = regexp.MustCompile(`^(invoice|payout|manual):[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	longDigitRun         = regexp.MustCompile(`[0-9]{9,}`)
)

type Statement struct {
	ID               uint64
	RequestKey       string
	PartyType        PartyType
	PartyID          uint64
	Cadence          Cadence
	PeriodStart      time.Time
	PeriodEnd        time.Time
	Currency         string
	SourceAmount     Money
	AdjustmentAmount Money
	TotalAmount      Money
	Status           Status
	SupersedesID     sql.NullInt64
	ExternalRef      sql.NullString
	CreatedBy        string
	ConfirmedBy      sql.NullString
	SettledBy        sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// StatementScope is explicit so a missing party filter cannot accidentally
// become a cross-account listing. All is reserved for offline operator export.
type StatementScope struct {
	PartyType PartyType
	PartyID   uint64
	All       bool
}

func PartyStatementScope(party PartyType, partyID uint64) StatementScope {
	return StatementScope{PartyType: party, PartyID: partyID}
}

func AllStatementScope() StatementScope { return StatementScope{All: true} }

type CreateInput struct {
	RequestKey  string
	PartyType   PartyType
	PartyID     uint64
	Cadence     Cadence
	PeriodStart time.Time
	PeriodEnd   time.Time
	Actor       string
	Reason      string
}

type TransitionInput struct {
	StatementID uint64
	To          Status
	Actor       string
	Reason      string
	ExternalRef string
}

type AdjustmentInput struct {
	StatementID uint64
	Amount      Money
	Actor       string
	Reason      string
}

type CorrectionInput struct {
	StatementID uint64
	RequestKey  string
	Actor       string
	Reason      string
}

type Discrepancy struct {
	StatementID uint64
	Expected    Money
	Actual      Money
	Difference  Money
}

type MiddlemanReconciliation struct {
	PeriodStart    time.Time
	PeriodEnd      time.Time
	Currency       string
	Impressions    uint64
	Charge         Money
	Pay            Money
	Margin         Money
	ExpectedMargin Money
	Difference     Money
}

type Service struct{ DB *sql.DB }

func (s Service) ListStatements(ctx context.Context, scope StatementScope) ([]Statement, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("accounting database is nil")
	}
	if scope.All {
		if scope.PartyType != "" || scope.PartyID != 0 {
			return nil, fmt.Errorf("all-party statement scope cannot include a party")
		}
	} else if (scope.PartyType != PartyAdvertiser && scope.PartyType != PartyPublisher) || scope.PartyID == 0 {
		return nil, fmt.Errorf("an explicit authorized statement party scope is required")
	}
	query := `
SELECT statement_id, request_key, party_type, party_id, cadence, period_start,
       period_end, currency, CAST(source_amount AS CHAR),
       CAST(adjustment_amount AS CHAR), CAST(total_amount AS CHAR), status,
       supersedes_id, external_ref, created_by, confirmed_by, settled_by,
       created_at, updated_at
FROM acct_statement`
	var args []any
	if !scope.All {
		query += " WHERE party_type=? AND party_id=?"
		args = append(args, scope.PartyType, scope.PartyID)
	}
	query += " ORDER BY period_start DESC, statement_id DESC"
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var statements []Statement
	for rows.Next() {
		var statement Statement
		var sourceRaw, adjustmentRaw, totalRaw string
		if err := rows.Scan(
			&statement.ID, &statement.RequestKey, &statement.PartyType, &statement.PartyID,
			&statement.Cadence, &statement.PeriodStart, &statement.PeriodEnd, &statement.Currency,
			&sourceRaw, &adjustmentRaw, &totalRaw, &statement.Status, &statement.SupersedesID,
			&statement.ExternalRef, &statement.CreatedBy, &statement.ConfirmedBy, &statement.SettledBy,
			&statement.CreatedAt, &statement.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if statement.SourceAmount, err = ParseMoney(sourceRaw); err != nil {
			return nil, err
		}
		if statement.AdjustmentAmount, err = ParseMoney(adjustmentRaw); err != nil {
			return nil, err
		}
		if statement.TotalAmount, err = ParseMoney(totalRaw); err != nil {
			return nil, err
		}
		statements = append(statements, statement)
	}
	return statements, rows.Err()
}

func (s Service) CreateStatement(ctx context.Context, input CreateInput) (uint64, error) {
	if err := validateCreate(input); err != nil {
		return 0, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := requireParty(ctx, tx, input.PartyType, input.PartyID); err != nil {
		return 0, err
	}
	amount, err := sourceAmount(ctx, tx, input.PartyType, input.PartyID, input.PeriodStart, input.PeriodEnd)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO acct_statement (
  request_key, party_type, party_id, cadence, period_start, period_end,
  currency, source_amount, adjustment_amount, total_amount, status,
  created_by, created_at, updated_at
) VALUES (?,?,?,?,?,?,'USD',?,0,?,'Draft',?,UTC_TIMESTAMP(),UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE statement_id=LAST_INSERT_ID(statement_id)`,
		input.RequestKey, input.PartyType, input.PartyID, input.Cadence,
		dateString(input.PeriodStart), dateString(input.PeriodEnd), amount.String(), amount.String(), input.Actor)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	// MySQL may report zero for the no-op duplicate branch when a BEFORE UPDATE
	// integrity trigger is installed, even though LAST_INSERT_ID(id) is present.
	// Resolve the durable request identity inside the same transaction.
	if id == 0 {
		if err := tx.QueryRowContext(ctx, `SELECT statement_id FROM acct_statement WHERE request_key=?`, input.RequestKey).Scan(&id); err != nil {
			return 0, err
		}
	}
	if err := verifyIdempotentCreate(ctx, tx, uint64(id), input); err != nil {
		return 0, err
	}
	if err := insertCreatedAudit(ctx, tx, uint64(id), input.Actor, input.Reason); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(id), nil
}

func verifyIdempotentCreate(ctx context.Context, tx *sql.Tx, statementID uint64, input CreateInput) error {
	var party PartyType
	var partyID uint64
	var cadence Cadence
	var start, end time.Time
	var createdBy string
	if err := tx.QueryRowContext(ctx, `
SELECT party_type, party_id, cadence, period_start, period_end, created_by
FROM acct_statement WHERE statement_id=? FOR UPDATE`, statementID).
		Scan(&party, &partyID, &cadence, &start, &end, &createdBy); err != nil {
		return err
	}
	if party != input.PartyType || partyID != input.PartyID || cadence != input.Cadence ||
		dateString(start) != dateString(input.PeriodStart) || dateString(end) != dateString(input.PeriodEnd) ||
		createdBy != input.Actor {
		return fmt.Errorf("%w: request key belongs to a different statement operation", ErrConflict)
	}
	return nil
}

func (s Service) Transition(ctx context.Context, input TransitionInput) error {
	if input.StatementID == 0 || !validActor(input.Actor) || strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 500 {
		return fmt.Errorf("statement, actor, and bounded reason are required")
	}
	if input.ExternalRef != "" && (!safeReference.MatchString(input.ExternalRef) || containsAccountNumber(input.ExternalRef)) {
		return fmt.Errorf("external reference must be an opaque invoice, payout, or manual reference and must not contain account numbers")
	}
	if input.To == StatusSettled && input.ExternalRef == "" {
		return fmt.Errorf("settlement requires an external evidence reference")
	}
	if input.To != StatusSettled && input.ExternalRef != "" {
		return fmt.Errorf("external reference is accepted only for settlement")
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var from Status
	var createdBy string
	var confirmedBy sql.NullString
	var party PartyType
	var partyID uint64
	var periodStart, periodEnd time.Time
	var sourceRaw string
	if err := tx.QueryRowContext(ctx, `
SELECT status, created_by, confirmed_by, party_type, party_id, period_start,
       period_end, CAST(source_amount AS CHAR)
FROM acct_statement WHERE statement_id=? FOR UPDATE`, input.StatementID).
		Scan(&from, &createdBy, &confirmedBy, &party, &partyID, &periodStart, &periodEnd, &sourceRaw); err != nil {
		return err
	}
	if !transitionAllowed(from, input.To) {
		return fmt.Errorf("%w: %s to %s", ErrConflict, from, input.To)
	}
	if input.To == StatusConfirmed && input.Actor == createdBy {
		return fmt.Errorf("%w: statement creator cannot approve the same statement", ErrConflict)
	}
	if input.To == StatusConfirmed {
		expected, err := ParseMoney(sourceRaw)
		if err != nil {
			return err
		}
		actual, err := sourceAmount(ctx, tx, party, partyID, periodStart, periodEnd)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("%w: expected %s, current source %s", ErrSourceDiscrepancy, expected, actual)
		}
	}
	if input.To == StatusSettled && confirmedBy.Valid && input.Actor == confirmedBy.String {
		return fmt.Errorf("%w: statement approver cannot record its settlement", ErrConflict)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE acct_statement SET status=?, external_ref=NULLIF(?,''),
  confirmed_by=IF(?='Confirmed',?,confirmed_by),
  settled_by=IF(?='Settled',?,settled_by), updated_at=UTC_TIMESTAMP()
WHERE statement_id=? AND status=?`, input.To, input.ExternalRef,
		input.To, input.Actor, input.To, input.Actor, input.StatementID, from)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, input.StatementID, input.Actor, "transition", string(from), string(input.To), input.Reason); err != nil {
		return err
	}
	return tx.Commit()
}

func containsAccountNumber(value string) bool {
	if longDigitRun.MatchString(value) {
		return true
	}
	digits := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
			if digits >= 9 {
				return true
			}
		case r == '-' || r == '.' || r == '_':
		default:
			digits = 0
		}
	}
	return false
}

func (s Service) AddAdjustment(ctx context.Context, input AdjustmentInput) error {
	if input.StatementID == 0 || input.Amount == 0 || !validActor(input.Actor) || strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 500 {
		return fmt.Errorf("statement, non-zero amount, actor, and bounded reason are required")
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status Status
	var sourceRaw, adjustmentRaw string
	if err := tx.QueryRowContext(ctx, `
SELECT status, CAST(source_amount AS CHAR), CAST(adjustment_amount AS CHAR)
FROM acct_statement WHERE statement_id=? FOR UPDATE`, input.StatementID).Scan(&status, &sourceRaw, &adjustmentRaw); err != nil {
		return err
	}
	if status != StatusDraft && status != StatusHeld {
		return fmt.Errorf("%w: adjustments require Draft or Held status", ErrConflict)
	}
	source, err := ParseMoney(sourceRaw)
	if err != nil {
		return err
	}
	adjustment, err := ParseMoney(adjustmentRaw)
	if err != nil {
		return err
	}
	newAdjustment, err := adjustment.Add(input.Amount)
	if err != nil {
		return err
	}
	total, err := source.Add(newAdjustment)
	if err != nil {
		return err
	}
	if total < 0 {
		return fmt.Errorf("statement total cannot be negative")
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO acct_adjustment (statement_id, amount, reason, created_by, created_at)
VALUES (?,?,?,?,UTC_TIMESTAMP())`, input.StatementID, input.Amount.String(), input.Reason, input.Actor); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE acct_statement SET adjustment_amount=?, total_amount=?, updated_at=UTC_TIMESTAMP()
WHERE statement_id=?`, newAdjustment.String(), total.String(), input.StatementID); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, input.StatementID, input.Actor, "adjustment", string(status), string(status), input.Reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s Service) Correct(ctx context.Context, input CorrectionInput) (uint64, error) {
	if input.StatementID == 0 || !safeRequestKey.MatchString(input.RequestKey) || !validActor(input.Actor) || strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 500 {
		return 0, fmt.Errorf("statement, request key, actor, and reason are required")
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var party PartyType
	var partyID uint64
	var cadence Cadence
	var start, end time.Time
	var status Status
	if err := tx.QueryRowContext(ctx, `
SELECT party_type, party_id, cadence, period_start, period_end, status
FROM acct_statement WHERE statement_id=? FOR UPDATE`, input.StatementID).
		Scan(&party, &partyID, &cadence, &start, &end, &status); err != nil {
		return 0, err
	}
	if status == StatusCorrected {
		var replacementID uint64
		var createdBy string
		err := tx.QueryRowContext(ctx, `
SELECT statement_id, created_by FROM acct_statement
WHERE request_key=? AND supersedes_id=?`, input.RequestKey, input.StatementID).
			Scan(&replacementID, &createdBy)
		if err == nil {
			if createdBy != input.Actor {
				return 0, fmt.Errorf("%w: correction request key belongs to another actor", ErrConflict)
			}
			if err := tx.Commit(); err != nil {
				return 0, err
			}
			return replacementID, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
		return 0, fmt.Errorf("%w: corrected statement has no matching replacement", ErrConflict)
	}
	if status != StatusConfirmed && status != StatusSettled {
		return 0, fmt.Errorf("%w: only Confirmed or Settled statements can be corrected", ErrConflict)
	}
	amount, err := sourceAmount(ctx, tx, party, partyID, start, end)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO acct_statement (
  request_key, party_type, party_id, cadence, period_start, period_end,
  currency, source_amount, adjustment_amount, total_amount, status,
  supersedes_id, created_by, created_at, updated_at
) VALUES (?,?,?,?,?,?,'USD',?,0,?,'Draft',?,?,UTC_TIMESTAMP(),UTC_TIMESTAMP())`,
		input.RequestKey, party, partyID, cadence, dateString(start), dateString(end),
		amount.String(), amount.String(), input.StatementID, input.Actor)
	if err != nil {
		return 0, err
	}
	newID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE acct_statement SET status='Corrected', updated_at=UTC_TIMESTAMP() WHERE statement_id=?`, input.StatementID); err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, input.StatementID, input.Actor, "corrected", string(status), string(StatusCorrected), input.Reason); err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, uint64(newID), input.Actor, "created_from_correction", "", string(StatusDraft), input.Reason); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(newID), nil
}

func (s Service) ReconcileStatement(ctx context.Context, statementID uint64) (Discrepancy, error) {
	if s.DB == nil || statementID == 0 {
		return Discrepancy{}, fmt.Errorf("database and statement are required")
	}
	var party PartyType
	var partyID uint64
	var start, end time.Time
	var sourceRaw string
	if err := s.DB.QueryRowContext(ctx, `
SELECT party_type, party_id, period_start, period_end, CAST(source_amount AS CHAR)
FROM acct_statement WHERE statement_id=?`, statementID).Scan(&party, &partyID, &start, &end, &sourceRaw); err != nil {
		return Discrepancy{}, err
	}
	expected, err := ParseMoney(sourceRaw)
	if err != nil {
		return Discrepancy{}, err
	}
	if expected < 0 {
		return Discrepancy{StatementID: statementID, Expected: expected}, ErrSourceDiscrepancy
	}
	actual, err := sourceAmount(ctx, s.DB, party, partyID, start, end)
	if err != nil {
		return Discrepancy{}, err
	}
	difference, err := actual.Sub(expected)
	if err != nil {
		return Discrepancy{}, err
	}
	result := Discrepancy{StatementID: statementID, Expected: expected, Actual: actual, Difference: difference}
	if result.Difference != 0 {
		return result, ErrSourceDiscrepancy
	}
	return result, nil
}

func (s Service) ReconcileMiddleman(ctx context.Context, start, end time.Time) (MiddlemanReconciliation, error) {
	result := MiddlemanReconciliation{
		PeriodStart: midnightUTC(start), PeriodEnd: midnightUTC(end), Currency: "USD",
	}
	if s.DB == nil {
		return result, fmt.Errorf("accounting database is nil")
	}
	if start.IsZero() || end.IsZero() || result.PeriodStart.After(result.PeriodEnd) {
		return result, fmt.Errorf("valid middleman reconciliation period is required")
	}
	var chargeRaw, payRaw, marginRaw string
	if err := s.DB.QueryRowContext(ctx, `
SELECT COALESCE(SUM(m.imps),0),
       CAST(COALESCE(ROUND(SUM(m.charge_spend),6),0) AS CHAR),
       CAST(COALESCE(ROUND(SUM(m.pay_spend),6),0) AS CHAR),
       CAST(COALESCE(ROUND(SUM(m.margin_spend),6),0) AS CHAR)
FROM daily_mid m INNER JOIN daily_log l USING (log_id)
WHERE l.daily BETWEEN ? AND ?`, dateString(start), dateString(end)).
		Scan(&result.Impressions, &chargeRaw, &payRaw, &marginRaw); err != nil {
		return result, err
	}
	var err error
	if result.Charge, err = ParseMoney(chargeRaw); err != nil {
		return result, err
	}
	if result.Pay, err = ParseMoney(payRaw); err != nil {
		return result, err
	}
	if result.Margin, err = ParseMoney(marginRaw); err != nil {
		return result, err
	}
	if result.Charge < 0 || result.Pay < 0 || result.Margin < 0 || result.Pay > result.Charge {
		return result, ErrSourceDiscrepancy
	}
	result.ExpectedMargin, err = result.Charge.Sub(result.Pay)
	if err != nil {
		return result, err
	}
	result.Difference, err = result.Margin.Sub(result.ExpectedMargin)
	if err != nil {
		return result, err
	}
	if result.Difference != 0 {
		return result, ErrSourceDiscrepancy
	}
	return result, nil
}

func WriteCSV(w io.Writer, statements []Statement) error {
	csvw := csv.NewWriter(w)
	if err := csvw.Write([]string{"statement_id", "request_key", "party_type", "party_id", "cadence", "period_start", "period_end", "currency", "source_amount", "adjustment_amount", "total_amount", "status", "external_ref"}); err != nil {
		return err
	}
	for _, statement := range statements {
		if err := csvw.Write([]string{
			strconv.FormatUint(statement.ID, 10), statement.RequestKey, string(statement.PartyType),
			strconv.FormatUint(statement.PartyID, 10), string(statement.Cadence), dateString(statement.PeriodStart),
			dateString(statement.PeriodEnd), statement.Currency, statement.SourceAmount.String(),
			statement.AdjustmentAmount.String(), statement.TotalAmount.String(), string(statement.Status),
			statement.ExternalRef.String,
		}); err != nil {
			return err
		}
	}
	csvw.Flush()
	return csvw.Error()
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func sourceAmount(ctx context.Context, db queryer, party PartyType, partyID uint64, start, end time.Time) (Money, error) {
	query := ""
	switch party {
	case PartyAdvertiser:
		query = `SELECT CAST(COALESCE(ROUND(SUM(a.spend),6),0) AS CHAR)
FROM daily_adv a INNER JOIN daily_log l USING (log_id)
WHERE a.adv_id=? AND l.daily BETWEEN ? AND ?`
	case PartyPublisher:
		query = `SELECT CAST(COALESCE(ROUND(SUM(p.spend),6),0) AS CHAR)
FROM daily_pub p INNER JOIN daily_log l USING (log_id)
WHERE p.pub_id=? AND l.daily BETWEEN ? AND ?`
	default:
		return 0, fmt.Errorf("invalid party type")
	}
	var raw string
	if err := db.QueryRowContext(ctx, query, partyID, dateString(start), dateString(end)).Scan(&raw); err != nil {
		return 0, err
	}
	amount, err := ParseMoney(raw)
	if err != nil {
		return 0, err
	}
	if amount < 0 {
		return 0, ErrSourceDiscrepancy
	}
	return amount, nil
}

func requireParty(ctx context.Context, db queryer, party PartyType, partyID uint64) error {
	table, key := "", ""
	switch party {
	case PartyAdvertiser:
		table, key = "adv", "adv_id"
	case PartyPublisher:
		table, key = "pub", "pub_id"
	default:
		return fmt.Errorf("invalid party type")
	}
	var one int
	return db.QueryRowContext(ctx, "SELECT 1 FROM "+table+" WHERE "+key+"=?", partyID).Scan(&one)
}

func (s Service) begin(ctx context.Context) (*sql.Tx, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("accounting database is nil")
	}
	return s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

func validateCreate(input CreateInput) error {
	if !safeRequestKey.MatchString(input.RequestKey) || input.PartyID == 0 || !validActor(input.Actor) || strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 500 {
		return fmt.Errorf("request key, party, actor, and bounded reason are required")
	}
	if input.PeriodStart.IsZero() || input.PeriodEnd.IsZero() {
		return fmt.Errorf("statement period dates are required")
	}
	if input.PartyType != PartyAdvertiser && input.PartyType != PartyPublisher {
		return fmt.Errorf("invalid party type")
	}
	start, end := midnightUTC(input.PeriodStart), midnightUTC(input.PeriodEnd)
	if start.After(end) {
		return fmt.Errorf("period start must not follow period end")
	}
	days := int(end.Sub(start)/(24*time.Hour)) + 1
	switch input.Cadence {
	case CadenceDaily:
		if days != 1 {
			return fmt.Errorf("daily statement must cover one UTC day")
		}
	case CadenceWeekly:
		if days != 7 || start.Weekday() != time.Monday {
			return fmt.Errorf("weekly statement must cover Monday through Sunday in UTC")
		}
	case CadenceMonthly:
		last := start.AddDate(0, 1, -start.Day())
		if start.Day() != 1 || end != last {
			return fmt.Errorf("monthly statement must cover one UTC calendar month")
		}
	default:
		return fmt.Errorf("invalid cadence")
	}
	return nil
}

func transitionAllowed(from, to Status) bool {
	switch from {
	case StatusDraft:
		return to == StatusHeld || to == StatusConfirmed
	case StatusHeld:
		return to == StatusDraft || to == StatusConfirmed
	case StatusConfirmed:
		return to == StatusHeld || to == StatusSettled
	default:
		return false
	}
}

func insertAudit(ctx context.Context, tx *sql.Tx, statementID uint64, actor, event, from, to, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO acct_audit (statement_id, actor, event, status_from, status_to, reason, created_at)
VALUES (?,?,?,?,?,?,UTC_TIMESTAMP())`, statementID, actor, event, nullEmpty(from), nullEmpty(to), reason)
	return err
}

func insertCreatedAudit(ctx context.Context, tx *sql.Tx, statementID uint64, actor, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO acct_audit (statement_id, actor, event, status_from, status_to, reason, created_at)
SELECT ?,?,'created',NULL,'Draft',?,UTC_TIMESTAMP() FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM acct_audit WHERE statement_id=? AND event='created'
)`, statementID, actor, reason, statementID)
	return err
}

func validActor(actor string) bool { return safeActor.MatchString(actor) }

func nullEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func midnightUTC(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dateString(value time.Time) string { return value.UTC().Format("2006-01-02") }

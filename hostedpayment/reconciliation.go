package hostedpayment

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/guruperl/aofei/accounting"
)

const hostedRetentionCleanupTimeout = 2 * time.Second

type Reconciliation struct {
	ID                  uint64
	RequestKey          string
	OperationID         sql.NullInt64
	BindingID           sql.NullInt64
	EventID             sql.NullInt64
	Category            string
	ProviderObjectToken sql.NullString
	Currency            string
	Amount              accounting.Money
	Fee                 accounting.Money
	Net                 accounting.Money
	Status              string
	Reason              string
	CreatedBy           string
	ResolvedBy          sql.NullString
	CreatedAt           time.Time
	ResolvedAt          sql.NullTime
}

type ReconciliationSummary struct {
	OperationID uint64
	Matched     bool
	Exceptions  uint64
	Amount      accounting.Money
	Fee         accounting.Money
	Net         accounting.Money
}

type SecretReadiness struct {
	APIKeyPresent          bool
	WebhookSecretPresent   bool
	PreviousWebhookPresent bool
	ModeMatches            bool
}

const maxBalanceTransactionsPerOperation = 32

func (s *Service) ReconcileOperation(ctx context.Context, actor Actor, operationID uint64, reason string) (ReconciliationSummary, error) {
	operation, err := s.Operation(ctx, operationID)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	if err := authorize(actor, PermissionReconcile, Scope{PartyType: operation.PartyType, PartyID: operation.PartyID}, true); err != nil {
		return ReconciliationSummary{}, err
	}
	if validateReason(reason) != nil {
		return ReconciliationSummary{}, fmt.Errorf("bounded reconciliation reason is required")
	}
	providerFacts, err := s.retrieveBalanceTransactions(ctx, operationID)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return ReconciliationSummary{}, err
	}
	defer tx.Rollback()
	operation, err = operationByIDTx(ctx, tx, operationID, true)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	for _, fact := range providerFacts {
		if err := recordBalanceTransaction(ctx, tx, operation, fact, actorName(actor)); err != nil {
			return ReconciliationSummary{}, err
		}
	}
	summary, err := summarizeOperationReconciliation(ctx, tx, operation)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	if operationNeedsSettlement(operation.Status) && !summary.Matched {
		category := "MissingEvent"
		exceptionReason := "completed provider operation has no matching available balance transaction"
		if summary.Amount != 0 {
			category = "ProviderMismatch"
			exceptionReason = "provider settlement total differs from approved operation amount"
		}
		requestKey := fmt.Sprintf("reconcile:%d:%s", operationID, category)
		result, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO hosted_reconciliation (request_key,operation_id,category,currency,
 amount,fee,net,status,reason,created_by,created_at)
VALUES (?,? ,?,'USD',?,?,?,'Unresolved',?,?,UTC_TIMESTAMP())`, requestKey, operationID, category,
			summary.Amount.String(), summary.Fee.String(), summary.Net.String(), exceptionReason, actorName(actor))
		if err != nil {
			return ReconciliationSummary{}, err
		}
		if affected, _ := result.RowsAffected(); affected == 1 {
			summary.Exceptions++
			metricReconciliationUnresolved.Add(1)
		}
	}
	if err := insertAudit(ctx, tx, "Operation", operationID, actorName(actor), "OperationReconciled", string(operation.Status), string(operation.Status), reason, operation.CurrentObjectToken.String); err != nil {
		return ReconciliationSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReconciliationSummary{}, err
	}
	return summary, nil
}

func operationNeedsSettlement(status OperationStatus) bool {
	switch status {
	case OperationSucceeded, OperationDisputed, OperationPartiallyRefunded, OperationRefunded:
		return true
	default:
		return false
	}
}

func (s *Service) retrieveBalanceTransactions(ctx context.Context, operationID uint64) ([]BalanceTransaction, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT provider_token FROM hosted_provider_object
WHERE provider='stripe' AND operation_id=? AND object_kind='BalanceTransaction'
ORDER BY object_id LIMIT ?`, operationID, maxBalanceTransactionsPerOperation+1)
	if err != nil {
		return nil, err
	}
	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			rows.Close()
			return nil, err
		}
		tokens = append(tokens, token)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(tokens) > maxBalanceTransactionsPerOperation {
		return nil, fmt.Errorf("%w: provider operation has too many balance transactions", ErrConflict)
	}
	facts := make([]BalanceTransaction, 0, len(tokens))
	for _, token := range tokens {
		fact, err := s.Provider.RetrieveBalanceTransaction(ctx, token)
		if err != nil {
			return nil, err
		}
		if fact.Token != token {
			return nil, invalidProviderResponse()
		}
		facts = append(facts, fact)
	}
	return facts, nil
}

func recordBalanceTransaction(ctx context.Context, tx *sql.Tx, operation Operation, fact BalanceTransaction, actor string) error {
	category := "Settlement"
	if operation.Kind == OperationRefund {
		category = "Refund"
	}
	status := "Matched"
	reason := "available provider balance transaction reconciles to the approved operation"
	amount, fee, net := centsMoney(fact.AmountCents), centsMoney(fact.FeeCents), centsMoney(fact.NetCents)
	mismatch := func(why string, zeroValues bool) {
		category, status, reason = "ProviderMismatch", "Unresolved", why
		if zeroValues {
			amount, fee, net = 0, 0, 0
		}
	}
	if err := ValidateOpaqueToken(fact.Token, "txn_"); err != nil {
		return invalidProviderResponse()
	}
	if strings.ToUpper(fact.Currency) != "USD" || amount == accounting.Money(math.MinInt64) ||
		fee == accounting.Money(math.MinInt64) || net == accounting.Money(math.MinInt64) {
		mismatch("provider balance currency or amount is outside the supported contract", true)
	} else if fact.FeeCents < 0 || subtractOverflows(fact.AmountCents, fact.FeeCents) || fact.AmountCents-fact.FeeCents != fact.NetCents {
		mismatch("provider balance amount, fee, and net are arithmetically inconsistent", fact.FeeCents < 0)
	} else if fact.Status != "available" {
		mismatch("provider balance transaction is not yet available", false)
	} else if absoluteMoney(amount) != operation.Amount || operation.Kind == OperationFunding && amount <= 0 ||
		(operation.Kind == OperationPayout || operation.Kind == OperationRefund) && amount >= 0 {
		mismatch("provider balance amount or direction differs from the approved operation", false)
	}
	var objectKind string
	var objectOwner uint64
	if err := tx.QueryRowContext(ctx, `SELECT object_kind,operation_id FROM hosted_provider_object WHERE provider='stripe' AND provider_token=? AND operation_id IS NOT NULL FOR UPDATE`, fact.Token).Scan(&objectKind, &objectOwner); err != nil {
		return err
	}
	if objectKind != "BalanceTransaction" || objectOwner != operation.ID {
		return fmt.Errorf("%w: provider balance transaction belongs to another owner", ErrConflict)
	}
	var sourceOwner uint64
	if ValidateOpaqueToken(fact.SourceToken) != nil {
		mismatch("provider balance source has no exact operation ownership", false)
	} else {
		sourceErr := tx.QueryRowContext(ctx, `SELECT operation_id FROM hosted_provider_object WHERE provider='stripe' AND provider_token=? AND operation_id IS NOT NULL FOR UPDATE`, fact.SourceToken).Scan(&sourceOwner)
		if sourceErr != nil && !errors.Is(sourceErr, sql.ErrNoRows) {
			return sourceErr
		}
		if sourceErr != nil || sourceOwner != operation.ID {
			mismatch("provider balance source has no exact operation ownership", false)
		}
	}
	tokenDigest := sha256.Sum256([]byte(fact.Token))
	requestKey := fmt.Sprintf("provider:%x:%s", tokenDigest[:16], category)
	result, err := tx.ExecContext(ctx, `
INSERT IGNORE INTO hosted_reconciliation (request_key,operation_id,category,provider_object_token,
 currency,amount,fee,net,status,reason,created_by,created_at)
VALUES (?,?,?,?,'USD',?,?,?,?,?,?,UTC_TIMESTAMP())`, requestKey, operation.ID, category, fact.Token,
		amount.String(), fee.String(), net.String(), status, reason, actor)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 1 && status == "Unresolved" {
		metricReconciliationUnresolved.Add(1)
	}
	return nil
}

func summarizeOperationReconciliation(ctx context.Context, tx *sql.Tx, operation Operation) (ReconciliationSummary, error) {
	var matchedCount, exceptionCount uint64
	var amountRaw, feeRaw, netRaw string
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(status='Matched'),0),COALESCE(SUM(status='Unresolved'),0),
 CAST(COALESCE(SUM(CASE WHEN status='Matched' THEN amount ELSE 0 END),0) AS CHAR),
 CAST(COALESCE(SUM(CASE WHEN status='Matched' THEN fee ELSE 0 END),0) AS CHAR),
 CAST(COALESCE(SUM(CASE WHEN status='Matched' THEN net ELSE 0 END),0) AS CHAR)
FROM hosted_reconciliation WHERE operation_id=?`, operation.ID).Scan(&matchedCount, &exceptionCount, &amountRaw, &feeRaw, &netRaw); err != nil {
		return ReconciliationSummary{}, err
	}
	amount, err := accounting.ParseMoney(amountRaw)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	fee, err := accounting.ParseMoney(feeRaw)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	net, err := accounting.ParseMoney(netRaw)
	if err != nil {
		return ReconciliationSummary{}, err
	}
	matched := matchedCount > 0 && absoluteMoney(amount) == operation.Amount
	return ReconciliationSummary{OperationID: operation.ID, Matched: matched, Exceptions: exceptionCount, Amount: amount, Fee: fee, Net: net}, nil
}

func (s *Service) ResolveReconciliation(ctx context.Context, actor Actor, id uint64, reason string) error {
	if id == 0 || validateReason(reason) != nil {
		return fmt.Errorf("reconciliation id and bounded reason are required")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	item, scope, err := reconciliationByIDTx(ctx, tx, id)
	if err != nil {
		return err
	}
	if err := authorize(actor, PermissionReconcile, scope, true); err != nil {
		return err
	}
	switch item.Category {
	case "Refund", "Dispute", "Chargeback", "PayoutFailure":
		if err := authorize(actor, PermissionDisputeHandle, scope, true); err != nil {
			return err
		}
	}
	if item.Status != "Unresolved" {
		return fmt.Errorf("%w: reconciliation is not unresolved", ErrConflict)
	}
	if item.CreatedBy == actorName(actor) {
		return fmt.Errorf("%w: exception recorder cannot resolve the same reconciliation", ErrConflict)
	}
	result, err := tx.ExecContext(ctx, `UPDATE hosted_reconciliation SET status='Resolved',resolved_by=?,resolved_at=UTC_TIMESTAMP() WHERE reconciliation_id=? AND status='Unresolved'`, actorName(actor), id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrConflict
	}
	if err := insertAudit(ctx, tx, "Reconciliation", id, actorName(actor), "ReconciliationResolved", "Unresolved", "Resolved", reason, item.ProviderObjectToken.String); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ListReconciliations(ctx context.Context, actor Actor, scope Scope) ([]Reconciliation, error) {
	if err := authorize(actor, PermissionRead, scope, false); err != nil {
		return nil, err
	}
	query := reconciliationSelect + `
LEFT JOIN hosted_operation o ON o.operation_id=r.operation_id
LEFT JOIN hosted_binding b ON b.binding_id=r.binding_id`
	var args []any
	if scope.PartyType != "" {
		query += ` WHERE COALESCE(o.party_type,b.party_type)=? AND COALESCE(o.party_id,b.party_id)=?`
		args = append(args, scope.PartyType, scope.PartyID)
	}
	query += ` ORDER BY r.reconciliation_id DESC LIMIT 200`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Reconciliation
	for rows.Next() {
		item, err := scanReconciliation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) CheckSecretReadiness(ctx context.Context, actor Actor, reason string) (SecretReadiness, error) {
	if err := authorize(actor, PermissionSecretReadiness, Scope{}, true); err != nil {
		return SecretReadiness{}, err
	}
	if actor.Role != "admin" || validateReason(reason) != nil {
		return SecretReadiness{}, fmt.Errorf("administrator and bounded readiness reason are required")
	}
	apiKey := strings.TrimSpace(os.Getenv(s.Config.APIKeyEnv))
	webhook := strings.TrimSpace(os.Getenv(s.Config.WebhookSecretEnv))
	previous := ""
	if s.Config.WebhookPreviousSecretEnv != "" {
		previous = strings.TrimSpace(os.Getenv(s.Config.WebhookPreviousSecretEnv))
	}
	modeMatches := apiKeyMatchesMode(apiKey, s.Config.LiveMode)
	readiness := SecretReadiness{
		APIKeyPresent: apiKey != "", WebhookSecretPresent: len(webhook) >= 16,
		PreviousWebhookPresent: previous != "", ModeMatches: modeMatches,
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return SecretReadiness{}, err
	}
	defer tx.Rollback()
	state := "NotReady"
	if readiness.APIKeyPresent && readiness.WebhookSecretPresent && readiness.ModeMatches {
		state = "Ready"
	}
	if err := insertAudit(ctx, tx, "SecretReadiness", 0, actorName(actor), "SecretReadinessChecked", "Unknown", state, reason, ""); err != nil {
		return SecretReadiness{}, err
	}
	if err := tx.Commit(); err != nil {
		return SecretReadiness{}, err
	}
	return readiness, nil
}

func (s *Service) pruneEvents(ctx context.Context, actor Actor, limit int, reason string) (deleted int64, err error) {
	if err := authorizeMaintenance(actor, PermissionRetentionPrune); err != nil {
		return 0, err
	}
	if actor.Role != "admin" || limit < 1 || limit > 10_000 || validateReason(reason) != nil {
		return 0, fmt.Errorf("administrator, bounded limit, and reason are required")
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return 0, err
	}
	clean := false
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), hostedRetentionCleanupTimeout)
		defer cancel()
		if _, cleanupErr := conn.ExecContext(cleanupCtx, `SET @aofei_hosted_event_retention=0`); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("clear hosted-event retention mode: %w", cleanupErr), driver.ErrBadConn)
		} else {
			clean = true
		}
		if !clean {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
		err = errors.Join(err, conn.Close())
	}()
	if _, err := conn.ExecContext(ctx, `SET @aofei_hosted_event_retention=1`); err != nil {
		return 0, err
	}
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
DELETE FROM hosted_event
WHERE received_at<UTC_TIMESTAMP()-INTERVAL ? DAY
  AND NOT EXISTS (
    SELECT 1 FROM hosted_reconciliation r
    WHERE r.event_id=hosted_event.event_id
  )
ORDER BY received_at LIMIT ?`, s.Config.EventRetentionDays, limit)
	if err != nil {
		return 0, err
	}
	deleted, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, "Event", 0, actorName(actor), "ProviderEventsPruned", "Expired", fmt.Sprintf("Deleted:%d", deleted), reason, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (m *MaintenanceService) PruneEvents(ctx context.Context, actor Actor, limit int, reason string) (int64, error) {
	if m == nil || m.service == nil {
		return 0, sql.ErrConnDone
	}
	return m.service.pruneEvents(ctx, actor, limit, reason)
}

const reconciliationSelect = `SELECT r.reconciliation_id,r.request_key,r.operation_id,r.binding_id,
r.event_id,r.category,r.provider_object_token,r.currency,CAST(r.amount AS CHAR),
CAST(r.fee AS CHAR),CAST(r.net AS CHAR),r.status,r.reason,r.created_by,r.resolved_by,
r.created_at,r.resolved_at FROM hosted_reconciliation r `

func scanReconciliation(row scanner) (Reconciliation, error) {
	var item Reconciliation
	var amountRaw, feeRaw, netRaw string
	err := row.Scan(&item.ID, &item.RequestKey, &item.OperationID, &item.BindingID, &item.EventID,
		&item.Category, &item.ProviderObjectToken, &item.Currency, &amountRaw, &feeRaw, &netRaw,
		&item.Status, &item.Reason, &item.CreatedBy, &item.ResolvedBy, &item.CreatedAt, &item.ResolvedAt)
	if err != nil {
		return item, err
	}
	if item.Amount, err = accounting.ParseMoney(amountRaw); err != nil {
		return item, err
	}
	if item.Fee, err = accounting.ParseMoney(feeRaw); err != nil {
		return item, err
	}
	if item.Net, err = accounting.ParseMoney(netRaw); err != nil {
		return item, err
	}
	return item, nil
}

func reconciliationByIDTx(ctx context.Context, tx *sql.Tx, id uint64) (Reconciliation, Scope, error) {
	item, err := scanReconciliation(tx.QueryRowContext(ctx, reconciliationSelect+`
WHERE r.reconciliation_id=? FOR UPDATE`, id))
	if err != nil {
		return item, Scope{}, err
	}
	var party sql.NullString
	var partyID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(o.party_type,b.party_type),COALESCE(o.party_id,b.party_id)
FROM hosted_reconciliation r
LEFT JOIN hosted_operation o ON o.operation_id=r.operation_id
LEFT JOIN hosted_binding b ON b.binding_id=r.binding_id
WHERE r.reconciliation_id=?`, id).Scan(&party, &partyID); err != nil {
		return item, Scope{}, err
	}
	if party.Valid != partyID.Valid {
		return item, Scope{}, fmt.Errorf("reconciliation owner is incomplete")
	}
	var scope Scope
	if party.Valid {
		scope = Scope{PartyType: PartyType(party.String), PartyID: uint64(partyID.Int64)}
	}
	return item, scope, nil
}

func absoluteMoney(value accounting.Money) accounting.Money {
	if int64(value) == math.MinInt64 {
		return accounting.Money(math.MaxInt64)
	}
	if value < 0 {
		return -value
	}
	return value
}

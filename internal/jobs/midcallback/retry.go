package midcallback

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/guruperl/aofei/internal/opsmetrics"
	"github.com/guruperl/aofei/internal/safehttp"
)

const (
	StatusPending    = "Pending"
	StatusRetrying   = "Retrying"
	StatusProcessing = "Processing"
	StatusSucceeded  = "Succeeded"
	StatusAbandoned  = "Abandoned"
)

type Failure struct {
	Token          string
	Source         string
	CallbackURL    string
	BidderID       uint32
	GroupID        uint32
	RouteBidderID  uint32
	TargetID       uint32
	AuctionID      string
	ImpID          string
	AuctionBidID   string
	ChargePrice    float64
	PayPrice       float64
	Currency       string
	HTTPStatus     int
	ForwardOutcome string
	NextAttemptAt  time.Time
}

type Row struct {
	RetryID     uint64
	Token       string
	Source      string
	CallbackURL string
	Attempts    int
}

type Options struct {
	Limit                int
	MaxAttempts          int
	Timeout              time.Duration
	BaseBackoff          time.Duration
	MaxBackoff           time.Duration
	StaleProcessingAfter time.Duration
	DryRun               bool
	Client               *http.Client
	Now                  func() time.Time
}

type Result struct {
	Selected    int
	Forwarded   int
	Succeeded   int
	Retrying    int
	Abandoned   int
	StateErrors int
}

// PostForwardStateError means callback processing completed but the fixed-state
// database transition was not durably confirmed. Forwarded distinguishes true
// at-least-once delivery uncertainty from a failure after local rejection. The
// error deliberately carries no retry id, token, URL, payload, bidder, or
// auction identifier.
type PostForwardStateError struct {
	ForwardOutcome string
	TargetState    string
	Forwarded      bool
}

func (err *PostForwardStateError) Error() string {
	if !err.Forwarded {
		return fmt.Sprintf("callback outcome %s has unconfirmed %s state", err.ForwardOutcome, err.TargetState)
	}
	return fmt.Sprintf("callback forward outcome %s has uncertain %s state", err.ForwardOutcome, err.TargetState)
}

type BacklogStats struct {
	Due             int
	StaleProcessing int
}

func Enqueue(ctx context.Context, db *sql.DB, failure Failure) error {
	if db == nil {
		return nil
	}
	if failure.Token == "" {
		return fmt.Errorf("callback retry token is required")
	}
	if failure.Source == "" {
		return fmt.Errorf("callback retry source is required")
	}
	if failure.CallbackURL == "" {
		return fmt.Errorf("callback retry URL is required")
	}
	if failure.Currency == "" {
		failure.Currency = "USD"
	}
	if failure.NextAttemptAt.IsZero() {
		failure.NextAttemptAt = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO mid_callback_retry
	(token, source, callback_url, bidder_id, group_id, route_bidder_id, target_id,
	 auction_id, imp_id, auction_bid_id, charge_price, pay_price, currency,
	 attempts, next_attempt_at, status, last_http_status, last_error, created, updated)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, 'Pending', ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE
	callback_url=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), callback_url, VALUES(callback_url)),
	bidder_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), bidder_id, VALUES(bidder_id)),
	group_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), group_id, VALUES(group_id)),
	route_bidder_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), route_bidder_id, VALUES(route_bidder_id)),
	target_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), target_id, VALUES(target_id)),
	auction_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), auction_id, VALUES(auction_id)),
	imp_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), imp_id, VALUES(imp_id)),
	auction_bid_id=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), auction_bid_id, VALUES(auction_bid_id)),
	charge_price=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), charge_price, VALUES(charge_price)),
	pay_price=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), pay_price, VALUES(pay_price)),
	currency=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), currency, VALUES(currency)),
	last_http_status=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), last_http_status, VALUES(last_http_status)),
	last_error=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), last_error, VALUES(last_error)),
	updated=IF(status IN ('Processing', 'Succeeded', 'Abandoned'), updated, NOW())`,
		failure.Token, failure.Source, failure.CallbackURL, failure.BidderID, failure.GroupID,
		failure.RouteBidderID, failure.TargetID, failure.AuctionID, failure.ImpID,
		failure.AuctionBidID, failure.ChargePrice, failure.PayPrice, failure.Currency,
		failure.NextAttemptAt, nullableHTTPStatus(failure.HTTPStatus), nullableError(storedForwardEvidence(failure.ForwardOutcome)))
	return err
}

func Run(ctx context.Context, db *sql.DB, opts Options) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("database is nil")
	}
	opts = opts.withDefaults()
	rowSelector := claimDueRows
	if opts.DryRun {
		rowSelector = dueRows
	}
	now := opts.now().UTC()
	staleCutoff := now.Add(-opts.StaleProcessingAfter)
	rows, err := rowSelector(ctx, db, opts.Limit, opts.MaxAttempts, now, staleCutoff)
	if err != nil {
		return Result{}, err
	}
	result := Result{Selected: len(rows)}
	if opts.DryRun {
		return result, nil
	}
	for _, row := range rows {
		status, code, lastErr, forwarded := forward(ctx, row.CallbackURL, opts)
		if forwarded {
			result.Forwarded++
		}
		attempts := row.Attempts + 1
		switch {
		case status == "ok":
			opsmetrics.RecordCallbackRetry("forward_succeeded")
			if err := markSucceeded(ctx, db, row.RetryID, attempts, code); err != nil {
				return postForwardStateFailure(result, status, StatusSucceeded, forwarded)
			}
			opsmetrics.RecordCallbackRetry("state_succeeded")
			result.Succeeded++
		case !RetryableForward(status, code) || attempts >= opts.MaxAttempts:
			if forwarded {
				opsmetrics.RecordCallbackRetry("forward_abandoned")
			} else {
				opsmetrics.RecordCallbackRetry("forward_rejected")
			}
			if err := markAbandoned(ctx, db, row.RetryID, attempts, code, lastErr); err != nil {
				return postForwardStateFailure(result, status, StatusAbandoned, forwarded)
			}
			opsmetrics.RecordCallbackRetry("state_abandoned")
			result.Abandoned++
		default:
			opsmetrics.RecordCallbackRetry("forward_retryable")
			next := opts.now().Add(backoff(attempts, opts.BaseBackoff, opts.MaxBackoff))
			if err := markRetrying(ctx, db, row.RetryID, attempts, code, lastErr, next); err != nil {
				return postForwardStateFailure(result, status, StatusRetrying, forwarded)
			}
			opsmetrics.RecordCallbackRetry("state_retrying")
			result.Retrying++
		}
	}
	return result, nil
}

func postForwardStateFailure(result Result, forwardOutcome, targetState string, forwarded bool) (Result, error) {
	result.StateErrors++
	stateOutcome := "state_error_before_forward"
	if forwarded {
		stateOutcome = "state_error_after_forward"
	}
	opsmetrics.RecordCallbackRetry(stateOutcome)
	return result, &PostForwardStateError{
		ForwardOutcome: normalizeForwardOutcome(forwardOutcome),
		TargetState:    normalizeTargetState(targetState),
		Forwarded:      forwarded,
	}
}

func normalizeForwardOutcome(outcome string) string {
	switch outcome {
	case "ok", "error", "request_error", "http_error", "invalid_url":
		return outcome
	default:
		return "other"
	}
}

// storedForwardEvidence deliberately maps the internal forward disposition to
// a closed vocabulary. Guarded URL and transport errors can contain callback
// hosts, resolved addresses, redirect targets, or dependency detail and must
// never be copied into durable retry/recovery evidence.
func storedForwardEvidence(outcome string) string {
	switch normalizeForwardOutcome(outcome) {
	case "ok":
		return ""
	case "invalid_url":
		return "callback URL rejected"
	case "request_error":
		return "callback request rejected"
	case "error":
		return "callback request failed"
	case "http_error":
		return "callback HTTP rejected"
	default:
		return "callback outcome unavailable"
	}
}

func normalizeTargetState(state string) string {
	switch state {
	case StatusSucceeded, StatusRetrying, StatusAbandoned:
		return state
	default:
		return "Unknown"
	}
}

func RetryableForward(status string, code int) bool {
	switch status {
	case "error", "request_error":
		return true
	case "http_error":
		return code == http.StatusTooManyRequests || code >= 500
	default:
		return false
	}
}

func Backlog(ctx context.Context, db *sql.DB, opts Options) (BacklogStats, error) {
	if db == nil {
		return BacklogStats{}, fmt.Errorf("database is nil")
	}
	opts = opts.withDefaults()
	now := opts.now().UTC()
	staleCutoff := now.Add(-opts.StaleProcessingAfter)
	var due, stale sql.NullInt64
	err := db.QueryRowContext(ctx, `
SELECT
  SUM(CASE WHEN status IN ('Pending', 'Retrying') AND next_attempt_at <= ? THEN 1 ELSE 0 END) AS due_rows,
  SUM(CASE WHEN status = 'Processing' AND (claimed_at IS NULL OR claimed_at < ?) THEN 1 ELSE 0 END) AS stale_processing_rows
FROM mid_callback_retry
WHERE attempts < ?`, now, staleCutoff, opts.MaxAttempts).Scan(&due, &stale)
	if err != nil {
		return BacklogStats{}, err
	}
	stats := BacklogStats{}
	if due.Valid {
		stats.Due = int(due.Int64)
	}
	if stale.Valid {
		stats.StaleProcessing = int(stale.Int64)
	}
	return stats, nil
}

func dueRows(ctx context.Context, db *sql.DB, limit, maxAttempts int, now, staleCutoff time.Time) ([]Row, error) {
	rows, err := db.QueryContext(ctx, `
SELECT retry_id, token, source, callback_url, attempts
FROM mid_callback_retry
WHERE attempts < ?
  AND (
    (status IN ('Pending', 'Retrying') AND next_attempt_at <= ?)
    OR (status = 'Processing' AND (claimed_at IS NULL OR claimed_at < ?))
  )
ORDER BY next_attempt_at ASC, retry_id ASC
LIMIT ?`, maxAttempts, now, staleCutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.RetryID, &row.Token, &row.Source, &row.CallbackURL, &row.Attempts); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func claimDueRows(ctx context.Context, db *sql.DB, limit, maxAttempts int, now, staleCutoff time.Time) ([]Row, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	rows, err := tx.QueryContext(ctx, `
	SELECT retry_id, token, source, callback_url, attempts
	FROM mid_callback_retry
	WHERE attempts < ?
	  AND (
	    (status IN ('Pending', 'Retrying') AND next_attempt_at <= ?)
	    OR (status = 'Processing' AND (claimed_at IS NULL OR claimed_at < ?))
	  )
	ORDER BY next_attempt_at ASC, retry_id ASC
	LIMIT ?
	FOR UPDATE`, maxAttempts, now, staleCutoff, limit)
	if err != nil {
		return nil, err
	}
	var out []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.RetryID, &row.Token, &row.Source, &row.CallbackURL, &row.Attempts); err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		committed = true
		return out, nil
	}

	ids := make([]any, 0, len(out))
	placeholders := ""
	for i, row := range out {
		if i != 0 {
			placeholders += ","
		}
		placeholders += "?"
		ids = append(ids, row.RetryID)
	}
	result, err := tx.ExecContext(ctx, `
	UPDATE mid_callback_retry
	SET status='Processing', claimed_at=?, updated=NOW()
	WHERE retry_id IN (`+placeholders+`)`, append([]any{now}, ids...)...)
	if err != nil {
		return nil, err
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if claimed != int64(len(out)) {
		return nil, fmt.Errorf("callback claim affected %d rows, want %d", claimed, len(out))
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return out, nil
}

func forward(ctx context.Context, raw string, opts Options) (string, int, string, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "invalid_url", 0, "invalid callback URL", false
	}
	if err := safehttp.ValidateCallbackURL(ctx, raw); err != nil {
		return "invalid_url", 0, storedForwardEvidence("invalid_url"), false
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, raw, nil)
	if err != nil {
		return "request_error", 0, storedForwardEvidence("request_error"), false
	}
	client := safehttp.NewCallbackClient(opts.Client)
	resp, err := client.Do(req)
	if err != nil {
		return "error", 0, storedForwardEvidence("error"), true
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "http_error", resp.StatusCode, storedForwardEvidence("http_error"), true
	}
	return "ok", resp.StatusCode, "", true
}

func markSucceeded(ctx context.Context, db *sql.DB, retryID uint64, attempts, code int) error {
	result, err := db.ExecContext(ctx, `
	UPDATE mid_callback_retry
	SET status='Succeeded', attempts=?, claimed_at=NULL, last_http_status=?, last_error=NULL, updated=NOW()
	WHERE retry_id=? AND status='Processing'`, attempts, nullableHTTPStatus(code), retryID)
	return requireSingleStateTransition(result, err)
}

func markRetrying(ctx context.Context, db *sql.DB, retryID uint64, attempts, code int, lastErr string, next time.Time) error {
	result, err := db.ExecContext(ctx, `
	UPDATE mid_callback_retry
	SET status='Retrying', attempts=?, next_attempt_at=?, claimed_at=NULL, last_http_status=?, last_error=?, updated=NOW()
	WHERE retry_id=? AND status='Processing'`, attempts, next.UTC(), nullableHTTPStatus(code), nullableError(lastErr), retryID)
	return requireSingleStateTransition(result, err)
}

func markAbandoned(ctx context.Context, db *sql.DB, retryID uint64, attempts, code int, lastErr string) error {
	result, err := db.ExecContext(ctx, `
	UPDATE mid_callback_retry
	SET status='Abandoned', attempts=?, claimed_at=NULL, last_http_status=?, last_error=?, updated=NOW()
	WHERE retry_id=? AND status='Processing'`, attempts, nullableHTTPStatus(code), nullableError(lastErr), retryID)
	return requireSingleStateTransition(result, err)
}

func requireSingleStateTransition(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("callback state transition affected %d rows, want 1", rows)
	}
	return nil
}

func (opts Options) withDefaults() Options {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 5
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Second
	}
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = time.Minute
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = time.Hour
	}
	if opts.StaleProcessingAfter <= 0 {
		opts.StaleProcessingAfter = 10 * time.Minute
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return opts
}

func (opts Options) now() time.Time {
	if opts.Now == nil {
		return time.Now()
	}
	return opts.Now()
}

func backoff(attempts int, base, max time.Duration) time.Duration {
	if attempts <= 1 {
		return base
	}
	multiplier := math.Pow(2, float64(attempts-1))
	delay := time.Duration(float64(base) * multiplier)
	if delay > max {
		return max
	}
	return delay
}

func nullableHTTPStatus(code int) interface{} {
	if code <= 0 {
		return nil
	}
	return code
}

func nullableError(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

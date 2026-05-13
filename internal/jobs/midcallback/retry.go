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
)

const (
	StatusPending    = "Pending"
	StatusRetrying   = "Retrying"
	StatusProcessing = "Processing"
	StatusSucceeded  = "Succeeded"
	StatusAbandoned  = "Abandoned"
)

type Failure struct {
	Token         string
	Source        string
	CallbackURL   string
	BidderID      uint32
	GroupID       uint32
	RouteBidderID uint32
	TargetID      uint32
	AuctionID     string
	ImpID         string
	AuctionBidID  string
	ChargePrice   float64
	PayPrice      float64
	Currency      string
	HTTPStatus    int
	LastError     string
	NextAttemptAt time.Time
}

type Row struct {
	RetryID     uint64
	Token       string
	Source      string
	CallbackURL string
	Attempts    int
}

type Options struct {
	Limit       int
	MaxAttempts int
	Timeout     time.Duration
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	DryRun      bool
	Client      *http.Client
	Now         func() time.Time
}

type Result struct {
	Selected  int
	Succeeded int
	Retrying  int
	Abandoned int
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
	callback_url=VALUES(callback_url),
	bidder_id=VALUES(bidder_id),
	group_id=VALUES(group_id),
	route_bidder_id=VALUES(route_bidder_id),
	target_id=VALUES(target_id),
	auction_id=VALUES(auction_id),
	imp_id=VALUES(imp_id),
	auction_bid_id=VALUES(auction_bid_id),
	charge_price=VALUES(charge_price),
	pay_price=VALUES(pay_price),
	currency=VALUES(currency),
	next_attempt_at=IF(status IN ('Succeeded', 'Abandoned'), next_attempt_at, VALUES(next_attempt_at)),
	status=IF(status IN ('Succeeded', 'Abandoned'), status, 'Pending'),
	last_http_status=VALUES(last_http_status),
	last_error=VALUES(last_error),
	updated=NOW()`,
		failure.Token, failure.Source, failure.CallbackURL, failure.BidderID, failure.GroupID,
		failure.RouteBidderID, failure.TargetID, failure.AuctionID, failure.ImpID,
		failure.AuctionBidID, failure.ChargePrice, failure.PayPrice, failure.Currency,
		failure.NextAttemptAt, nullableHTTPStatus(failure.HTTPStatus), nullableError(failure.LastError))
	return err
}

func Run(ctx context.Context, db *sql.DB, opts Options) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("database is nil")
	}
	opts = opts.withDefaults()
	rows, err := dueRows(ctx, db, opts.Limit, opts.MaxAttempts)
	if err != nil {
		return Result{}, err
	}
	result := Result{Selected: len(rows)}
	if opts.DryRun {
		return result, nil
	}
	for _, row := range rows {
		status, code, lastErr := forward(ctx, row.CallbackURL, opts)
		attempts := row.Attempts + 1
		switch {
		case status == "ok":
			if err := markSucceeded(ctx, db, row.RetryID, attempts, code); err != nil {
				return result, err
			}
			result.Succeeded++
		case !RetryableForward(status, code) || attempts >= opts.MaxAttempts:
			if err := markAbandoned(ctx, db, row.RetryID, attempts, code, lastErr); err != nil {
				return result, err
			}
			result.Abandoned++
		default:
			next := opts.now().Add(backoff(attempts, opts.BaseBackoff, opts.MaxBackoff))
			if err := markRetrying(ctx, db, row.RetryID, attempts, code, lastErr, next); err != nil {
				return result, err
			}
			result.Retrying++
		}
	}
	return result, nil
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

func dueRows(ctx context.Context, db *sql.DB, limit, maxAttempts int) ([]Row, error) {
	rows, err := db.QueryContext(ctx, `
SELECT retry_id, token, source, callback_url, attempts
FROM mid_callback_retry
WHERE status IN ('Pending', 'Retrying')
  AND next_attempt_at <= NOW()
  AND attempts < ?
ORDER BY next_attempt_at ASC, retry_id ASC
LIMIT ?`, maxAttempts, limit)
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

func forward(ctx context.Context, raw string, opts Options) (string, int, string) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "invalid_url", 0, "invalid callback URL"
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, raw, nil)
	if err != nil {
		return "request_error", 0, err.Error()
	}
	client := opts.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "error", 0, err.Error()
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "http_error", resp.StatusCode, resp.Status
	}
	return "ok", resp.StatusCode, ""
}

func markSucceeded(ctx context.Context, db *sql.DB, retryID uint64, attempts, code int) error {
	_, err := db.ExecContext(ctx, `
UPDATE mid_callback_retry
SET status='Succeeded', attempts=?, last_http_status=?, last_error=NULL, updated=NOW()
WHERE retry_id=?`, attempts, nullableHTTPStatus(code), retryID)
	return err
}

func markRetrying(ctx context.Context, db *sql.DB, retryID uint64, attempts, code int, lastErr string, next time.Time) error {
	_, err := db.ExecContext(ctx, `
UPDATE mid_callback_retry
SET status='Retrying', attempts=?, next_attempt_at=?, last_http_status=?, last_error=?, updated=NOW()
WHERE retry_id=?`, attempts, next.UTC(), nullableHTTPStatus(code), nullableError(lastErr), retryID)
	return err
}

func markAbandoned(ctx context.Context, db *sql.DB, retryID uint64, attempts, code int, lastErr string) error {
	_, err := db.ExecContext(ctx, `
UPDATE mid_callback_retry
SET status='Abandoned', attempts=?, last_http_status=?, last_error=?, updated=NOW()
WHERE retry_id=?`, attempts, nullableHTTPStatus(code), nullableError(lastErr), retryID)
	return err
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

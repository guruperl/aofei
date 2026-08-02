package managementapi

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const apiAuditCleanupTimeout = 2 * time.Second

// PruneAudit applies the shared account-security retention class to immutable
// management API evidence. Callers must use the separated maintenance database
// principal; the HTTP principal must not have DELETE permission.
func PruneAudit(ctx context.Context, db *sql.DB, actor Actor, retentionDays, limit int, reason string) (deleted int64, err error) {
	if actor.Role != "admin" || actor.ID == 0 {
		return 0, fmt.Errorf("an administrator actor is required")
	}
	if retentionDays < 365 || retentionDays > 2555 {
		return 0, fmt.Errorf("audit retention must be between 365 and 2555 days")
	}
	if limit < 1 || limit > 10000 {
		return 0, fmt.Errorf("prune limit must be between 1 and 10000")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 255 || strings.ContainsAny(reason, "\r\n\x00") {
		return 0, fmt.Errorf("a single-line reason of at most 255 bytes is required")
	}
	if db == nil {
		return 0, fmt.Errorf("management API audit database is nil")
	}
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -retentionDays)
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SET @aofei_api_audit_retention=1`); err != nil {
		return 0, err
	}
	bypassActive := true
	defer func() {
		if bypassActive {
			err = errors.Join(err, clearAPIAuditRetentionBypass(conn))
		}
	}()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM api_audit WHERE created_at<? ORDER BY audit_id LIMIT ?`, cutoff, limit)
	if err != nil {
		return 0, err
	}
	deleted, err = result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := insertAPIAudit(ctx, tx, apiAudit{
		Actor: actor, Event: "ManagementAPIAuditPruned", ObjectType: "audit",
		PriorState: strconv.FormatInt(deleted, 10) + "ExpiredRows",
		NewState:   "RetentionApplied", Reason: reason, Outcome: "Success", CreatedAt: now,
	}); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if err := clearAPIAuditRetentionBypass(conn); err != nil {
		return 0, err
	}
	bypassActive = false
	return deleted, nil
}

func clearAPIAuditRetentionBypass(conn *sql.Conn) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), apiAuditCleanupTimeout)
	defer cancel()
	if _, err := conn.ExecContext(cleanupCtx, `SET @aofei_api_audit_retention=0`); err != nil {
		// A connection with an unconfirmed session-scoped deletion bypass must
		// never return to database/sql's idle pool.
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		return fmt.Errorf("clear management API audit retention bypass: %w", err)
	}
	return nil
}

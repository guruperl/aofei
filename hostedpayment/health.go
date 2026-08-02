package hostedpayment

import (
	"context"
	"database/sql"
)

type OperationalHealth struct {
	ApprovedHeldStatements  uint64 `json:"approved_held_statements"`
	StuckSubmitting         uint64 `json:"stuck_submitting"`
	StuckCanceling          uint64 `json:"stuck_canceling"`
	StaleSubmitted          uint64 `json:"stale_submitted"`
	UnresolvedExceptions    uint64 `json:"unresolved_exceptions"`
	UnresolvedPastPolicy    uint64 `json:"unresolved_past_policy"`
	OldestUnresolvedSeconds uint64 `json:"oldest_unresolved_seconds"`
	WebhookEvents24Hours    uint64 `json:"webhook_events_24_hours"`
}

func (s *Service) OperationalHealth(ctx context.Context) (OperationalHealth, error) {
	var health OperationalHealth
	var oldest sql.NullInt64
	err := s.DB.QueryRowContext(ctx, `
SELECT
 (SELECT COUNT(*) FROM hosted_operation o JOIN acct_statement a ON a.statement_id=o.statement_id
   WHERE o.status IN ('Approved','Submitting','Submitted','Canceling','Succeeded','Disputed','PartiallyRefunded','Refunded') AND a.status='Held'),
 (SELECT COUNT(*) FROM hosted_operation WHERE status='Submitting' AND updated_at<UTC_TIMESTAMP()-INTERVAL 2 MINUTE),
 (SELECT COUNT(*) FROM hosted_operation WHERE status='Canceling' AND updated_at<UTC_TIMESTAMP()-INTERVAL 2 MINUTE),
 (SELECT COUNT(*) FROM hosted_operation WHERE status='Submitted' AND updated_at<UTC_TIMESTAMP()-INTERVAL 1 HOUR),
 (SELECT COUNT(*) FROM hosted_reconciliation WHERE status='Unresolved'),
 (SELECT COUNT(*) FROM hosted_reconciliation WHERE status='Unresolved' AND created_at<UTC_TIMESTAMP()-INTERVAL ? DAY),
 (SELECT TIMESTAMPDIFF(SECOND,MIN(created_at),UTC_TIMESTAMP()) FROM hosted_reconciliation WHERE status='Unresolved'),
 (SELECT COUNT(*) FROM hosted_event WHERE received_at>=UTC_TIMESTAMP()-INTERVAL 24 HOUR)`, s.Config.ReconciliationMaxAgeDays).
		Scan(&health.ApprovedHeldStatements, &health.StuckSubmitting, &health.StuckCanceling, &health.StaleSubmitted,
			&health.UnresolvedExceptions, &health.UnresolvedPastPolicy, &oldest, &health.WebhookEvents24Hours)
	if err != nil {
		return OperationalHealth{}, err
	}
	if oldest.Valid && oldest.Int64 > 0 {
		health.OldestUnresolvedSeconds = uint64(oldest.Int64)
	}
	return health, nil
}

func (m *MaintenanceService) OperationalHealth(ctx context.Context) (OperationalHealth, error) {
	if m == nil || m.service == nil {
		return OperationalHealth{}, sql.ErrConnDone
	}
	return m.service.OperationalHealth(ctx)
}

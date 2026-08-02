// Package action provides bounded maintenance operations for R01 analytical
// action and attribution facts. It never reads or mutates delivery reservation,
// accounting statement, adjustment, or settlement tables.
package action

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

var pseudonymPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Service struct {
	DB *sql.DB
}

type actionCandidate struct {
	ID       uint64
	Lineage  []byte
	Occurred time.Time
}

// Reconcile attributes previously-unattributed action facts after a tracking
// touch that arrived or was restored later. Existing click/view decisions are
// immutable; click has precedence over view for candidates still unattributed.
func (s Service) Reconcile(ctx context.Context, clickWindow, viewWindow time.Duration, limit int) (int64, error) {
	if s.DB == nil || clickWindow <= 0 || viewWindow <= 0 || viewWindow > clickWindow || limit < 1 || limit > 10000 {
		return 0, fmt.Errorf("invalid action reconciliation configuration")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
SELECT action_id, lineage_hash, occurred_at
FROM measurement_action
WHERE attribution_type='unattributed' AND expires_at>UTC_TIMESTAMP(6)
ORDER BY action_id
LIMIT ? FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, err
	}
	var candidates []actionCandidate
	for rows.Next() {
		var candidate actionCandidate
		if err := rows.Scan(&candidate.ID, &candidate.Lineage, &candidate.Occurred); err != nil {
			rows.Close()
			return 0, err
		}
		if len(candidate.Lineage) != 32 {
			rows.Close()
			return 0, fmt.Errorf("action %d has invalid lineage hash", candidate.ID)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var updated int64
	for _, candidate := range candidates {
		var touchType string
		var touchAt time.Time
		err := tx.QueryRowContext(ctx, `
SELECT touch_type, occurred_at
FROM measurement_touch
WHERE lineage_hash=? AND occurred_at<=?
  AND ((touch_type='click' AND occurred_at>=?) OR (touch_type='view' AND occurred_at>=?))
ORDER BY (touch_type='click') DESC, occurred_at DESC
LIMIT 1`, candidate.Lineage, candidate.Occurred, candidate.Occurred.Add(-clickWindow), candidate.Occurred.Add(-viewWindow)).Scan(&touchType, &touchAt)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return 0, err
		}
		result, err := tx.ExecContext(ctx, `
UPDATE measurement_action
SET attribution_type=?, touch_at=?
WHERE action_id=? AND attribution_type='unattributed'`, touchType, touchAt.UTC(), candidate.ID)
		if err != nil {
			return 0, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		updated += count
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return updated, nil
}

// Prune removes expired action and touch rows in bounded batches.
func (s Service) Prune(ctx context.Context, limit int) (actions, touches int64, err error) {
	if s.DB == nil || limit < 1 || limit > 100000 {
		return 0, 0, fmt.Errorf("invalid action prune configuration")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM measurement_action WHERE expires_at<=UTC_TIMESTAMP(6) ORDER BY action_id LIMIT ?`, limit)
	if err != nil {
		return 0, 0, err
	}
	if actions, err = result.RowsAffected(); err != nil {
		return 0, 0, err
	}
	result, err = tx.ExecContext(ctx, `DELETE FROM measurement_touch WHERE expires_at<=UTC_TIMESTAMP(6) ORDER BY touch_id LIMIT ?`, limit)
	if err != nil {
		return 0, 0, err
	}
	if touches, err = result.RowsAffected(); err != nil {
		return 0, 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return actions, touches, nil
}

// DeletePseudonym removes the scoped action facts and their now-orphaned
// tracking touches. The pseudonym is the domain-separated R01 value returned
// by an authorized export process, never a raw browser or consent identifier.
func (s Service) DeletePseudonym(ctx context.Context, pseudonym string) (actions, touches int64, err error) {
	if s.DB == nil || !pseudonymPattern.MatchString(pseudonym) {
		return 0, 0, fmt.Errorf("invalid action pseudonym")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT lineage_hash FROM measurement_action WHERE action_pseudonym=? FOR UPDATE`, pseudonym)
	if err != nil {
		return 0, 0, err
	}
	var lineages [][]byte
	for rows.Next() {
		var lineage []byte
		if err := rows.Scan(&lineage); err != nil {
			rows.Close()
			return 0, 0, err
		}
		lineages = append(lineages, append([]byte(nil), lineage...))
	}
	if err := rows.Close(); err != nil {
		return 0, 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM measurement_action WHERE action_pseudonym=?`, pseudonym)
	if err != nil {
		return 0, 0, err
	}
	if actions, err = result.RowsAffected(); err != nil {
		return 0, 0, err
	}
	for _, lineage := range lineages {
		result, err = tx.ExecContext(ctx, `DELETE t FROM measurement_touch t WHERE t.lineage_hash=? AND NOT EXISTS (SELECT 1 FROM measurement_action a WHERE a.lineage_hash=t.lineage_hash)`, lineage)
		if err != nil {
			return 0, 0, err
		}
		count, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return 0, 0, rowsErr
		}
		touches += count
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return actions, touches, nil
}

// ExportPseudonym writes a bounded, operator-scoped CSV export. It excludes
// token hashes, auction IDs, and internal delivery/accounting identities.
func (s Service) ExportPseudonym(ctx context.Context, pseudonym string, output io.Writer) error {
	if s.DB == nil || output == nil || !pseudonymPattern.MatchString(pseudonym) {
		return fmt.Errorf("invalid action export configuration")
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT action_id, adv_id, campaign_id, item_id, creative_id, event_id, event_type,
COALESCE(action_name,''), occurred_at, COALESCE(CAST(value_usd AS CHAR),''),
currency, attribution_type, COALESCE(DATE_FORMAT(touch_at,'%Y-%m-%dT%H:%i:%s.%fZ'),''), late,
privacy_mode, privacy_reason, action_pseudonym
FROM measurement_action WHERE action_pseudonym=? ORDER BY action_id`, pseudonym)
	if err != nil {
		return err
	}
	defer rows.Close()
	writer := csv.NewWriter(output)
	if err := writer.Write([]string{"action_id", "adv_id", "campaign_id", "item_id", "creative_id", "event_id", "event_type", "action_name", "occurred_at", "value_usd", "currency", "attribution_type", "touch_at", "late", "privacy_mode", "privacy_reason", "action_pseudonym"}); err != nil {
		return err
	}
	for rows.Next() {
		var actionID, advID, campaignID, itemID, creativeID uint64
		var eventID, eventType, actionName, value, currency, attribution, touchAt, privacyMode, privacyReason, resultPseudonym string
		var occurred time.Time
		var late bool
		if err := rows.Scan(&actionID, &advID, &campaignID, &itemID, &creativeID, &eventID, &eventType, &actionName, &occurred, &value, &currency, &attribution, &touchAt, &late, &privacyMode, &privacyReason, &resultPseudonym); err != nil {
			return err
		}
		record := []string{strconv.FormatUint(actionID, 10), strconv.FormatUint(advID, 10), strconv.FormatUint(campaignID, 10), strconv.FormatUint(itemID, 10), strconv.FormatUint(creativeID, 10), eventID, eventType, actionName, occurred.UTC().Format(time.RFC3339Nano), value, currency, attribution, touchAt, strconv.FormatBool(late), privacyMode, privacyReason, resultPseudonym}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

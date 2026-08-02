package action

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestReconcileUsesClickPrecedenceAndOnlyUpdatesUnattributed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)
	lineage := bytes.Repeat([]byte{1}, 32)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT action_id, lineage_hash, occurred_at\nFROM measurement_action")).
		WithArgs(100).WillReturnRows(sqlmock.NewRows([]string{"action_id", "lineage_hash", "occurred_at"}).AddRow(7, lineage, now))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT touch_type, occurred_at\nFROM measurement_touch")).
		WithArgs(lineage, now, now.Add(-30*24*time.Hour), now.Add(-7*24*time.Hour)).
		WillReturnRows(sqlmock.NewRows([]string{"touch_type", "occurred_at"}).AddRow("click", now.Add(-time.Minute)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE measurement_action\nSET attribution_type=?, touch_at=?\nWHERE action_id=? AND attribution_type='unattributed'")).
		WithArgs("click", now.Add(-time.Minute), uint64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	updated, err := (Service{DB: db}).Reconcile(context.Background(), 30*24*time.Hour, 7*24*time.Hour, 100)
	if err != nil || updated != 1 {
		t.Fatalf("updated=%d err=%v", updated, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPruneIsBoundedAndTransactional(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM measurement_action WHERE expires_at<=UTC_TIMESTAMP(6) ORDER BY action_id LIMIT ?")).WithArgs(10).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM measurement_touch WHERE expires_at<=UTC_TIMESTAMP(6) ORDER BY touch_id LIMIT ?")).WithArgs(10).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	actions, touches, err := (Service{DB: db}).Prune(context.Background(), 10)
	if err != nil || actions != 3 || touches != 2 {
		t.Fatalf("actions=%d touches=%d err=%v", actions, touches, err)
	}
}

func TestExportPseudonymExcludesTokenAndAuctionIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pseudonym := strings.Repeat("a", 64)
	now := time.Now().UTC().Truncate(time.Microsecond)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT action_id, adv_id, campaign_id, item_id, creative_id, event_id, event_type,")).
		WithArgs(pseudonym).WillReturnRows(sqlmock.NewRows([]string{"action_id", "adv_id", "campaign_id", "item_id", "creative_id", "event_id", "event_type", "action_name", "occurred_at", "value", "currency", "attribution_type", "touch_at", "late", "privacy_mode", "privacy_reason", "action_pseudonym"}).
		AddRow(1, 2, 3, 4, 5, "order:1", "purchase", "", now, "12.50", "USD", "click", now.Format("2006-01-02T15:04:05.000000Z"), false, "contextual", "signed_lineage_contextual", pseudonym))
	var output bytes.Buffer
	if err := (Service{DB: db}).ExportPseudonym(context.Background(), pseudonym, &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "order:1") || strings.Contains(text, "token_hash") || strings.Contains(text, "auction_id") {
		t.Fatalf("unexpected export: %s", text)
	}
}

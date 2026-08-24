package accounting

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// TestServiceIntegration intentionally requires a disposable database named
// aofei_a01_test. Accounting rows are immutable, so this test refuses to run
// against any other schema instead of attempting destructive cleanup.
func TestServiceIntegration(t *testing.T) {
	dsn := os.Getenv("AOFEI_ACCOUNTING_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("AOFEI_ACCOUNTING_INTEGRATION_DSN is not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	var schema string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	if schema != "aofei_a01_test" {
		t.Fatalf("refusing accounting integration test against database %q", schema)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO adv (adv_id,email,passwd,active,created) VALUES (1,'a01-adv@example.invalid','fixture','Yes',UTC_TIMESTAMP())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pub (pub_id,email,passwd,active,created) VALUES (2,'a01-pub@example.invalid','fixture','Yes',UTC_TIMESTAMP())`); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := db.ExecContext(ctx, `
INSERT INTO daily_log (daily, spend, imps, clis, created)
VALUES ('2099-01-01',1.250000,1,0,UTC_TIMESTAMP())`)
	if err != nil {
		t.Fatal(err)
	}
	logID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO daily_adv (log_id,creative_id,item_id,campaign_id,adv_id,spend,imps,clis)
VALUES (?,1,1,1,1,1.250000,1,0)`, logID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO daily_pub (log_id,slot_id,site_id,pub_id,spend,imps,clis)
VALUES (?,1,1,2,1.250000,1,0)`, logID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO daily_mid (
  log_id,bidder_id,group_id,route_bidder_id,target_id,
  adv_id,campaign_id,item_id,creative_id,pub_id,site_id,slot_id,
  imps,charge_spend,pay_spend,margin_spend
) VALUES (?,1,1,1,1,1,1,1,1,2,1,1,1,0.001250,0.001000,0.000250)`, logID); err != nil {
		t.Fatal(err)
	}

	service := Service{DB: db}
	input := CreateInput{
		RequestKey: "a01-integration-adv", PartyType: PartyAdvertiser, PartyID: 1,
		Cadence: CadenceDaily, PeriodStart: day, PeriodEnd: day,
		Actor: "unix-uid:1001", Reason: "integration close",
	}
	statementID, err := service.CreateStatement(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	retryID, err := service.CreateStatement(ctx, input)
	if err != nil || retryID != statementID {
		t.Fatalf("idempotent retry = %d, %v", retryID, err)
	}
	if err := service.AddAdjustment(ctx, AdjustmentInput{
		StatementID: statementID, Amount: Money(250000), Actor: "unix-uid:1001", Reason: "integration adjustment",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Transition(ctx, TransitionInput{
		StatementID: statementID, To: StatusConfirmed, Actor: "unix-uid:1002", Reason: "integration approval",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Transition(ctx, TransitionInput{
		StatementID: statementID, To: StatusSettled, Actor: "unix-uid:1003", Reason: "integration settlement",
		ExternalRef: "invoice:a01-integration",
	}); err != nil {
		t.Fatal(err)
	}
	if discrepancy, err := service.ReconcileStatement(ctx, statementID); err != nil || discrepancy.Difference != 0 {
		t.Fatalf("reconcile = %#v, %v", discrepancy, err)
	}
	if reconciliation, err := service.ReconcileMiddleman(ctx, day, day); err != nil ||
		reconciliation.Charge != Money(1250) || reconciliation.Pay != Money(1000) ||
		reconciliation.Margin != Money(250) || reconciliation.Difference != 0 {
		t.Fatalf("middleman reconcile = %#v, %v", reconciliation, err)
	}
	replacementID, err := service.Correct(ctx, CorrectionInput{
		StatementID: statementID, RequestKey: "a01-integration-adv-correction",
		Actor: "unix-uid:1001", Reason: "integration correction",
	})
	if err != nil || replacementID == statementID {
		t.Fatalf("correction = %d, %v", replacementID, err)
	}
	retryReplacementID, err := service.Correct(ctx, CorrectionInput{
		StatementID: statementID, RequestKey: "a01-integration-adv-correction",
		Actor: "unix-uid:1001", Reason: "integration correction retry",
	})
	if err != nil || retryReplacementID != replacementID {
		t.Fatalf("idempotent correction retry = %d, %v", retryReplacementID, err)
	}

	publisherID, err := service.CreateStatement(ctx, CreateInput{
		RequestKey: "a01-integration-pub", PartyType: PartyPublisher, PartyID: 2,
		Cadence: CadenceDaily, PeriodStart: day, PeriodEnd: day,
		Actor: "unix-uid:1001", Reason: "integration publisher close",
	})
	if err != nil {
		t.Fatal(err)
	}
	statements, err := service.ListStatements(ctx, PartyStatementScope(PartyPublisher, 2))
	if err != nil || len(statements) != 1 || statements[0].ID != publisherID || statements[0].SourceAmount != Money(1250000) {
		t.Fatalf("publisher statements = %#v, %v", statements, err)
	}

	if _, err := db.ExecContext(ctx, `UPDATE acct_adjustment SET amount=2 WHERE statement_id=?`, statementID); err == nil {
		t.Fatal("immutable adjustment update succeeded")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM acct_audit WHERE statement_id=?`, statementID); err == nil {
		t.Fatal("immutable audit delete succeeded")
	}
	if _, err := db.ExecContext(ctx, `UPDATE acct_statement SET party_id=99 WHERE statement_id=?`, statementID); err == nil {
		t.Fatal("protected statement party update succeeded")
	}
	if _, err := db.ExecContext(ctx, `UPDATE acct_statement SET source_amount=99,total_amount=99 WHERE statement_id=?`, replacementID); err == nil {
		t.Fatal("protected statement amount update succeeded")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM acct_statement WHERE statement_id=?`, replacementID); err == nil {
		t.Fatal("immutable statement delete succeeded")
	}
	var sensitive int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema=DATABASE() AND table_name IN ('pay_cc','pay_cheque')
  AND column_name IN ('cardnumber','routing_number','account_number','expire','bank','account')`).Scan(&sensitive); err != nil {
		t.Fatal(err)
	}
	if sensitive != 0 {
		t.Fatalf("found %d sensitive compatibility columns", sensitive)
	}
	if _, err := service.ReconcileStatement(ctx, 0); err == nil || errors.Is(err, ErrSourceDiscrepancy) {
		t.Fatalf("invalid reconciliation error = %v", err)
	}
}

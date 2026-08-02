package accounting

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateStatementSnapshotsSourceAndAuditAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM adv WHERE adv_id=?")).WithArgs(uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectQuery("SELECT CAST\\(COALESCE\\(ROUND\\(SUM\\(a.spend\\),6\\),0\\) AS CHAR\\)").
		WithArgs(uint64(7), "2026-07-31", "2026-07-31").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("12.345678"))
	mock.ExpectExec("INSERT INTO acct_statement").
		WithArgs("invoice-20260731-7", PartyAdvertiser, uint64(7), CadenceDaily, "2026-07-31", "2026-07-31", "12.345678", "12.345678", "admin:1").
		WillReturnResult(sqlmock.NewResult(44, 1))
	mock.ExpectQuery("SELECT party_type, party_id, cadence").WithArgs(uint64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"party", "party_id", "cadence", "start", "end", "created_by"}).
			AddRow(PartyAdvertiser, 7, CadenceDaily, day, day, "admin:1"))
	mock.ExpectExec("INSERT INTO acct_audit").
		WithArgs(uint64(44), "admin:1", "period close", uint64(44)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	id, err := (Service{DB: db}).CreateStatement(context.Background(), CreateInput{
		RequestKey: "invoice-20260731-7", PartyType: PartyAdvertiser, PartyID: 7,
		Cadence: CadenceDaily, PeriodStart: day, PeriodEnd: day, Actor: "admin:1", Reason: "period close",
	})
	if err != nil || id != 44 {
		t.Fatalf("CreateStatement = %d, %v", id, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateStatementIdempotentRetryDoesNotDuplicateAudit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM adv WHERE adv_id=?")).WithArgs(uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	mock.ExpectQuery("SELECT CAST\\(COALESCE\\(ROUND\\(SUM\\(a.spend\\),6\\),0\\) AS CHAR\\)").
		WithArgs(uint64(7), "2026-07-31", "2026-07-31").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("12.500000"))
	mock.ExpectExec("INSERT INTO acct_statement").
		WithArgs("invoice-20260731-7", PartyAdvertiser, uint64(7), CadenceDaily, "2026-07-31", "2026-07-31", "12.500000", "12.500000", "unix-uid:1001").
		WillReturnResult(sqlmock.NewResult(44, 0))
	mock.ExpectQuery("SELECT party_type, party_id, cadence").WithArgs(uint64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"party", "party_id", "cadence", "start", "end", "created_by"}).
			AddRow(PartyAdvertiser, 7, CadenceDaily, day, day, "unix-uid:1001"))
	mock.ExpectExec("INSERT INTO acct_audit").
		WithArgs(uint64(44), "unix-uid:1001", "retry", uint64(44)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	id, err := (Service{DB: db}).CreateStatement(context.Background(), CreateInput{
		RequestKey: "invoice-20260731-7", PartyType: PartyAdvertiser, PartyID: 7,
		Cadence: CadenceDaily, PeriodStart: day, PeriodEnd: day, Actor: "unix-uid:1001", Reason: "retry",
	})
	if err != nil || id != 44 {
		t.Fatalf("CreateStatement retry = %d, %v", id, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionRequiresEvidenceAndEnforcesStateMachine(t *testing.T) {
	service := Service{}
	if err := service.Transition(context.Background(), TransitionInput{StatementID: 1, To: StatusSettled, Actor: "admin:1", Reason: "paid"}); err == nil {
		t.Fatal("settlement without evidence succeeded")
	}
	if err := service.Transition(context.Background(), TransitionInput{StatementID: 1, To: StatusSettled, Actor: "admin:1", Reason: "paid", ExternalRef: "payout:1234567890123456"}); err == nil {
		t.Fatal("card-like reference succeeded")
	}
	if transitionAllowed(StatusSettled, StatusDraft) || !transitionAllowed(StatusConfirmed, StatusSettled) {
		t.Fatal("invalid statement transition table")
	}
}

func TestTransitionEnforcesMakerCheckerSeparation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, created_by, confirmed_by").WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "created_by", "confirmed_by", "party", "party_id", "start", "end", "source"}).
			AddRow(StatusConfirmed, "operator:maker", "operator:approver", PartyPublisher, 2,
				time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), "1.000000"))
	mock.ExpectRollback()
	err = (Service{DB: db}).Transition(context.Background(), TransitionInput{
		StatementID: 1, To: StatusSettled, Actor: "operator:approver",
		Reason: "record settlement", ExternalRef: "payout:ticket-abc",
	})
	if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "approver") {
		t.Fatalf("maker-checker transition error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmationRejectsChangedSourceSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, created_by, confirmed_by").WithArgs(uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "created_by", "confirmed_by", "party", "party_id", "start", "end", "source"}).
			AddRow(StatusDraft, "unix-uid:1001", nil, PartyAdvertiser, 7, start, end, "5.000000"))
	mock.ExpectQuery("SELECT CAST\\(COALESCE\\(ROUND\\(SUM\\(a.spend\\),6\\),0\\) AS CHAR\\)").
		WithArgs(uint64(7), "2026-07-01", "2026-07-31").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("5.250000"))
	mock.ExpectRollback()
	err = (Service{DB: db}).Transition(context.Background(), TransitionInput{
		StatementID: 2, To: StatusConfirmed, Actor: "unix-uid:1002",
		Reason: "review complete",
	})
	if !errors.Is(err, ErrSourceDiscrepancy) {
		t.Fatalf("confirmation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCorrectRetryReturnsExistingReplacement(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	day := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT party_type, party_id, cadence").WithArgs(uint64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"party", "party_id", "cadence", "start", "end", "status"}).
			AddRow(PartyAdvertiser, 7, CadenceDaily, day, day, StatusCorrected))
	mock.ExpectQuery("SELECT statement_id, created_by FROM acct_statement").
		WithArgs("correction-44", uint64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"statement_id", "created_by"}).AddRow(45, "unix-uid:1001"))
	mock.ExpectCommit()
	id, err := (Service{DB: db}).Correct(context.Background(), CorrectionInput{
		StatementID: 44, RequestKey: "correction-44", Actor: "unix-uid:1001", Reason: "retry",
	})
	if err != nil || id != 45 {
		t.Fatalf("Correct retry = %d, %v", id, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateStatementRejectsMissingDates(t *testing.T) {
	_, err := (Service{}).CreateStatement(context.Background(), CreateInput{
		RequestKey: "missing-dates", PartyType: PartyAdvertiser, PartyID: 7,
		Cadence: CadenceDaily, Actor: "unix-uid:1001", Reason: "invalid close",
	})
	if err == nil || !strings.Contains(err.Error(), "dates") {
		t.Fatalf("CreateStatement missing dates error = %v", err)
	}
}

func TestAdjustmentIsImmutableAndCannotMakeNegativeTotal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, CAST\\(source_amount AS CHAR\\)").WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "source", "adjustment"}).AddRow(StatusDraft, "10.000000", "0.000000"))
	mock.ExpectRollback()
	err = (Service{DB: db}).AddAdjustment(context.Background(), AdjustmentInput{
		StatementID: 9, Amount: Money(-11 * moneyScale), Actor: "admin:1", Reason: "unsupported correction",
	})
	if err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("negative adjustment error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReconcileReportsChangedSourceWithoutMutatingStatement(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT party_type, party_id, period_start").WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"party", "party_id", "start", "end", "source"}).
			AddRow(PartyPublisher, 8, start, end, "5.000000"))
	mock.ExpectQuery("SELECT CAST\\(COALESCE\\(ROUND\\(SUM\\(p.spend\\),6\\),0\\) AS CHAR\\)").
		WithArgs(uint64(8), "2026-07-01", "2026-07-31").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow("5.250000"))
	result, err := (Service{DB: db}).ReconcileStatement(context.Background(), 3)
	if !errors.Is(err, ErrSourceDiscrepancy) || result.Difference != Money(250000) {
		t.Fatalf("reconcile = %#v, %v", result, err)
	}
}

func TestReconcileMiddlemanReportsChargePayMarginDifference(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(m.imps\\),0\\)").
		WithArgs("2026-07-01", "2026-07-31").
		WillReturnRows(sqlmock.NewRows([]string{"imps", "charge", "pay", "margin"}).
			AddRow(1000, "2.500000", "2.000000", "0.499999"))
	result, err := (Service{DB: db}).ReconcileMiddleman(context.Background(), start, end)
	if !errors.Is(err, ErrSourceDiscrepancy) || result.ExpectedMargin != Money(500000) || result.Difference != Money(-1) {
		t.Fatalf("middleman reconcile = %#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteCSVUsesStableMoneyAndNoSensitiveFields(t *testing.T) {
	var out bytes.Buffer
	err := WriteCSV(&out, []Statement{{
		ID: 1, RequestKey: "invoice-1", PartyType: PartyAdvertiser, PartyID: 2,
		Cadence: CadenceDaily, PeriodStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Currency: "USD",
		SourceAmount: Money(1250), TotalAmount: Money(1250), Status: StatusDraft,
		ExternalRef: sql.NullString{String: "invoice:ticket-1", Valid: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "0.001250") || strings.Contains(strings.ToLower(out.String()), "account_number") {
		t.Fatalf("CSV = %q", out.String())
	}
}

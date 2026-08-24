package midcallback

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

const safeRetryOrigin = "http://8.8.8.8"

type safeRetryRoundTripper struct {
	handler http.Handler
}

func (t safeRetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, req)
	response := recorder.Result()
	response.Request = req
	return response, nil
}

func (safeRetryRoundTripper) SafeHTTPNonNetworkTransport() {}

type failingSafeRetryRoundTripper struct {
	err error
}

func (t failingSafeRetryRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

func (failingSafeRetryRoundTripper) SafeHTTPNonNetworkTransport() {}

func TestRetryableForward(t *testing.T) {
	tests := []struct {
		status string
		code   int
		want   bool
	}{
		{"error", 0, true},
		{"request_error", 0, true},
		{"http_error", http.StatusTooManyRequests, true},
		{"http_error", http.StatusInternalServerError, true},
		{"http_error", http.StatusBadRequest, false},
		{"missing", 0, false},
		{"invalid_url", 0, false},
		{"duplicate", 0, false},
		{"ok", http.StatusNoContent, false},
	}
	for _, tt := range tests {
		if got := RetryableForward(tt.status, tt.code); got != tt.want {
			t.Fatalf("RetryableForward(%q, %d) = %v, want %v", tt.status, tt.code, got, tt.want)
		}
	}
}

func TestForwardRejectsPrivateTargetBeforeInjectedClient(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: safeRetryRoundTripper{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	})}}
	status, _, _, forwarded := forward(context.Background(), "http://127.0.0.1/callback", Options{
		Timeout: time.Second,
		Client:  client,
	})
	if status != "invalid_url" {
		t.Fatalf("status = %q, want invalid_url", status)
	}
	if forwarded {
		t.Fatal("rejected callback was reported as forwarded")
	}
	if calls != 0 {
		t.Fatalf("injected client calls = %d, want zero", calls)
	}
}

func TestStoredForwardEvidenceUsesClosedVocabulary(t *testing.T) {
	tests := map[string]string{
		"ok":                              "",
		"invalid_url":                     "callback URL rejected",
		"request_error":                   "callback request rejected",
		"error":                           "callback request failed",
		"http_error":                      "callback HTTP rejected",
		"https://secret.example/callback": "callback outcome unavailable",
	}
	for outcome, want := range tests {
		if got := storedForwardEvidence(outcome); got != want {
			t.Fatalf("storedForwardEvidence(%q) = %q, want %q", outcome, got, want)
		}
	}
}

func TestRunPersistsFixedForwardEvidence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry.*FOR UPDATE`).
		WithArgs(5, now, now.Add(-10*time.Minute), 10).
		WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
			AddRow(uint64(75), "secret-token", "win", safeRetryOrigin+"/credential", 0))
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Processing', claimed_at=\?, updated=NOW\(\)\s+WHERE retry_id IN \(\?\)`).
		WithArgs(now, uint64(75)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Retrying'.*last_error=\?.*WHERE retry_id=\? AND status='Processing'`).
		WithArgs(1, now.Add(time.Minute), nil, "callback request failed", uint64(75)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := Run(context.Background(), db, Options{
		Limit:       10,
		MaxAttempts: 5,
		Timeout:     time.Second,
		Now:         func() time.Time { return now },
		Client: &http.Client{Transport: failingSafeRetryRoundTripper{
			err: errors.New("dial secret.example at 203.0.113.75 with credential"),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected != 1 || result.Forwarded != 1 || result.Retrying != 1 {
		t.Fatalf("result = %#v, want one forwarded retry", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBackoff(t *testing.T) {
	base := time.Minute
	max := 5 * time.Minute
	if got := backoff(1, base, max); got != time.Minute {
		t.Fatalf("attempt 1 backoff = %s", got)
	}
	if got := backoff(3, base, max); got != 4*time.Minute {
		t.Fatalf("attempt 3 backoff = %s", got)
	}
	if got := backoff(10, base, max); got != max {
		t.Fatalf("capped backoff = %s", got)
	}
}

func TestRunClaimsDueRowsBeforeForwarding(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry.*FOR UPDATE`).
		WithArgs(5, sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
			AddRow(uint64(7), "tok", "win", safeRetryOrigin+"/win", 0))
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Processing', claimed_at=\?, updated=NOW\(\)\s+WHERE retry_id IN \(\?\)`).
		WithArgs(sqlmock.AnyArg(), uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Succeeded', attempts=\?, claimed_at=NULL, last_http_status=\?, last_error=NULL, updated=NOW\(\)\s+WHERE retry_id=\? AND status='Processing'`).
		WithArgs(1, http.StatusNoContent, uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := Run(context.Background(), db, Options{
		Limit:       10,
		MaxAttempts: 5,
		Timeout:     time.Second,
		Client:      &http.Client{Transport: safeRetryRoundTripper{handler: downstream.Config.Handler}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected != 1 || result.Succeeded != 1 {
		t.Fatalf("result = %#v, want one selected and succeeded", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunReportsPostForwardStateFailureWithoutIdentifiers(t *testing.T) {
	tests := []struct {
		name   string
		result driver.Result
		err    error
	}{
		{name: "database error", err: errors.New("database unavailable")},
		{name: "lost claim", result: sqlmock.NewResult(0, 0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			const token = "sensitive-token"

			mock.ExpectBegin()
			mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry.*FOR UPDATE`).
				WithArgs(5, sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
				WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
					AddRow(uint64(71), token, "win", safeRetryOrigin+"/credential", 0))
			mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Processing', claimed_at=\?, updated=NOW\(\)\s+WHERE retry_id IN \(\?\)`).
				WithArgs(sqlmock.AnyArg(), uint64(71)).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			expect := mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Succeeded', attempts=\?, claimed_at=NULL, last_http_status=\?, last_error=NULL, updated=NOW\(\)\s+WHERE retry_id=\? AND status='Processing'`).
				WithArgs(1, http.StatusNoContent, uint64(71))
			if test.err != nil {
				expect.WillReturnError(test.err)
			} else {
				expect.WillReturnResult(test.result)
			}

			result, err := Run(context.Background(), db, Options{
				Limit: 10, MaxAttempts: 5, Timeout: time.Second,
				Client: &http.Client{Transport: safeRetryRoundTripper{handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				})}},
			})
			if result.Selected != 1 || result.Forwarded != 1 || result.Succeeded != 0 || result.StateErrors != 1 {
				t.Fatalf("result = %#v, want one forwarded row and one state error", result)
			}
			var stateErr *PostForwardStateError
			if !errors.As(err, &stateErr) {
				t.Fatalf("error = %v, want PostForwardStateError", err)
			}
			if stateErr.ForwardOutcome != "ok" || stateErr.TargetState != StatusSucceeded || !stateErr.Forwarded {
				t.Fatalf("state error = %#v", stateErr)
			}
			for _, secret := range []string{token, safeRetryOrigin, "credential", "71"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("identifier %q leaked in error %q", secret, err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunRejectsPartialClaimBeforeForwarding(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry.*FOR UPDATE`).
		WithArgs(5, sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
			AddRow(uint64(72), "secret", "win", "https://callback.example/secret", 0))
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Processing', claimed_at=\?, updated=NOW\(\)\s+WHERE retry_id IN \(\?\)`).
		WithArgs(sqlmock.AnyArg(), uint64(72)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	result, err := Run(context.Background(), db, Options{Limit: 10, MaxAttempts: 5})
	if err == nil || !strings.Contains(err.Error(), "callback claim affected 0 rows, want 1") {
		t.Fatalf("error = %v", err)
	}
	if result != (Result{}) {
		t.Fatalf("result = %#v, want empty pre-forward result", result)
	}
	for _, secret := range []string{"secret", "callback.example", "72"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("identifier %q leaked in error %q", secret, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunDistinguishesStateFailureBeforeForward(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry.*FOR UPDATE`).
		WithArgs(5, sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
			AddRow(uint64(73), "secret", "win", "http://127.0.0.1/secret", 0))
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Processing', claimed_at=\?, updated=NOW\(\)\s+WHERE retry_id IN \(\?\)`).
		WithArgs(sqlmock.AnyArg(), uint64(73)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Abandoned'.*WHERE retry_id=\? AND status='Processing'`).
		WithArgs(1, nil, sqlmock.AnyArg(), uint64(73)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	result, err := Run(context.Background(), db, Options{Limit: 10, MaxAttempts: 5})
	if result.Selected != 1 || result.Forwarded != 0 || result.StateErrors != 1 {
		t.Fatalf("result = %#v, want local rejection and state error", result)
	}
	var stateErr *PostForwardStateError
	if !errors.As(err, &stateErr) || stateErr.Forwarded || stateErr.ForwardOutcome != "invalid_url" {
		t.Fatalf("state error = %#v, err = %v", stateErr, err)
	}
	for _, secret := range []string{"secret", "127.0.0.1", "73"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("identifier %q leaked in error %q", secret, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunDryRunDoesNotClaimRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry\s+WHERE attempts < \?.*status = 'Processing'.*LIMIT \?`).
		WithArgs(5, sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
			AddRow(uint64(7), "tok", "win", "https://downstream.example/win", 0))

	result, err := Run(context.Background(), db, Options{Limit: 10, MaxAttempts: 5, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected != 1 || result.Succeeded != 0 || result.Retrying != 0 || result.Abandoned != 0 {
		t.Fatalf("dry-run result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBacklogReportsDueAndStaleProcessingRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT\s+SUM\(CASE WHEN status IN \('Pending', 'Retrying'\).*FROM mid_callback_retry\s+WHERE attempts < \?`).
		WithArgs(now, now.Add(-10*time.Minute), 5).
		WillReturnRows(sqlmock.NewRows([]string{"due_rows", "stale_processing_rows"}).AddRow(3, 1))

	stats, err := Backlog(context.Background(), db, Options{
		MaxAttempts: 5,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Due != 3 || stats.StaleProcessing != 1 {
		t.Fatalf("backlog = %#v, want due 3 stale 1", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunReclaimsStaleProcessingRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT retry_id, token, source, callback_url, attempts\s+FROM mid_callback_retry.*status = 'Processing'.*FOR UPDATE`).
		WithArgs(5, now, now.Add(-10*time.Minute), 10).
		WillReturnRows(sqlmock.NewRows([]string{"retry_id", "token", "source", "callback_url", "attempts"}).
			AddRow(uint64(9), "tok", "win", "http://127.0.0.1/win", 4))
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Processing', claimed_at=\?, updated=NOW\(\)\s+WHERE retry_id IN \(\?\)`).
		WithArgs(now, uint64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE mid_callback_retry\s+SET status='Abandoned', attempts=\?, claimed_at=NULL, last_http_status=\?, last_error=\?, updated=NOW\(\)\s+WHERE retry_id=\? AND status='Processing'`).
		WithArgs(5, nil, sqlmock.AnyArg(), uint64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := Run(context.Background(), db, Options{
		Limit:       10,
		MaxAttempts: 5,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected != 1 || result.Abandoned != 1 {
		t.Fatalf("result = %#v, want one stale row abandoned after final failed attempt", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

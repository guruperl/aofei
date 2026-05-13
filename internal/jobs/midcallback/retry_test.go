package midcallback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

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
			AddRow(uint64(7), "tok", "win", downstream.URL+"/win", 0))
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
		Client:      downstream.Client(),
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

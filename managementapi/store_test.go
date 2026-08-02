package managementapi

import (
	"context"
	"crypto/sha256"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestClaimIdempotencyReplaysCompletedResponse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := sha256.Sum256([]byte("key"))
	request := sha256.Sum256([]byte("request"))
	claim := []byte("0123456789abcdef")
	otherClaim := []byte("fedcba9876543210")
	body := []byte(`{"data":{"ok":true}}`)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO api_idempotency")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT request_hash, claim_token, state, response_status, response_body")).
		WithArgs(uint64(9), key[:]).WillReturnRows(sqlmock.NewRows([]string{"request_hash", "claim_token", "state", "response_status", "response_body"}).AddRow(request[:], otherClaim, "Complete", 202, body))
	mock.ExpectRollback()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	claimed, result, err := claimIdempotency(context.Background(), tx, Principal{CredentialID: 9, AdvertiserID: 7}, key[:], request[:], claim, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if claimed || !result.Replay || result.Status != 202 || string(result.Body) != string(body) {
		t.Fatalf("unexpected replay result: claimed=%t result=%#v", claimed, result)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClaimIdempotencyRejectsDifferentRequest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := sha256.Sum256([]byte("key"))
	request := sha256.Sum256([]byte("request"))
	other := sha256.Sum256([]byte("other"))
	claim := []byte("0123456789abcdef")
	otherClaim := []byte("fedcba9876543210")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO api_idempotency")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT request_hash, claim_token, state, response_status, response_body")).
		WithArgs(uint64(9), key[:]).WillReturnRows(sqlmock.NewRows([]string{"request_hash", "claim_token", "state", "response_status", "response_body"}).AddRow(other[:], otherClaim, "Complete", 202, []byte(`{}`)))
	mock.ExpectRollback()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = claimIdempotency(context.Background(), tx, Principal{CredentialID: 9, AdvertiserID: 7}, key[:], request[:], claim, time.Now())
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v", err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOperationReportsDelayedWithoutRewritingStoredState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT operation_id").WithArgs("00112233445566778899aabbccddeeff", uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"operation_id", "resource_type", "resource_id", "accepted_version", "configuration_state", "activation_state", "accepted_at", "activation_deadline", "activated_at", "publication_mode"}).
			AddRow("00112233445566778899aabbccddeeff", "campaign", 10, 2, "Accepted", "Pending", now.Add(-20*time.Minute), now.Add(-5*time.Minute), nil, ""))
	service := &Service{db: db, now: func() time.Time { return now }}
	op, err := service.operation(context.Background(), 7, "00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatal(err)
	}
	if op.ActivationState != "Delayed" {
		t.Fatalf("activation state = %q", op.ActivationState)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

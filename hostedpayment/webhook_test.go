package hostedpayment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/accounting"
)

func webhookService(t *testing.T, db *sql.DB) *Service {
	t.Helper()
	config := Config{
		Enabled: true, Provider: ProviderStripe, APIBaseURL: "http://127.0.0.1:4242",
		APIKeyEnv: "STRIPE_API_KEY", WebhookSecretEnv: "STRIPE_WEBHOOK_SECRET",
		PublicBaseURL: "http://127.0.0.1:8080", RequestTimeoutMS: 1000,
		MaxBodyBytes: 8192, WebhookToleranceSeconds: 300, MaxAttempts: 1,
		RetryBaseMS: 25, EventRetentionDays: 400, ReconciliationMaxAgeDays: 90,
	}
	return &Service{DB: db, Config: config, WebhookSecret: []byte("whsec_fixture_at_least_16"), now: func() time.Time {
		return time.Unix(1_785_600_000, 0).UTC()
	}}
}

func signedHeader(secret []byte, when int64, payload []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "%d.", when)
	_, _ = mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", when, hex.EncodeToString(mac.Sum(nil)))
}

func TestWebhookSignatureRequiresRawBodyAndFreshTimestamp(t *testing.T) {
	service := webhookService(t, nil)
	payload := []byte(`{"id":"evt_fixture"}`)
	header := signedHeader(service.WebhookSecret, service.now().Unix(), payload)
	if err := service.VerifyWebhookSignature(payload, header, service.now()); err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyWebhookSignature(append(payload, ' '), header, service.now()); err == nil {
		t.Fatal("mutated payload passed signature verification")
	}
	if err := service.VerifyWebhookSignature(payload, header, service.now().Add(10*time.Minute)); err == nil {
		t.Fatal("stale webhook passed signature verification")
	}
}

func TestWebhookSignatureAcceptsBoundedPreviousSecretDuringRotation(t *testing.T) {
	service := webhookService(t, nil)
	previous := []byte("whsec_previous_at_least_16")
	service.WebhookSecrets = [][]byte{service.WebhookSecret, previous}
	payload := []byte(`{"id":"evt_rotation"}`)
	if err := service.VerifyWebhookSignature(payload, signedHeader(previous, service.now().Unix(), payload), service.now()); err != nil {
		t.Fatal(err)
	}
	service.WebhookSecrets = [][]byte{service.WebhookSecret}
	if err := service.VerifyWebhookSignature(payload, signedHeader(previous, service.now().Unix(), payload), service.now()); err == nil {
		t.Fatal("removed previous webhook secret still verified")
	}
}

func TestWebhookHandlerRejectsBadSignatureBeforeDatabase(t *testing.T) {
	service := webhookService(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{"id":"evt_fixture"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=1,v1=00")
	response := httptest.NewRecorder()
	service.WebhookHandler().ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestPrunedEventAgeCannotReenterReplayWindow(t *testing.T) {
	service := webhookService(t, nil)
	service.Config.EventRetentionDays = 30
	created := service.now().AddDate(0, 0, -31).Unix()
	payload := []byte(fmt.Sprintf(`{"id":"evt_pruned","type":"future.object.changed","created":%d,"livemode":false,"data":{"object":{"id":"obj_pruned","object":"future.object"}}}`, created))
	if _, err := service.IngestWebhook(context.Background(), payload); err == nil {
		t.Fatal("event older than its retention window reached the database boundary")
	}
}

func TestWebhookEventVersionMismatchStopsBeforeDatabase(t *testing.T) {
	service := webhookService(t, nil)
	payload := []byte(`{"id":"evt_wrong_version","type":"future.object.changed","created":1785600000,"livemode":false,"api_version":"2023-10-16","data":{"object":{"id":"obj_fixture","object":"future.object"}}}`)
	if _, err := service.IngestWebhook(context.Background(), payload); err == nil {
		t.Fatal("mismatched provider event API version reached the database boundary")
	}
}

func TestUnknownSignedEventIsDurablyIgnoredWithoutRawPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	payload := []byte(`{"id":"evt_unknown","type":"future.object.changed","created":1785600000,"livemode":false,"data":{"object":{"id":"obj_fixture","object":"future.object"}}}`)
	hash := sha256.Sum256(payload)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT operation_id FROM hosted_provider_object").WithArgs("obj_fixture").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO hosted_event").
		WithArgs("evt_unknown", "future.object.changed", "future.object", "obj_fixture", nil, nil,
			time.Unix(1_785_600_000, 0).UTC(), hash[:], "Ignored", "").
		WillReturnResult(sqlmock.NewResult(17, 1))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || result.EventID != 17 || result.Disposition != "Ignored" {
		t.Fatalf("IngestWebhook=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConnectedAccountDirectChargeCannotClaimPlatformOperationMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	payload := []byte(`{"id":"evt_connected_direct","type":"payment_intent.succeeded","created":1785600000,"livemode":false,"account":"acct_publisher_fixture","data":{"object":{"id":"pi_connected_fixture","object":"payment_intent","currency":"usd","amount_received":1000,"metadata":{"aofei_operation_id":"9","aofei_statement_id":"44","aofei_party_id":"7"}}}}`)
	hash := sha256.Sum256(payload)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO hosted_event").
		WithArgs("evt_connected_direct", "payment_intent.succeeded", "payment_intent", "pi_connected_fixture", nil, nil,
			time.Unix(1_785_600_000, 0).UTC(), hash[:], "Ignored", "connected_account_event_out_of_scope").
		WillReturnResult(sqlmock.NewResult(18, 1))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || result.EventID != 18 || result.Disposition != "Ignored" {
		t.Fatalf("IngestWebhook=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConnectedAccountReadinessIgnoresOperationMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	now := service.now()
	payload := []byte(`{"id":"evt_account_binding","type":"account.updated","created":1785600000,"livemode":false,"account":"acct_publisher_fixture","data":{"object":{"id":"acct_publisher_fixture","object":"account","payouts_enabled":true,"details_submitted":true,"capabilities":{"transfers":"active"},"metadata":{"aofei_operation_id":"9","aofei_statement_id":"44","aofei_party_id":"7"}}}}`)
	hash := sha256.Sum256(payload)
	mock.ExpectBegin()
	expectPayoutBindingRow(mock, "acct_publisher_fixture", now)
	mock.ExpectQuery("SELECT MAX\\(provider_created_at\\) FROM hosted_event").WithArgs(uint64(41)).
		WillReturnRows(sqlmock.NewRows([]string{"MAX(provider_created_at)"}).AddRow(nil))
	mock.ExpectExec("INSERT INTO hosted_event").
		WithArgs("evt_account_binding", "account.updated", "account", "acct_publisher_fixture", nil, uint64(41),
			now, hash[:], "Applied", "").
		WillReturnResult(sqlmock.NewResult(23, 1))
	mock.ExpectExec("INSERT IGNORE INTO hosted_provider_object").
		WithArgs("PayoutAccount", "acct_publisher_fixture", uint64(41)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT binding_id,object_kind,operation_id FROM hosted_provider_object").
		WithArgs("acct_publisher_fixture").
		WillReturnRows(sqlmock.NewRows([]string{"binding_id", "object_kind", "operation_id"}).AddRow(41, "PayoutAccount", nil))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || result.EventID != 23 || result.Disposition != "Applied" {
		t.Fatalf("account readiness=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConnectedPayoutFailureIgnoresOperationMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	now := service.now()
	payload := []byte(`{"id":"evt_payout_binding","type":"payout.failed","created":1785600000,"livemode":false,"account":"acct_publisher_fixture","data":{"object":{"id":"po_publisher_fixture","object":"payout","currency":"usd","amount":500,"metadata":{"aofei_operation_id":"9","aofei_statement_id":"44","aofei_party_id":"7"}}}}`)
	hash := sha256.Sum256(payload)
	mock.ExpectBegin()
	expectPayoutBindingRow(mock, "acct_publisher_fixture", now)
	mock.ExpectExec("INSERT INTO hosted_event").
		WithArgs("evt_payout_binding", "payout.failed", "payout", "po_publisher_fixture", nil, uint64(41),
			now, hash[:], "Applied", "").
		WillReturnResult(sqlmock.NewResult(24, 1))
	mock.ExpectExec("INSERT IGNORE INTO hosted_provider_object").
		WithArgs("Payout", "po_publisher_fixture", uint64(41)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT binding_id,object_kind,operation_id FROM hosted_provider_object").
		WithArgs("po_publisher_fixture").
		WillReturnRows(sqlmock.NewRows([]string{"binding_id", "object_kind", "operation_id"}).AddRow(41, "Payout", nil))
	mock.ExpectExec("INSERT INTO hosted_reconciliation").
		WithArgs("event:24:PayoutFailure", nil, uint64(41), uint64(24), "PayoutFailure", "po_publisher_fixture",
			"5.000000", "0.000000", "0.000000", "Unresolved", "provider reported a failed connected-account payout; exact operation requires manual reconciliation").
		WillReturnResult(sqlmock.NewResult(34, 1))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || result.EventID != 24 || result.Disposition != "Applied" {
		t.Fatalf("payout failure=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectPayoutBindingRow(mock sqlmock.Sqlmock, token string, now time.Time) {
	mock.ExpectQuery("FROM hosted_binding").WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "request_key", "provider", "party_type", "party_id", "binding_kind",
			"provider_token", "country", "status", "provider_ready", "version", "created_by",
			"approved_by", "revoked_by", "reason", "created_at", "updated_at",
		}).AddRow(41, "payout-binding", "stripe", "Publisher", 7, "PayoutAccount", token, "US", "Ready", true, 3,
			"pub:7", nil, nil, "test payout binding", now, now))
}

func TestDuplicateWebhookDoesNotRepeatSideEffects(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	payload := []byte(`{"id":"evt_duplicate","type":"future.object.changed","created":1785600000,"livemode":false,"data":{"object":{"id":"obj_fixture","object":"future.object"}}}`)
	hash := sha256.Sum256(payload)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT operation_id FROM hosted_provider_object").WithArgs("obj_fixture").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO hosted_event").WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate provider event"})
	mock.ExpectQuery("SELECT event_id,payload_sha256,disposition FROM hosted_event").WithArgs("evt_duplicate").
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "payload_sha256", "disposition"}).AddRow(8, hash[:], "Ignored"))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || !result.Duplicate || result.EventID != 8 {
		t.Fatalf("duplicate=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAmountMismatchCreatesExceptionWithoutFundingStateChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	payload := []byte(`{"id":"evt_amount","type":"payment_intent.succeeded","created":1785600000,"livemode":false,"data":{"object":{"id":"pi_fixture","object":"payment_intent","status":"succeeded","currency":"usd","amount_received":999,"metadata":{"aofei_operation_id":"9","aofei_statement_id":"44","aofei_party_id":"7"}}}}`)
	hash := sha256.Sum256(payload)
	now := service.now()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT operation_id FROM hosted_provider_object").WithArgs("pi_fixture").WillReturnError(sql.ErrNoRows)
	expectOperationRow(mock, 9, OperationFunding, OperationSubmitted, now)
	mock.ExpectExec("INSERT INTO hosted_event").
		WithArgs("evt_amount", "payment_intent.succeeded", "payment_intent", "pi_fixture", uint64(9), nil,
			now, hash[:], "Unresolved", "currency_or_amount_mismatch").
		WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectExec("INSERT INTO hosted_reconciliation").
		WithArgs("event:21:ProviderMismatch", uint64(9), nil, uint64(21), "ProviderMismatch", "pi_fixture",
			"0.000000", "0.000000", "0.000000", "Unresolved", "provider event amount or currency differs from the approved operation").
		WillReturnResult(sqlmock.NewResult(31, 1))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || result.Disposition != "Unresolved" {
		t.Fatalf("amount mismatch=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFastWebhookRequiresExactStatementAndPartyMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := webhookService(t, db)
	payload := []byte(`{"id":"evt_owner","type":"payment_intent.succeeded","created":1785600000,"livemode":false,"data":{"object":{"id":"pi_owner","object":"payment_intent","status":"succeeded","currency":"usd","amount_received":1000,"metadata":{"aofei_operation_id":"9","aofei_statement_id":"45","aofei_party_id":"7"}}}}`)
	hash := sha256.Sum256(payload)
	now := service.now()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT operation_id FROM hosted_provider_object").WithArgs("pi_owner").WillReturnError(sql.ErrNoRows)
	expectOperationRow(mock, 9, OperationFunding, OperationSubmitted, now)
	mock.ExpectExec("INSERT INTO hosted_event").
		WithArgs("evt_owner", "payment_intent.succeeded", "payment_intent", "pi_owner", uint64(9), nil,
			now, hash[:], "Unresolved", "provider_metadata_mismatch").
		WillReturnResult(sqlmock.NewResult(22, 1))
	mock.ExpectExec("INSERT INTO hosted_reconciliation").
		WithArgs("event:22:ProviderMismatch", uint64(9), nil, uint64(22), "ProviderMismatch", "pi_owner",
			"0.000000", "0.000000", "0.000000", "Unresolved", "provider object metadata does not match the exact Aofei statement and party").
		WillReturnResult(sqlmock.NewResult(32, 1))
	mock.ExpectCommit()
	result, err := service.IngestWebhook(context.Background(), payload)
	if err != nil || result.Disposition != "Unresolved" {
		t.Fatalf("metadata mismatch=%#v, %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReorderedProviderEventsDoNotRegressState(t *testing.T) {
	operation := Operation{Status: OperationSucceeded, ProviderEventCreatedAt: sql.NullTime{Time: time.Unix(200, 0), Valid: true}}
	if eventShouldApply(operation, "payment_intent.payment_failed", OperationFailed, time.Unix(199, 0)) {
		t.Fatal("older failed event regressed succeeded operation")
	}
	if eventShouldApply(operation, "payment_intent.payment_failed", OperationFailed, time.Unix(200, 0)) {
		t.Fatal("equal-time weaker event regressed succeeded operation")
	}
	if !eventShouldApply(operation, "charge.refunded", OperationRefunded, time.Unix(200, 0)) {
		t.Fatal("equal-time stronger refund event was ignored")
	}
	if !eventShouldApply(operation, "transfer.reversed", OperationDisputed, time.Unix(201, 0)) {
		t.Fatal("later transfer reversal was ignored")
	}
	if eventShouldApply(operation, "charge.failed", OperationFailed, time.Unix(201, 0)) {
		t.Fatal("generic later failure regressed a succeeded operation")
	}
	if !eventMatchesOperation("checkout.session.async_payment_succeeded", OperationFunding) ||
		!needsAmountMatch("checkout.session.async_payment_succeeded", OperationSucceeded) {
		t.Fatal("asynchronous Checkout success is not an amount-checked funding event")
	}
	if !eventMatchesOperation("charge.refund.updated", OperationRefund) {
		t.Fatal("charge.refund.updated is not associated with the refund operation")
	}
	refunded := Operation{Status: OperationRefunded, ProviderEventCreatedAt: sql.NullTime{Time: time.Unix(200, 0), Valid: true}}
	if eventShouldApply(refunded, "charge.dispute.closed", OperationSucceeded, time.Unix(201, 0)) {
		t.Fatal("a later dispute event erased terminal refund state")
	}
}

func TestProviderCentConversionUsesExplicitOverflowSentinel(t *testing.T) {
	if got := centsMoney(math.MaxInt64); got != accounting.Money(math.MinInt64) {
		t.Fatalf("overflow cents=%s", got.String())
	}
}

func TestStripeDisputeUsesDocumentedOpaqueTokenPrefix(t *testing.T) {
	if got := prefixForStripeObject("dispute"); got != "du_" {
		t.Fatalf("dispute prefix=%q, want du_", got)
	}
	if err := ValidateOpaqueToken("du_fixture", prefixForObjectKind("Dispute")); err != nil {
		t.Fatalf("documented Stripe dispute token was rejected: %v", err)
	}
	if err := ValidateOpaqueToken("dp_fixture", prefixForObjectKind("Dispute")); err == nil {
		t.Fatal("undocumented dispute prefix was accepted")
	}
}

func expectOperationRow(mock sqlmock.Sqlmock, id uint64, kind OperationKind, status OperationStatus, now time.Time) {
	amount, _ := accounting.ParseMoney("10.000000")
	columns := []string{
		"operation_id", "request_key", "provider", "operation_kind", "parent_operation_id",
		"binding_id", "statement_id", "party_type", "party_id", "amount", "currency", "status", "current_object_token",
		"version", "created_by", "approved_by", "executed_by", "reason", "failure_code", "attempt_count",
		"provider_event_created_at", "created_at", "updated_at",
	}
	mock.ExpectQuery(regexp.QuoteMeta(operationSelect + ` WHERE operation_id=? FOR UPDATE`)).WithArgs(id).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(id, "funding-request", "stripe", kind, nil, nil, 44,
			PartyAdvertiser, 7, amount.String(), "USD", status, "cs_fixture", 3, "adv:7", "admin:2",
			"adv:7", "approved funding", nil, 1, nil, now, now))
}

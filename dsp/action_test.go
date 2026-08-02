package dsp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/guruperl/aofei/match"
)

const actionTestSecret = "action-test-secret"

func actionTestWinLoss(now time.Time) *WinLoss {
	return &WinLoss{
		Current:      now,
		AuctionID:    "auction-1",
		AuctionBidID: "bid-1",
		AuctionImpID: "imp-1",
		RAdv: match.RAdv{Demand: match.Demand{
			AdvID: 11, CampaignID: 12, ItemID: 13, CreativeID: 14,
		}},
		RPub: match.RPub{PubID: 21, SiteID: 22, SlotID: 23},
	}
}

func actionTestToken(t *testing.T, now time.Time) string {
	t.Helper()
	token, err := newActionToken(actionTestSecret, actionTestWinLoss(now), 30*24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestActionTokenIsBoundedTamperEvidentAndReservationFree(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	wl := actionTestWinLoss(now)
	wl.DeliveryReservation = "must-never-leave-w8m"
	token, err := newActionToken(actionTestSecret, wl, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := validateActionToken(actionTestSecret, token, now.Add(time.Minute), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if payload.AdvID != 11 || payload.AuctionImpID != "imp-1" {
		t.Fatalf("unexpected lineage: %+v", payload)
	}
	if bytes.Contains([]byte(token), []byte(wl.DeliveryReservation)) {
		t.Fatal("action token exposed the delivery reservation")
	}
	if _, err := validateActionToken(actionTestSecret, token+"0", now, time.Minute); err == nil {
		t.Fatal("tampered token was accepted")
	}
	if _, err := validateActionToken(actionTestSecret, token, now.Add(time.Hour), time.Minute); err == nil {
		t.Fatal("token was accepted at its exclusive expiry boundary")
	}
}

func TestClickRedirectCarriesActionTokenAndPreservesLandingURL(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	wl := actionTestWinLoss(now)
	wl.serverURL = "https://w8m.example"
	wl.WithTrackingSecret(actionTestSecret).withActionTokenTTL(time.Hour)
	trackingURL, err := url.Parse(wl.ClkRedirectURL("https://advertiser.example/buy?sku=42#details"))
	if err != nil {
		t.Fatal(err)
	}
	landing, err := url.Parse(trackingURL.Query().Get("redirect"))
	if err != nil {
		t.Fatal(err)
	}
	if landing.Query().Get("sku") != "42" || landing.Fragment != "details" {
		t.Fatalf("landing URL was not preserved: %s", landing)
	}
	if _, err := validateActionToken(actionTestSecret, landing.Query().Get("w8m_action_token"), now, time.Minute); err != nil {
		t.Fatalf("redirect action token: %v", err)
	}
	second, err := url.Parse(wl.ClkRedirectURL("https://advertiser.example/buy?sku=42#details"))
	if err != nil {
		t.Fatal(err)
	}
	secondLanding, _ := url.Parse(second.Query().Get("redirect"))
	if secondLanding.Query().Get("w8m_action_token") != landing.Query().Get("w8m_action_token") {
		t.Fatal("one served lineage generated multiple action tokens")
	}
}

func TestValidateActionPayloadTaxonomyAndLifecycle(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	tokenText := actionTestToken(t, now.Add(-time.Hour))
	token, err := validateActionToken(actionTestSecret, tokenText, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	base := actionRequest{Version: 1, Token: tokenText, EventID: "order:42", EventType: "purchase", OccurredAt: now.Format(time.RFC3339Nano), ValueUSD: "12.340000"}
	if _, err := validateActionPayload(base, token, now, 90*24*time.Hour, time.Minute, actionTestSecret); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*actionRequest){
		func(r *actionRequest) { r.EventID = "<script>" },
		func(r *actionRequest) { r.ValueUSD = "0.000000" },
		func(r *actionRequest) { r.ValueUSD = "1e6" },
		func(r *actionRequest) { r.EventType, r.ValueUSD, r.ActionName = "custom", "", `shop.</script>` },
		func(r *actionRequest) { r.EventType, r.ValueUSD, r.ActionName = "custom", "", "https://evil.example/x" },
		func(r *actionRequest) { r.OccurredAt = now.Add(time.Minute + time.Second).Format(time.RFC3339Nano) },
	} {
		request := base
		mutate(&request)
		if _, err := validateActionPayload(request, token, now, 90*24*time.Hour, time.Minute, actionTestSecret); err == nil {
			t.Fatalf("invalid action accepted: %+v", request)
		}
	}
}

func signedActionRequest(t *testing.T, token string, request actionRequest, now time.Time) (*http.Request, []byte) {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	httpRequest := httptest.NewRequest(http.MethodPost, "/action", bytes.NewReader(raw))
	httpRequest.Header.Set("Content-Type", "application/json")
	timestamp := now.Unix()
	timestampText := strconv.FormatInt(timestamp, 10)
	httpRequest.Header.Set("X-W8M-Action-Timestamp", timestampText)
	message := append([]byte("w8m-action-request-v1\n"+timestampText+"\n"), raw...)
	httpRequest.Header.Set("X-W8M-Action-Signature", hex.EncodeToString(hmacSHA256([]byte(token), message)))
	return httpRequest, raw
}

func TestServeActionRejectsBeforeStorage(t *testing.T) {
	now := time.Now().UTC()
	token := actionTestToken(t, now)
	request := actionRequest{Version: 1, Token: token, EventID: "event-1", EventType: "conversion", OccurredAt: now.Format(time.RFC3339Nano)}
	r, _ := signedActionRequest(t, token, request, now)
	r.Header.Set("X-W8M-Action-Signature", string(make([]byte, sha256.Size)))
	recorder := httptest.NewRecorder()
	controller := &Controller{C: &Config{TrackingSecret: actionTestSecret}}
	controller.ServeAction(recorder, r)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestStoreActionUsesClickPrecedenceAndIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)
	tokenText := actionTestToken(t, now.Add(-time.Hour))
	token, err := validateActionToken(actionTestSecret, tokenText, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := validateActionPayload(actionRequest{Version: 1, Token: tokenText, EventID: "event-1", EventType: "conversion", OccurredAt: now.Format(time.RFC3339Nano)}, token, now, 90*24*time.Hour, time.Minute, actionTestSecret)
	if err != nil {
		t.Fatal(err)
	}
	query := regexp.QuoteMeta("SELECT touch_type, occurred_at\nFROM measurement_touch")
	insert := regexp.QuoteMeta("INSERT IGNORE INTO measurement_action")
	touchAt := now.Add(-time.Minute)
	mock.ExpectQuery(query).WithArgs(sqlmock.AnyArg(), fact.OccurredAt, fact.OccurredAt.Add(-30*24*time.Hour), fact.OccurredAt.Add(-7*24*time.Hour)).
		WillReturnRows(sqlmock.NewRows([]string{"touch_type", "occurred_at"}).AddRow("click", touchAt))
	mock.ExpectExec(insert).WithArgs(
		11, 12, 13, 14, 21, 22, 23,
		sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "event-1", "conversion", nil,
		fact.OccurredAt, nil, "click", touchAt, false, sqlmock.AnyArg(), 168,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	controller := &Controller{C: &Config{ActionRetentionHours: 168}, DB: db}
	result, err := controller.storeAction(context.Background(), &fact, 30*24*time.Hour, 7*24*time.Hour)
	if err != nil || result != actionStored || fact.Attribution != "click" {
		t.Fatalf("result=%v attribution=%q err=%v", result, fact.Attribution, err)
	}

	mock.ExpectQuery(query).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(insert).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT event_fingerprint FROM measurement_action WHERE adv_id=? AND event_id=?")).
		WithArgs(uint32(11), "event-1").WillReturnRows(sqlmock.NewRows([]string{"event_fingerprint"}).AddRow(fact.Fingerprint[:]))
	result, err = controller.storeAction(context.Background(), &fact, 30*24*time.Hour, 7*24*time.Hour)
	if err != nil || result != actionDuplicate {
		t.Fatalf("duplicate result=%v err=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestServeActionPersistsOnceWithoutDeliveryOrBillingMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	token := actionTestToken(t, now.Add(-time.Minute))
	request := actionRequest{Version: 1, Token: token, EventID: "checkout:9001", EventType: "purchase", OccurredAt: now.Format(time.RFC3339Nano), ValueUSD: "8.250000"}
	r, _ := signedActionRequest(t, token, request, now)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT touch_type, occurred_at\nFROM measurement_touch")).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO measurement_action")).
		WithArgs(11, 12, 13, 14, 21, 22, 23, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "checkout:9001", "purchase", nil, sqlmock.AnyArg(), "8.250000", "unattributed", nil, false, sqlmock.AnyArg(), 168).
		WillReturnResult(sqlmock.NewResult(1, 1))
	controller := &Controller{C: &Config{TrackingSecret: actionTestSecret, ActionRetentionHours: 168}, DB: db}
	recorder := httptest.NewRecorder()
	controller.ServeAction(recorder, r)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreActionRejectsConflictingAdvertiserEventID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	tokenText := actionTestToken(t, now.Add(-time.Minute))
	token, err := validateActionToken(actionTestSecret, tokenText, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := validateActionPayload(actionRequest{Version: 1, Token: tokenText, EventID: "same-id", EventType: "download", OccurredAt: now.Format(time.RFC3339Nano)}, token, now, 90*24*time.Hour, time.Minute, actionTestSecret)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT touch_type, occurred_at\nFROM measurement_touch")).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO measurement_action")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT event_fingerprint FROM measurement_action WHERE adv_id=? AND event_id=?")).
		WithArgs(uint32(11), "same-id").WillReturnRows(sqlmock.NewRows([]string{"event_fingerprint"}).AddRow(bytes.Repeat([]byte{0xff}, sha256.Size)))
	result, err := (&Controller{C: &Config{}, DB: db}).storeAction(context.Background(), &fact, 30*24*time.Hour, 7*24*time.Hour)
	if err != nil || result != actionConflict {
		t.Fatalf("result=%v err=%v", result, err)
	}
}

func TestRecordAttributionTouchIsFailOpenAndBounded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	wl := actionTestWinLoss(time.Now().UTC())
	wl.Status = StatusTrackClk
	mock.ExpectExec(regexp.QuoteMeta("INSERT IGNORE INTO measurement_touch")).
		WithArgs(sqlmock.AnyArg(), "click", sqlmock.AnyArg(), 11, 12, 13, 14, 21, 22, 23, "auction-1", "bid-1", "imp-1", 24).
		WillReturnError(context.DeadlineExceeded)
	(&Controller{C: &Config{ActionRetentionHours: 24}, DB: db}).recordAttributionTouch(wl)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

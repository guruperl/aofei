package managementapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func TestHandlerReturnsStableUnauthenticatedError(t *testing.T) {
	h := newHandler(&Service{config: (Config{Enabled: true}).WithDefaults(900)})
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/advertiser", nil))
	if recorder.Code != http.StatusUnauthorized || recorder.Header().Get("WWW-Authenticate") == "" || recorder.Header().Get("X-Request-ID") == "" {
		t.Fatalf("status/headers = %d %#v", recorder.Code, recorder.Header())
	}
	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "unauthorized" || response.Error.RequestID == "" {
		t.Fatalf("error = %#v", response.Error)
	}
}

func TestDecodeBodyReturnsMoneyStringDeprecation(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/items", strings.NewReader(`{"price_cpm_usd":2.5}`))
	request.Header.Set("Content-Type", "application/json")
	var input itemWrite
	_, err := decodeBody(recorder, request, 1024, &input)
	var clientErr clientError
	if !errors.As(err, &clientErr) || clientErr.code != "money_string_required" {
		t.Fatalf("numeric money decode error = %#v", err)
	}
}

func TestHandlerDerivesAccountScopeFromCredential(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	redisServer := miniredis.RunT(t)
	redisClient, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", redisServer.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer redisClient.Close()
	key := sha256.Sum256([]byte("management-key"))
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	publicID := "00112233445566778899aabbccddeeff"
	token := "w8m_v1_" + publicID + "_" + strings.Repeat("A", 20) + "_" + strings.Repeat("A", 22)
	service := &Service{config: (Config{Enabled: true, KeyEnv: "KEY"}).WithDefaults(900), db: db, redis: redisClient, key: key[:], now: func() time.Time { return now }}
	digest := service.digest("management-api-token-v1", token)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT credential_id, adv_id, credential_name, token_digest, permissions, expires_at, revoked_at")).
		WithArgs(publicID).WillReturnRows(sqlmock.NewRows([]string{"credential_id", "adv_id", "credential_name", "token_digest", "permissions", "expires_at", "revoked_at"}).
		AddRow(9, 7, "reader", digest, ScopeCampaignRead, now.Add(time.Hour), nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE api_credential SET last_used_at=")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM adv_campaign c").WithArgs(uint64(7), uint64(99)).WillReturnError(sqlmock.ErrCancelled)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/99", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	service.Handler().ServeHTTP(recorder, request)
	// The sentinel database error is deliberately mapped to a dependency error;
	// the SQL expectation proves request adv_id cannot override credential 7.
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandlerDistinguishesCredentialStoreFailureFromInvalidBearer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	key := sha256.Sum256([]byte("management-key"))
	publicID := "00112233445566778899aabbccddeeff"
	token := "w8m_v1_" + publicID + "_" + strings.Repeat("A", 43)
	service := &Service{config: (Config{Enabled: true, KeyEnv: "KEY"}).WithDefaults(900), db: db, key: key[:], now: time.Now}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT credential_id, adv_id, credential_name, token_digest, permissions, expires_at, revoked_at")).
		WithArgs(publicID).WillReturnError(errors.New("credential database unavailable"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/advertiser", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	service.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable || recorder.Header().Get("Retry-After") == "" || recorder.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("status/headers=%d %#v body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestQuotaCountsCredentialAndAccountIndependently(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	service := &Service{config: Config{CredentialRequestsMinute: 2, AccountRequestsMinute: 3}, redis: client, now: func() time.Time { return now }}
	for index := 0; index < 2; index++ {
		allowed, err := service.allowQuota(context.Background(), Principal{CredentialID: 9, AdvertiserID: 7})
		if err != nil || !allowed {
			t.Fatalf("request %d allowed=%t err=%v", index, allowed, err)
		}
	}
	allowed, err := service.allowQuota(context.Background(), Principal{CredentialID: 9, AdvertiserID: 7})
	if err != nil || allowed {
		t.Fatalf("third credential request allowed=%t err=%v", allowed, err)
	}
	allowed, err = service.allowQuota(context.Background(), Principal{CredentialID: 10, AdvertiserID: 7})
	if err != nil || !allowed {
		t.Fatalf("independent credential request allowed=%t err=%v", allowed, err)
	}
	allowed, err = service.allowQuota(context.Background(), Principal{CredentialID: 10, AdvertiserID: 7})
	if err != nil || allowed {
		t.Fatalf("account quota request allowed=%t err=%v", allowed, err)
	}
	foundTTL := false
	for _, key := range server.Keys() {
		if strings.HasPrefix(key, "aofei:management-api:quota:credential:9:") && server.TTL(key) > 0 {
			foundTTL = true
		}
	}
	if !foundTTL {
		t.Fatal("credential quota key has no TTL")
	}
}

func TestDispatchChecksNestedScopeBeforeOwnershipLookup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h := &handler{service: &Service{db: db}}
	for _, path := range []string{"/api/v1/campaigns/7/items", "/api/v1/items/8/creatives"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if _, err := h.dispatch(recorder, request, "request", Principal{}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("dispatch %s error=%v want forbidden", path, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestParseIfMatchRequiresExactQuotedPositiveVersion(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/1", nil)
	request.Header.Set("If-Match", `"v12"`)
	if version, err := parseIfMatch(request); err != nil || version != 12 {
		t.Fatalf("valid If-Match version=%d err=%v", version, err)
	}
	for _, raw := range []string{"v12", `"v0"`, `"v01"`, `W/"v12"`, `""v12""`} {
		request.Header.Set("If-Match", raw)
		if _, err := parseIfMatch(request); err == nil {
			t.Errorf("If-Match %q was accepted", raw)
		}
	}
}

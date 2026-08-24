package managementapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mediocregopher/radix/v4"
)

const (
	integrationAdvOne = uint64(4000000001)
	integrationAdvTwo = uint64(4000000002)
)

func TestMySQLManagementAPILifecycle(t *testing.T) {
	dsn := os.Getenv("MANAGEMENT_API_INTEGRATION_DSN")
	redisAddr := os.Getenv("MANAGEMENT_API_INTEGRATION_REDIS")
	if dsn == "" || redisAddr == "" {
		t.Skip("MANAGEMENT_API_INTEGRATION_DSN and MANAGEMENT_API_INTEGRATION_REDIS are required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	redisClient, err := (radix.PoolConfig{Size: 8}).New(context.Background(), "tcp", redisAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer redisClient.Close()

	cleanupIntegrationFixtures(t, db)
	t.Cleanup(func() { cleanupIntegrationFixtures(t, db) })
	for id, email := range map[uint64]string{integrationAdvOne: "i03-one@example.invalid", integrationAdvTwo: "i03-two@example.invalid"} {
		if _, err := db.Exec(`INSERT INTO adv (adv_id, email, passwd, active, created) VALUES (?, ?, '$2a$12$integrationfixtureonly000000000000000000000000000000', 'Yes', UTC_TIMESTAMP())`, id, email); err != nil {
			t.Fatal(err)
		}
	}
	key := sha256.Sum256([]byte("disposable-i03-integration-key"))
	now := time.Now().UTC().Truncate(time.Microsecond)
	service := &Service{
		config: Config{Enabled: true, KeyEnv: "INTEGRATION", RequestTimeoutMS: 5000, MaxBodyBytes: 256 << 10, CredentialRequestsMinute: 1000, AccountRequestsMinute: 2000, CacheActivationSeconds: 900},
		db:     db, redis: redisClient, key: key[:], now: func() time.Time { return now }, random: rand.Reader,
	}
	credential, token, err := service.IssueCredential(context.Background(), Actor{Role: "adv", ID: integrationAdvOne}, integrationAdvOne, "integration", []string{ScopeCampaignRead, ScopeCampaignWrite, ScopeCreativeRead, ScopeCreativeWrite, ScopeTargetingRead, ScopeTargetingWrite, ScopeReportRead}, now.Add(24*time.Hour), "I03 disposable integration")
	if err != nil {
		t.Fatal(err)
	}
	tokenParts := strings.SplitN(token, "_", 4)
	if len(tokenParts) != 4 || tokenParts[0] != "w8m" || tokenParts[1] != "v1" || tokenParts[2] != credential.PublicID || len(tokenParts[3]) != 43 {
		t.Fatalf("malformed issued token shape")
	}
	var storedDigest []byte
	if err := db.QueryRow(`SELECT token_digest FROM api_credential WHERE credential_id=?`, credential.ID).Scan(&storedDigest); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(storedDigest, []byte(token)) || len(storedDigest) != sha256.Size {
		t.Fatal("credential was not stored as a fixed digest")
	}

	handler := service.Handler()
	spend := ExactDecimal("25.000000000")
	imps := uint64(1000)
	schedule := strings.Repeat("1", 168)
	campaignInput := CampaignInput{Name: "I03 campaign", ExternalID: "integration-1", TargetType: "Web", Delivery: DeliveryPolicy{Timezone: "Asia/Shanghai", WeeklySchedule: &schedule, Pacing: "Even", TotalLimits: Limits{SpendUSD: &spend, Imps: &imps}, DailyLimits: Limits{Imps: &imps}}}
	created := apiRequest(t, handler, http.MethodPost, "/api/v1/campaigns", token, "campaign-create-key", "", campaignInput)
	if created.Code != http.StatusAccepted {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var campaignAccepted struct {
		Data mutationPayload `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &campaignAccepted); err != nil {
		t.Fatal(err)
	}
	campaign, ok := campaignAccepted.Data.Resource.(map[string]any)
	if !ok {
		t.Fatalf("campaign response = %#v", campaignAccepted)
	}
	campaignID := uint64(campaign["id"].(float64))
	operationID := campaignAccepted.Data.Operation.ID
	if operationID == "" {
		t.Fatal("create response omitted activation operation")
	}

	replay := apiRequest(t, handler, http.MethodPost, "/api/v1/campaigns", token, "campaign-create-key", "", campaignInput)
	if replay.Code != 202 || replay.Header().Get("Idempotency-Replayed") != "true" || replay.Body.String() != created.Body.String() {
		t.Fatalf("replay status/header/body=%d %q %s", replay.Code, replay.Header().Get("Idempotency-Replayed"), replay.Body.String())
	}
	changed := campaignInput
	changed.Name = "different"
	conflict := apiRequest(t, handler, http.MethodPost, "/api/v1/campaigns", token, "campaign-create-key", "", changed)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), "idempotency_conflict") {
		t.Fatalf("changed replay=%d %s", conflict.Code, conflict.Body.String())
	}

	otherCampaign, err := db.Exec(`INSERT INTO adv_campaign (adv_id, campaign_name, delivery_timezone, pacing_mode, active, created) VALUES (?, 'other account', 'UTC', 'Fast', 'New', UTC_TIMESTAMP())`, integrationAdvTwo)
	if err != nil {
		t.Fatal(err)
	}
	otherID, _ := otherCampaign.LastInsertId()
	cross := apiRequest(t, handler, http.MethodGet, fmt.Sprintf("/api/v1/campaigns/%d", otherID), token, "", "", nil)
	if cross.Code != http.StatusNotFound {
		t.Fatalf("cross-account status=%d body=%s", cross.Code, cross.Body.String())
	}

	itemInput := ItemInput{Name: "I03 item", LandingURL: "https://example.invalid/landing", PriceCPMUSD: "2.500000", Delivery: DeliveryPolicy{Pacing: "Fast", TotalLimits: Limits{}, DailyLimits: Limits{}}}
	const goroutines = 6
	responses := make([]*httptest.ResponseRecorder, goroutines)
	var wg sync.WaitGroup
	for index := range responses {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			responses[index] = apiRequest(t, handler, http.MethodPost, fmt.Sprintf("/api/v1/campaigns/%d/items", campaignID), token, "concurrent-item-key", "", itemInput)
		}(index)
	}
	wg.Wait()
	replayed := 0
	for _, response := range responses {
		if response.Code != http.StatusAccepted {
			t.Fatalf("concurrent status=%d body=%s", response.Code, response.Body.String())
		}
		if response.Header().Get("Idempotency-Replayed") == "true" {
			replayed++
		}
	}
	if replayed != goroutines-1 {
		t.Fatalf("replayed responses=%d want=%d", replayed, goroutines-1)
	}
	var itemCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM adv_item i INNER JOIN adv_campaign c USING (campaign_id) WHERE c.adv_id=? AND i.item_name='I03 item'`, integrationAdvOne).Scan(&itemCount); err != nil || itemCount != 1 {
		t.Fatalf("item count=%d err=%v", itemCount, err)
	}
	var itemID uint64
	if err := db.QueryRow(`SELECT i.item_id FROM adv_item i INNER JOIN adv_campaign c USING (campaign_id) WHERE c.adv_id=? AND i.item_name='I03 item'`, integrationAdvOne).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	itemRead := apiRequest(t, handler, http.MethodGet, fmt.Sprintf("/api/v1/items/%d", itemID), token, "", "", nil)
	if itemRead.Code != http.StatusOK || !strings.Contains(itemRead.Body.String(), `"timezone":"Asia/Shanghai"`) || strings.Contains(itemRead.Body.String(), `"timezone":"inherited"`) {
		t.Fatalf("item inherited timezone status=%d body=%s", itemRead.Code, itemRead.Body.String())
	}

	creativeInput := CreativeInput{Name: "I03 creative", Width: 300, Height: 250, MediaType: "Banner", SourceURL: "https://cdn.example.invalid/ad.png", Weight: 1, Status: "Yes"}
	creativeResponse := apiRequest(t, handler, http.MethodPost, fmt.Sprintf("/api/v1/items/%d/creatives", itemID), token, "creative-create-key", "", creativeInput)
	if creativeResponse.Code != 202 {
		t.Fatalf("creative status=%d body=%s", creativeResponse.Code, creativeResponse.Body.String())
	}
	targetingInput := TargetingInput{SiteTypes: []string{"Web"}, Languages: []string{"EN", "ZH"}, DeviceTypes: []string{"1", "2"}, Positions: []string{"1", "3"}, AccessOrder: "Inherit", ChannelOrder: "Black"}
	targetingResponse := apiRequest(t, handler, http.MethodPatch, fmt.Sprintf("/api/v1/items/%d/targeting", itemID), token, "targeting-update-key", `"v1"`, targetingInput)
	if targetingResponse.Code != 202 {
		t.Fatalf("targeting status=%d body=%s", targetingResponse.Code, targetingResponse.Body.String())
	}

	if _, err := db.Exec(`UPDATE adv_campaign SET description='human portal update' WHERE campaign_id=?`, campaignID); err != nil {
		t.Fatal(err)
	}
	stale := apiRequest(t, handler, http.MethodPatch, fmt.Sprintf("/api/v1/campaigns/%d", campaignID), token, "stale-campaign-key", `"v1"`, campaignInput)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "version_conflict") {
		t.Fatalf("stale update=%d %s", stale.Code, stale.Body.String())
	}

	publicationToken, err := PrepareOperationsPublication(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	lateCreated := apiRequest(t, handler, http.MethodPost, "/api/v1/campaigns", token, "late-campaign-key", "", campaignInput)
	if lateCreated.Code != http.StatusAccepted {
		t.Fatalf("late create status=%d body=%s", lateCreated.Code, lateCreated.Body.String())
	}
	var lateAccepted struct {
		Data mutationPayload `json:"data"`
	}
	if err := json.Unmarshal(lateCreated.Body.Bytes(), &lateAccepted); err != nil {
		t.Fatal(err)
	}
	if err := MarkOperationsActive(context.Background(), db, "redis", publicationToken, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	active := apiRequest(t, handler, http.MethodGet, "/api/v1/operations/"+operationID, token, "", "", nil)
	if active.Code != 200 || !strings.Contains(active.Body.String(), `"activation_state":"Active"`) {
		t.Fatalf("operation=%d %s", active.Code, active.Body.String())
	}
	late := apiRequest(t, handler, http.MethodGet, "/api/v1/operations/"+lateAccepted.Data.Operation.ID, token, "", "", nil)
	if late.Code != 200 || !strings.Contains(late.Body.String(), `"activation_state":"Pending"`) {
		t.Fatalf("late operation crossed publication generation=%d %s", late.Code, late.Body.String())
	}
	firstPage := apiRequest(t, handler, http.MethodGet, "/api/v1/campaigns?limit=1", token, "", "", nil)
	var firstPageBody struct {
		Data []Campaign `json:"data"`
		Meta pageMeta   `json:"meta"`
	}
	if firstPage.Code != http.StatusOK || json.Unmarshal(firstPage.Body.Bytes(), &firstPageBody) != nil || len(firstPageBody.Data) != 1 || firstPageBody.Meta.NextCursor == "" {
		t.Fatalf("first campaign page status=%d body=%s", firstPage.Code, firstPage.Body.String())
	}
	secondPage := apiRequest(t, handler, http.MethodGet, "/api/v1/campaigns?limit=1&cursor="+firstPageBody.Meta.NextCursor, token, "", "", nil)
	var secondPageBody struct {
		Data []Campaign `json:"data"`
	}
	if secondPage.Code != http.StatusOK || json.Unmarshal(secondPage.Body.Bytes(), &secondPageBody) != nil || len(secondPageBody.Data) != 1 || secondPageBody.Data[0].ID == firstPageBody.Data[0].ID {
		t.Fatalf("second campaign page status=%d body=%s", secondPage.Code, secondPage.Body.String())
	}

	if _, err := db.Exec(`UPDATE api_audit SET outcome='Failure' WHERE adv_id=? LIMIT 1`, integrationAdvOne); err == nil {
		t.Fatal("API audit update was not rejected")
	}
	if _, err := db.Exec(`DELETE FROM api_audit WHERE adv_id=? LIMIT 1`, integrationAdvOne); err == nil {
		t.Fatal("API audit delete was not rejected")
	}
	if _, err := db.Exec(`
INSERT INTO api_audit
 (event_name, actor_role, actor_id, adv_id, object_type, prior_state, new_state, reason, outcome, created_at)
VALUES ('IntegrationExpiredEvidence','admin',999,?,'audit','Active','Expired','I03 disposable retention fixture','Success',?)`,
		integrationAdvOne, now.AddDate(0, 0, -401)); err != nil {
		t.Fatal(err)
	}
	deleted, err := PruneAudit(context.Background(), db, Actor{Role: "admin", ID: 999}, 400, 1000, "I03 disposable API audit retention")
	if err != nil || deleted < 1 {
		t.Fatalf("API audit retention deleted=%d err=%v", deleted, err)
	}
	var expiredCount, pruneEvidence int
	if err := db.QueryRow(`SELECT COUNT(*) FROM api_audit WHERE event_name='IntegrationExpiredEvidence' AND adv_id=?`, integrationAdvOne).Scan(&expiredCount); err != nil || expiredCount != 0 {
		t.Fatalf("expired API audit count=%d err=%v", expiredCount, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM api_audit WHERE event_name='ManagementAPIAuditPruned' AND actor_id=999 AND reason='I03 disposable API audit retention'`).Scan(&pruneEvidence); err != nil || pruneEvidence != 1 {
		t.Fatalf("API prune evidence count=%d err=%v", pruneEvidence, err)
	}
	if _, err := db.Exec(`DELETE FROM api_audit WHERE event_name='ManagementAPIAuditPruned' LIMIT 1`); err == nil {
		t.Fatal("API audit retention bypass escaped its dedicated connection")
	}
	var leaked int
	if err := db.QueryRow(`SELECT COUNT(*) FROM api_audit WHERE adv_id=? AND CONCAT_WS('|', event_name, actor_role, prior_state, new_state, reason) LIKE ?`, integrationAdvOne, "%"+token+"%").Scan(&leaked); err != nil || leaked != 0 {
		t.Fatalf("audit token leakage count=%d err=%v", leaked, err)
	}

	_, rotatedToken, err := service.RotateCredential(context.Background(), Actor{Role: "adv", ID: integrationAdvOne}, integrationAdvOne, credential.ID, "I03 disposable rotation")
	if err != nil {
		t.Fatal(err)
	}
	old := apiRequest(t, handler, http.MethodGet, "/api/v1/advertiser", token, "", "", nil)
	if old.Code != http.StatusUnauthorized {
		t.Fatalf("old token status=%d", old.Code)
	}
	newResponse := apiRequest(t, handler, http.MethodGet, "/api/v1/advertiser", rotatedToken, "", "", nil)
	if newResponse.Code != http.StatusOK {
		t.Fatalf("rotated token status=%d body=%s", newResponse.Code, newResponse.Body.String())
	}
	if err := service.RevokeCredential(context.Background(), Actor{Role: "adv", ID: integrationAdvOne}, integrationAdvOne, credential.ID, "I03 disposable revocation"); err != nil {
		t.Fatal(err)
	}
	revoked := apiRequest(t, handler, http.MethodGet, "/api/v1/advertiser", rotatedToken, "", "", nil)
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status=%d", revoked.Code)
	}
}

func apiRequest(t *testing.T, handler http.Handler, method, path, token, idempotency, ifMatch string, input any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if input != nil {
		var err error
		body, err = json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotency != "" {
		request.Header.Set("Idempotency-Key", idempotency)
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func cleanupIntegrationFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SET @aofei_api_audit_retention=1`); err != nil {
		t.Fatal(err)
	}
	_, deleteErr := conn.ExecContext(ctx, `DELETE FROM api_audit WHERE adv_id IN (4000000001,4000000002) OR (event_name='ManagementAPIAuditPruned' AND reason='I03 disposable API audit retention')`)
	_, resetErr := conn.ExecContext(ctx, `SET @aofei_api_audit_retention=0`)
	if deleteErr != nil || resetErr != nil {
		t.Fatalf("audit cleanup delete=%v reset=%v", deleteErr, resetErr)
	}
	statements := []string{
		`CREATE TEMPORARY TABLE IF NOT EXISTS api_test_balance_ids (balance_id INT UNSIGNED PRIMARY KEY)`,
		`TRUNCATE TABLE api_test_balance_ids`,
		`INSERT IGNORE INTO api_test_balance_ids SELECT total_balance_id FROM adv_campaign WHERE adv_id IN (4000000001,4000000002) AND total_balance_id IS NOT NULL`,
		`INSERT IGNORE INTO api_test_balance_ids SELECT daily_balance_id FROM adv_campaign WHERE adv_id IN (4000000001,4000000002) AND daily_balance_id IS NOT NULL`,
		`INSERT IGNORE INTO api_test_balance_ids SELECT i.total_balance_id FROM adv_item i INNER JOIN adv_campaign c USING (campaign_id) WHERE c.adv_id IN (4000000001,4000000002) AND i.total_balance_id IS NOT NULL`,
		`INSERT IGNORE INTO api_test_balance_ids SELECT i.daily_balance_id FROM adv_item i INNER JOIN adv_campaign c USING (campaign_id) WHERE c.adv_id IN (4000000001,4000000002) AND i.daily_balance_id IS NOT NULL`,
		`DELETE FROM api_idempotency WHERE adv_id IN (4000000001,4000000002)`,
		`DELETE FROM api_operation WHERE adv_id IN (4000000001,4000000002)`,
		`DELETE FROM api_credential WHERE adv_id IN (4000000001,4000000002)`,
		`DELETE cr FROM adv_creative cr INNER JOIN adv_item i USING (item_id) INNER JOIN adv_campaign c USING (campaign_id) WHERE c.adv_id IN (4000000001,4000000002)`,
		`DELETE i FROM adv_item i INNER JOIN adv_campaign c USING (campaign_id) WHERE c.adv_id IN (4000000001,4000000002)`,
		`DELETE FROM adv_campaign WHERE adv_id IN (4000000001,4000000002)`,
		`DELETE b FROM adv_balance b INNER JOIN api_test_balance_ids x USING (balance_id)`,
		`DELETE FROM adv WHERE adv_id IN (4000000001,4000000002)`,
	}
	for _, statement := range statements {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatalf("cleanup %q: %v", statement, err)
		}
	}
	// Prove the connection-local audit deletion bypass did not escape cleanup.
	var bypass int
	if err := conn.QueryRowContext(ctx, `SELECT COALESCE(@aofei_api_audit_retention,0)`).Scan(&bypass); err != nil || bypass != 0 {
		t.Fatalf("audit cleanup bypass=%d err=%v", bypass, err)
	}
}

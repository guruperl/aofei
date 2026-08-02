package dsp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	actionContractVersion = 1
	maxActionBodyBytes    = 64 << 10
	actionDBTimeout       = 2 * time.Second
)

var (
	actionEventIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	customActionPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}\.[a-z][a-z0-9_]{0,31}$`)
	decimalUSDPattern    = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(\.[0-9]{1,6})?$`)
)

type actionTokenPayload struct {
	Version      int    `json:"v"`
	IssuedAt     int64  `json:"iat"`
	ExpiresAt    int64  `json:"exp"`
	JTI          string `json:"jti"`
	AuctionID    string `json:"auction_id"`
	AuctionBidID string `json:"auction_bid_id"`
	AuctionImpID string `json:"auction_imp_id"`
	AdvID        uint32 `json:"adv_id"`
	CampaignID   uint32 `json:"campaign_id"`
	ItemID       uint32 `json:"item_id"`
	CreativeID   uint32 `json:"creative_id"`
	PubID        uint32 `json:"pub_id"`
	SiteID       uint32 `json:"site_id"`
	SlotID       uint32 `json:"slot_id"`
}

type actionRequest struct {
	Version    int    `json:"version"`
	Token      string `json:"token"`
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	ActionName string `json:"action_name,omitempty"`
	OccurredAt string `json:"occurred_at"`
	ValueUSD   string `json:"value_usd,omitempty"`
}

type actionFact struct {
	Token       actionTokenPayload
	TokenHash   [32]byte
	Fingerprint [32]byte
	EventID     string
	EventType   string
	ActionName  string
	OccurredAt  time.Time
	ValueUSD    string
	Attribution string
	TouchAt     sql.NullTime
	Late        bool
	Pseudonym   string
}

type actionStoreResult uint8

const (
	actionStored actionStoreResult = iota
	actionDuplicate
	actionConflict
)

func (self *Controller) actionPolicy() (clickWindow, viewWindow, maxAge, tokenTTL, requestSkew time.Duration) {
	clickHours, viewHours, maxHours, tokenSeconds, skewSeconds := (*Config)(nil).actionPolicyValues()
	if self != nil {
		clickHours, viewHours, maxHours, tokenSeconds, skewSeconds = self.C.actionPolicyValues()
	}
	return time.Duration(clickHours) * time.Hour,
		time.Duration(viewHours) * time.Hour,
		time.Duration(maxHours) * time.Hour,
		time.Duration(tokenSeconds) * time.Second,
		time.Duration(skewSeconds) * time.Second
}

func newActionToken(secret string, wl *WinLoss, ttl time.Duration, now time.Time) (string, error) {
	if secret == "" || wl == nil || ttl <= 0 {
		return "", fmt.Errorf("action token configuration is incomplete")
	}
	if wl.AuctionID == "" || wl.AuctionBidID == "" || wl.AuctionImpID == "" || wl.RAdv.AdvID == 0 || wl.RAdv.CampaignID == 0 || wl.RAdv.ItemID == 0 || wl.RAdv.CreativeID == 0 {
		return "", fmt.Errorf("action token lineage is incomplete")
	}
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	payload := actionTokenPayload{
		Version: actionContractVersion, IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix(),
		JTI:       base64.RawURLEncoding.EncodeToString(nonce),
		AuctionID: wl.AuctionID, AuctionBidID: wl.AuctionBidID, AuctionImpID: wl.AuctionImpID,
		AdvID: wl.RAdv.AdvID, CampaignID: wl.RAdv.CampaignID, ItemID: wl.RAdv.ItemID, CreativeID: wl.RAdv.CreativeID,
		PubID: wl.RPub.PubID, SiteID: wl.RPub.SiteID, SlotID: wl.RPub.SlotID,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	signature := hmacSHA256([]byte(secret), []byte("w8m-action-token-v1\n"+encoded))
	return encoded + "." + hex.EncodeToString(signature), nil
}

func validateActionToken(secret, token string, now time.Time, skew time.Duration) (actionTokenPayload, error) {
	var payload actionTokenPayload
	encoded, signatureText, ok := strings.Cut(token, ".")
	if !ok || encoded == "" || signatureText == "" {
		return payload, fmt.Errorf("action token is malformed")
	}
	signature, err := hex.DecodeString(signatureText)
	if err != nil || len(signature) != sha256.Size {
		return payload, fmt.Errorf("action token signature is malformed")
	}
	expected := hmacSHA256([]byte(secret), []byte("w8m-action-token-v1\n"+encoded))
	if !hmac.Equal(signature, expected) {
		return payload, fmt.Errorf("action token signature is invalid")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) > 4096 || json.Unmarshal(raw, &payload) != nil {
		return actionTokenPayload{}, fmt.Errorf("action token payload is invalid")
	}
	if payload.Version != actionContractVersion || payload.JTI == "" || payload.IssuedAt <= 0 || payload.ExpiresAt <= payload.IssuedAt ||
		payload.AdvID == 0 || payload.CampaignID == 0 || payload.ItemID == 0 || payload.CreativeID == 0 ||
		payload.AuctionID == "" || payload.AuctionBidID == "" || payload.AuctionImpID == "" {
		return actionTokenPayload{}, fmt.Errorf("action token lineage is invalid")
	}
	if now.Before(time.Unix(payload.IssuedAt, 0).Add(-skew)) || !now.Before(time.Unix(payload.ExpiresAt, 0)) {
		return actionTokenPayload{}, fmt.Errorf("action token is expired or not yet valid")
	}
	return payload, nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

func appendActionToken(rawURL, token string) string {
	if rawURL == "" || token == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return rawURL
	}
	query := u.Query()
	query.Set("w8m_action_token", token)
	u.RawQuery = query.Encode()
	return u.String()
}

// ServeAction accepts a signed analytical action and persists it idempotently.
// It never mutates delivery reservations, caps, statements, or CPM spend.
func (self *Controller) ServeAction(w http.ResponseWriter, r *http.Request) {
	metricActionRequests.Add(1)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost {
		recordActionRejection("method")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if contentType != "application/json" {
		recordActionRejection("content_type")
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxActionBodyBytes+1))
	if err != nil || len(raw) > maxActionBodyBytes {
		recordActionRejection("body")
		http.Error(w, "invalid action body", http.StatusRequestEntityTooLarge)
		return
	}
	var request actionRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		recordActionRejection("payload")
		http.Error(w, "invalid action payload", http.StatusBadRequest)
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		recordActionRejection("payload")
		http.Error(w, "invalid action payload", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	clickWindow, viewWindow, maxAge, _, requestSkew := self.actionPolicy()
	token, err := validateActionToken(self.trackingSecret(), request.Token, now, requestSkew)
	if err != nil {
		recordActionRejection("token")
		http.Error(w, "invalid action token", http.StatusUnauthorized)
		return
	}
	if err := validateActionRequestSignature(r.Header, request.Token, raw, now, requestSkew); err != nil {
		recordActionRejection("signature")
		http.Error(w, "invalid action signature", http.StatusUnauthorized)
		return
	}
	fact, err := validateActionPayload(request, token, now, maxAge, requestSkew, self.trackingSecret())
	if err != nil {
		recordActionRejection("payload")
		http.Error(w, "invalid action payload", http.StatusBadRequest)
		return
	}
	if self == nil || self.DB == nil {
		recordActionRejection("dependency")
		http.Error(w, "action storage unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), actionDBTimeout)
	defer cancel()
	result, err := self.storeAction(ctx, &fact, clickWindow, viewWindow)
	if err != nil {
		recordActionRejection("dependency")
		http.Error(w, "action storage unavailable", http.StatusServiceUnavailable)
		return
	}
	switch result {
	case actionConflict:
		recordActionRejection("conflict")
		http.Error(w, "action event id conflicts with an existing event", http.StatusConflict)
		return
	case actionDuplicate:
		metricActionDuplicates.Add(1)
	default:
		metricActionAccepted.Add(1)
		recordActionAttribution(fact.Attribution)
	}
	w.WriteHeader(http.StatusNoContent)
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func validateActionRequestSignature(header http.Header, token string, raw []byte, now time.Time, skew time.Duration) error {
	timestampText := strings.TrimSpace(header.Get("X-W8M-Action-Timestamp"))
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return fmt.Errorf("action timestamp is invalid")
	}
	when := time.Unix(timestamp, 0)
	if when.Before(now.Add(-skew)) || when.After(now.Add(skew)) {
		return fmt.Errorf("action timestamp is outside the accepted window")
	}
	signature, err := hex.DecodeString(strings.TrimSpace(header.Get("X-W8M-Action-Signature")))
	if err != nil || len(signature) != sha256.Size {
		return fmt.Errorf("action signature is malformed")
	}
	message := append([]byte("w8m-action-request-v1\n"+timestampText+"\n"), raw...)
	if !hmac.Equal(signature, hmacSHA256([]byte(token), message)) {
		return fmt.Errorf("action signature is invalid")
	}
	return nil
}

func validateActionPayload(request actionRequest, token actionTokenPayload, now time.Time, maxAge, skew time.Duration, secret string) (actionFact, error) {
	fact := actionFact{Token: token, EventID: request.EventID, EventType: request.EventType, ActionName: request.ActionName, ValueUSD: request.ValueUSD}
	if request.Version != actionContractVersion || !actionEventIDPattern.MatchString(request.EventID) {
		return fact, fmt.Errorf("action version or event id is invalid")
	}
	switch request.EventType {
	case "conversion", "download", "video_complete":
		if request.ActionName != "" || request.ValueUSD != "" {
			return fact, fmt.Errorf("action type does not accept name or value")
		}
	case "purchase":
		if request.ActionName != "" || !decimalUSDPattern.MatchString(request.ValueUSD) || request.ValueUSD == "0" || strings.HasPrefix(request.ValueUSD, "0.0") && strings.Trim(request.ValueUSD[2:], "0") == "" {
			return fact, fmt.Errorf("purchase requires a positive USD value")
		}
	case "custom":
		if !customActionPattern.MatchString(request.ActionName) || request.ValueUSD != "" {
			return fact, fmt.Errorf("custom action name is invalid")
		}
	default:
		return fact, fmt.Errorf("action type is unsupported")
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, request.OccurredAt)
	if err != nil {
		return fact, fmt.Errorf("action occurred_at is invalid")
	}
	occurredAt = occurredAt.UTC()
	issuedAt := time.Unix(token.IssuedAt, 0).UTC()
	if occurredAt.Before(issuedAt.Add(-skew)) || occurredAt.After(now.Add(skew)) || occurredAt.Before(now.Add(-maxAge)) {
		return fact, fmt.Errorf("action occurred_at is outside the accepted lifecycle")
	}
	fact.OccurredAt = occurredAt
	fact.TokenHash = sha256.Sum256([]byte(request.Token))
	fingerprintInput, _ := json.Marshal(struct {
		TokenHash  string `json:"token_hash"`
		EventType  string `json:"event_type"`
		ActionName string `json:"action_name"`
		OccurredAt string `json:"occurred_at"`
		ValueUSD   string `json:"value_usd"`
	}{hex.EncodeToString(fact.TokenHash[:]), request.EventType, request.ActionName, occurredAt.Format(time.RFC3339Nano), request.ValueUSD})
	fact.Fingerprint = sha256.Sum256(fingerprintInput)
	fact.Pseudonym = hex.EncodeToString(hmacSHA256([]byte(secret), []byte("w8m-action-pseudonym-v1\x00"+token.JTI)))
	return fact, nil
}

func actionLineageHash(token actionTokenPayload) [32]byte {
	return sha256.Sum256([]byte(strings.Join([]string{
		token.AuctionID, token.AuctionBidID, token.AuctionImpID,
		strconv.FormatUint(uint64(token.AdvID), 10), strconv.FormatUint(uint64(token.CampaignID), 10),
		strconv.FormatUint(uint64(token.ItemID), 10), strconv.FormatUint(uint64(token.CreativeID), 10),
		strconv.FormatUint(uint64(token.PubID), 10), strconv.FormatUint(uint64(token.SiteID), 10),
		strconv.FormatUint(uint64(token.SlotID), 10),
	}, "\x00")))
}

func actionLineageHashFromWinLoss(wl *WinLoss) [32]byte {
	if wl == nil {
		return [32]byte{}
	}
	return actionLineageHash(actionTokenPayload{
		AuctionID: wl.AuctionID, AuctionBidID: wl.AuctionBidID, AuctionImpID: wl.AuctionImpID,
		AdvID: wl.RAdv.AdvID, CampaignID: wl.RAdv.CampaignID, ItemID: wl.RAdv.ItemID, CreativeID: wl.RAdv.CreativeID,
		PubID: wl.RPub.PubID, SiteID: wl.RPub.SiteID, SlotID: wl.RPub.SlotID,
	})
}

func (self *Controller) storeAction(ctx context.Context, fact *actionFact, clickWindow, viewWindow time.Duration) (actionStoreResult, error) {
	lineage := actionLineageHash(fact.Token)
	clickStart, viewStart := fact.OccurredAt.Add(-clickWindow), fact.OccurredAt.Add(-viewWindow)
	var touchType string
	var touchAt time.Time
	err := self.DB.QueryRowContext(ctx, `
SELECT touch_type, occurred_at
FROM measurement_touch
WHERE lineage_hash=? AND occurred_at<=?
  AND ((touch_type='click' AND occurred_at>=?) OR (touch_type='view' AND occurred_at>=?))
ORDER BY (touch_type='click') DESC, occurred_at DESC
LIMIT 1`, lineage[:], fact.OccurredAt, clickStart, viewStart).Scan(&touchType, &touchAt)
	if err != nil && err != sql.ErrNoRows {
		return actionStored, err
	}
	fact.Attribution = "unattributed"
	if err == nil {
		fact.Attribution = touchType
		fact.TouchAt = sql.NullTime{Time: touchAt.UTC(), Valid: true}
	}
	fact.Late = time.Since(fact.OccurredAt) > clickWindow
	value := any(nil)
	if fact.ValueUSD != "" {
		value = fact.ValueUSD
	}
	touchValue := any(nil)
	if fact.TouchAt.Valid {
		touchValue = fact.TouchAt.Time
	}
	result, err := self.DB.ExecContext(ctx, `
INSERT IGNORE INTO measurement_action (
  adv_id, campaign_id, item_id, creative_id, pub_id, site_id, slot_id,
  lineage_hash, token_hash, event_fingerprint, event_id, event_type, action_name,
  occurred_at, received_at, value_usd, currency, attribution_type, touch_at,
  late, privacy_mode, privacy_reason, action_pseudonym, expires_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),?,'USD',?,?,?,'contextual','signed_lineage_contextual',?,DATE_ADD(UTC_TIMESTAMP(6), INTERVAL ? HOUR))`,
		fact.Token.AdvID, fact.Token.CampaignID, fact.Token.ItemID, fact.Token.CreativeID,
		fact.Token.PubID, fact.Token.SiteID, fact.Token.SlotID,
		lineage[:], fact.TokenHash[:], fact.Fingerprint[:], fact.EventID, fact.EventType, nullActionString(fact.ActionName),
		fact.OccurredAt, value, fact.Attribution, touchValue, fact.Late, fact.Pseudonym, actionRetentionHours(self.C))
	if err != nil {
		return actionStored, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return actionStored, err
	}
	if rows == 1 {
		return actionStored, nil
	}
	var existing []byte
	if err := self.DB.QueryRowContext(ctx, `SELECT event_fingerprint FROM measurement_action WHERE adv_id=? AND event_id=?`, fact.Token.AdvID, fact.EventID).Scan(&existing); err != nil {
		return actionStored, err
	}
	if hmac.Equal(existing, fact.Fingerprint[:]) {
		return actionDuplicate, nil
	}
	return actionConflict, nil
}

func nullActionString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (self *Controller) recordAttributionTouch(wl *WinLoss) {
	if self == nil || self.DB == nil || wl == nil || (wl.Status != StatusTrackImp && wl.Status != StatusTrackClk) {
		return
	}
	lineage := actionLineageHashFromWinLoss(wl)
	if lineage == [32]byte{} || wl.RAdv.AdvID == 0 || wl.RAdv.CampaignID == 0 || wl.RAdv.ItemID == 0 || wl.RAdv.CreativeID == 0 {
		return
	}
	touchType := "view"
	if wl.Status == StatusTrackClk {
		touchType = "click"
	}
	ctx, cancel := context.WithTimeout(context.Background(), actionDBTimeout)
	defer cancel()
	_, err := self.DB.ExecContext(ctx, `
INSERT IGNORE INTO measurement_touch (
  lineage_hash, touch_type, occurred_at, adv_id, campaign_id, item_id, creative_id,
  pub_id, site_id, slot_id, auction_id, auction_bid_id, auction_imp_id,
  privacy_mode, privacy_reason, expires_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,'contextual','signed_tracking_contextual',DATE_ADD(UTC_TIMESTAMP(6), INTERVAL ? HOUR))`,
		lineage[:], touchType, wl.Current.UTC(), wl.RAdv.AdvID, wl.RAdv.CampaignID, wl.RAdv.ItemID, wl.RAdv.CreativeID,
		wl.RPub.PubID, wl.RPub.SiteID, wl.RPub.SlotID, wl.AuctionID, wl.AuctionBidID, wl.AuctionImpID, actionRetentionHours(self.C))
	if err != nil {
		metricActionTouchErrors.Add(1)
	} else {
		metricActionTouches.Add(touchType, 1)
	}
}

func actionRetentionHours(config *Config) int {
	hours := 90 * 24
	if config != nil {
		hours = config.ActionRetentionHours
		if hours == 0 {
			_, _, maxAge, _, _ := config.actionPolicyValues()
			hours = maxAge
		}
	}
	if hours < 1 {
		return 1
	}
	return hours
}

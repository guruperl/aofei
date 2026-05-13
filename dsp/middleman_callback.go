package dsp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/genelet/winter/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const (
	middlemanCallbackKeyPrefix = "middleman:cb:"
	middlemanClickKeyPrefix    = "middleman:click:"
	middlemanBillKeyPrefix     = "middleman:bill:"
	middlemanNotifyKeyPrefix   = "middleman:notify:"
)

type middlemanCallbackStore interface {
	SetCallback(context.Context, string, middlemanCallbackContext, time.Duration) error
	GetCallback(context.Context, string) (middlemanCallbackContext, error)
	SetClick(context.Context, string, string, string, time.Duration) error
	GetClick(context.Context, string, string) (string, error)
	SetBillOnce(context.Context, string, time.Duration) (bool, error)
	ClearBill(context.Context, string) error
	SetNotifyOnce(context.Context, string, string, time.Duration) (bool, error)
	ClearNotify(context.Context, string, string) error
}

type redisMiddlemanCallbackStore struct {
	redis radix.Client
}

type middlemanCallbackContext struct {
	Token              string     `json:"token"`
	Created            time.Time  `json:"created"`
	RequestID          string     `json:"request_id"`
	ImpID              string     `json:"imp_id"`
	ResponseBidID      string     `json:"response_bid_id"`
	DownstreamBidID    string     `json:"downstream_bid_id"`
	DownstreamSeat     string     `json:"downstream_seat,omitempty"`
	DownstreamAdID     string     `json:"downstream_ad_id,omitempty"`
	DownstreamNURL     string     `json:"downstream_nurl,omitempty"`
	DownstreamBURL     string     `json:"downstream_burl,omitempty"`
	DownstreamLURL     string     `json:"downstream_lurl,omitempty"`
	DownstreamBidPrice float64    `json:"downstream_bid_price"`
	UpstreamBidPrice   float64    `json:"upstream_bid_price"`
	MarginCPM          float64    `json:"margin_cpm"`
	BillOnWin          bool       `json:"bill_on_win"`
	BidderID           uint32     `json:"bidder_id,omitempty"`
	GroupID            uint32     `json:"group_id,omitempty"`
	RouteBidderID      uint32     `json:"route_bidder_id,omitempty"`
	TargetID           uint32     `json:"target_id,omitempty"`
	RPub               match.RPub `json:"rpub,omitempty"`
	RAdv               match.RAdv `json:"radv,omitempty"`
}

type middlemanPrices struct {
	ChargePrice float64
	PayPrice    float64
	Currency    string
}

func (s redisMiddlemanCallbackStore) SetCallback(ctx context.Context, token string, value middlemanCallbackContext, ttl time.Duration) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.redis.Do(ctx, radix.Cmd(nil, "SETEX", middlemanCallbackKey(token), strconv.Itoa(ttlSeconds(ttl)), string(data)))
}

func (s redisMiddlemanCallbackStore) GetCallback(ctx context.Context, token string) (middlemanCallbackContext, error) {
	if s.redis == nil {
		return middlemanCallbackContext{}, fmt.Errorf("middleman callback redis is not configured")
	}
	var data []byte
	if err := s.redis.Do(ctx, radix.Cmd(&data, "GET", middlemanCallbackKey(token))); err != nil {
		return middlemanCallbackContext{}, err
	}
	if len(data) == 0 {
		return middlemanCallbackContext{}, fmt.Errorf("middleman callback token not found")
	}
	var value middlemanCallbackContext
	if err := json.Unmarshal(data, &value); err != nil {
		return middlemanCallbackContext{}, err
	}
	return value, nil
}

func (s redisMiddlemanCallbackStore) SetClick(ctx context.Context, requestToken, impID, token string, ttl time.Duration) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	return s.redis.Do(ctx, radix.Cmd(nil, "SETEX", middlemanClickKey(requestToken, impID), strconv.Itoa(ttlSeconds(ttl)), token))
}

func (s redisMiddlemanCallbackStore) GetClick(ctx context.Context, requestToken, impID string) (string, error) {
	if s.redis == nil {
		return "", fmt.Errorf("middleman callback redis is not configured")
	}
	var token string
	if err := s.redis.Do(ctx, radix.Cmd(&token, "GET", middlemanClickKey(requestToken, impID))); err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("middleman click token not found")
	}
	return token, nil
}

func (s redisMiddlemanCallbackStore) SetBillOnce(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("middleman callback redis is not configured")
	}
	var result string
	err := s.redis.Do(ctx, radix.Cmd(&result, "SET", middlemanBillKey(token), "1", "EX", strconv.Itoa(ttlSeconds(ttl)), "NX"))
	return result == "OK", err
}

func (s redisMiddlemanCallbackStore) ClearBill(ctx context.Context, token string) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	return s.redis.Do(ctx, radix.Cmd(nil, "DEL", middlemanBillKey(token)))
}

func (s redisMiddlemanCallbackStore) SetNotifyOnce(ctx context.Context, token, source string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("middleman callback redis is not configured")
	}
	var result string
	err := s.redis.Do(ctx, radix.Cmd(&result, "SET", middlemanNotifyKey(token, source), "1", "EX", strconv.Itoa(ttlSeconds(ttl)), "NX"))
	return result == "OK", err
}

func (s redisMiddlemanCallbackStore) ClearNotify(ctx context.Context, token, source string) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	return s.redis.Do(ctx, radix.Cmd(nil, "DEL", middlemanNotifyKey(token, source)))
}

func middlemanCallbackKey(token string) string {
	return middlemanCallbackKeyPrefix + token
}

func middlemanClickKey(requestToken, impID string) string {
	return middlemanClickKeyPrefix + requestToken + ":" + url.PathEscape(impID)
}

func middlemanBillKey(token string) string {
	return middlemanBillKeyPrefix + token
}

func middlemanNotifyKey(token, source string) string {
	return middlemanNotifyKeyPrefix + source + ":" + token
}

func ttlSeconds(ttl time.Duration) int {
	seconds := int(math.Ceil(ttl.Seconds()))
	if seconds <= 0 {
		return 1
	}
	return seconds
}

func (self *Controller) callbackStore() middlemanCallbackStore {
	if self.middlemanStore != nil {
		return self.middlemanStore
	}
	return redisMiddlemanCallbackStore{redis: self.Redis}
}

func (self *Controller) middlemanCallbackTTL() time.Duration {
	if self == nil || self.C == nil || self.C.MiddlemanCallbackTTLSeconds <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(self.C.MiddlemanCallbackTTLSeconds) * time.Second
}

func (self *Controller) middlemanCallbackTimeout() time.Duration {
	if self == nil || self.C == nil || self.C.MiddlemanCallbackTimeoutMS <= 0 {
		return time.Second
	}
	return time.Duration(self.C.MiddlemanCallbackTimeoutMS) * time.Millisecond
}

func (self *Controller) middlemanCallbackBaseURL() string {
	if self == nil || self.C == nil {
		return ""
	}
	base := self.C.MiddlemanCallbackBaseURL
	if base == "" {
		base = self.C.ServerURL
	}
	return strings.TrimRight(base, "/")
}

func newMiddlemanToken() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func addMiddlemanSignature(secret, path string, args url.Values) error {
	if secret == "" {
		return fmt.Errorf("tracking signature secret is not configured")
	}
	signable := middlemanSignableValues(path, args)
	args.Set(trackingSignatureParam, signTrackingValues(secret, path, signable))
	return nil
}

func validateMiddlemanSignature(secret, path string, args url.Values) error {
	if secret == "" {
		return fmt.Errorf("tracking signature secret is not configured")
	}
	got := args.Get(trackingSignatureParam)
	if got == "" {
		return fmt.Errorf("tracking signature missing")
	}
	signable := middlemanSignableValues(path, args)
	want := signTrackingValues(secret, path, signable)
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return fmt.Errorf("tracking signature malformed")
	}
	wantBytes, err := hex.DecodeString(want)
	if err != nil {
		return err
	}
	if !hmac.Equal(gotBytes, wantBytes) {
		return fmt.Errorf("tracking signature invalid")
	}
	return nil
}

func middlemanSignableValues(path string, args url.Values) url.Values {
	out := url.Values{}
	switch path {
	case "/mid/click":
		out.Set("rt", args.Get("rt"))
		out.Set("imp", args.Get("imp"))
	default:
		out.Set("t", args.Get("t"))
	}
	return out
}

func (self *Controller) middlemanProxyURL(path, token string) (string, error) {
	base := self.middlemanCallbackBaseURL()
	if base == "" {
		return "", fmt.Errorf("middleman callback base URL is not configured")
	}
	args := url.Values{}
	args.Set("t", token)
	args.Set("auction_price", `${AUCTION_PRICE}`)
	args.Set("auction_currency", `${AUCTION_CURRENCY}`)
	if err := addMiddlemanSignature(self.trackingSecret(), path, args); err != nil {
		return "", err
	}
	return base + path + "?" + encodeOpenRTBMacroQuery(args), nil
}

func (self *Controller) middlemanClickProxyURL(requestToken, impID string) (string, error) {
	base := self.middlemanCallbackBaseURL()
	if base == "" {
		return "", fmt.Errorf("middleman callback base URL is not configured")
	}
	args := url.Values{}
	args.Set("rt", requestToken)
	args.Set("imp", impID)
	if err := addMiddlemanSignature(self.trackingSecret(), "/mid/click", args); err != nil {
		return "", err
	}
	return base + "/mid/click?" + args.Encode(), nil
}

func encodeOpenRTBMacroQuery(args url.Values) string {
	encoded := args.Encode()
	for _, macro := range []string{
		`${AUCTION_ID}`,
		`${AUCTION_BID_ID}`,
		`${AUCTION_IMP_ID}`,
		`${AUCTION_SEAT_ID}`,
		`${AUCTION_AD_ID}`,
		`${AUCTION_PRICE}`,
		`${AUCTION_CURRENCY}`,
	} {
		encoded = strings.ReplaceAll(encoded, url.QueryEscape(macro), macro)
	}
	return encoded
}

func (self *Controller) prepareMiddlemanCallback(ctx context.Context, bid *openrtb2.BidRequest, selected *middlemanDownstreamBid) error {
	if bid == nil || selected == nil {
		return fmt.Errorf("middleman callback bid is missing")
	}
	token, err := newMiddlemanToken()
	if err != nil {
		return err
	}
	winURL, err := self.middlemanProxyURL("/mid/win", token)
	if err != nil {
		return err
	}
	lossURL, err := self.middlemanProxyURL("/mid/loss", token)
	if err != nil {
		return err
	}

	billOnWin := selected.DownstreamBURL == ""
	ctxValue := middlemanCallbackContext{
		Token:              token,
		Created:            time.Now(),
		RequestID:          bid.ID,
		ImpID:              selected.Bid.ImpID,
		ResponseBidID:      selected.ResponseBidID,
		DownstreamBidID:    selected.Bid.ID,
		DownstreamSeat:     selected.DownstreamSeat,
		DownstreamAdID:     selected.DownstreamAdID,
		DownstreamNURL:     selected.DownstreamNURL,
		DownstreamBURL:     selected.DownstreamBURL,
		DownstreamLURL:     selected.DownstreamLURL,
		DownstreamBidPrice: selected.DownstreamBidPrice,
		UpstreamBidPrice:   selected.UpstreamBidPrice,
		MarginCPM:          selected.UpstreamBidPrice - selected.DownstreamBidPrice,
		BillOnWin:          billOnWin,
		BidderID:           selected.Entry.BidderID,
		GroupID:            selected.Entry.GroupID,
		RouteBidderID:      selected.Entry.RouteBidderID,
		TargetID:           selected.Entry.TargetID,
		RAdv:               selected.Audit.One,
	}
	if selected.Audit.Attr != nil {
		ctxValue.RPub = selected.Audit.Attr.RPub
	}

	ttl := self.middlemanCallbackTTL()
	if err := self.callbackStore().SetCallback(ctx, token, ctxValue, ttl); err != nil {
		return err
	}
	if selected.ClickRequestToken != "" {
		if err := self.callbackStore().SetClick(ctx, selected.ClickRequestToken, selected.Bid.ImpID, token, ttl); err != nil {
			return err
		}
	}

	selected.Bid.NURL = winURL
	selected.Bid.LURL = lossURL
	if !billOnWin {
		burl, err := self.middlemanProxyURL("/mid/bill", token)
		if err != nil {
			return err
		}
		selected.Bid.BURL = burl
	} else {
		selected.Bid.BURL = ""
	}
	return nil
}

func (self *Controller) ServeMiddlemanCallback(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.URL.Path {
	case "/mid/win", "/mid/loss", "/mid/bill":
		err = self.serveMiddlemanStatus(r.Context(), r.URL.Path, r.URL.Query())
	case "/mid/click":
		err = self.serveMiddlemanClick(r.Context(), r.URL.Query())
	default:
		http.Error(w, "Invalid path", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (self *Controller) serveMiddlemanClick(ctx context.Context, args url.Values) error {
	if err := validateMiddlemanSignature(self.trackingSecret(), "/mid/click", args); err != nil {
		return err
	}
	requestToken := args.Get("rt")
	impID := args.Get("imp")
	if requestToken == "" || impID == "" {
		return fmt.Errorf("middleman click callback missing token or impression")
	}
	token, err := self.callbackStore().GetClick(ctx, requestToken, impID)
	if err != nil {
		return err
	}
	value, err := self.callbackStore().GetCallback(ctx, token)
	if err != nil {
		return err
	}
	prices := value.reconciledPrices("")
	return self.publishMiddlemanWinLoss(StatusTrackClk, value, "click", prices, "none", 0)
}

func (self *Controller) serveMiddlemanStatus(ctx context.Context, path string, args url.Values) error {
	if err := validateMiddlemanSignature(self.trackingSecret(), path, args); err != nil {
		return err
	}
	token := args.Get("t")
	if token == "" {
		return fmt.Errorf("middleman callback token is missing")
	}
	value, err := self.callbackStore().GetCallback(ctx, token)
	if err != nil {
		return err
	}
	prices := value.reconciledPrices(args.Get("auction_price"))

	switch path {
	case "/mid/win":
		status, code, err := self.forwardMiddlemanCallbackOnce(ctx, value, "win", value.DownstreamNURL, prices)
		if err != nil {
			return err
		}
		if err := self.publishMiddlemanWinLoss(StatusWin, value, "win", prices, status, code); err != nil {
			return err
		}
		if value.BillOnWin {
			if err := self.publishMiddlemanBillOnce(ctx, value, "win", prices, status, code); err != nil {
				return err
			}
		}
	case "/mid/bill":
		first, err := self.callbackStore().SetBillOnce(ctx, value.Token, self.middlemanCallbackTTL())
		if err != nil {
			return err
		}
		status, code := "duplicate", 0
		if first {
			status, code, err = self.forwardMiddlemanCallbackOnce(ctx, value, "bill", value.DownstreamBURL, prices)
			if err != nil {
				_ = self.callbackStore().ClearBill(ctx, value.Token)
				return err
			}
		}
		if first {
			if err := self.publishMiddlemanWinLoss(StatusTrackImp, value, "bill", prices, status, code); err != nil {
				_ = self.callbackStore().ClearBill(ctx, value.Token)
				return err
			}
		}
	case "/mid/loss":
		status, code, err := self.forwardMiddlemanCallbackOnce(ctx, value, "loss", value.DownstreamLURL, prices)
		if err != nil {
			return err
		}
		return self.publishMiddlemanWinLoss(StatusLoss, value, "loss", prices, status, code)
	default:
		return fmt.Errorf("invalid middleman callback path")
	}
	return nil
}

func (self *Controller) publishMiddlemanBillOnce(ctx context.Context, value middlemanCallbackContext, source string, prices middlemanPrices, forwardStatus string, forwardCode int) error {
	first, err := self.callbackStore().SetBillOnce(ctx, value.Token, self.middlemanCallbackTTL())
	if err != nil {
		return err
	}
	if !first {
		return nil
	}
	if err := self.publishMiddlemanWinLoss(StatusTrackImp, value, source, prices, forwardStatus, forwardCode); err != nil {
		_ = self.callbackStore().ClearBill(ctx, value.Token)
		return err
	}
	return nil
}

func (self *Controller) publishMiddlemanWinLoss(status Status, value middlemanCallbackContext, source string, prices middlemanPrices, forwardStatus string, forwardCode int) error {
	wl := &WinLoss{
		Current:      time.Now(),
		Status:       status,
		RPub:         value.RPub,
		RAdv:         value.RAdv,
		Seat:         value.DownstreamSeat,
		AuctionID:    value.RequestID,
		AuctionBidID: value.ResponseBidID,
		AuctionImpID: value.ImpID,
		AuctionAdID:  value.DownstreamAdID,
		Middleman: &MiddlemanWinLossMeta{
			BidderID:           value.BidderID,
			GroupID:            value.GroupID,
			RouteBidderID:      value.RouteBidderID,
			TargetID:           value.TargetID,
			Source:             source,
			ForwardStatus:      forwardStatus,
			ForwardHTTPStatus:  forwardCode,
			DownstreamBidPrice: value.DownstreamBidPrice,
			UpstreamBidPrice:   value.UpstreamBidPrice,
			ChargePrice:        prices.ChargePrice,
			PayPrice:           prices.PayPrice,
			MarginCPM:          value.MarginCPM,
			Currency:           prices.Currency,
		},
	}
	wl.RAdv.Cost = float32(prices.ChargePrice)
	wl.RAdv.CostType = 2
	data, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	return self.publishWinLoss(data)
}

func (value middlemanCallbackContext) reconciledPrices(rawAuctionPrice string) middlemanPrices {
	charge := value.UpstreamBidPrice
	if parsed, err := strconv.ParseFloat(rawAuctionPrice, 64); err == nil && parsed > 0 {
		charge = parsed
		if value.UpstreamBidPrice > 0 && charge > value.UpstreamBidPrice {
			charge = value.UpstreamBidPrice
		}
	}
	pay := charge - value.MarginCPM
	if pay < 0 {
		pay = 0
	}
	if pay > value.DownstreamBidPrice {
		pay = value.DownstreamBidPrice
	}
	return middlemanPrices{ChargePrice: charge, PayPrice: pay, Currency: "USD"}
}

func (self *Controller) forwardMiddlemanCallbackOnce(ctx context.Context, value middlemanCallbackContext, source, raw string, prices middlemanPrices) (string, int, error) {
	first, err := self.callbackStore().SetNotifyOnce(ctx, value.Token, source, self.middlemanCallbackTTL())
	if err != nil {
		return "", 0, err
	}
	if !first {
		return "duplicate", 0, nil
	}
	status, code := self.forwardMiddlemanCallback(ctx, raw, value, prices)
	if retryableMiddlemanForwardStatus(status) {
		_ = self.callbackStore().ClearNotify(ctx, value.Token, source)
	}
	return status, code, nil
}

func retryableMiddlemanForwardStatus(status string) bool {
	switch status {
	case "error", "http_error", "request_error":
		return true
	default:
		return false
	}
}

func (self *Controller) forwardMiddlemanCallback(ctx context.Context, raw string, value middlemanCallbackContext, prices middlemanPrices) (string, int) {
	if raw == "" {
		return "missing", 0
	}
	target, err := expandMiddlemanCallbackURL(raw, value, prices)
	if err != nil {
		return "invalid_url", 0
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, self.middlemanCallbackTimeout())
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, target, nil)
	if err != nil {
		return "request_error", 0
	}
	client := self.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "error", 0
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "http_error", resp.StatusCode
	}
	return "ok", resp.StatusCode
}

func expandMiddlemanCallbackURL(raw string, value middlemanCallbackContext, prices middlemanPrices) (string, error) {
	replacer := strings.NewReplacer(
		`${AUCTION_ID}`, value.RequestID,
		`${AUCTION_BID_ID}`, value.ResponseBidID,
		`${AUCTION_IMP_ID}`, value.ImpID,
		`${AUCTION_SEAT_ID}`, value.DownstreamSeat,
		`${AUCTION_AD_ID}`, value.DownstreamAdID,
		`${AUCTION_PRICE}`, fmt.Sprintf("%.3f", prices.PayPrice),
		`${AUCTION_CURRENCY}`, prices.Currency,
	)
	expanded := replacer.Replace(raw)
	u, err := url.Parse(expanded)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid callback scheme")
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid callback host")
	}
	return u.String(), nil
}

type memoryMiddlemanCallbackStore struct {
	callbacks map[string]middlemanCallbackContext
	clicks    map[string]string
	bills     map[string]struct{}
	notifies  map[string]struct{}
}

func newMemoryMiddlemanCallbackStore() *memoryMiddlemanCallbackStore {
	return &memoryMiddlemanCallbackStore{
		callbacks: make(map[string]middlemanCallbackContext),
		clicks:    make(map[string]string),
		bills:     make(map[string]struct{}),
		notifies:  make(map[string]struct{}),
	}
}

func (s *memoryMiddlemanCallbackStore) SetCallback(_ context.Context, token string, value middlemanCallbackContext, _ time.Duration) error {
	s.callbacks[token] = value
	return nil
}

func (s *memoryMiddlemanCallbackStore) GetCallback(_ context.Context, token string) (middlemanCallbackContext, error) {
	value, ok := s.callbacks[token]
	if !ok {
		return middlemanCallbackContext{}, fmt.Errorf("middleman callback token not found")
	}
	return value, nil
}

func (s *memoryMiddlemanCallbackStore) SetClick(_ context.Context, requestToken, impID, token string, _ time.Duration) error {
	s.clicks[middlemanClickKey(requestToken, impID)] = token
	return nil
}

func (s *memoryMiddlemanCallbackStore) GetClick(_ context.Context, requestToken, impID string) (string, error) {
	token, ok := s.clicks[middlemanClickKey(requestToken, impID)]
	if !ok {
		return "", fmt.Errorf("middleman click token not found")
	}
	return token, nil
}

func (s *memoryMiddlemanCallbackStore) SetBillOnce(_ context.Context, token string, _ time.Duration) (bool, error) {
	if _, ok := s.bills[token]; ok {
		return false, nil
	}
	s.bills[token] = struct{}{}
	return true, nil
}

func (s *memoryMiddlemanCallbackStore) ClearBill(_ context.Context, token string) error {
	delete(s.bills, token)
	return nil
}

func (s *memoryMiddlemanCallbackStore) SetNotifyOnce(_ context.Context, token, source string, _ time.Duration) (bool, error) {
	key := middlemanNotifyKey(token, source)
	if _, ok := s.notifies[key]; ok {
		return false, nil
	}
	s.notifies[key] = struct{}{}
	return true, nil
}

func (s *memoryMiddlemanCallbackStore) ClearNotify(_ context.Context, token, source string) error {
	delete(s.notifies, middlemanNotifyKey(token, source))
	return nil
}

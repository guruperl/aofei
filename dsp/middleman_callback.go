package dsp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/internal/jobs/midcallback"
	"github.com/guruperl/aofei/internal/safehttp"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

const (
	middlemanCallbackKeyPrefix = "middleman:cb:"
	middlemanClickKeyPrefix    = "middleman:click:"
	middlemanBillKeyPrefix     = "middleman:bill:"
	middlemanNotifyKeyPrefix   = "middleman:notify:"
	middlemanPublishKeyPrefix  = "middleman:publish:"
	middlemanProcessingPrefix  = "processing:"
	middlemanClaimDone         = "done"
	middlemanCleanupTimeout    = 2 * time.Second
)

type middlemanCallbackStore interface {
	SetCallback(context.Context, string, middlemanCallbackContext, time.Duration) error
	GetCallback(context.Context, string) (middlemanCallbackContext, error)
	SetClick(context.Context, string, string, string, time.Duration) error
	GetClick(context.Context, string, string) (string, error)
	SetBillOnce(context.Context, string, string, time.Duration) (bool, error)
	CompleteBill(context.Context, string, string, time.Duration) error
	ClearBill(context.Context, string, string) error
	SetNotifyOnce(context.Context, string, string, string, time.Duration) (bool, error)
	GetNotify(context.Context, string, string) (middlemanForwardState, error)
	CompleteNotify(context.Context, string, string, string, middlemanForwardState, time.Duration) error
	ClearNotify(context.Context, string, string, string) error
	SetPublishOnce(context.Context, string, string, string, time.Duration) (bool, error)
	CompletePublish(context.Context, string, string, string, time.Duration) error
	ClearPublish(context.Context, string, string, string) error
}

type redisMiddlemanCallbackStore struct {
	redis radix.Client
}

type middlemanCallbackContext struct {
	Token              string               `json:"token"`
	Created            time.Time            `json:"created"`
	RequestID          string               `json:"request_id"`
	ImpID              string               `json:"imp_id"`
	ResponseBidID      string               `json:"response_bid_id"`
	DownstreamBidID    string               `json:"downstream_bid_id"`
	DownstreamSeat     string               `json:"downstream_seat,omitempty"`
	DownstreamAdID     string               `json:"downstream_ad_id,omitempty"`
	DownstreamNURL     string               `json:"downstream_nurl,omitempty"`
	DownstreamBURL     string               `json:"downstream_burl,omitempty"`
	DownstreamLURL     string               `json:"downstream_lurl,omitempty"`
	DownstreamBidPrice float64              `json:"downstream_bid_price"`
	UpstreamBidPrice   float64              `json:"upstream_bid_price"`
	MarginCPM          float64              `json:"margin_cpm"`
	BillOnWin          bool                 `json:"bill_on_win"`
	BidderID           uint32               `json:"bidder_id,omitempty"`
	GroupID            uint32               `json:"group_id,omitempty"`
	RouteBidderID      uint32               `json:"route_bidder_id,omitempty"`
	TargetID           uint32               `json:"target_id,omitempty"`
	TriggerMode        string               `json:"trigger_mode,omitempty"`
	RPub               match.RPub           `json:"rpub,omitempty"`
	RAdv               match.RAdv           `json:"radv,omitempty"`
	Reporting          *ReportingDimensions `json:"reporting,omitempty"`
}

type middlemanPrices struct {
	ChargePrice float64
	PayPrice    float64
	Currency    string
}

type middlemanForwardState struct {
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Processing bool   `json:"-"`
}

var (
	errMiddlemanCallbackTokenNotFound = errors.New("middleman callback token not found")
	errMiddlemanClickTokenNotFound    = errors.New("middleman click token not found")
	errMiddlemanCallbackStateInvalid  = errors.New("middleman callback state is invalid")

	completeMiddlemanClaimScript = radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  redis.call("SET", KEYS[1], ARGV[2], "EX", ARGV[3])
  return 1
end
return 0`)
	releaseMiddlemanClaimScript = radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)
)

func middlemanProcessingValue(owner string) string {
	return middlemanProcessingPrefix + owner
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
		return middlemanCallbackContext{}, errMiddlemanCallbackTokenNotFound
	}
	var value middlemanCallbackContext
	if err := json.Unmarshal(data, &value); err != nil {
		return middlemanCallbackContext{}, fmt.Errorf("%w: %v", errMiddlemanCallbackStateInvalid, err)
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
		return "", errMiddlemanClickTokenNotFound
	}
	return token, nil
}

func (s redisMiddlemanCallbackStore) SetBillOnce(ctx context.Context, token, owner string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("middleman callback redis is not configured")
	}
	var result string
	err := s.redis.Do(ctx, radix.Cmd(&result, "SET", middlemanBillKey(token), middlemanProcessingValue(owner), "EX", strconv.Itoa(ttlSeconds(ttl)), "NX"))
	return result == "OK", err
}

func (s redisMiddlemanCallbackStore) CompleteBill(ctx context.Context, token, owner string, ttl time.Duration) error {
	return s.completeClaim(ctx, middlemanBillKey(token), owner, middlemanClaimDone, ttl)
}

func (s redisMiddlemanCallbackStore) ClearBill(ctx context.Context, token, owner string) error {
	return s.releaseClaim(ctx, middlemanBillKey(token), owner)
}

func (s redisMiddlemanCallbackStore) SetNotifyOnce(ctx context.Context, token, source, owner string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("middleman callback redis is not configured")
	}
	var result string
	err := s.redis.Do(ctx, radix.Cmd(&result, "SET", middlemanNotifyKey(token, source), middlemanProcessingValue(owner), "EX", strconv.Itoa(ttlSeconds(ttl)), "NX"))
	return result == "OK", err
}

func (s redisMiddlemanCallbackStore) GetNotify(ctx context.Context, token, source string) (middlemanForwardState, error) {
	if s.redis == nil {
		return middlemanForwardState{}, fmt.Errorf("middleman callback redis is not configured")
	}
	var data string
	if err := s.redis.Do(ctx, radix.Cmd(&data, "GET", middlemanNotifyKey(token, source))); err != nil {
		return middlemanForwardState{}, err
	}
	if strings.HasPrefix(data, middlemanProcessingPrefix) {
		return middlemanForwardState{Processing: true}, nil
	}
	if data == "" {
		return middlemanForwardState{}, fmt.Errorf("middleman notification state not found")
	}
	var state middlemanForwardState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return middlemanForwardState{}, err
	}
	if state.Status == "" {
		return middlemanForwardState{}, fmt.Errorf("middleman notification state is incomplete")
	}
	return state, nil
}

func (s redisMiddlemanCallbackStore) CompleteNotify(ctx context.Context, token, source, owner string, state middlemanForwardState, ttl time.Duration) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return s.completeClaim(ctx, middlemanNotifyKey(token, source), owner, string(data), ttl)
}

func (s redisMiddlemanCallbackStore) ClearNotify(ctx context.Context, token, source, owner string) error {
	return s.releaseClaim(ctx, middlemanNotifyKey(token, source), owner)
}

func (s redisMiddlemanCallbackStore) SetPublishOnce(ctx context.Context, token, source, owner string, ttl time.Duration) (bool, error) {
	if s.redis == nil {
		return false, fmt.Errorf("middleman callback redis is not configured")
	}
	var result string
	err := s.redis.Do(ctx, radix.Cmd(&result, "SET", middlemanPublishKey(token, source), middlemanProcessingValue(owner), "EX", strconv.Itoa(ttlSeconds(ttl)), "NX"))
	return result == "OK", err
}

func (s redisMiddlemanCallbackStore) CompletePublish(ctx context.Context, token, source, owner string, ttl time.Duration) error {
	return s.completeClaim(ctx, middlemanPublishKey(token, source), owner, middlemanClaimDone, ttl)
}

func (s redisMiddlemanCallbackStore) ClearPublish(ctx context.Context, token, source, owner string) error {
	return s.releaseClaim(ctx, middlemanPublishKey(token, source), owner)
}

func (s redisMiddlemanCallbackStore) completeClaim(ctx context.Context, key, owner, value string, ttl time.Duration) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	var completed int
	if err := s.redis.Do(ctx, completeMiddlemanClaimScript.Cmd(&completed, []string{key}, middlemanProcessingValue(owner), value, strconv.Itoa(ttlSeconds(ttl)))); err != nil {
		return err
	}
	if completed != 1 {
		return fmt.Errorf("middleman callback claim ownership lost before completion")
	}
	return nil
}

func (s redisMiddlemanCallbackStore) releaseClaim(ctx context.Context, key, owner string) error {
	if s.redis == nil {
		return fmt.Errorf("middleman callback redis is not configured")
	}
	var released int
	if err := s.redis.Do(ctx, releaseMiddlemanClaimScript.Cmd(&released, []string{key}, middlemanProcessingValue(owner))); err != nil {
		return err
	}
	if released != 1 {
		return fmt.Errorf("middleman callback claim ownership lost before release")
	}
	return nil
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

func middlemanPublishKey(token, source string) string {
	return middlemanPublishKeyPrefix + source + ":" + token
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
		return defaultTrackingSignatureTTL + maxTrackingSignatureFutureSkew
	}
	return time.Duration(self.C.MiddlemanCallbackTTLSeconds) * time.Second
}

func (self *Controller) middlemanCallbackTimeout() time.Duration {
	if self == nil || self.C == nil || self.C.MiddlemanCallbackTimeoutMS <= 0 {
		return time.Second
	}
	return time.Duration(self.C.MiddlemanCallbackTimeoutMS) * time.Millisecond
}

func (self *Controller) middlemanProcessingTTL() time.Duration {
	ttl := self.middlemanCallbackTimeout() + 5*time.Second
	if ttl < defaultTrackingProcessingTTL {
		return defaultTrackingProcessingTTL
	}
	return ttl
}

func (self *Controller) trackingSignatureTTL() time.Duration {
	if self == nil || self.C == nil || self.C.TrackingSignatureTTLSeconds <= 0 {
		return defaultTrackingSignatureTTL
	}
	return time.Duration(self.C.TrackingSignatureTTLSeconds) * time.Second
}

func (self *Controller) capStateTTL() time.Duration {
	if self == nil || self.C == nil || self.C.CapStateTTLSeconds <= 0 {
		return defaultCapStateTTL
	}
	return time.Duration(self.C.CapStateTTLSeconds) * time.Second
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
	args.Set(trackingSignatureTimestampParam, strconv.FormatInt(time.Now().Unix(), 10))
	signable := middlemanSignableValues(path, args)
	args.Set(trackingSignatureParam, signTrackingValues(secret, path, signable))
	return nil
}

func validateMiddlemanSignature(secret, path string, args url.Values, ttl time.Duration) error {
	if secret == "" {
		return fmt.Errorf("tracking signature secret is not configured")
	}
	got := args.Get(trackingSignatureParam)
	if got == "" {
		return fmt.Errorf("tracking signature missing")
	}
	if _, err := validateTrackingSignatureTimestamp(args, ttl); err != nil {
		return err
	}
	signable := middlemanSignableValues(path, args)
	want := signTrackingValues(secret, path, signable)
	if !equalHexSignature(got, want) {
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
	out.Set(trackingSignatureTimestampParam, args.Get(trackingSignatureTimestampParam))
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
		TriggerMode:        selected.Entry.TriggerMode,
		RAdv:               selected.Audit.One,
	}
	if selected.Audit.Attr != nil {
		ctxValue.RPub = selected.Audit.Attr.RPub
		ctxValue.Reporting = reportingDimensionsFromAttribute(selected.Audit.Attr)
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
		status := http.StatusBadRequest
		var retryableErr *retryableCallbackError
		if errors.As(err, &retryableErr) {
			status = http.StatusServiceUnavailable
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (self *Controller) serveMiddlemanClick(ctx context.Context, args url.Values) error {
	if err := validateMiddlemanSignature(self.trackingSecret(), "/mid/click", args, self.trackingSignatureTTL()); err != nil {
		return err
	}
	requestToken := args.Get("rt")
	impID := args.Get("imp")
	if requestToken == "" || impID == "" {
		return fmt.Errorf("middleman click callback missing token or impression")
	}
	token, err := self.callbackStore().GetClick(ctx, requestToken, impID)
	if err != nil {
		return middlemanLookupError(err)
	}
	value, err := self.callbackStore().GetCallback(ctx, token)
	if err != nil {
		return middlemanLookupError(err)
	}
	prices := value.reconciledPrices()
	return self.publishMiddlemanEventOnce(ctx, StatusTrackClk, value, "click", prices, "none", 0)
}

func (self *Controller) serveMiddlemanStatus(ctx context.Context, path string, args url.Values) error {
	if err := validateMiddlemanSignature(self.trackingSecret(), path, args, self.trackingSignatureTTL()); err != nil {
		return err
	}
	token := args.Get("t")
	if token == "" {
		return fmt.Errorf("middleman callback token is missing")
	}
	value, err := self.callbackStore().GetCallback(ctx, token)
	if err != nil {
		return middlemanLookupError(err)
	}
	prices := value.reconciledPrices()

	switch path {
	case "/mid/win":
		forward, ready, forwardErr := self.forwardMiddlemanCallbackOnce(ctx, value, "win", value.DownstreamNURL, prices)
		if forwardErr != nil && !isRetryableCallbackError(forwardErr) {
			return forwardErr
		}
		if !ready {
			return forwardErr
		}
		if err := self.publishMiddlemanEventOnce(ctx, StatusWin, value, "win", prices, forward.Status, forward.HTTPStatus); err != nil {
			return errors.Join(forwardErr, err)
		}
		if value.BillOnWin {
			if err := self.publishMiddlemanBillOnce(ctx, value, "win", prices, forward.Status, forward.HTTPStatus); err != nil {
				return errors.Join(forwardErr, err)
			}
		}
		return forwardErr
	case "/mid/bill":
		forward, ready, forwardErr := self.forwardMiddlemanCallbackOnce(ctx, value, "bill", value.DownstreamBURL, prices)
		if forwardErr != nil && !isRetryableCallbackError(forwardErr) {
			return forwardErr
		}
		if !ready {
			return forwardErr
		}
		return errors.Join(forwardErr, self.publishMiddlemanBillOnce(ctx, value, "bill", prices, forward.Status, forward.HTTPStatus))
	case "/mid/loss":
		forward, ready, forwardErr := self.forwardMiddlemanCallbackOnce(ctx, value, "loss", value.DownstreamLURL, prices)
		if forwardErr != nil && !isRetryableCallbackError(forwardErr) {
			return forwardErr
		}
		if !ready {
			return forwardErr
		}
		return errors.Join(forwardErr, self.publishMiddlemanEventOnce(ctx, StatusLoss, value, "loss", prices, forward.Status, forward.HTTPStatus))
	default:
		return fmt.Errorf("invalid middleman callback path")
	}
}

func isRetryableCallbackError(err error) bool {
	var retryableErr *retryableCallbackError
	return errors.As(err, &retryableErr)
}

func middlemanLookupError(err error) error {
	if err == nil || errors.Is(err, errMiddlemanCallbackTokenNotFound) || errors.Is(err, errMiddlemanClickTokenNotFound) || errors.Is(err, errMiddlemanCallbackStateInvalid) {
		return err
	}
	return retryableCallback(err)
}

func (self *Controller) publishMiddlemanEventOnce(ctx context.Context, status Status, value middlemanCallbackContext, source string, prices middlemanPrices, forwardStatus string, forwardCode int) error {
	owner, err := newMiddlemanToken()
	if err != nil {
		return retryableCallback(err)
	}
	store := self.callbackStore()
	first, err := store.SetPublishOnce(ctx, value.Token, source, owner, self.middlemanProcessingTTL())
	if err != nil {
		return retryableCallback(err)
	}
	if !first {
		recordMiddlemanCallbackOutcome("publish_duplicate")
		return nil
	}
	if err := self.publishMiddlemanWinLoss(status, value, source, prices, forwardStatus, forwardCode); err != nil {
		recordMiddlemanCallbackOutcome("local_publish_retryable")
		if clearErr := self.clearMiddlemanPublishClaim(ctx, value.Token, source, owner); clearErr != nil {
			return retryableCallback(errors.Join(err, clearErr))
		}
		return retryableCallback(err)
	}
	stateCtx, cancel := middlemanStateContext(ctx)
	defer cancel()
	if err := store.CompletePublish(stateCtx, value.Token, source, owner, self.middlemanCallbackTTL()); err != nil {
		recordMiddlemanCallbackOutcome("claim_completion_error")
		return retryableCallback(err)
	}
	return nil
}

func (self *Controller) publishMiddlemanBillOnce(ctx context.Context, value middlemanCallbackContext, source string, prices middlemanPrices, forwardStatus string, forwardCode int) error {
	owner, err := newMiddlemanToken()
	if err != nil {
		return retryableCallback(err)
	}
	store := self.callbackStore()
	first, err := store.SetBillOnce(ctx, value.Token, owner, self.middlemanProcessingTTL())
	if err != nil {
		return retryableCallback(err)
	}
	if !first {
		recordMiddlemanCallbackOutcome("bill_duplicate")
		return nil
	}
	if err := self.publishMiddlemanWinLoss(StatusTrackImp, value, source, prices, forwardStatus, forwardCode); err != nil {
		recordMiddlemanCallbackOutcome("local_publish_retryable")
		if clearErr := self.clearMiddlemanBillClaim(ctx, value.Token, owner); clearErr != nil {
			return retryableCallback(errors.Join(err, clearErr))
		}
		return retryableCallback(err)
	}
	stateCtx, cancel := middlemanStateContext(ctx)
	defer cancel()
	if err := store.CompleteBill(stateCtx, value.Token, owner, self.middlemanCallbackTTL()); err != nil {
		recordMiddlemanCallbackOutcome("claim_completion_error")
		return retryableCallback(err)
	}
	return nil
}

func (self *Controller) publishMiddlemanWinLoss(status Status, value middlemanCallbackContext, source string, prices middlemanPrices, forwardStatus string, forwardCode int) error {
	chargeCPM, err := protocolCPM(prices.ChargePrice)
	if err != nil {
		return err
	}
	payCPM, err := protocolCPM(prices.PayPrice)
	if err != nil {
		return err
	}
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
		Reporting:    value.Reporting,
		Middleman: &MiddlemanWinLossMeta{
			BidderID:           value.BidderID,
			GroupID:            value.GroupID,
			RouteBidderID:      value.RouteBidderID,
			TargetID:           value.TargetID,
			TriggerMode:        value.TriggerMode,
			Source:             source,
			ForwardStatus:      forwardStatus,
			ForwardHTTPStatus:  forwardCode,
			DownstreamBidPrice: value.DownstreamBidPrice,
			UpstreamBidPrice:   value.UpstreamBidPrice,
			ChargePrice:        prices.ChargePrice,
			PayPrice:           prices.PayPrice,
			ChargeCPM:          chargeCPM,
			PayCPM:             payCPM,
			MarginCPM:          value.MarginCPM,
			Currency:           prices.Currency,
		},
	}
	wl.RAdv.Cost = float32(prices.ChargePrice)
	wl.RAdv.CostCPM = chargeCPM
	wl.RAdv.CostType = match.CostTypeCPM
	data, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	if err := self.publishWinLoss(data); err != nil {
		return err
	}
	self.recordAttributionTouch(wl)
	return nil
}

func protocolCPM(value float64) (accounting.CPM, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, fmt.Errorf("invalid protocol USD CPM %v", value)
	}
	return accounting.ParseCPM(strconv.FormatFloat(value, 'f', 6, 64))
}

func (value middlemanCallbackContext) reconciledPrices() middlemanPrices {
	charge := value.UpstreamBidPrice
	if charge < 0 {
		charge = 0
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

func (self *Controller) forwardMiddlemanCallbackOnce(ctx context.Context, value middlemanCallbackContext, source, raw string, prices middlemanPrices) (middlemanForwardState, bool, error) {
	store := self.callbackStore()
	owner, err := newMiddlemanToken()
	if err != nil {
		return middlemanForwardState{}, false, retryableCallback(err)
	}
	first, err := store.SetNotifyOnce(ctx, value.Token, source, owner, self.middlemanProcessingTTL())
	if err != nil {
		return middlemanForwardState{}, false, retryableCallback(err)
	}
	if !first {
		state, err := store.GetNotify(ctx, value.Token, source)
		if err != nil {
			return middlemanForwardState{}, false, retryableCallback(err)
		}
		if state.Processing {
			recordMiddlemanCallbackOutcome("forward_inflight_duplicate")
		} else {
			recordMiddlemanCallbackOutcome("forward_completed_duplicate")
		}
		return state, !state.Processing, nil
	}
	target, status, code := self.forwardMiddlemanCallback(ctx, raw, value, prices)
	state := middlemanForwardState{Status: status, HTTPStatus: code}
	persist := true
	if midcallback.RetryableForward(status, code) {
		queued, err := self.enqueueMiddlemanCallbackRetry(ctx, value, source, target, prices, status, code)
		if err != nil {
			recordMiddlemanCallbackOutcome("retry_enqueue_error")
			if self.Logger != nil {
				self.Logger.Warn("middleman callback retry enqueue failed",
					zap.String("source", source),
					zap.String("reason", "retry_dependency"),
				)
			}
		}
		if queued {
			recordMiddlemanCallbackOutcome("retry_queued")
		}
		if !queued {
			recordMiddlemanCallbackOutcome("retry_unavailable")
			persist = false
			if err := self.clearMiddlemanNotifyClaim(ctx, value.Token, source, owner); err != nil {
				return middlemanForwardState{}, false, retryableCallback(err)
			}
			return state, true, retryableCallback(fmt.Errorf("middleman callback retry is unavailable"))
		}
	}
	if persist {
		stateCtx, cancel := middlemanStateContext(ctx)
		err := store.CompleteNotify(stateCtx, value.Token, source, owner, state, self.middlemanCallbackTTL())
		cancel()
		if err != nil {
			recordMiddlemanCallbackOutcome("claim_completion_error")
			if clearErr := self.clearMiddlemanNotifyClaim(ctx, value.Token, source, owner); clearErr != nil {
				return middlemanForwardState{}, false, retryableCallback(errors.Join(err, clearErr))
			}
			return middlemanForwardState{}, false, retryableCallback(err)
		}
	}
	return state, true, nil
}

func (self *Controller) clearMiddlemanNotifyClaim(ctx context.Context, token, source, owner string) error {
	stateCtx, cancel := middlemanStateContext(ctx)
	defer cancel()
	err := self.callbackStore().ClearNotify(stateCtx, token, source, owner)
	recordMiddlemanClaimRelease(err)
	return err
}

func (self *Controller) clearMiddlemanPublishClaim(ctx context.Context, token, source, owner string) error {
	stateCtx, cancel := middlemanStateContext(ctx)
	defer cancel()
	err := self.callbackStore().ClearPublish(stateCtx, token, source, owner)
	recordMiddlemanClaimRelease(err)
	return err
}

func (self *Controller) clearMiddlemanBillClaim(ctx context.Context, token, owner string) error {
	stateCtx, cancel := middlemanStateContext(ctx)
	defer cancel()
	err := self.callbackStore().ClearBill(stateCtx, token, owner)
	recordMiddlemanClaimRelease(err)
	return err
}

func middlemanStateContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), middlemanCleanupTimeout)
}

func recordMiddlemanClaimRelease(err error) {
	if err != nil {
		recordMiddlemanCallbackOutcome("claim_release_error")
		return
	}
	recordMiddlemanCallbackOutcome("claim_released")
}

func (self *Controller) enqueueMiddlemanCallbackRetry(ctx context.Context, value middlemanCallbackContext, source, target string, prices middlemanPrices, status string, code int) (bool, error) {
	if self == nil || self.DB == nil || target == "" {
		return false, nil
	}
	err := midcallback.Enqueue(ctx, self.DB, midcallback.Failure{
		Token:          value.Token,
		Source:         source,
		CallbackURL:    target,
		BidderID:       value.BidderID,
		GroupID:        value.GroupID,
		RouteBidderID:  value.RouteBidderID,
		TargetID:       value.TargetID,
		AuctionID:      value.RequestID,
		ImpID:          value.ImpID,
		AuctionBidID:   value.ResponseBidID,
		ChargePrice:    prices.ChargePrice,
		PayPrice:       prices.PayPrice,
		Currency:       prices.Currency,
		HTTPStatus:     code,
		ForwardOutcome: status,
	})
	return err == nil, err
}

func (self *Controller) forwardMiddlemanCallback(ctx context.Context, raw string, value middlemanCallbackContext, prices middlemanPrices) (string, string, int) {
	if raw == "" {
		metricMiddlemanForwardErrors.Add(1)
		return "", "missing", 0
	}
	target, err := expandMiddlemanCallbackURL(raw, value, prices)
	if err != nil {
		metricMiddlemanForwardErrors.Add(1)
		return "", "invalid_url", 0
	}
	if err := self.validateMiddlemanCallbackTarget(ctx, target); err != nil {
		metricMiddlemanForwardErrors.Add(1)
		return target, "invalid_url", 0
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, self.middlemanCallbackTimeout())
	defer cancel()
	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodGet, target, nil)
	if err != nil {
		metricMiddlemanForwardErrors.Add(1)
		return target, "request_error", 0
	}
	client := self.client
	client = safehttp.NewCallbackClient(client)
	resp, err := client.Do(req)
	if err != nil {
		metricMiddlemanForwardErrors.Add(1)
		return target, "error", 0
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		metricMiddlemanForwardErrors.Add(1)
		return target, "http_error", resp.StatusCode
	}
	metricMiddlemanForwardOK.Add(1)
	return target, "ok", resp.StatusCode
}

func (self *Controller) validateMiddlemanCallbackTarget(ctx context.Context, target string) error {
	if err := safehttp.ValidateCallbackURL(ctx, target); err != nil {
		return err
	}
	if self != nil && self.callbackURLGuard != nil {
		return self.callbackURLGuard(ctx, target)
	}
	return nil
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
	mu        sync.Mutex
	callbacks map[string]middlemanCallbackContext
	clicks    map[string]string
	bills     map[string]string
	notifies  map[string]memoryMiddlemanNotify
	publishes map[string]string
}

type memoryMiddlemanNotify struct {
	owner string
	state middlemanForwardState
}

func newMemoryMiddlemanCallbackStore() *memoryMiddlemanCallbackStore {
	return &memoryMiddlemanCallbackStore{
		callbacks: make(map[string]middlemanCallbackContext),
		clicks:    make(map[string]string),
		bills:     make(map[string]string),
		notifies:  make(map[string]memoryMiddlemanNotify),
		publishes: make(map[string]string),
	}
}

func (s *memoryMiddlemanCallbackStore) SetCallback(_ context.Context, token string, value middlemanCallbackContext, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks[token] = value
	return nil
}

func (s *memoryMiddlemanCallbackStore) GetCallback(_ context.Context, token string) (middlemanCallbackContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.callbacks[token]
	if !ok {
		return middlemanCallbackContext{}, errMiddlemanCallbackTokenNotFound
	}
	return value, nil
}

func (s *memoryMiddlemanCallbackStore) SetClick(_ context.Context, requestToken, impID, token string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clicks[middlemanClickKey(requestToken, impID)] = token
	return nil
}

func (s *memoryMiddlemanCallbackStore) GetClick(_ context.Context, requestToken, impID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.clicks[middlemanClickKey(requestToken, impID)]
	if !ok {
		return "", errMiddlemanClickTokenNotFound
	}
	return token, nil
}

func (s *memoryMiddlemanCallbackStore) SetBillOnce(_ context.Context, token, owner string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bills[token]; ok {
		return false, nil
	}
	s.bills[token] = middlemanProcessingValue(owner)
	return true, nil
}

func (s *memoryMiddlemanCallbackStore) CompleteBill(_ context.Context, token, owner string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bills[token] != middlemanProcessingValue(owner) {
		return fmt.Errorf("middleman callback claim ownership lost before completion")
	}
	s.bills[token] = middlemanClaimDone
	return nil
}

func (s *memoryMiddlemanCallbackStore) ClearBill(_ context.Context, token, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bills[token] != middlemanProcessingValue(owner) {
		return fmt.Errorf("middleman callback claim ownership lost before release")
	}
	delete(s.bills, token)
	return nil
}

func (s *memoryMiddlemanCallbackStore) SetNotifyOnce(_ context.Context, token, source, owner string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := middlemanNotifyKey(token, source)
	if _, ok := s.notifies[key]; ok {
		return false, nil
	}
	s.notifies[key] = memoryMiddlemanNotify{owner: owner, state: middlemanForwardState{Processing: true}}
	return true, nil
}

func (s *memoryMiddlemanCallbackStore) GetNotify(_ context.Context, token, source string) (middlemanForwardState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.notifies[middlemanNotifyKey(token, source)]
	if !ok {
		return middlemanForwardState{}, fmt.Errorf("middleman notification state not found")
	}
	return entry.state, nil
}

func (s *memoryMiddlemanCallbackStore) CompleteNotify(_ context.Context, token, source, owner string, state middlemanForwardState, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := middlemanNotifyKey(token, source)
	entry, ok := s.notifies[key]
	if !ok || entry.owner != owner || !entry.state.Processing {
		return fmt.Errorf("middleman callback claim ownership lost before completion")
	}
	state.Processing = false
	s.notifies[key] = memoryMiddlemanNotify{state: state}
	return nil
}

func (s *memoryMiddlemanCallbackStore) ClearNotify(_ context.Context, token, source, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := middlemanNotifyKey(token, source)
	entry, ok := s.notifies[key]
	if !ok || entry.owner != owner || !entry.state.Processing {
		return fmt.Errorf("middleman callback claim ownership lost before release")
	}
	delete(s.notifies, key)
	return nil
}

func (s *memoryMiddlemanCallbackStore) SetPublishOnce(_ context.Context, token, source, owner string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := middlemanPublishKey(token, source)
	if _, ok := s.publishes[key]; ok {
		return false, nil
	}
	s.publishes[key] = middlemanProcessingValue(owner)
	return true, nil
}

func (s *memoryMiddlemanCallbackStore) CompletePublish(_ context.Context, token, source, owner string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := middlemanPublishKey(token, source)
	if s.publishes[key] != middlemanProcessingValue(owner) {
		return fmt.Errorf("middleman callback claim ownership lost before completion")
	}
	s.publishes[key] = middlemanClaimDone
	return nil
}

func (s *memoryMiddlemanCallbackStore) ClearPublish(_ context.Context, token, source, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := middlemanPublishKey(token, source)
	if s.publishes[key] != middlemanProcessingValue(owner) {
		return fmt.Errorf("middleman callback claim ownership lost before release")
	}
	delete(s.publishes, key)
	return nil
}

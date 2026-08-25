package dsp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/publisherauth"
)

const (
	sspUserCookieName = "aofei_pz_uid"
)

var legacyDirectSSPTokenCodec = acl.NewLegacyDirectTokenCodec()

var errSSPPublisherCacheUnavailable = errors.New("direct publisher cache is unavailable")

// ServeSSP handles the direct publisher JSON contract on /pz.
func (self *Controller) ServeSSP(w http.ResponseWriter, r *http.Request) {
	metricSSPRequests.Add(1)
	current := time.Now()
	ctx := r.Context()

	rawRequest, err := io.ReadAll(io.LimitReader(r.Body, maxBidRequestBodyBytes+1))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	if len(rawRequest) > maxBidRequestBodyBytes {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	sspReq, err := ParseSSPRequest(rawRequest)
	if err != nil {
		metricSSPMalformed.Add(1)
		writeSSPError(w, http.StatusBadRequest)
		return
	}
	responseFormat, err := sspReq.NormalizedResponseFormat()
	if err != nil {
		metricSSPMalformed.Add(1)
		writeSSPError(w, http.StatusBadRequest)
		return
	}
	var requestProof *publisherauth.RequestProof
	publisherAuthOutcome := ""
	if sspPlatformIsSDK(sspReq.Platform) {
		publisherAuthOutcome = "compatibility"
		defer func() { recordSSPPublisherAuthOutcome(publisherAuthOutcome) }()
	}
	if sspPlatformIsSDK(sspReq.Platform) && self.directSSPAuthRequired() {
		publisherAuthOutcome = "dependency_error"
		if self.publisherAuth == nil {
			writeSSPPublisherAuthError(w, publisherauth.ErrUnavailable)
			return
		}
		requestProof, err = self.publisherAuth.VerifyRequest(r, rawRequest)
		if err != nil {
			publisherAuthOutcome = sspPublisherAuthErrorOutcome(err)
			writeSSPPublisherAuthError(w, err)
			return
		}
		publisherAuthOutcome = "inventory_rejected"
	}

	pub, units, tokenVersion, err := self.validateSSPSupplyWithVersion(ctx, sspReq)
	recordSSPInventoryTokenOutcome(sspInventoryTokenOutcome(tokenVersion, err))
	if err != nil {
		metricSSPValidationErrors.Add(1)
		status := http.StatusBadRequest
		if errors.Is(err, errSSPPublisherCacheUnavailable) {
			status = http.StatusServiceUnavailable
		}
		writeSSPError(w, status)
		return
	}
	if requestProof != nil {
		publisherAuthOutcome = "scope_rejected"
		unit := units[0].RPub
		if err := requestProof.AuthorizeScope(uint64(unit.PubID), uint64(unit.SiteID)); err != nil {
			writeSSPPublisherAuthError(w, err)
			return
		}
		publisherAuthOutcome = "policy_rejected"
	}
	if err := validateSSPRequestPolicy(r, sspReq, units); err != nil {
		metricSSPPolicyRejections.Add(1)
		writeSSPError(w, http.StatusForbidden)
		return
	}
	if requestProof != nil {
		if err := self.publisherAuth.ClaimReplay(ctx, requestProof); err != nil {
			publisherAuthOutcome = sspPublisherAuthErrorOutcome(err)
			writeSSPPublisherAuthError(w, err)
			return
		}
		publisherAuthOutcome = "accepted"
	}
	privacy := self.privacyDecision(r, sspReq.Regs, sspReq.User, sspReq.Device)
	privacy = sspClientClaimPrivacy(sspReq.Platform, requestProof != nil, privacy)
	recordPrivacyDecision(privacy)

	cookieUserID := ""
	if privacy.AllowCookie && sspPlatformUsesBrowserCookie(sspReq.Platform) {
		cookieUserID = readSSPUserCookie(r)
	}
	bid, err := self.openRTBFromValidatedSSP(r, sspReq, pub, units, cookieUserID)
	if err != nil {
		metricSSPValidationErrors.Add(1)
		writeSSPError(w, http.StatusBadRequest)
		return
	}
	if err := self.applyPrivacyPolicy(bid, privacy); err != nil {
		metricSSPValidationErrors.Add(1)
		http.Error(w, "privacy policy could not be applied", http.StatusBadRequest)
		return
	}
	if privacy.AllowCookie && sspPlatformUsesBrowserCookie(sspReq.Platform) && cookieUserID == "" {
		self.setSSPUserCookie(w, r)
	}

	rawMiddlemanRequest, err := json.Marshal(bid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	finalWinners, localWinners := self.auctionBidWinners(ctx, bid, pub.Pub, current, pub.Domain, rawMiddlemanRequest, nil, privacy)
	_, _, audits, _, materialized := self.materializeBidWinners(ctx, bid, finalWinners, localWinners, nil)
	recordWinnerSourceLatency(current, materialized)
	results := sspBidResultsFromMaterialized(bid, materialized)
	for _, result := range results {
		if !result.Filled {
			metricSSPNoFillAdUnits.Add(1)
		} else {
			metricSSPFilledAdUnits.Add(1)
		}
	}

	rawResponse, err := renderSSPResponse(responseFormat, bid, results)
	if err != nil {
		self.releaseMaterializedDeliveries(ctx, materialized)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if written, writeErr := w.Write(rawResponse); writeErr != nil || written != len(rawResponse) {
		self.releaseMaterializedDeliveries(ctx, materialized)
		return
	}

	elapsed := time.Since(current)
	for i := range audits {
		audits[i].Elapsed = elapsed
		audits[i].PrivacyMode = string(privacy.Mode)
		audits[i].PrivacyReason = privacy.Reason
	}
	privacyAuditRequest, err := json.Marshal(sspReq)
	if err == nil {
		privacyAuditResponse, auditErr := privacySSPAuditResponse(results)
		if auditErr == nil {
			_ = self.publishSSPBidAuditsWithPrivacy(privacyAuditRequest, privacyAuditResponse, audits, privacy)
		}
	}
}

// sspClientClaimPrivacy prevents an unauthenticated SDK compatibility request
// from upgrading its own identity, geo, demographic, or uploaded-audience
// claims into personalized targeting. Restrictive signals are evaluated first
// and keep their more specific reason; only an otherwise-personalized decision
// is downgraded. A valid publisher/App proof authenticates the asserting
// publisher, not the truth of an individual device claim.
func sspClientClaimPrivacy(platform string, authenticated bool, decision privacyDecision) privacyDecision {
	decision = decision.normalized()
	if !sspPlatformIsSDK(platform) || authenticated || decision.Mode != privacyModePersonalized {
		return decision
	}
	decision.Mode = privacyModeContextual
	decision.Reason = "sdk_unauthenticated"
	decision.AllowCookie = false
	decision.AllowIdentity = false
	return decision
}

func writeSSPPublisherAuthError(w http.ResponseWriter, err error) {
	metricSSPValidationErrors.Add(1)
	status := http.StatusUnauthorized
	if errors.Is(err, publisherauth.ErrUnavailable) {
		status = http.StatusServiceUnavailable
	}
	writeSSPError(w, status)
}

func sspPublisherAuthErrorOutcome(err error) string {
	switch {
	case errors.Is(err, publisherauth.ErrRequired):
		return "required_rejected"
	case errors.Is(err, publisherauth.ErrStale):
		return "stale_rejected"
	case errors.Is(err, publisherauth.ErrScope):
		return "scope_rejected"
	case errors.Is(err, publisherauth.ErrReplay):
		return "replay_rejected"
	case errors.Is(err, publisherauth.ErrUnavailable):
		return "dependency_error"
	default:
		return "invalid_rejected"
	}
}

// writeSSPError keeps the public pre-auction rejection surface stable without
// reflecting cached publisher/App identity, inventory ids, hostnames, or
// credential state. Detailed validation errors stay inside the process and
// fixed-cardinality metrics identify the failing boundary.
func writeSSPError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (self *Controller) openRTBFromSSP(ctx context.Context, r *http.Request, req *SSPRequest, cookieUserID ...string) (*openrtb2.BidRequest, *acl.DirectPub, []SSPValidatedUnit, error) {
	pub, units, err := self.validateSSPSupply(ctx, req)
	if err != nil {
		return nil, nil, nil, err
	}
	userID := ""
	if len(cookieUserID) != 0 && sspPlatformUsesBrowserCookie(req.Platform) {
		userID = cookieUserID[0]
	}
	bid, err := self.openRTBFromValidatedSSP(r, req, pub, units, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	return bid, pub, units, nil
}

func (self *Controller) validateSSPSupply(ctx context.Context, req *SSPRequest) (*acl.DirectPub, []SSPValidatedUnit, error) {
	pub, units, _, err := self.validateSSPSupplyWithVersion(ctx, req)
	return pub, units, err
}

func (self *Controller) validateSSPSupplyWithVersion(ctx context.Context, req *SSPRequest) (*acl.DirectPub, []SSPValidatedUnit, acl.DirectTokenVersion, error) {
	if req == nil {
		return nil, nil, acl.DirectTokenUnknown, fmt.Errorf("ssp request is nil")
	}
	if req.Site == "" {
		return nil, nil, acl.DirectTokenUnknown, fmt.Errorf("ssp request missing site")
	}
	codec := self.directSSPInventoryTokenCodec()
	pubID, siteID, version, err := codec.UnpackSite(string(req.Site))
	if err != nil {
		return nil, nil, version, fmt.Errorf("invalid site token: %w", err)
	}
	pub, err := self.sspPubByID(ctx, pubID)
	if err != nil {
		return nil, nil, version, err
	}
	units, validatedVersion, err := req.validateSupplyWithTokens(pub, codec, pubID, siteID, version)
	if err != nil {
		return nil, nil, validatedVersion, err
	}
	return pub, units, validatedVersion, nil
}

func (self *Controller) directSSPInventoryTokenCodec() *acl.DirectTokenCodec {
	if self != nil && self.directTokens != nil {
		return self.directTokens
	}
	// A manually embedded controller must not silently bypass an enabled v2
	// policy when its codec was never initialized. A nil codec rejects both v2
	// and legacy tokens; ordinary zero-config test embeddings retain v1.
	if self != nil && self.C != nil && self.C.DirectSSPTokens.Enabled {
		return nil
	}
	return legacyDirectSSPTokenCodec
}

func sspInventoryTokenOutcome(version acl.DirectTokenVersion, err error) string {
	if err == nil {
		return version.String() + "_accepted"
	}
	if errors.Is(err, acl.ErrLegacyDirectTokenDisabled) {
		return "legacy_disabled"
	}
	if errors.Is(err, errSSPMixedDirectTokenVersions) {
		return "mixed_rejected"
	}
	return version.String() + "_rejected"
}

func (self *Controller) openRTBFromValidatedSSP(r *http.Request, req *SSPRequest, pub *acl.DirectPub, units []SSPValidatedUnit, cookieUserID string) (*openrtb2.BidRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("ssp request is nil")
	}
	if pub == nil {
		return nil, fmt.Errorf("publisher is nil")
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("ssp request has no validated ad units")
	}
	if !sspPlatformUsesBrowserCookie(req.Platform) {
		cookieUserID = ""
	}
	bid := &openrtb2.BidRequest{
		ID:     req.ID,
		Device: self.deviceFromSSPHeaders(r),
		User:   &openrtb2.User{},
		Regs:   req.Regs,
		Source: sourceFromApprovedSeller(pub.Pub.Seller),
		Imp:    make([]openrtb2.Imp, 0, len(units)),
	}
	if sspPlatformIsSDK(req.Platform) {
		app, err := appFromSSP(req.App, pub, units[0])
		if err != nil {
			return nil, err
		}
		bid.App = app
		bid.Device = self.deviceFromSSP(r, req.Device)
		if req.User != nil {
			user := *req.User
			bid.User = &user
		}
	} else {
		bid.Site = siteFromSSP(r, pub, units[0])
		if req.User != nil {
			bid.User.Consent = req.User.Consent
		}
	}
	if cookieUserID != "" && !sspPlatformIsSDK(req.Platform) {
		bid.User.ID = cookieUserID
		bid.User.BuyerUID = cookieUserID
	}
	if bid.ID == "" {
		bid.ID = "ssp-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	for i, unit := range units {
		imp, err := openRTBImpFromSSPUnit(req.AdUnits[i], unit, i)
		if err != nil {
			return nil, fmt.Errorf("adUnits[%d] %w", i, err)
		}
		bid.Imp = append(bid.Imp, imp)
	}
	return bid, nil
}

type sspBidResult struct {
	Filled        bool
	Bid           openrtb2.Bid
	ResponseBidID string
	Seat          string
	ImpressionURL string
	ClickURL      string
}

type sspJSONAdUnitResponse struct {
	Filled        bool            `json:"filled"`
	AdM           string          `json:"adm,omitempty"`
	Native        json.RawMessage `json:"native,omitempty"`
	ImpressionURL string          `json:"impressionUrl,omitempty"`
	ClickURL      string          `json:"clickUrl,omitempty"`
	Price         float64         `json:"price,omitempty"`
	Currency      string          `json:"currency,omitempty"`
	AdID          string          `json:"adId,omitempty"`
	CampaignID    string          `json:"campaignId,omitempty"`
	CreativeID    string          `json:"creativeId,omitempty"`
	Width         int64           `json:"width,omitempty"`
	Height        int64           `json:"height,omitempty"`
}

func sspBidResultsFromMaterialized(bid *openrtb2.BidRequest, winners []bidWinner) []sspBidResult {
	n := 0
	if bid != nil {
		n = len(bid.Imp)
	}
	results := make([]sspBidResult, n)
	for _, winner := range winners {
		if winner.ImpIndex < 0 || winner.ImpIndex >= len(results) {
			continue
		}
		result := sspBidResult{
			Filled:        true,
			Bid:           winner.Bid,
			ResponseBidID: winner.ResponseBidID,
			Seat:          winner.Seat,
		}
		if winner.Local {
			result.ImpressionURL = winner.ImpressionURL
			result.ClickURL = winner.ClickURL
		}
		results[winner.ImpIndex] = result
	}
	return results
}

func clickURLForSSPBid(dspBid *DSP, winloss *WinLoss) string {
	if dspBid == nil || dspBid.creative == nil || winloss == nil {
		return ""
	}
	landing, err := dspBid.creative.LandingURL(winloss.Macro(), dspBid.Macro())
	if err != nil {
		return ""
	}
	return winloss.ClkRedirectURL(landing)
}

func renderSSPResponse(format string, bid *openrtb2.BidRequest, results []sspBidResult) ([]byte, error) {
	switch format {
	case "json":
		return json.Marshal(sspJSONResponse(results))
	case "openrtb":
		return json.Marshal(sspOpenRTBResponse(bid, results))
	default:
		return json.Marshal(sspHTMLResponse(results))
	}
}

func sspHTMLResponse(results []sspBidResult) []string {
	html := make([]string, len(results))
	for i, result := range results {
		if result.Filled {
			html[i] = result.Bid.AdM
		}
	}
	return html
}

func sspJSONResponse(results []sspBidResult) []sspJSONAdUnitResponse {
	response := make([]sspJSONAdUnitResponse, len(results))
	for i, result := range results {
		if !result.Filled {
			response[i] = sspJSONAdUnitResponse{Filled: false}
			continue
		}
		response[i] = sspJSONAdUnitResponse{
			Filled:        true,
			AdM:           result.Bid.AdM,
			ImpressionURL: result.ImpressionURL,
			ClickURL:      result.ClickURL,
			Price:         result.Bid.Price,
			Currency:      "USD",
			AdID:          result.Bid.AdID,
			CampaignID:    result.Bid.CID,
			CreativeID:    result.Bid.CrID,
			Width:         result.Bid.W,
			Height:        result.Bid.H,
		}
		if native := nativeRawFromAdM(result.Bid.AdM); len(native) != 0 {
			response[i].Native = native
		}
	}
	return response
}

func privacySSPAuditResponse(results []sspBidResult) ([]byte, error) {
	response := sspJSONResponse(results)
	for i := range response {
		response[i].AdM = ""
		response[i].Native = nil
		response[i].ImpressionURL = ""
		response[i].ClickURL = ""
	}
	return json.Marshal(response)
}

func nativeRawFromAdM(adm string) json.RawMessage {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(adm), &raw); err != nil {
		return nil
	}
	native := raw["native"]
	if len(native) == 0 {
		return nil
	}
	return native
}

func sspOpenRTBResponse(bid *openrtb2.BidRequest, results []sspBidResult) *openrtb2.BidResponse {
	response := &openrtb2.BidResponse{Cur: "USD"}
	if bid != nil {
		response.ID = bid.ID
	}
	seatBids := make(map[string][]openrtb2.Bid)
	seatOrder := make([]string, 0)
	for _, result := range results {
		if !result.Filled {
			continue
		}
		seat := result.Seat
		if seat == "" {
			seat = result.Bid.CID
		}
		if seat == "" {
			seat = "aofei"
		}
		if _, ok := seatBids[seat]; !ok {
			seatOrder = append(seatOrder, seat)
		}
		if response.BidID == "" {
			response.BidID = result.ResponseBidID
			if response.BidID == "" {
				response.BidID = result.Bid.ID
			}
		}
		seatBids[seat] = append(seatBids[seat], result.Bid)
	}
	for _, seat := range seatOrder {
		response.SeatBid = append(response.SeatBid, openrtb2.SeatBid{
			Seat:  seat,
			Group: 0,
			Bid:   seatBids[seat],
		})
	}
	return response
}

func validateSSPRequestPolicy(r *http.Request, req *SSPRequest, units []SSPValidatedUnit) error {
	if req == nil {
		return fmt.Errorf("ssp request is nil")
	}
	if len(units) == 0 {
		return fmt.Errorf("ssp request has no validated ad units")
	}
	siteHost := strings.TrimSpace(units[0].SiteStr)
	if siteHost == "" {
		return fmt.Errorf("ssp request has no validated site host")
	}

	origins, hasOrigin, originErr := validatedSSPHeaderHosts(r, "Origin")
	if originErr != nil {
		return originErr
	}
	referers, hasReferer, refererErr := validatedSSPHeaderHosts(r, "Referer")
	if refererErr != nil {
		return refererErr
	}
	if !sspPlatformMayOmitBrowserHeaders(req.Platform) && !hasOrigin && !hasReferer {
		return fmt.Errorf("browser SSP request requires matching Origin or Referer")
	}
	for _, origin := range origins {
		if !strings.EqualFold(origin, siteHost) {
			return fmt.Errorf("Origin host %q does not match site host %q", origin, siteHost)
		}
	}
	for _, referer := range referers {
		if !strings.EqualFold(referer, siteHost) {
			return fmt.Errorf("Referer host %q does not match site host %q", referer, siteHost)
		}
	}
	return nil
}

func sspPlatformMayOmitBrowserHeaders(platform string) bool {
	return sspPlatformIsSDK(platform)
}

func sspPlatformUsesBrowserCookie(platform string) bool {
	normalized := strings.TrimSpace(platform)
	return normalized == "" || strings.EqualFold(normalized, "browser")
}

func sspPlatformIsSDK(platform string) bool {
	return strings.EqualFold(strings.TrimSpace(platform), "sdk")
}

func validatedSSPHeaderHosts(r *http.Request, header string) ([]string, bool, error) {
	if r == nil {
		return nil, false, nil
	}
	values := nonEmptyHeaderValues(r.Header.Values(header))
	if len(values) == 0 {
		return nil, false, nil
	}
	hosts := make([]string, 0, len(values))
	for _, value := range values {
		if strings.EqualFold(value, "null") {
			return nil, true, fmt.Errorf("%s header is not an allowed origin", header)
		}
		u, err := url.Parse(value)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Hostname() == "" {
			return nil, true, fmt.Errorf("%s header must be an absolute URL", header)
		}
		hosts = append(hosts, u.Hostname())
	}
	return hosts, true, nil
}

func nonEmptyHeaderValues(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func (self *Controller) resolveSSPUserCookie(w http.ResponseWriter, r *http.Request) string {
	if value := readSSPUserCookie(r); value != "" {
		return value
	}
	self.setSSPUserCookie(w, r)
	return ""
}

func readSSPUserCookie(r *http.Request) string {
	if r == nil {
		return ""
	}
	if cookie, err := r.Cookie(sspUserCookieName); err == nil && validSSPUserCookie(cookie.Value) {
		return cookie.Value
	}
	return ""
}

func (self *Controller) setSSPUserCookie(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	value, err := newSSPUserCookieValue()
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sspUserCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(self.privacyBrowserIDTTL().Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   self.sspCookieSecure(r),
	})
}

func (self *Controller) resolveSSPUserCookieForPlatform(w http.ResponseWriter, r *http.Request, platform string) string {
	if !sspPlatformUsesBrowserCookie(platform) {
		return ""
	}
	return self.resolveSSPUserCookie(w, r)
}

func validSSPUserCookie(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func newSSPUserCookieValue() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (self *Controller) sspCookieSecure(r *http.Request) bool {
	if r != nil {
		if r.TLS != nil {
			return true
		}
		if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			return true
		}
	}
	if self == nil || self.C == nil || self.C.ServerURL == "" {
		return false
	}
	u, err := url.Parse(self.C.ServerURL)
	return err == nil && strings.EqualFold(u.Scheme, "https")
}

func (self *Controller) sspPubByID(ctx context.Context, pubID uint32) (*acl.DirectPub, error) {
	var pub *acl.DirectPub
	var err error
	if self != nil && self.C != nil && self.C.IsLocal {
		pub, err = self.localPubByID(self.C.Spread, pubID)
	} else {
		if self == nil || self.Redis == nil {
			return nil, errSSPPublisherCacheUnavailable
		}
		pub, err = acl.PubByIDFromRedis(ctx, self.Redis, pubID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errSSPPublisherCacheUnavailable, err)
	}
	if pub == nil || pub.Pub == nil {
		return nil, fmt.Errorf("publisher %d not found", pubID)
	}
	return pub, nil
}

func siteFromSSP(r *http.Request, pub *acl.DirectPub, unit SSPValidatedUnit) *openrtb2.Site {
	host := ""
	ref := ""
	if r != nil {
		host = r.Host
		ref = r.Header.Get("Referer")
	}
	site := &openrtb2.Site{
		ID:     unit.Site,
		Name:   host,
		Domain: unit.SiteStr,
		Ref:    ref,
		Publisher: &openrtb2.Publisher{
			ID: strconv.FormatUint(uint64(unit.RPub.PubID), 10),
		},
	}
	if pub != nil {
		site.Publisher.Domain = pub.Domain
	}
	return site
}

func appFromSSP(body *openrtb2.App, pub *acl.DirectPub, unit SSPValidatedUnit) (*openrtb2.App, error) {
	appID := strings.TrimSpace(unit.SiteStr)
	if appID == "" {
		return nil, fmt.Errorf("sdk SSP request has no validated app identity")
	}
	app := &openrtb2.App{}
	if body != nil {
		if err := validateSSPAppIdentity("app.id", body.ID, appID); err != nil {
			return nil, err
		}
		if err := validateSSPAppIdentity("app.bundle", body.Bundle, appID); err != nil {
			return nil, err
		}
		if err := validateSSPAppIdentity("app.domain", body.Domain, appID); err != nil {
			return nil, err
		}
		*app = *body
	}
	app.ID = appID
	app.Bundle = appID
	app.Domain = appID
	app.Publisher = &openrtb2.Publisher{
		ID: strconv.FormatUint(uint64(unit.RPub.PubID), 10),
	}
	if pub != nil {
		app.Publisher.Domain = pub.Domain
	}
	return app, nil
}

func validateSSPAppIdentity(field, value, want string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == want {
		return nil
	}
	return fmt.Errorf("%s %q does not match validated site %q", field, value, want)
}

func (self *Controller) deviceFromSSPHeaders(r *http.Request) *openrtb2.Device {
	device := &openrtb2.Device{}
	if r == nil {
		return device
	}
	device.UA = r.Header.Get("User-Agent")
	device.IP = self.browserIP(r)
	if lang := r.Header.Get("Accept-Language"); lang != "" {
		device.Language = strings.TrimSpace(strings.Split(lang, ",")[0])
	}
	if raw := strings.TrimSpace(r.Header.Get("DNT")); raw == "0" || raw == "1" {
		value := int8(raw[0] - '0')
		device.DNT = &value
	}
	return device
}

func (self *Controller) deviceFromSSP(r *http.Request, body *openrtb2.Device) *openrtb2.Device {
	headers := self.deviceFromSSPHeaders(r)
	if body == nil {
		return headers
	}
	device := *body
	if device.UA == "" {
		device.UA = headers.UA
	}
	if device.IP == "" {
		device.IP = headers.IP
	}
	if device.Language == "" {
		device.Language = headers.Language
	}
	return &device
}

func (self *Controller) browserIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	remoteIP := remoteAddrIP(r.RemoteAddr)
	if self.trustsProxyIP(remoteIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			for _, part := range strings.Split(xff, ",") {
				if ip := strings.TrimSpace(part); ip != "" {
					return ip
				}
			}
		}
		if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
			return xrip
		}
	}
	if remoteIP != "" {
		return remoteIP
	}
	return r.RemoteAddr
}

func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

func (self *Controller) trustsProxyIP(raw string) bool {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || self == nil || self.C == nil || len(self.C.TrustedProxyCIDRs) == 0 {
		return false
	}
	networks, err := parseTrustedProxyCIDRs(self.C.TrustedProxyCIDRs)
	if err != nil {
		return false
	}
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseTrustedProxyCIDRs(values []string) ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if ip := net.ParseIP(value); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			networks = append(networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, fmt.Errorf("trusted_proxy_cidrs entry %q is invalid", value)
		}
		networks = append(networks, network)
	}
	return networks, nil
}

func openRTBImpFromSSPUnit(adUnit SSPAdUnit, unit SSPValidatedUnit, index int) (openrtb2.Imp, error) {
	media := adUnit.EffectiveMediaTypes()
	if err := media.Validate(); err != nil {
		return openrtb2.Imp{}, err
	}
	w, h := match.SizeID1To2(unit.RPub.SizeID)
	if w == 0 || h == 0 {
		return openrtb2.Imp{}, fmt.Errorf("slot token has empty size")
	}
	wi, hi := int64(w), int64(h)
	requestFloor, err := protocolCPM(adUnit.Floor)
	if err != nil {
		return openrtb2.Imp{}, err
	}
	configuredFloor := unit.ConfiguredFloorCPM
	if unit.AccountingVersion == accounting.ExactMoneyContract {
		if configuredFloor < 0 || configuredFloor > accounting.MaxCPM {
			return openrtb2.Imp{}, fmt.Errorf("configured floor is outside the exact USD CPM range")
		}
	} else {
		// Compatibility for callers and old cache generations that predate the
		// fixed-point field. New validated units always carry the v3 marker.
		configuredFloor, err = protocolCPM(unit.ConfiguredFloor)
		if err != nil {
			return openrtb2.Imp{}, err
		}
	}
	if configuredFloor > requestFloor {
		requestFloor = configuredFloor
	}
	imp := openrtb2.Imp{
		ID:          adUnit.Code,
		TagID:       unit.SlotStr,
		BidFloor:    requestFloor.Float64(),
		BidFloorCur: "USD",
	}
	if imp.ID == "" {
		imp.ID = "ssp-" + strconv.Itoa(index+1)
	}
	switch {
	case media.Native != nil:
		nativeRequest, err := nativeRequestFromSSP(media.Native, w, h)
		if err != nil {
			return openrtb2.Imp{}, err
		}
		imp.Native = &openrtb2.Native{Request: nativeRequest, Ver: "1.1"}
	case media.Video != nil:
		mimes := media.Video.MIMEs
		if len(mimes) == 0 {
			mimes = []string{"video/mp4"}
		}
		imp.Video = &openrtb2.Video{W: &wi, H: &hi, MIMEs: mimes}
	default:
		imp.Banner = &openrtb2.Banner{W: &wi, H: &hi}
	}
	return imp, nil
}

func nativeRequestFromSSP(native *SSPNative, w, h uint16) (string, error) {
	assets := []map[string]any{{
		"id":       1,
		"required": 1,
		"img": map[string]any{
			"wmin": int64(w),
			"hmin": int64(h),
		},
	}}
	nextID := 2
	if native != nil && native.Title {
		assets = append(assets, map[string]any{
			"id":       nextID,
			"required": 1,
			"title":    map[string]any{"len": 90},
		})
		nextID++
	}
	if native != nil && native.Body {
		assets = append(assets, map[string]any{
			"id":       nextID,
			"required": 0,
			"data":     map[string]any{"type": 2, "len": 140},
		})
		nextID++
	}
	if native != nil && native.SponsoredBy {
		assets = append(assets, map[string]any{
			"id":       nextID,
			"required": 0,
			"data":     map[string]any{"type": 1, "len": 60},
		})
	}
	data := map[string]any{
		"native": map[string]any{
			"ver":    "1.1",
			"assets": assets,
		},
	}
	bs, err := json.Marshal(data)
	return string(bs), err
}

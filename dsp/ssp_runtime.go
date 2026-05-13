package dsp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
)

const (
	sspUserCookieName   = "aofei_pz_uid"
	sspUserCookieMaxAge = 365 * 24 * 60 * 60
)

// ServeSSP handles the direct publisher JSON contract on /pz.
func (self *Controller) ServeSSP(w http.ResponseWriter, r *http.Request) {
	metricSSPRequests.Add(1)
	current := time.Now()
	ctx := r.Context()

	rawRequest, err := io.ReadAll(io.LimitReader(r.Body, maxBidRequestBodyBytes+1))
	if err != nil {
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pub, units, err := self.validateSSPSupply(ctx, sspReq)
	if err != nil {
		metricSSPValidationErrors.Add(1)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateSSPRequestPolicy(r, sspReq, units); err != nil {
		metricSSPPolicyRejections.Add(1)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	cookieUserID := ""
	if sspPlatformUsesBrowserCookie(sspReq.Platform) {
		cookieUserID = self.resolveSSPUserCookieForPlatform(nil, r, sspReq.Platform)
		if cookieUserID == "" {
			self.resolveSSPUserCookieForPlatform(w, r, sspReq.Platform)
		}
	}
	bid, err := openRTBFromValidatedSSP(r, sspReq, pub, units, cookieUserID)
	if err != nil {
		metricSSPValidationErrors.Add(1)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	html := make([]string, len(units))
	audits := make([]bidAudit, 0, len(units))
	for impIndex := range bid.Imp {
		dspBid, audit, err := self.bidForImp(ctx, bid, pub.Pub, current, pub.Domain, impIndex)
		if err != nil {
			continue
		}
		winloss := dspBid.WinLoss(StatusBid)
		rspBid, err := dspBid.NewBid(winloss)
		if err != nil {
			continue
		}
		html[impIndex] = rspBid.AdM
		audits = append(audits, audit)
	}

	for _, value := range html {
		if value == "" {
			metricSSPNoFillAdUnits.Add(1)
		} else {
			metricSSPFilledAdUnits.Add(1)
		}
	}

	rawResponse, err := json.Marshal(html)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(rawResponse)

	elapsed := time.Since(current)
	for i := range audits {
		audits[i].Elapsed = elapsed
	}
	_ = self.publishSSPBidAudits(rawRequest, rawResponse, audits)
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
	bid, err := openRTBFromValidatedSSP(r, req, pub, units, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	return bid, pub, units, nil
}

func (self *Controller) validateSSPSupply(ctx context.Context, req *SSPRequest) (*acl.DirectPub, []SSPValidatedUnit, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("ssp request is nil")
	}
	pubID, _, err := acl.UnpackDirectToken(string(req.Site))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid site token: %w", err)
	}
	pub, err := self.sspPubByID(ctx, pubID)
	if err != nil {
		return nil, nil, err
	}
	units, err := req.ValidateSupply(pub)
	if err != nil {
		return nil, nil, err
	}
	return pub, units, nil
}

func openRTBFromValidatedSSP(r *http.Request, req *SSPRequest, pub *acl.DirectPub, units []SSPValidatedUnit, cookieUserID string) (*openrtb2.BidRequest, error) {
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
		Site:   siteFromSSP(r, pub, units[0]),
		Device: deviceFromSSPHeaders(r),
		User:   &openrtb2.User{},
		Imp:    make([]openrtb2.Imp, 0, len(units)),
	}
	if cookieUserID != "" {
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
	return strings.EqualFold(strings.TrimSpace(platform), "sdk")
}

func sspPlatformUsesBrowserCookie(platform string) bool {
	normalized := strings.TrimSpace(platform)
	return normalized == "" || strings.EqualFold(normalized, "browser")
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
	if r == nil {
		return ""
	}
	if cookie, err := r.Cookie(sspUserCookieName); err == nil && validSSPUserCookie(cookie.Value) {
		return cookie.Value
	}
	value, err := newSSPUserCookieValue()
	if err != nil || w == nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sspUserCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   sspUserCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   self.sspCookieSecure(r),
	})
	return ""
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
			return nil, fmt.Errorf("redis publisher cache is unavailable")
		}
		pub, err = acl.PubByIDFromRedis(ctx, self.Redis, pubID)
	}
	if err != nil {
		return nil, err
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

func deviceFromSSPHeaders(r *http.Request) *openrtb2.Device {
	device := &openrtb2.Device{}
	if r == nil {
		return device
	}
	device.UA = r.Header.Get("User-Agent")
	device.IP = browserIP(r)
	if lang := r.Header.Get("Accept-Language"); lang != "" {
		device.Language = strings.TrimSpace(strings.Split(lang, ",")[0])
	}
	return device
}

func browserIP(r *http.Request) string {
	if r == nil {
		return ""
	}
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
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
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
	imp := openrtb2.Imp{
		ID:          adUnit.Code,
		TagID:       unit.SlotStr,
		BidFloor:    adUnit.Floor,
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

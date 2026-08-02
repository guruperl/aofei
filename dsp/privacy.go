package dsp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const (
	defaultPrivacyTCFMinPolicyVersion = 5
	defaultPrivacyBrowserIDTTL        = 30 * 24 * time.Hour
	defaultPrivacyLogRetention        = 7 * 24 * time.Hour
	defaultPrivacyAudienceTTL         = 30 * 24 * time.Hour
)

var defaultPrivacyTCFPurposeIDs = []int{1, 3, 4}

type privacyMode string

const (
	privacyModePersonalized privacyMode = "personalized"
	privacyModeContextual   privacyMode = "contextual"
	privacyModeRestricted   privacyMode = "restricted"
)

type privacyDecision struct {
	Mode           privacyMode
	Reason         string
	InvalidSignal  bool
	AllowCookie    bool
	AllowMiddleman bool
	AllowIdentity  bool
}

func (d privacyDecision) normalized() privacyDecision {
	if d.Mode == "" {
		d.Mode = privacyModeContextual
		d.Reason = "missing_signal"
	}
	return d
}

func (self *Controller) privacyDecisionForBid(r *http.Request, bid *openrtb2.BidRequest) privacyDecision {
	var regs *openrtb2.Regs
	var user *openrtb2.User
	var device *openrtb2.Device
	if bid != nil {
		regs = bid.Regs
		user = bid.User
		device = bid.Device
	}
	return self.privacyDecision(r, regs, user, device)
}

func (self *Controller) privacyDecision(r *http.Request, regs *openrtb2.Regs, user *openrtb2.User, device *openrtb2.Device) privacyDecision {
	contextualMiddleman := self != nil && self.C != nil && self.C.PrivacyContextualMiddleman
	decision := func(mode privacyMode, reason string, invalid bool) privacyDecision {
		return privacyDecision{
			Mode:           mode,
			Reason:         reason,
			InvalidSignal:  invalid,
			AllowCookie:    mode == privacyModePersonalized,
			AllowMiddleman: contextualMiddleman && mode != privacyModeRestricted,
			AllowIdentity:  mode == privacyModePersonalized,
		}
	}

	if regs != nil && regs.COPPA != 0 {
		return decision(privacyModeRestricted, "coppa", regs.COPPA != 1)
	}
	if r != nil {
		if raw, present := headerValue(r, "Sec-GPC"); present {
			if raw == "1" {
				return decision(privacyModeContextual, "global_privacy_control", false)
			}
			return decision(privacyModeContextual, "invalid_global_privacy_control", true)
		}
		if raw, present := headerValue(r, "DNT"); present {
			if raw == "1" {
				return decision(privacyModeContextual, "do_not_track", false)
			}
			if raw != "0" {
				return decision(privacyModeContextual, "invalid_do_not_track", true)
			}
		}
	}
	if device != nil {
		if device.DNT != nil {
			if *device.DNT == 1 {
				return decision(privacyModeContextual, "do_not_track", false)
			}
			if *device.DNT != 0 {
				return decision(privacyModeContextual, "invalid_do_not_track", true)
			}
		}
		if device.Lmt != nil {
			if *device.Lmt == 1 {
				return decision(privacyModeContextual, "limit_ad_tracking", false)
			}
			if *device.Lmt != 0 {
				return decision(privacyModeContextual, "invalid_limit_ad_tracking", true)
			}
		}
	}

	if regs != nil && regs.USPrivacy != "" {
		optedOut, valid := parseUSPrivacy(regs.USPrivacy)
		if !valid {
			return decision(privacyModeContextual, "invalid_us_privacy", true)
		}
		if optedOut {
			return decision(privacyModeContextual, "us_privacy_opt_out", false)
		}
	}
	if regs != nil && (regs.GPP != "" || len(regs.GPPSID) != 0) {
		if !validGPPEnvelope(regs.GPP, regs.GPPSID) {
			return decision(privacyModeContextual, "invalid_gpp", true)
		}
		// W8M does not infer a sale/share or purpose grant from an unreviewed
		// jurisdiction section. The signal is retained in the scrubbed request,
		// but personalized processing stays off.
		return decision(privacyModeContextual, "gpp_requires_policy_mapping", false)
	}

	if regs != nil && regs.GDPR != nil {
		if *regs.GDPR != 0 && *regs.GDPR != 1 {
			return decision(privacyModeContextual, "invalid_gdpr", true)
		}
		if *regs.GDPR == 1 {
			if user == nil || strings.TrimSpace(user.Consent) == "" {
				return decision(privacyModeContextual, "gdpr_consent_missing", false)
			}
			vendorID := 0
			minimumPolicy := defaultPrivacyTCFMinPolicyVersion
			purposes := defaultPrivacyTCFPurposeIDs
			secretAvailable := false
			if self != nil && self.C != nil {
				vendorID = self.C.PrivacyTCFVendorID
				if self.C.PrivacyTCFMinPolicyVersion > 0 {
					minimumPolicy = self.C.PrivacyTCFMinPolicyVersion
				}
				if len(self.C.PrivacyTCFPurposeIDs) != 0 {
					purposes = self.C.PrivacyTCFPurposeIDs
				}
				secretAvailable = strings.TrimSpace(self.C.TrackingSecret) != ""
			}
			if vendorID == 0 || !secretAvailable {
				return decision(privacyModeContextual, "gdpr_contract_not_configured", false)
			}
			valid, granted := parseTCFConsent(user.Consent, vendorID, purposes, minimumPolicy)
			if !valid {
				return decision(privacyModeContextual, "invalid_tcf", true)
			}
			if !granted {
				return decision(privacyModeContextual, "gdpr_consent_denied", false)
			}
			return decision(privacyModePersonalized, "gdpr_tcf_granted", false)
		}
		return decision(privacyModeContextual, "gdpr_not_applicable", false)
	}
	if user != nil && strings.TrimSpace(user.Consent) != "" {
		return decision(privacyModeContextual, "tcf_without_gdpr", true)
	}
	return decision(privacyModeContextual, "missing_signal", false)
}

func headerValue(r *http.Request, name string) (string, bool) {
	values := nonEmptyHeaderValues(r.Header.Values(name))
	if len(values) == 0 {
		return "", false
	}
	if len(values) != 1 {
		return "", true
	}
	return values[0], true
}

func parseUSPrivacy(raw string) (bool, bool) {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if len(raw) != 4 || raw[0] != '1' {
		return false, false
	}
	for i := 1; i < len(raw); i++ {
		if raw[i] != 'Y' && raw[i] != 'N' && raw[i] != '-' {
			return false, false
		}
	}
	return raw[2] == 'Y' || raw[3] == 'Y', true
}

func validGPPEnvelope(gpp string, sectionIDs []int8) bool {
	if strings.TrimSpace(gpp) == "" || len(gpp) > 8192 || len(sectionIDs) == 0 {
		return false
	}
	for _, sectionID := range sectionIDs {
		if sectionID <= 0 {
			return false
		}
	}
	for _, segment := range strings.Split(gpp, "~") {
		if segment == "" {
			return false
		}
		for _, r := range segment {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				continue
			}
			return false
		}
	}
	return true
}

type privacyBitReader struct {
	data []byte
	bit  int
}

func (r *privacyBitReader) read(n int) (uint64, bool) {
	if n < 0 || n > 64 || r.bit+n > len(r.data)*8 {
		return 0, false
	}
	var out uint64
	for range n {
		out = out<<1 | uint64((r.data[r.bit/8]>>(7-(r.bit%8)))&1)
		r.bit++
	}
	return out, true
}

func parseTCFConsent(raw string, vendorID int, purposes []int, minimumPolicy int) (bool, bool) {
	if len(raw) == 0 || len(raw) > 8192 || strings.TrimSpace(raw) != raw {
		return false, false
	}
	segments := strings.Split(raw, ".")
	core := segments[0]
	decoded, err := base64.RawURLEncoding.DecodeString(core)
	if err != nil || len(decoded) == 0 {
		return false, false
	}
	r := &privacyBitReader{data: decoded}
	version, ok := r.read(6)
	if !ok || version != 2 {
		return false, false
	}
	created, ok := r.read(36)
	if !ok || created == 0 {
		return false, false
	}
	updated, ok := r.read(36)
	if !ok || updated < created {
		return false, false
	}
	cmpID, ok := r.read(12)
	if !ok || cmpID == 0 {
		return false, false
	}
	cmpVersion, ok := r.read(12)
	if !ok || cmpVersion == 0 {
		return false, false
	}
	if _, ok := r.read(6 + 12); !ok { // consent screen and language
		return false, false
	}
	vendorListVersion, ok := r.read(12)
	if !ok || vendorListVersion == 0 {
		return false, false
	}
	policy, ok := r.read(6)
	if !ok || int(policy) < minimumPolicy {
		return false, false
	}
	serviceSpecific, ok := r.read(1)
	if !ok || serviceSpecific != 1 {
		return false, false
	}
	if _, ok := r.read(1 + 12); !ok { // non-standard stacks and special features
		return false, false
	}
	purposeBits, ok := r.read(24)
	if !ok {
		return false, false
	}
	if _, ok := r.read(24 + 1 + 12); !ok { // LI, purpose-one treatment, publisher CC
		return false, false
	}
	for _, purposeID := range purposes {
		if purposeID < 1 || purposeID > 24 || purposeBits&(uint64(1)<<uint(24-purposeID)) == 0 {
			return true, false
		}
	}
	valid, granted := parseTCFVendorVector(r, vendorID)
	if !valid {
		return false, false
	}
	if valid, _ := parseTCFVendorVector(r, vendorID); !valid { // legitimate-interest vector
		return false, false
	}
	restrictionCount, ok := r.read(12)
	if !ok {
		return false, false
	}
	restricted := false
	configuredPurposes := make(map[int]struct{}, len(purposes))
	for _, purposeID := range purposes {
		configuredPurposes[purposeID] = struct{}{}
	}
	for range int(restrictionCount) {
		purposeID, ok := r.read(6)
		if !ok || purposeID < 1 || purposeID > 24 {
			return false, false
		}
		restrictionType, ok := r.read(2)
		if !ok || restrictionType > 2 {
			return false, false
		}
		entryCount, ok := r.read(12)
		if !ok {
			return false, false
		}
		var previousEnd uint64
		for range int(entryCount) {
			isRange, ok := r.read(1)
			if !ok {
				return false, false
			}
			start, ok := r.read(16)
			if !ok || start == 0 || start <= previousEnd {
				return false, false
			}
			end := start
			if isRange == 1 {
				end, ok = r.read(16)
				if !ok || end < start {
					return false, false
				}
			}
			previousEnd = end
			if _, relevantPurpose := configuredPurposes[int(purposeID)]; relevantPurpose && uint64(vendorID) >= start && uint64(vendorID) <= end {
				// W8M authorizes the configured purposes on consent. A
				// publisher restriction that forbids the purpose or requires
				// legitimate interest is therefore incompatible; an explicit
				// require-consent restriction is satisfied by the checks above.
				restricted = restricted || restrictionType != 1
			}
		}
	}
	if !privacyZeroPadding(r) {
		return false, false
	}
	foundDisclosed := false
	disclosed := false
	seenSegmentTypes := make(map[uint64]struct{}, len(segments)-1)
	for _, rawSegment := range segments[1:] {
		decodedSegment, err := base64.RawURLEncoding.DecodeString(rawSegment)
		if err != nil || len(decodedSegment) == 0 {
			return false, false
		}
		segmentReader := &privacyBitReader{data: decodedSegment}
		segmentType, ok := segmentReader.read(3)
		if !ok || segmentType < 1 || segmentType > 3 {
			return false, false
		}
		if _, duplicate := seenSegmentTypes[segmentType]; duplicate {
			return false, false
		}
		seenSegmentTypes[segmentType] = struct{}{}
		switch segmentType {
		case 1, 2:
			valid, listed := parseTCFVendorVector(segmentReader, vendorID)
			if !valid || !privacyZeroPadding(segmentReader) {
				return false, false
			}
			if segmentType == 1 {
				foundDisclosed = true
				disclosed = listed
			}
		case 3:
			if _, ok := segmentReader.read(24 + 24); !ok {
				return false, false
			}
			customPurposeCount, ok := segmentReader.read(6)
			if !ok || customPurposeCount > 63 {
				return false, false
			}
			if _, ok := segmentReader.read(int(customPurposeCount) * 2); !ok || !privacyZeroPadding(segmentReader) {
				return false, false
			}
		}
	}
	// TCF v2.3 requires the disclosed-vendors segment. Missing disclosure
	// evidence therefore cannot authorize personalized processing.
	if !foundDisclosed {
		return false, false
	}
	return true, granted && disclosed && !restricted
}

func privacyZeroPadding(r *privacyBitReader) bool {
	if r == nil {
		return false
	}
	remaining := len(r.data)*8 - r.bit
	if remaining < 0 || remaining > 7 {
		return false
	}
	padding, ok := r.read(remaining)
	return ok && padding == 0
}

func parseTCFVendorVector(r *privacyBitReader, vendorID int) (bool, bool) {
	maxVendor, ok := r.read(16)
	if !ok || vendorID < 1 {
		return false, false
	}
	rangeEncoding, ok := r.read(1)
	if !ok {
		return false, false
	}
	if rangeEncoding == 0 {
		listed := false
		for current := 1; current <= int(maxVendor); current++ {
			value, ok := r.read(1)
			if !ok {
				return false, false
			}
			if current == vendorID {
				listed = value == 1
			}
		}
		return true, listed
	}
	defaultConsent, ok := r.read(1)
	if !ok {
		return false, false
	}
	numEntries, ok := r.read(12)
	if !ok {
		return false, false
	}
	listed := defaultConsent == 1
	var previousEnd uint64
	for range int(numEntries) {
		isRange, ok := r.read(1)
		if !ok {
			return false, false
		}
		start, ok := r.read(16)
		if !ok || start == 0 || start > maxVendor || start <= previousEnd {
			return false, false
		}
		end := start
		if isRange == 1 {
			end, ok = r.read(16)
			if !ok || end < start || end > maxVendor {
				return false, false
			}
		}
		previousEnd = end
		if uint64(vendorID) >= start && uint64(vendorID) <= end {
			listed = !listed
		}
	}
	return true, listed
}

func (self *Controller) applyPrivacyPolicy(bid *openrtb2.BidRequest, decision privacyDecision) error {
	if bid == nil {
		return nil
	}
	minimizePreciseLocation(bid)
	decision = decision.normalized()
	if decision.Mode == privacyModePersonalized {
		return nil
	}
	raw, err := json.Marshal(bid)
	if err != nil {
		return err
	}
	raw, err = privacySanitizeJSON(raw, false)
	if err != nil {
		return err
	}
	var sanitized openrtb2.BidRequest
	if err := json.Unmarshal(raw, &sanitized); err != nil {
		return err
	}
	*bid = sanitized
	return nil
}

func minimizePreciseLocation(bid *openrtb2.BidRequest) {
	if bid == nil {
		return
	}
	minimizeGeo := func(geo *openrtb2.Geo) {
		if geo == nil {
			return
		}
		hadCoordinates := geo.Lat != nil || geo.Lon != nil
		geo.Lat = nil
		geo.Lon = nil
		geo.Accuracy = 0
		geo.LastFix = 0
		if hadCoordinates {
			geo.Type = 0
		}
	}
	if bid.Device != nil {
		minimizeGeo(bid.Device.Geo)
	}
	if bid.User != nil {
		minimizeGeo(bid.User.Geo)
	}
}

func privacySanitizeJSON(raw []byte, audit bool) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("null"), nil
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	stripPrivacyExtensions(root)
	if object, ok := root.(map[string]any); ok {
		sanitizePrivacyRoot(object, audit)
	}
	return json.Marshal(root)
}

func stripPrivacyExtensions(value any) {
	switch typed := value.(type) {
	case map[string]any:
		delete(typed, "ext")
		for _, child := range typed {
			stripPrivacyExtensions(child)
		}
	case []any:
		for _, child := range typed {
			stripPrivacyExtensions(child)
		}
	}
}

func sanitizePrivacyRoot(root map[string]any, audit bool) {
	delete(root, "user")
	if device, ok := root["device"].(map[string]any); ok {
		for _, key := range []string{
			"geo", "ua", "sua", "ip", "ipv6", "make", "model", "osv", "hwv",
			"h", "w", "ppi", "pxratio", "geofetch", "flashver", "carrier", "mccmnc",
			"ifa", "didsha1", "didmd5", "dpidsha1", "dpidmd5", "macsha1", "macmd5",
		} {
			delete(device, key)
		}
	}
	for _, supplyKey := range []string{"site", "app", "dooh"} {
		if supply, ok := root[supplyKey].(map[string]any); ok {
			delete(supply, "search")
			delete(supply, "keywords")
			delete(supply, "kwarray")
			for _, key := range []string{"page", "ref", "storeurl"} {
				if rawURL, ok := supply[key].(string); ok {
					if safe := privacySafeURL(rawURL); safe == "" {
						delete(supply, key)
					} else {
						supply[key] = safe
					}
				}
			}
			if content, ok := supply["content"].(map[string]any); ok {
				delete(content, "data")
				delete(content, "keywords")
				delete(content, "kwarray")
				if rawURL, ok := content["url"].(string); ok {
					if safe := privacySafeURL(rawURL); safe == "" {
						delete(content, "url")
					} else {
						content["url"] = safe
					}
				}
			}
		}
	}
	if audit {
		if regs, ok := root["regs"].(map[string]any); ok {
			delete(regs, "gpp")
			delete(regs, "us_privacy")
		}
	}
}

func privacySafeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	return u.String()
}

func (self *Controller) protectPrivacyAttribute(attr *match.Attribute, decision privacyDecision) {
	if attr == nil {
		return
	}
	decision = decision.normalized()
	if decision.Mode != privacyModePersonalized || self == nil || self.C == nil || strings.TrimSpace(self.C.TrackingSecret) == "" {
		attr.IFA = ""
		attr.UserID = ""
		return
	}
	attr.IFA = privacyPseudonym(self.C.TrackingSecret, "ifa", attr.IFA)
	attr.UserID = privacyPseudonym(self.C.TrackingSecret, "user", attr.UserID)
}

func (self *Controller) protectPrivacyAttributeForBid(attr *match.Attribute, bid *openrtb2.BidRequest, decision privacyDecision) {
	if attr == nil {
		return
	}
	if !privacyBidHasExplicitIdentity(bid) {
		attr.IFA = ""
		attr.UserID = ""
	}
	self.protectPrivacyAttribute(attr, decision)
}

func privacyBidHasExplicitIdentity(bid *openrtb2.BidRequest) bool {
	if bid == nil {
		return false
	}
	if bid.User != nil && (bid.User.BuyerUID != "" || bid.User.ID != "") {
		return true
	}
	if bid.Device == nil {
		return false
	}
	device := bid.Device
	return device.IFA != "" || device.DIDSHA1 != "" || device.DIDMD5 != "" ||
		device.DPIDSHA1 != "" || device.DPIDMD5 != "" || device.MACSHA1 != "" || device.MACMD5 != ""
}

func privacyPseudonym(secret, domain, value string) string {
	if secret == "" || value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("aofei/privacy/v1/" + domain + "\x00" + value))
	return "p1_" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func privacySafeAttribute(attr *match.Attribute) *match.Attribute {
	if attr == nil {
		return nil
	}
	copyAttr := *attr
	copyAttr.IFA = ""
	copyAttr.UserID = ""
	copyAttr.Demo = nil
	if attr.Geo != nil {
		geo := *attr.Geo
		geo.CityID = 0
		geo.DmaID = 0
		geo.IspID = 0
		geo.ZipID = 0
		geo.Location.Lat = 0
		geo.Location.Lon = 0
		geo.Location.Accuracy = 0
		geo.Location.LastFix = 0
		copyAttr.Geo = &geo
	}
	return &copyAttr
}

func (self *Controller) privacyBrowserIDTTL() time.Duration {
	if self != nil && self.C != nil && self.C.PrivacyBrowserIDTTLSeconds > 0 {
		return time.Duration(self.C.PrivacyBrowserIDTTLSeconds) * time.Second
	}
	return defaultPrivacyBrowserIDTTL
}

func privacyDecisionLabel(decision privacyDecision) string {
	decision = decision.normalized()
	return fmt.Sprintf("%s:%s", decision.Mode, decision.Reason)
}

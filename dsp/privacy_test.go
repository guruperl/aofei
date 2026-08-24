package dsp

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestPrivacyDecisionMatrixUsesMostRestrictiveSignal(t *testing.T) {
	zero, one, invalid := int8(0), int8(1), int8(2)
	granted := testTCFConsent(t, 7, true, []int{1, 3, 4})
	denied := testTCFConsent(t, 7, false, []int{1, 3, 4})
	controller := &Controller{C: &Config{
		TrackingSecret:             "privacy-secret",
		PrivacyTCFVendorID:         7,
		PrivacyTCFMinPolicyVersion: 5,
		PrivacyTCFPurposeIDs:       []int{1, 3, 4},
		PrivacyContextualMiddleman: true,
	}}
	tests := []struct {
		name    string
		regs    *openrtb2.Regs
		user    *openrtb2.User
		device  *openrtb2.Device
		headers map[string]string
		mode    privacyMode
		reason  string
		invalid bool
	}{
		{name: "missing", mode: privacyModeContextual, reason: "missing_signal"},
		{name: "coppa", regs: &openrtb2.Regs{COPPA: 1}, mode: privacyModeRestricted, reason: "coppa"},
		{name: "gpc overrides grant", regs: &openrtb2.Regs{GDPR: &one}, user: &openrtb2.User{Consent: granted}, headers: map[string]string{"Sec-GPC": "1"}, mode: privacyModeContextual, reason: "global_privacy_control"},
		{name: "dnt", device: &openrtb2.Device{DNT: &one}, mode: privacyModeContextual, reason: "do_not_track"},
		{name: "invalid dnt", device: &openrtb2.Device{DNT: &invalid}, mode: privacyModeContextual, reason: "invalid_do_not_track", invalid: true},
		{name: "limit tracking", device: &openrtb2.Device{Lmt: &one}, mode: privacyModeContextual, reason: "limit_ad_tracking"},
		{name: "us opt out", regs: &openrtb2.Regs{USPrivacy: "1YYN"}, mode: privacyModeContextual, reason: "us_privacy_opt_out"},
		{name: "invalid us privacy", regs: &openrtb2.Regs{USPrivacy: "bad"}, mode: privacyModeContextual, reason: "invalid_us_privacy", invalid: true},
		{name: "gpp requires mapping", regs: &openrtb2.Regs{GPP: "DBABLA~BUENAA", GPPSID: []int8{7}}, mode: privacyModeContextual, reason: "gpp_requires_policy_mapping"},
		{name: "invalid gpp", regs: &openrtb2.Regs{GPP: "bad!", GPPSID: []int8{7}}, mode: privacyModeContextual, reason: "invalid_gpp", invalid: true},
		{name: "gdpr missing", regs: &openrtb2.Regs{GDPR: &one}, mode: privacyModeContextual, reason: "gdpr_consent_missing"},
		{name: "gdpr denied", regs: &openrtb2.Regs{GDPR: &one}, user: &openrtb2.User{Consent: denied}, mode: privacyModeContextual, reason: "gdpr_consent_denied"},
		{name: "gdpr granted", regs: &openrtb2.Regs{GDPR: &one}, user: &openrtb2.User{Consent: granted}, mode: privacyModePersonalized, reason: "gdpr_tcf_granted"},
		{name: "gdpr not applicable remains contextual", regs: &openrtb2.Regs{GDPR: &zero}, mode: privacyModeContextual, reason: "gdpr_not_applicable"},
		{name: "invalid gdpr", regs: &openrtb2.Regs{GDPR: &invalid}, mode: privacyModeContextual, reason: "invalid_gdpr", invalid: true},
		{name: "tcf without gdpr", user: &openrtb2.User{Consent: granted}, mode: privacyModeContextual, reason: "tcf_without_gdpr", invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/bid", nil)
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}
			got := controller.privacyDecision(r, tt.regs, tt.user, tt.device)
			if got.Mode != tt.mode || got.Reason != tt.reason || got.InvalidSignal != tt.invalid {
				t.Fatalf("decision = %#v, want mode=%s reason=%s invalid=%t", got, tt.mode, tt.reason, tt.invalid)
			}
			if got.AllowIdentity != (tt.mode == privacyModePersonalized) || got.AllowCookie != (tt.mode == privacyModePersonalized) {
				t.Fatalf("identity/cookie grants = %t/%t for mode %s", got.AllowIdentity, got.AllowCookie, tt.mode)
			}
			if got.AllowMiddleman != (tt.mode != privacyModeRestricted) {
				t.Fatalf("middleman grant = %t for mode %s", got.AllowMiddleman, tt.mode)
			}
		})
	}
}

func TestSSPClientClaimsRequireAuthenticatedSDKForPersonalization(t *testing.T) {
	personalized := privacyDecision{
		Mode: privacyModePersonalized, Reason: "gdpr_tcf_granted",
		AllowCookie: true, AllowMiddleman: true, AllowIdentity: true,
	}
	unauthenticated := sspClientClaimPrivacy("sdk", false, personalized)
	if unauthenticated.Mode != privacyModeContextual || unauthenticated.Reason != "sdk_unauthenticated" || unauthenticated.AllowCookie || unauthenticated.AllowIdentity || !unauthenticated.AllowMiddleman {
		t.Fatalf("unauthenticated SDK decision = %+v", unauthenticated)
	}
	if got := sspClientClaimPrivacy("sdk", true, personalized); got != personalized {
		t.Fatalf("authenticated SDK decision = %+v, want %+v", got, personalized)
	}
	if got := sspClientClaimPrivacy("browser", false, personalized); got != personalized {
		t.Fatalf("browser decision = %+v, want %+v", got, personalized)
	}
	restricted := privacyDecision{Mode: privacyModeRestricted, Reason: "coppa"}
	if got := sspClientClaimPrivacy("sdk", false, restricted); got != restricted {
		t.Fatalf("restrictive SDK decision = %+v, want %+v", got, restricted)
	}

	lat, lon := 37.7, -122.4
	claimed := &openrtb2.BidRequest{
		Device: &openrtb2.Device{IP: "198.51.100.9", IFA: "claimed-ifa", Geo: &openrtb2.Geo{
			Country: "US", Region: "CA", Lat: &lat, Lon: &lon,
		}},
		User: &openrtb2.User{ID: "claimed-user", BuyerUID: "claimed-buyer"},
	}
	controller := &Controller{}
	if err := controller.applyPrivacyPolicy(claimed, unauthenticated); err != nil {
		t.Fatal(err)
	}
	if claimed.User != nil || claimed.Device == nil || claimed.Device.IP != "" || claimed.Device.IFA != "" || claimed.Device.Geo != nil {
		t.Fatalf("unauthenticated SDK claims survived contextualization: %#v", claimed)
	}

	claimed = &openrtb2.BidRequest{
		Device: &openrtb2.Device{IP: "198.51.100.9", IFA: "claimed-ifa", Geo: &openrtb2.Geo{
			Country: "US", Region: "CA", Lat: &lat, Lon: &lon,
		}},
		User: &openrtb2.User{ID: "claimed-user"},
	}
	if err := controller.applyPrivacyPolicy(claimed, personalized); err != nil {
		t.Fatal(err)
	}
	if claimed.User == nil || claimed.User.ID != "claimed-user" || claimed.Device.IP != "198.51.100.9" || claimed.Device.Geo == nil || claimed.Device.Geo.Country != "US" || claimed.Device.Geo.Region != "CA" || claimed.Device.Geo.Lat != nil || claimed.Device.Geo.Lon != nil {
		t.Fatalf("authenticated publisher assertions were not bounded as documented: %#v", claimed)
	}
}

func TestPrivacyDecisionRequiresConfiguredContractAndCurrentTCF(t *testing.T) {
	one := int8(1)
	consent := testTCFConsent(t, 7, true, []int{1, 3, 4})
	controller := &Controller{C: &Config{TrackingSecret: "secret"}}
	got := controller.privacyDecision(nil, &openrtb2.Regs{GDPR: &one}, &openrtb2.User{Consent: consent}, nil)
	if got.Mode != privacyModeContextual || got.Reason != "gdpr_contract_not_configured" {
		t.Fatalf("decision = %#v", got)
	}

	controller.C.PrivacyTCFVendorID = 7
	controller.C.PrivacyTCFMinPolicyVersion = 6
	got = controller.privacyDecision(nil, &openrtb2.Regs{GDPR: &one}, &openrtb2.User{Consent: consent}, nil)
	if got.Reason != "invalid_tcf" || !got.InvalidSignal {
		t.Fatalf("old-policy decision = %#v", got)
	}
}

func TestPrivacyDecisionHonorsTCFPublisherRestrictions(t *testing.T) {
	one := int8(1)
	controller := &Controller{C: &Config{
		TrackingSecret:             "secret",
		PrivacyTCFVendorID:         7,
		PrivacyTCFMinPolicyVersion: 5,
		PrivacyTCFPurposeIDs:       []int{1, 3, 4},
	}}
	tests := []struct {
		name            string
		restrictionType int
		wantMode        privacyMode
		wantReason      string
	}{
		{name: "not allowed", restrictionType: 0, wantMode: privacyModeContextual, wantReason: "gdpr_consent_denied"},
		{name: "requires consent", restrictionType: 1, wantMode: privacyModePersonalized, wantReason: "gdpr_tcf_granted"},
		{name: "requires legitimate interest", restrictionType: 2, wantMode: privacyModeContextual, wantReason: "gdpr_consent_denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consent := testTCFConsentWithRestrictions(t, 7, true, []int{1, 3, 4}, []testTCFRestriction{{
				purposeID: 3,
				typeID:    tt.restrictionType,
				start:     7,
				end:       7,
			}})
			got := controller.privacyDecision(nil, &openrtb2.Regs{GDPR: &one}, &openrtb2.User{Consent: consent}, nil)
			if got.Mode != tt.wantMode || got.Reason != tt.wantReason {
				t.Fatalf("decision = %#v, want mode=%s reason=%s", got, tt.wantMode, tt.wantReason)
			}
		})
	}
}

func TestPrivacyDecisionCannotUndoRestrictiveTCFEntry(t *testing.T) {
	one := int8(1)
	controller := &Controller{C: &Config{
		TrackingSecret:             "secret",
		PrivacyTCFVendorID:         7,
		PrivacyTCFMinPolicyVersion: 5,
		PrivacyTCFPurposeIDs:       []int{1, 3, 4},
	}}
	consent := testTCFConsentWithRestrictions(t, 7, true, []int{1, 3, 4}, []testTCFRestriction{
		{purposeID: 3, typeID: 0, start: 7, end: 7},
		{purposeID: 3, typeID: 1, start: 7, end: 7},
	})
	got := controller.privacyDecision(nil, &openrtb2.Regs{GDPR: &one}, &openrtb2.User{Consent: consent}, nil)
	if got.Mode != privacyModeContextual || got.Reason != "gdpr_consent_denied" {
		t.Fatalf("decision = %#v, want publisher restriction to remain effective", got)
	}
}

func TestPrivacyDecisionRejectsMissingOrDuplicateDisclosedVendorSegment(t *testing.T) {
	one := int8(1)
	controller := &Controller{C: &Config{
		TrackingSecret:             "secret",
		PrivacyTCFVendorID:         7,
		PrivacyTCFMinPolicyVersion: 5,
		PrivacyTCFPurposeIDs:       []int{1, 3, 4},
	}}
	consent := testTCFConsent(t, 7, true, []int{1, 3, 4})
	segments := strings.Split(consent, ".")
	for name, malformed := range map[string]string{
		"missing":   segments[0],
		"duplicate": consent + "." + segments[1],
	} {
		t.Run(name, func(t *testing.T) {
			got := controller.privacyDecision(nil, &openrtb2.Regs{GDPR: &one}, &openrtb2.User{Consent: malformed}, nil)
			if got.Mode != privacyModeContextual || got.Reason != "invalid_tcf" || !got.InvalidSignal {
				t.Fatalf("decision = %#v, want invalid TCF", got)
			}
		})
	}
}

func TestPrivacyPolicyAlwaysRemovesPreciseLocation(t *testing.T) {
	lat, lon := 39.9042, 116.4074
	bid := &openrtb2.BidRequest{
		Device: &openrtb2.Device{Geo: &openrtb2.Geo{Lat: &lat, Lon: &lon, Accuracy: 5, LastFix: 2, Country: "CHN", Region: "BJ"}},
		User:   &openrtb2.User{Geo: &openrtb2.Geo{Lat: &lat, Lon: &lon, Accuracy: 10, LastFix: 3, Country: "CHN"}},
	}
	controller := &Controller{}
	if err := controller.applyPrivacyPolicy(bid, privacyDecision{Mode: privacyModePersonalized}); err != nil {
		t.Fatal(err)
	}
	for name, geo := range map[string]*openrtb2.Geo{"device": bid.Device.Geo, "user": bid.User.Geo} {
		if geo.Lat != nil || geo.Lon != nil || geo.Accuracy != 0 || geo.LastFix != 0 {
			t.Fatalf("%s precise geo survived: %#v", name, geo)
		}
		if geo.Country != "CHN" {
			t.Fatalf("%s coarse country was removed: %#v", name, geo)
		}
	}
}

func TestPrivacySanitizerRemovesIdentityAndEveryUncontrolledExtension(t *testing.T) {
	raw := []byte(`{
		"id":"request-1",
		"user":{"id":"raw-user","buyeruid":"raw-buyer","consent":"consent-signal","data":[{"ext":{"secret":1}}],"ext":{"debug":true}},
		"device":{"ip":"203.0.113.7","ua":"precise UA","ifa":"raw-ifa","os":"Linux","language":"en","geo":{"lat":1,"lon":2},"ext":{"secret":1}},
		"site":{"domain":"example.com","page":"https://person:pass@example.com/article?email=p@example.com#x","search":"private query","ext":{"secret":1}},
		"imp":[{"id":"one","banner":{"w":300,"h":250,"ext":{"secret":1}},"ext":{"debug":true}}],
		"ext":{"internal_debug":{"secret":1}},"unknown":{"personal":"value"}
	}`)
	safe, err := privacySanitizeJSON(raw, false)
	if err != nil {
		t.Fatal(err)
	}
	text := string(safe)
	for _, forbidden := range []string{"raw-user", "raw-buyer", "raw-ifa", "203.0.113.7", "precise UA", "private query", "email=", "person:pass", "internal_debug", "secret", `"ext"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("sanitized request contains %q: %s", forbidden, text)
		}
	}
	for _, required := range []string{"example.com/article", `"os":"Linux"`, `"language":"en"`} {
		if !strings.Contains(text, required) {
			t.Fatalf("sanitized request missing %q: %s", required, text)
		}
	}

	audit, err := privacySanitizeJSON(safe, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(safe), "consent-signal") || strings.Contains(string(safe), `"user"`) {
		t.Fatalf("contextual request contains consent/user data: %s", safe)
	}
	if strings.Contains(string(audit), "consent-signal") || strings.Contains(string(audit), `"user"`) {
		t.Fatalf("audit contains consent/user data: %s", audit)
	}
}

func TestPrivacyAttributeUsesDomainSeparatedPseudonymsAndAuditRedaction(t *testing.T) {
	controller := &Controller{C: &Config{TrackingSecret: "secret"}}
	attr := &match.Attribute{IFA: "same-raw-id", UserID: "same-raw-id"}
	controller.protectPrivacyAttribute(attr, privacyDecision{Mode: privacyModePersonalized})
	if attr.IFA == "" || attr.UserID == "" || attr.IFA == attr.UserID {
		t.Fatalf("pseudonyms = %q/%q, want populated domain-separated values", attr.IFA, attr.UserID)
	}
	if strings.Contains(attr.IFA, "same-raw-id") || strings.Contains(attr.UserID, "same-raw-id") {
		t.Fatalf("pseudonyms expose raw identifier: %#v", attr)
	}
	firstUser := attr.UserID
	second := &match.Attribute{UserID: "same-raw-id"}
	controller.protectPrivacyAttribute(second, privacyDecision{Mode: privacyModePersonalized})
	if second.UserID != firstUser {
		t.Fatalf("stable pseudonym = %q, want %q", second.UserID, firstUser)
	}
	redacted := privacySafeAttribute(attr)
	if redacted.UserID != "" || redacted.IFA != "" {
		t.Fatalf("audit attribute retained identity: %#v", redacted)
	}
}

func TestPrivacyAttributeDoesNotTurnIPAndUAIntoIdentity(t *testing.T) {
	controller := &Controller{C: &Config{TrackingSecret: "secret"}}
	attr := &match.Attribute{IFA: "legacy-ip-ua-hash", UserID: "legacy-ip-ua-hash"}
	bid := &openrtb2.BidRequest{Device: &openrtb2.Device{IP: "203.0.113.7", UA: "raw-ua"}}
	controller.protectPrivacyAttributeForBid(attr, bid, privacyDecision{Mode: privacyModePersonalized})
	if attr.IFA != "" || attr.UserID != "" {
		t.Fatalf("IP+UA fallback survived runtime privacy boundary: %#v", attr)
	}

	attr = &match.Attribute{IFA: "raw-ifa", UserID: "raw-ifa"}
	bid.Device.IFA = "raw-ifa"
	controller.protectPrivacyAttributeForBid(attr, bid, privacyDecision{Mode: privacyModePersonalized})
	if attr.IFA == "" || attr.UserID == "" || attr.IFA == "raw-ifa" || attr.UserID == "raw-ifa" {
		t.Fatalf("explicit identity was not pseudonymized: %#v", attr)
	}
}

func TestServeSSPSetsShortLivedIdentityCookieOnlyForConfiguredGrant(t *testing.T) {
	controller := newLocalBidPathController(t)
	controller.C.TrackingSecret = "privacy-secret"
	controller.C.PrivacyTCFVendorID = 7
	controller.C.PrivacyTCFMinPolicyVersion = 5
	controller.C.PrivacyTCFPurposeIDs = []int{1, 3, 4}
	controller.C.PrivacyBrowserIDTTLSeconds = 3600
	body := sspRequestBodyWithFields(t, sspRequestBodyWithPlatform(t, "browser", 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	}), map[string]any{
		"regs": map[string]any{"gdpr": 1},
		"user": map[string]any{"consent": testTCFConsent(t, 7, true, []int{1, 3, 4}), "id": "must-not-be-accepted-from-browser"},
	})
	rr := serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://example.com"})
	if rr.Code != 200 {
		t.Fatalf("ServeSSP status = %d: %s", rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sspUserCookieName || !validSSPUserCookie(cookies[0].Value) {
		t.Fatalf("cookies = %#v, want one generated identity cookie", cookies)
	}
	if cookies[0].MaxAge != 3600 {
		t.Fatalf("cookie MaxAge = %d, want 3600", cookies[0].MaxAge)
	}

	body = sspRequestBodyWithFields(t, body, map[string]any{
		"regs": map[string]any{"gdpr": 1, "coppa": 1},
	})
	rr = serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://example.com"})
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("COPPA response cookies = %#v, want none", cookies)
	}
}

func TestMiddlemanAssignmentGetsOnlyItsImpressionsAndContextualData(t *testing.T) {
	raw := []byte(`{"id":"request-1","user":{"id":"raw-user"},"device":{"ip":"203.0.113.7","ifa":"raw-ifa"},"imp":[{"id":"one","ext":{"private":1}},{"id":"two"}],"ext":{"private":1}}`)
	body, err := middlemanRequestBodyForAssignmentImps(raw, "exchange.example", map[string]string{"two": "https://exchange.example/click"}, map[string]struct{}{"two": {}})
	if err != nil {
		t.Fatal(err)
	}
	var bid openrtb2.BidRequest
	if err := json.Unmarshal(body, &bid); err != nil {
		t.Fatal(err)
	}
	if len(bid.Imp) != 1 || bid.Imp[0].ID != "two" {
		t.Fatalf("impressions = %#v, want only two", bid.Imp)
	}
	text := string(body)
	for _, forbidden := range []string{"raw-user", "raw-ifa", "203.0.113.7", `"private"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("bidder body contains %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "request_domain") || !strings.Contains(text, "click_notify_urls") {
		t.Fatalf("controlled middleman metadata missing: %s", text)
	}
}

func TestMiddlemanAssignmentRejectsAmbiguousImpressionIdentity(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		allowed map[string]struct{}
	}{
		{name: "duplicate request id", raw: `{"imp":[{"id":"one"},{"id":"one"}]}`, allowed: map[string]struct{}{"one": {}}},
		{name: "empty request id", raw: `{"imp":[{"id":""}]}`, allowed: map[string]struct{}{"": {}}},
		{name: "missing assigned id", raw: `{"imp":[{"id":"one"}]}`, allowed: map[string]struct{}{"two": {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := middlemanRequestBodyForAssignmentImps([]byte(tt.raw), "exchange.example", nil, tt.allowed); err == nil {
				t.Fatal("expected ambiguous impression identity to be rejected")
			}
		})
	}
}

func TestAuditPublisherRedactsRequestAndDerivedIdentity(t *testing.T) {
	controller := &Controller{audit: newAuditPublisher(nil, 10)}
	attr := &match.Attribute{IFA: "derived-ifa", UserID: "derived-user"}
	raw := []byte(`{"id":"request-1","user":{"id":"raw-user","consent":"raw-consent"},"device":{"ip":"203.0.113.7","ua":"raw-ua","ifa":"raw-ifa"},"imp":[{"id":"one"}]}`)
	audits := []bidAudit{{
		Attr:          attr,
		PrivacyMode:   string(privacyModePersonalized),
		PrivacyReason: "gdpr_tcf_granted",
	}}
	if err := controller.publishBidAudits(raw, []byte(`{"id":"request-1"}`), audits); err != nil {
		t.Fatal(err)
	}
	messages := drainAuditMessages(controller.audit)
	if len(messages) != 3 {
		t.Fatalf("messages = %d, want 3", len(messages))
	}
	for _, forbidden := range []string{"raw-user", "raw-consent", "203.0.113.7", "raw-ua", "raw-ifa"} {
		if strings.Contains(string(messages[0].data), forbidden) {
			t.Fatalf("request audit contains %q: %s", forbidden, messages[0].data)
		}
	}
	var plus match.AttributePlus
	if err := json.Unmarshal(messages[2].data, &plus); err != nil {
		t.Fatal(err)
	}
	if plus.UserID != "" || plus.IFA != "" {
		t.Fatalf("attribute audit retained identity: %#v", plus.Attribute)
	}
	if plus.PrivacyMode != string(privacyModePersonalized) || plus.PrivacyReason != "gdpr_tcf_granted" {
		t.Fatalf("privacy evidence = %q/%q", plus.PrivacyMode, plus.PrivacyReason)
	}
}

func TestWinLossPublisherRemovesEmbeddedPseudonymousIdentity(t *testing.T) {
	var published []byte
	controller := &Controller{publishWinLossFunc: func(data []byte) error {
		published = append([]byte(nil), data...)
		return nil
	}}
	const when = int64(0x1234567890abcdef)
	rawID := bidID{When: when, UserID: "p1_stable-pseudonym"}.String()
	raw, err := json.Marshal(WinLoss{AuctionBidID: rawID, AuctionID: "auction", AuctionImpID: "imp"})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.publishWinLoss(raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(published), "p1_stable-pseudonym") {
		t.Fatalf("win/loss log retained user pseudonym: %s", published)
	}
	var safe WinLoss
	if err := json.Unmarshal(published, &safe); err != nil {
		t.Fatal(err)
	}
	unpacked, err := UnpackBidID(safe.AuctionBidID)
	if err != nil || unpacked.When != when || unpacked.UserID != "" {
		t.Fatalf("safe auction bid ID = %q / %#v / %v", safe.AuctionBidID, unpacked, err)
	}
}

func TestCreativeMacrosNeverDiscloseRawDeviceData(t *testing.T) {
	connection := adcom1.ConnectionType(2)
	bid := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP: "203.0.113.7", UA: "raw-user-agent", IFA: "raw-ifa",
			DPIDMD5: "raw-gaid", DIDMD5: "raw-device", MACMD5: "raw-mac",
			OS: "Android", OSV: "99", Make: "raw-brand", Model: "raw-model",
			Carrier: "raw-carrier", Language: "zh", ConnectionType: &connection,
			Geo: &openrtb2.Geo{Country: "CHN", Region: "BJ", City: "raw-city"},
		},
		Imp: []openrtb2.Imp{{ID: "one"}},
	}
	d := &DSP{bid: bid, impIndex: 0, attribute: &match.Attribute{}, one: match.RAdv{}}
	macros := d.Macro()
	for _, key := range []string{"{DSP_IP}", "{DSP_CITY}", "{DSP_CARRIER}", "{DSP_USER_AGENT}", "{DSP_OS_VERSION}", "{DSP_DEVICE_BRAND}", "{DSP_DEVICE_MODEL}", "{DSP_GAID}", "{DSP_IDFA}", "{DSP_DEVICE_ID}", "{DSP_DEVICE_ID_MD5}", "{DSP_DEVICE_ID_SHA1}"} {
		if macros[key] != "" {
			t.Fatalf("privacy-sensitive macro %s = %q, want empty", key, macros[key])
		}
	}
	if macros["{DSP_COUNTRY}"] != "CHN" || macros["{DSP_OS}"] != "Android" || macros["{DSP_DEVICE_LANGUAGE}"] != "zh" {
		t.Fatalf("coarse contextual macros were removed: %#v", macros)
	}
}

type testBitWriter struct {
	bits []byte
}

func (w *testBitWriter) write(value uint64, width int) {
	for bit := width - 1; bit >= 0; bit-- {
		w.bits = append(w.bits, byte(value>>uint(bit)&1))
	}
}

func (w *testBitWriter) encode() string {
	data := make([]byte, (len(w.bits)+7)/8)
	for i, bit := range w.bits {
		data[i/8] |= bit << uint(7-(i%8))
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

type testTCFRestriction struct {
	purposeID int
	typeID    int
	start     int
	end       int
}

func testTCFConsent(t *testing.T, vendorID int, vendorGranted bool, purposes []int) string {
	t.Helper()
	return testTCFConsentWithRestrictions(t, vendorID, vendorGranted, purposes, nil)
}

func testTCFConsentWithRestrictions(t *testing.T, vendorID int, vendorGranted bool, purposes []int, restrictions []testTCFRestriction) string {
	t.Helper()
	maxVendorID := vendorID + 3 // Exercise vectors where the configured vendor is not last.
	w := &testBitWriter{}
	w.write(2, 6)
	w.write(1, 36) // created
	w.write(1, 36) // last updated
	w.write(1, 12) // CMP id
	w.write(1, 12) // CMP version
	w.write(0, 6)  // consent screen
	w.write(0, 12) // consent language fixture
	w.write(1, 12) // vendor-list version
	w.write(5, 6)
	w.write(1, 1) // service specific
	w.write(0, 1) // non-standard stacks
	w.write(0, 12)
	var purposeBits uint64
	for _, purposeID := range purposes {
		purposeBits |= uint64(1) << uint(24-purposeID)
	}
	w.write(purposeBits, 24)
	w.write(0, 24)
	w.write(0, 1)
	w.write(0, 12)
	w.write(uint64(maxVendorID), 16)
	w.write(0, 1) // bit-field vendor encoding
	for current := 1; current <= maxVendorID; current++ {
		if current == vendorID {
			w.write(1, 1)
		} else {
			w.write(0, 1)
		}
	}
	// Vendor legitimate-interest vector and publisher-restriction count.
	w.write(uint64(maxVendorID), 16)
	w.write(0, 1)
	for current := 1; current <= maxVendorID; current++ {
		w.write(0, 1)
	}
	w.write(uint64(len(restrictions)), 12)
	for _, restriction := range restrictions {
		w.write(uint64(restriction.purposeID), 6)
		w.write(uint64(restriction.typeID), 2)
		w.write(1, 12)
		if restriction.start == restriction.end {
			w.write(0, 1)
			w.write(uint64(restriction.start), 16)
		} else {
			w.write(1, 1)
			w.write(uint64(restriction.start), 16)
			w.write(uint64(restriction.end), 16)
		}
	}
	core := w.encode()
	disclosed := &testBitWriter{}
	disclosed.write(1, 3) // disclosed-vendors segment
	disclosed.write(uint64(maxVendorID), 16)
	disclosed.write(0, 1)
	for current := 1; current <= maxVendorID; current++ {
		if current == vendorID && vendorGranted {
			disclosed.write(1, 1)
		} else {
			disclosed.write(0, 1)
		}
	}
	return core + "." + disclosed.encode()
}

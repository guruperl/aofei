package dsp

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
)

func TestOpenRTBImpFromSSPUsesTokenSizeForSupportedMedia(t *testing.T) {
	unit := SSPValidatedUnit{
		Code:    "code",
		SlotStr: "slot",
		RPub:    match.RPub{SizeID: match.SizeID2To1(300, 250)},
	}
	tests := []struct {
		name   string
		adUnit SSPAdUnit
		want   string
	}{
		{
			name:   "banner",
			adUnit: SSPAdUnit{Code: "banner", MediaTypes: SSPMediaTypes{Banner: &SSPBanner{Size: []uint16{1, 1}}}},
			want:   "banner",
		},
		{
			name:   "video",
			adUnit: SSPAdUnit{Code: "video", MediaTypes: SSPMediaTypes{Video: &SSPVideo{PlayerSize: []uint16{1, 1}}}},
			want:   "video",
		},
		{
			name:   "native",
			adUnit: SSPAdUnit{Code: "native", MediaTypes: SSPMediaTypes{Native: &SSPNative{Image: []uint16{1, 1}, Title: true}}},
			want:   "native",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp, err := openRTBImpFromSSPUnit(tt.adUnit, unit, 0)
			if err != nil {
				t.Fatal(err)
			}
			switch tt.want {
			case "banner":
				if imp.Banner == nil || *imp.Banner.W != 300 || *imp.Banner.H != 250 {
					t.Fatalf("banner = %#v, want token size", imp.Banner)
				}
			case "video":
				if imp.Video == nil || *imp.Video.W != 300 || *imp.Video.H != 250 {
					t.Fatalf("video = %#v, want token size", imp.Video)
				}
			case "native":
				if imp.Native == nil || !strings.Contains(imp.Native.Request, `"wmin":300`) || !strings.Contains(imp.Native.Request, `"hmin":250`) {
					t.Fatalf("native request = %q, want token size", imp.Native.Request)
				}
			}
		})
	}
}

func TestServeSSPReturnsHTMLArrayInRequestOrder(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
		{Code: "unit-two", SlotID: 200, SizeID: match.SizeID2To1(320, 50), Banner: true, Floor: 1},
	})

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var html []string
	if err := json.Unmarshal(rr.Body.Bytes(), &html); err != nil {
		t.Fatal(err)
	}
	if len(html) != 2 {
		t.Fatalf("html len = %d, want 2", len(html))
	}
	if !strings.Contains(html[0], "one.html") || !strings.Contains(html[1], "two.html") {
		t.Fatalf("html order/content = %#v", html)
	}
}

func TestServeSSPPublishesSourceAuditsForFilledAndAllNoFillResponses(t *testing.T) {
	tests := []struct {
		name       string
		floor      float64
		wantAttrs  int
		wantFilled bool
	}{
		{name: "filled", floor: 1, wantAttrs: 1, wantFilled: true},
		{name: "all no fill", floor: 100, wantAttrs: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			controller.audit = newAuditPublisher(nil, 10)
			body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
				{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: tt.floor},
			})

			rr := serveSSP(t, controller, body)
			if rr.Code != http.StatusOK {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
			}

			messages := drainAuditMessages(controller.audit)
			if got, want := len(messages), 2+tt.wantAttrs; got != want {
				t.Fatalf("audit messages = %d, want %d: %#v", got, want, messages)
			}
			assertSSPAuditEnvelope(t, messages[0], SUBJECTRequest, body)
			assertSSPAuditEnvelope(t, messages[1], SUBJECTResponse, rr.Body.Bytes())
			for _, msg := range messages[2:] {
				if msg.subject != SUBJECTAttribute {
					t.Fatalf("attribute subject = %q", msg.subject)
				}
				var plus match.AttributePlus
				if err := json.Unmarshal(msg.data, &plus); err != nil {
					t.Fatal(err)
				}
				if plus.Source != "ssp" || plus.Contract != "pz-v1" {
					t.Fatalf("attribute source = %q/%q, want ssp/pz-v1", plus.Source, plus.Contract)
				}
			}

			var html []string
			if err := json.Unmarshal(rr.Body.Bytes(), &html); err != nil {
				t.Fatal(err)
			}
			if tt.wantFilled && html[0] == "" {
				t.Fatal("filled response should include markup")
			}
			if !tt.wantFilled && html[0] != "" {
				t.Fatalf("all-no-fill response = %#v, want empty markup", html)
			}
		})
	}
}

func TestServeSSPSmokeFixturesForDirectWebTagAndAppLikeAPI(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		code     string
	}{
		{name: "direct web tag", platform: "web", code: "pz-slot-100"},
		{name: "app-like api", platform: "sdk", code: "app-slot-100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			body := sspRequestBodyWithPlatform(t, tt.platform, 1, 10, []sspAdUnitSpec{
				{Code: tt.code, SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
			})

			rr := serveSSP(t, controller, body)
			if rr.Code != http.StatusOK {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
			}
			var html []string
			if err := json.Unmarshal(rr.Body.Bytes(), &html); err != nil {
				t.Fatal(err)
			}
			if len(html) != 1 || !strings.Contains(html[0], "one.html") {
				t.Fatalf("html = %#v, want one direct SSP markup response", html)
			}
		})
	}
}

func TestPublishBidAuditsKeepsADXRawRequestResponseAndMarksAttributes(t *testing.T) {
	controller := &Controller{audit: newAuditPublisher(nil, 10)}
	attr := &match.Attribute{UserID: "user"}
	if err := controller.publishBidAudits([]byte(`{"id":"adx"}`), []byte(`{"id":"rsp"}`), []bidAudit{{Attr: attr, One: match.RAdv{}, Elapsed: 3}}); err != nil {
		t.Fatal(err)
	}

	messages := drainAuditMessages(controller.audit)
	if len(messages) != 3 {
		t.Fatalf("audit messages = %d, want 3", len(messages))
	}
	if messages[0].subject != SUBJECTRequest || string(messages[0].data) != `{"id":"adx"}` {
		t.Fatalf("request audit = %q/%s, want raw ADX request", messages[0].subject, messages[0].data)
	}
	if messages[1].subject != SUBJECTResponse || string(messages[1].data) != `{"id":"rsp"}` {
		t.Fatalf("response audit = %q/%s, want raw ADX response", messages[1].subject, messages[1].data)
	}
	var plus match.AttributePlus
	if err := json.Unmarshal(messages[2].data, &plus); err != nil {
		t.Fatal(err)
	}
	if plus.Source != "adx" || plus.Contract != "openrtb" {
		t.Fatalf("attribute source = %q/%q, want adx/openrtb", plus.Source, plus.Contract)
	}
}

func TestServeSSPPartialFillReturnsEmptyString(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
		{Code: "unit-two", SlotID: 200, SizeID: match.SizeID2To1(320, 50), Banner: true, Floor: 100},
	})

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var html []string
	if err := json.Unmarshal(rr.Body.Bytes(), &html); err != nil {
		t.Fatal(err)
	}
	if len(html) != 2 || html[0] == "" || html[1] != "" {
		t.Fatalf("html = %#v, want first fill and second no-fill", html)
	}
}

func TestServeSSPRejectsMalformedOversizedAndInvalidInput(t *testing.T) {
	controller := newLocalBidPathController(t)
	site, slot := directTokens(t, 1, 10, 100, match.SizeID2To1(300, 250))
	_, wrongSizeSlot := directTokens(t, 1, 10, 100, match.SizeID2To1(320, 50))
	tests := []struct {
		name string
		body []byte
		want int
	}{
		{name: "malformed json", body: []byte("{"), want: http.StatusBadRequest},
		{name: "oversized", body: []byte(strings.Repeat(" ", maxBidRequestBodyBytes+1)), want: http.StatusRequestEntityTooLarge},
		{name: "missing slot", body: []byte(`{"site":"` + site + `","adUnits":[{"code":"x","mediaTypes":{"banner":{}}}]}`), want: http.StatusBadRequest},
		{name: "invalid site token", body: []byte(`{"site":"bad","adUnits":[{"code":"x","slot":"` + slot + `","mediaTypes":{"banner":{}}}]}`), want: http.StatusBadRequest},
		{name: "invalid slot token", body: []byte(`{"site":"` + site + `","adUnits":[{"code":"x","slot":"bad","mediaTypes":{"banner":{}}}]}`), want: http.StatusBadRequest},
		{name: "unknown publisher", body: sspRequestBody(t, 99, 10, []sspAdUnitSpec{{Code: "x", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true}}), want: http.StatusBadRequest},
		{name: "site slot mismatch", body: sspRequestBody(t, 1, 999, []sspAdUnitSpec{{Code: "x", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true}}), want: http.StatusBadRequest},
		{name: "slot size mismatch", body: []byte(`{"site":"` + site + `","adUnits":[{"code":"x","slot":"` + wrongSizeSlot + `","mediaTypes":{"banner":{}}}]}`), want: http.StatusBadRequest},
		{name: "missing media", body: []byte(`{"site":"` + site + `","adUnits":[{"code":"x","slot":"` + slot + `"}]}`), want: http.StatusBadRequest},
		{name: "unsupported media", body: []byte(`{"site":"` + site + `","adUnits":[{"code":"x","slot":"` + slot + `","mediaTypes":{"audio":{}}}]}`), want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := serveSSP(t, controller, tt.body)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", rr.Code, tt.want, rr.Body.String())
			}
		})
	}
}

func TestServeSSPDoesNotPublishAuditForInvalidToken(t *testing.T) {
	controller := newLocalBidPathController(t)
	controller.audit = newAuditPublisher(nil, 10)
	body := []byte(`{"site":"bad","adUnits":[{"code":"x","slot":"bad","mediaTypes":{"banner":{}}}]}`)

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
		t.Fatalf("audit messages = %#v, want none", messages)
	}
}

func TestOpenRTBFromSSPUsesHeaderMetadataAndValidatedSupply(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "debug-code", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1.25},
	})
	sspReq, err := ParseSSPRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/pz", nil)
	req.Host = "tag.example"
	req.RemoteAddr = "192.0.2.55:1234"
	req.Header.Set("User-Agent", "ua-test")
	req.Header.Set("Referer", "https://page.example/article")
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")

	bid, _, _, err := controller.openRTBFromSSP(req.Context(), req, sspReq)
	if err != nil {
		t.Fatal(err)
	}
	if bid.Device.UA != "ua-test" || bid.Device.IP != "203.0.113.9" {
		t.Fatalf("device = %#v", bid.Device)
	}
	if bid.Site.Domain != "example.com" || bid.Site.Ref != "https://page.example/article" || bid.Site.Name != "tag.example" {
		t.Fatalf("site = %#v", bid.Site)
	}
	if got := bid.Imp[0]; got.ID != "debug-code" || got.TagID != "slot-one" || got.Banner == nil || *got.Banner.W != 300 || *got.Banner.H != 250 {
		t.Fatalf("imp = %#v", got)
	}
}

func TestOpenRTBFromSSPUsesValidCookieUserAndCookieAbsenceFallsBack(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "debug-code", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1.25},
	})
	sspReq, err := ParseSSPRequest(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/pz", nil)
	req.RemoteAddr = "192.0.2.55:1234"
	req.Header.Set("User-Agent", "ua-test")
	bid, _, _, err := controller.openRTBFromSSP(req.Context(), req, sspReq)
	if err != nil {
		t.Fatal(err)
	}
	if bid.User.ID != "" || bid.User.BuyerUID != "" {
		t.Fatalf("user = %#v, want empty user so IP+UA fallback remains active", bid.User)
	}
	attr, err := match.NewAttributeForImp(req.Context(), nil, bid, 0, controller.local.pubByID[1].Pub, timeNowForTest(), "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if attr.UserID == "" {
		t.Fatal("attribute user should fall back to IP+UA when cookie is absent")
	}

	rr := httptest.NewRecorder()
	if got := controller.resolveSSPUserCookie(rr, req); got != "" {
		t.Fatalf("missing cookie resolved user = %q, want empty current-request user", got)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sspUserCookieName || !validSSPUserCookie(cookies[0].Value) {
		t.Fatalf("Set-Cookie = %#v, want valid %s cookie", cookies, sspUserCookieName)
	}

	cookieValue := "valid_cookie_user_123"
	req.AddCookie(&http.Cookie{Name: sspUserCookieName, Value: cookieValue})
	bid, _, _, err = controller.openRTBFromSSP(req.Context(), req, sspReq, controller.resolveSSPUserCookie(nil, req))
	if err != nil {
		t.Fatal(err)
	}
	if bid.User.ID != cookieValue || bid.User.BuyerUID != cookieValue {
		t.Fatalf("user = %#v, want valid cookie user", bid.User)
	}
}

func TestServeSSPMarkupTrackersFeedWinLoss(t *testing.T) {
	controller := newLocalBidPathController(t)
	var published []WinLoss
	controller.publishWinLossFunc = func(data []byte) error {
		var wl WinLoss
		if err := json.Unmarshal(data, &wl); err != nil {
			return err
		}
		published = append(published, wl)
		return nil
	}
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var htmlResponses []string
	if err := json.Unmarshal(rr.Body.Bytes(), &htmlResponses); err != nil {
		t.Fatal(err)
	}
	impURL := firstTrackerURL(t, htmlResponses[0], "/imp")
	clkURL := firstTrackerURL(t, htmlResponses[0], "/clk")

	for _, raw := range []string{impURL, clkURL} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
		trackRR := httptest.NewRecorder()
		controller.ServeWinLoss(trackRR, req)
		if u.Path == "/clk" {
			if trackRR.Code != http.StatusFound {
				t.Fatalf("%s status = %d, want redirect: %s", u.Path, trackRR.Code, trackRR.Body.String())
			}
			continue
		}
		if trackRR.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want no content: %s", u.Path, trackRR.Code, trackRR.Body.String())
		}
	}
	if len(published) != 2 {
		t.Fatalf("published winloss = %d, want 2", len(published))
	}
	if published[0].Status != StatusTrackImp || published[1].Status != StatusTrackClk {
		t.Fatalf("published statuses = %v/%v, want imp/click", published[0].Status, published[1].Status)
	}
	for _, wl := range published {
		if wl.RPub.PubID != 1 || wl.RPub.SiteID != 10 || wl.RPub.SlotID != 100 || wl.RAdv.CreativeID != 10000 || wl.RAdv.Cost <= 0 {
			t.Fatalf("winloss record = %#v, want SSP publisher and demand ids", wl)
		}
	}
}

type sspAdUnitSpec struct {
	Code   string
	SlotID uint32
	SizeID uint32
	Banner bool
	Video  bool
	Native bool
	Floor  float64
}

func sspRequestBody(t *testing.T, pubID, siteID uint32, specs []sspAdUnitSpec) []byte {
	t.Helper()
	return sspRequestBodyWithPlatform(t, "", pubID, siteID, specs)
}

func sspRequestBodyWithPlatform(t *testing.T, platform string, pubID, siteID uint32, specs []sspAdUnitSpec) []byte {
	t.Helper()
	site, err := acl.PackDirectToken(pubID, siteID)
	if err != nil {
		t.Fatal(err)
	}
	units := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		slot, err := acl.PackDirectToken(spec.SlotID, spec.SizeID)
		if err != nil {
			t.Fatal(err)
		}
		media := map[string]any{}
		switch {
		case spec.Video:
			media["video"] = map[string]any{"playerSize": []int{1, 1}}
		case spec.Native:
			media["native"] = map[string]any{"image": []int{1, 1}, "title": true}
		default:
			media["banner"] = map[string]any{"size": []int{1, 1}}
		}
		units = append(units, map[string]any{
			"code":       spec.Code,
			"slot":       slot,
			"floor":      spec.Floor,
			"mediaTypes": media,
		})
	}
	payload := map[string]any{
		"id":      "ssp-test",
		"site":    site,
		"adUnits": units,
	}
	if platform != "" {
		payload["platform"] = platform
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func directTokens(t *testing.T, pubID, siteID, slotID, sizeID uint32) (string, string) {
	t.Helper()
	site, err := acl.PackDirectToken(pubID, siteID)
	if err != nil {
		t.Fatal(err)
	}
	slot, err := acl.PackDirectToken(slotID, sizeID)
	if err != nil {
		t.Fatal(err)
	}
	return site, slot
}

func serveSSP(t *testing.T, controller *Controller, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	req.RemoteAddr = "203.0.113.1:4567"
	rr := httptest.NewRecorder()
	controller.ServeSSP(rr, req)
	return rr
}

func drainAuditMessages(publisher *auditPublisher) []auditMessage {
	var messages []auditMessage
	for {
		select {
		case msg := <-publisher.queue:
			messages = append(messages, msg)
		default:
			return messages
		}
	}
}

func assertSSPAuditEnvelope(t *testing.T, msg auditMessage, subject string, wantPayload []byte) {
	t.Helper()
	if msg.subject != subject {
		t.Fatalf("subject = %q, want %q", msg.subject, subject)
	}
	var envelope auditEnvelope
	if err := json.Unmarshal(msg.data, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Source != "ssp" || envelope.Contract != "pz-v1" {
		t.Fatalf("envelope source = %q/%q, want ssp/pz-v1", envelope.Source, envelope.Contract)
	}
	if string(envelope.Payload) != string(wantPayload) {
		t.Fatalf("payload = %s, want %s", envelope.Payload, wantPayload)
	}
}

func firstTrackerURL(t *testing.T, markup, path string) string {
	t.Helper()
	unescaped := html.UnescapeString(markup)
	idx := strings.Index(unescaped, "https://dsp.example"+path+"?")
	if idx < 0 {
		t.Fatalf("markup missing %s tracker: %s", path, markup)
	}
	end := len(unescaped)
	for _, sep := range []string{`"`, `'`, "<", ">"} {
		if next := strings.Index(unescaped[idx:], sep); next >= 0 && idx+next < end {
			end = idx + next
		}
	}
	return unescaped[idx:end]
}

func timeNowForTest() time.Time {
	return time.Now()
}

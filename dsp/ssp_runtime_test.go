package dsp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	data, err := json.Marshal(map[string]any{
		"id":      "ssp-test",
		"site":    site,
		"adUnits": units,
	})
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

package dsp

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/publisherauth"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestOpenRTBImpFromSSPUsesTokenSizeForSupportedMedia(t *testing.T) {
	unit := SSPValidatedUnit{
		AccountingVersion:  accounting.ExactMoneyContract,
		Code:               "code",
		SlotStr:            "slot",
		ConfiguredFloor:    9.5,
		ConfiguredFloorCPM: 1_500_000,
		RPub:               match.RPub{SizeID: match.SizeID2To1(300, 250)},
	}
	tests := []struct {
		name   string
		adUnit SSPAdUnit
		want   string
	}{
		{
			name:   "banner",
			adUnit: SSPAdUnit{Code: "banner", Floor: 1, MediaTypes: SSPMediaTypes{Banner: &SSPBanner{Size: []uint16{300, 250}}}},
			want:   "banner",
		},
		{
			name:   "video",
			adUnit: SSPAdUnit{Code: "video", Floor: 1, MediaTypes: SSPMediaTypes{Video: &SSPVideo{PlayerSize: []uint16{300, 250}}}},
			want:   "video",
		},
		{
			name:   "native",
			adUnit: SSPAdUnit{Code: "native", Floor: 1, MediaTypes: SSPMediaTypes{Native: &SSPNative{Image: []uint16{300, 250}, Title: true}}},
			want:   "native",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp, err := openRTBImpFromSSPUnit(tt.adUnit, unit, 0)
			if err != nil {
				t.Fatal(err)
			}
			if imp.BidFloor != 1.5 {
				t.Fatalf("bid floor = %v, want configured floor 1.5", imp.BidFloor)
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

func TestOpenRTBImpFromSSPRejectsUnknownFloorAccountingVersion(t *testing.T) {
	unit := SSPValidatedUnit{
		AccountingVersion: "future-version",
		ConfiguredFloor:   1.25,
		RPub:              match.RPub{SizeID: match.SizeID2To1(300, 250)},
	}
	adUnit := SSPAdUnit{Code: "banner", MediaTypes: SSPMediaTypes{Banner: &SSPBanner{Size: []uint16{300, 250}}}}
	if _, err := openRTBImpFromSSPUnit(adUnit, unit, 0); err == nil {
		t.Fatal("unknown floor accounting version was accepted")
	}
}

func TestOpenRTBFromSSPUsesGreaterOfRequestAndConfiguredFloor(t *testing.T) {
	controller := newLocalBidPathController(t)
	controller.local.pubByID[1].SlotFloors[10][100] = 1.75
	for _, test := range []struct {
		name         string
		requestFloor float64
		want         float64
	}{
		{name: "configured floor wins", requestFloor: 1.25, want: 1.75},
		{name: "higher valid request floor wins", requestFloor: 2.25, want: 2.25},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := sspRequestBodyWithPlatform(t, "browser", 1, 10, []sspAdUnitSpec{{
				Code: "floor-unit", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: test.requestFloor,
			}})
			req, err := ParseSSPRequest(body)
			if err != nil {
				t.Fatal(err)
			}
			httpRequest := httptest.NewRequest(http.MethodPost, "/pz", nil)
			bid, _, _, err := controller.openRTBFromSSP(httpRequest.Context(), httpRequest, req)
			if err != nil {
				t.Fatal(err)
			}
			if got := bid.Imp[0].BidFloor; got != test.want {
				t.Fatalf("bid floor = %v, want %v", got, test.want)
			}
		})
	}
}

func TestServeSSPRejectsCommercialInventoryMismatchBeforeSideEffects(t *testing.T) {
	site, slot := directTokens(t, 1, 10, 100, match.SizeID2To1(300, 250))
	appSite, _ := directTokens(t, 1, 11, 100, match.SizeID2To1(300, 250))
	tests := []struct {
		name        string
		body        []byte
		stalePolicy bool
	}{
		{
			name: "negative floor",
			body: []byte(`{"platform":"browser","site":"` + site + `","adUnits":[{"code":"unit","slot":"` + slot + `","floor":-1,"mediaTypes":{"banner":{"size":[300,250]}}}]}`),
		},
		{
			name: "declared size mismatch",
			body: []byte(`{"platform":"browser","site":"` + site + `","adUnits":[{"code":"unit","slot":"` + slot + `","mediaTypes":{"banner":{"size":[320,50]}}}]}`),
		},
		{
			name: "ambiguous media",
			body: []byte(`{"platform":"browser","site":"` + site + `","adUnits":[{"code":"unit","slot":"` + slot + `","mediaTypes":{"banner":{"size":[300,250]},"video":{"playerSize":[300,250]}}}]}`),
		},
		{
			name: "browser token points to app inventory",
			body: []byte(`{"platform":"browser","site":"` + appSite + `","adUnits":[{"code":"unit","slot":"` + slot + `","mediaTypes":{"banner":{"size":[300,250]}}}]}`),
		},
		{
			name:        "pre-P01 cache generation",
			body:        []byte(`{"platform":"browser","site":"` + site + `","adUnits":[{"code":"unit","slot":"` + slot + `","mediaTypes":{"banner":{"size":[300,250]}}}]}`),
			stalePolicy: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			controller.audit = newAuditPublisher(nil, 10)
			runtime := &recordingMiddlemanRuntime{}
			enableSSPMiddleman(controller, true, runtime)
			if test.stalePolicy {
				controller.local.pubByID[1].SlotFloors = nil
			}
			rr := serveSSPWithHeaders(t, controller, test.body, map[string]string{"Origin": "https://example.com"})
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
			}
			if runtime.calls != 0 {
				t.Fatalf("middleman calls = %d, want 0", runtime.calls)
			}
			if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
				t.Fatalf("audit messages = %#v, want none", messages)
			}
		})
	}
}

func TestServeSSPVersionedInventoryTokenMigration(t *testing.T) {
	allowCodec := directTokenV2TestCodec(t, true)
	denyCodec := directTokenV2TestCodec(t, false)
	v2Body := sspRequestBodyWithV2Tokens(t, allowCodec, 1, 10, []sspAdUnitSpec{{
		Code: "v2-unit", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1,
	}})
	legacyBody := sspRequestBody(t, 1, 10, []sspAdUnitSpec{{
		Code: "legacy-unit", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1,
	}})

	tests := []struct {
		name         string
		codec        *acl.DirectTokenCodec
		body         []byte
		wantStatus   int
		wantOutcome  string
		missingCodec bool
	}{
		{name: "v2 accepted", codec: allowCodec, body: v2Body, wantStatus: http.StatusOK, wantOutcome: "v2_accepted"},
		{name: "legacy accepted during overlap", codec: allowCodec, body: legacyBody, wantStatus: http.StatusOK, wantOutcome: "legacy_accepted"},
		{name: "legacy explicitly disabled", codec: denyCodec, body: legacyBody, wantStatus: http.StatusBadRequest, wantOutcome: "legacy_disabled"},
		{name: "enabled policy without codec fails closed", body: legacyBody, wantStatus: http.StatusBadRequest, wantOutcome: "legacy_disabled", missingCodec: true},
		{name: "tampered v2 rejected", codec: allowCodec, body: tamperSSPSlotToken(t, v2Body), wantStatus: http.StatusBadRequest, wantOutcome: "v2_rejected"},
		{name: "mixed versions rejected", codec: allowCodec, body: replaceSSPSlotWithLegacy(t, v2Body, 100, match.SizeID2To1(300, 250)), wantStatus: http.StatusBadRequest, wantOutcome: "mixed_rejected"},
		{name: "active cache still authoritative", codec: allowCodec, body: sspRequestBodyWithV2Tokens(t, allowCodec, 1, 10, []sspAdUnitSpec{{Code: "inactive", SlotID: 999, SizeID: match.SizeID2To1(300, 250), Banner: true}}), wantStatus: http.StatusBadRequest, wantOutcome: "v2_rejected"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			controller.directTokens = test.codec
			if test.missingCodec {
				controller.C.DirectSSPTokens.Enabled = true
			}
			runtime := &recordingMiddlemanRuntime{}
			enableSSPMiddleman(controller, true, runtime)
			if test.wantStatus != http.StatusOK {
				controller.audit = newAuditPublisher(nil, 10)
			}
			before := expvarMapInt64(metricSSPInventoryTokenOutcomes, test.wantOutcome)
			rr := serveSSP(t, controller, test.body)
			if rr.Code != test.wantStatus {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, test.wantStatus, rr.Body.String())
			}
			if got := expvarMapInt64(metricSSPInventoryTokenOutcomes, test.wantOutcome) - before; got != 1 {
				t.Fatalf("%s metric delta = %d, want 1", test.wantOutcome, got)
			}
			if test.wantStatus != http.StatusOK {
				if runtime.calls != 0 {
					t.Fatalf("middleman calls = %d, want 0", runtime.calls)
				}
				if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
					t.Fatalf("audit messages = %#v, want none", messages)
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

func TestServeSSPPolicyAcceptsMatchingOriginOrReferer(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
	}{
		{
			name:    "matching origin",
			headers: map[string]string{"Origin": "https://example.com"},
		},
		{
			name:    "matching referer without origin",
			headers: map[string]string{"Referer": "https://example.com/article/1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			body := sspRequestBodyWithPlatform(t, "browser", 1, 10, []sspAdUnitSpec{
				{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
			})

			rr := serveSSPWithHeaders(t, controller, body, tt.headers)
			if rr.Code != http.StatusOK {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
			}
		})
	}
}

func TestServeSSPPolicyRejectsBrowserHeaderFailures(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		headers  map[string]string
	}{
		{
			name:    "mismatched origin",
			headers: map[string]string{"Origin": "https://attacker.example"},
		},
		{
			name:    "mismatched referer",
			headers: map[string]string{"Referer": "https://attacker.example/page"},
		},
		{
			name:    "origin null",
			headers: map[string]string{"Origin": "null"},
		},
		{
			name:    "malformed origin",
			headers: map[string]string{"Origin": "://bad"},
		},
		{
			name: "no browser headers",
		},
		{
			name:     "sdk mismatched origin",
			platform: "sdk",
			headers:  map[string]string{"Origin": "https://attacker.example"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			body := sspRequestBodyWithPlatform(t, tt.platform, 1, 10, []sspAdUnitSpec{
				{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
			})

			rr := serveSSPWithHeaders(t, controller, body, tt.headers)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusForbidden, rr.Body.String())
			}
		})
	}
}

func TestServeSSPPolicyAllowsSDKWithoutBrowserHeaders(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBodyWithPlatform(t, "sdk", 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})

	before := expvarMapInt64(metricSSPPublisherAuthOutcomes, "compatibility")
	rr := serveSSPWithHeaders(t, controller, body, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("cookies = %#v, want none for sdk traffic", cookies)
	}
	if got := expvarMapInt64(metricSSPPublisherAuthOutcomes, "compatibility") - before; got != 1 {
		t.Fatalf("compatibility auth outcomes = %d, want 1", got)
	}
}

func TestServeSSPAuthenticatedSDKRequiresFreshOneUseBodyProof(t *testing.T) {
	controller := newLocalBidPathController(t)
	service, privateCredential, now, closeRedis := publisherAuthRuntimeService(t, 1, 11)
	defer closeRedis()
	controller.C.DirectSSPAuth = publisherauth.Config{Enabled: true}.WithDefaults()
	controller.publisherAuth = service
	body := sspRequestBodyWithPlatform(t, "sdk", 1, 11, []sspAdUnitSpec{
		{Code: "authenticated-sdk", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})
	headers := signedPublisherAuthHeaders(t, privateCredential, now, 0x61, body)

	acceptedBefore := expvarMapInt64(metricSSPPublisherAuthOutcomes, "accepted")
	replayBefore := expvarMapInt64(metricSSPPublisherAuthOutcomes, "replay_rejected")
	first := serveSSPWithHeaders(t, controller, body, headers)
	if first.Code != http.StatusOK {
		t.Fatalf("authenticated SDK status = %d: %s", first.Code, first.Body.String())
	}
	if cookies := first.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("authenticated SDK cookies = %#v", cookies)
	}
	second := serveSSPWithHeaders(t, controller, body, headers)
	if second.Code != http.StatusUnauthorized || strings.Contains(second.Body.String(), "replay") {
		t.Fatalf("replay response = %d %q", second.Code, second.Body.String())
	}
	if got := expvarMapInt64(metricSSPPublisherAuthOutcomes, "accepted") - acceptedBefore; got != 1 {
		t.Fatalf("accepted auth outcomes = %d, want 1", got)
	}
	if got := expvarMapInt64(metricSSPPublisherAuthOutcomes, "replay_rejected") - replayBefore; got != 1 {
		t.Fatalf("replay auth outcomes = %d, want 1", got)
	}
}

func TestServeSSPAuthenticationFailuresPrecedeAuctionSideEffects(t *testing.T) {
	body := sspRequestBodyWithPlatform(t, "sdk", 1, 11, []sspAdUnitSpec{
		{Code: "authenticated-sdk", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})
	tests := []struct {
		name           string
		credentialSite uint64
		headers        func(*testing.T, string, time.Time) map[string]string
		mutateBody     bool
		closeRedis     bool
		wantStatus     int
		wantOutcome    string
	}{
		{name: "missing proof", credentialSite: 11, wantStatus: http.StatusUnauthorized, wantOutcome: "required_rejected"},
		{name: "stale proof", credentialSite: 11, headers: func(t *testing.T, credential string, now time.Time) map[string]string {
			return signedPublisherAuthHeaders(t, credential, now.Add(-301*time.Second), 0x62, body)
		}, wantStatus: http.StatusUnauthorized, wantOutcome: "stale_rejected"},
		{name: "body mismatch", credentialSite: 11, headers: func(t *testing.T, credential string, now time.Time) map[string]string {
			return signedPublisherAuthHeaders(t, credential, now, 0x63, body)
		}, mutateBody: true, wantStatus: http.StatusUnauthorized, wantOutcome: "invalid_rejected"},
		{name: "cross App scope", credentialSite: 12, headers: func(t *testing.T, credential string, now time.Time) map[string]string {
			return signedPublisherAuthHeaders(t, credential, now, 0x64, body)
		}, wantStatus: http.StatusUnauthorized, wantOutcome: "scope_rejected"},
		{name: "replay store unavailable", credentialSite: 11, headers: func(t *testing.T, credential string, now time.Time) map[string]string {
			return signedPublisherAuthHeaders(t, credential, now, 0x65, body)
		}, closeRedis: true, wantStatus: http.StatusServiceUnavailable, wantOutcome: "dependency_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			service, privateCredential, now, closeRedis := publisherAuthRuntimeService(t, 1, test.credentialSite)
			controller.C.DirectSSPAuth = publisherauth.Config{Enabled: true}.WithDefaults()
			controller.publisherAuth = service
			runtime := &recordingMiddlemanRuntime{}
			enableSSPMiddleman(controller, true, runtime)
			controller.audit = newAuditPublisher(nil, 10)
			headers := map[string]string(nil)
			if test.headers != nil {
				headers = test.headers(t, privateCredential, now)
			}
			requestBody := body
			if test.mutateBody {
				requestBody = bytes.Replace(body, []byte("authenticated-sdk"), []byte("authenticated-bad"), 1)
			}
			if test.closeRedis {
				closeRedis()
			}
			before := expvarMapInt64(metricSSPPublisherAuthOutcomes, test.wantOutcome)
			rr := serveSSPWithHeaders(t, controller, requestBody, headers)
			if !test.closeRedis {
				closeRedis()
			}
			if rr.Code != test.wantStatus || strings.Contains(strings.ToLower(rr.Body.String()), "credential") {
				t.Fatalf("auth failure = %d %q, want %d generic", rr.Code, rr.Body.String(), test.wantStatus)
			}
			if runtime.calls != 0 {
				t.Fatalf("middleman calls = %d", runtime.calls)
			}
			if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
				t.Fatalf("audit messages = %#v", messages)
			}
			if got := expvarMapInt64(metricSSPPublisherAuthOutcomes, test.wantOutcome) - before; got != 1 {
				t.Fatalf("%s auth outcomes = %d, want 1", test.wantOutcome, got)
			}
		})
	}
}

func TestServeSSPAuthenticatedSDKCannotOverrideIndependentEnforcement(t *testing.T) {
	tests := []struct {
		name       string
		mutateBody func(*testing.T, []byte) []byte
		headers    map[string]string
		wantStatus int
		wantAuth   string
	}{
		{
			name: "App identity",
			mutateBody: func(t *testing.T, body []byte) []byte {
				return sspRequestBodyWithFields(t, body, map[string]any{
					"app": map[string]any{"bundle": "attacker.example"},
				})
			},
			wantStatus: http.StatusBadRequest,
			wantAuth:   "accepted",
		},
		{
			name: "active inventory",
			mutateBody: func(t *testing.T, body []byte) []byte {
				var payload map[string]any
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatal(err)
				}
				unit := payload["adUnits"].([]any)[0].(map[string]any)
				slot, err := acl.PackDirectToken(999, match.SizeID2To1(300, 250))
				if err != nil {
					t.Fatal(err)
				}
				unit["slot"] = slot
				out, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				return out
			},
			wantStatus: http.StatusBadRequest,
			wantAuth:   "inventory_rejected",
		},
		{
			name:       "browser provenance headers",
			headers:    map[string]string{"Origin": "https://attacker.example"},
			wantStatus: http.StatusForbidden,
			wantAuth:   "policy_rejected",
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			service, privateCredential, now, closeStores := publisherAuthRuntimeService(t, 1, 11)
			defer closeStores()
			controller.C.DirectSSPAuth = publisherauth.Config{Enabled: true}.WithDefaults()
			controller.publisherAuth = service
			controller.audit = newAuditPublisher(nil, 10)
			runtime := &recordingMiddlemanRuntime{}
			enableSSPMiddleman(controller, true, runtime)

			body := sspRequestBodyWithPlatform(t, "sdk", 1, 11, []sspAdUnitSpec{
				{Code: "authenticated-sdk", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
			})
			if test.mutateBody != nil {
				body = test.mutateBody(t, body)
			}
			headers := signedPublisherAuthHeaders(t, privateCredential, now, byte(0x70+index), body)
			for name, value := range test.headers {
				headers[name] = value
			}

			before := expvarMapInt64(metricSSPPublisherAuthOutcomes, test.wantAuth)
			rr := serveSSPWithHeaders(t, controller, body, headers)
			if rr.Code != test.wantStatus || rr.Body.String() != http.StatusText(test.wantStatus)+"\n" {
				t.Fatalf("rejection = %d %q, want generic %d", rr.Code, rr.Body.String(), test.wantStatus)
			}
			if runtime.calls != 0 {
				t.Fatalf("middleman calls = %d", runtime.calls)
			}
			if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
				t.Fatalf("audit messages = %#v", messages)
			}
			if cookies := rr.Result().Cookies(); len(cookies) != 0 {
				t.Fatalf("cookies = %#v", cookies)
			}
			if got := expvarMapInt64(metricSSPPublisherAuthOutcomes, test.wantAuth) - before; got != 1 {
				t.Fatalf("%s auth outcomes = %d, want 1", test.wantAuth, got)
			}
		})
	}
}

func TestServeSSPPublisherCacheFailureIsGenericAndRetryable(t *testing.T) {
	controller := newLocalBidPathController(t)
	controller.C.IsLocal = false
	controller.Redis = nil
	controller.audit = newAuditPublisher(nil, 10)
	runtime := &recordingMiddlemanRuntime{}
	enableSSPMiddleman(controller, true, runtime)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})

	rr := serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://example.com"})
	if rr.Code != http.StatusServiceUnavailable || rr.Body.String() != "Service Unavailable\n" {
		t.Fatalf("cache failure = %d %q, want generic 503", rr.Code, rr.Body.String())
	}
	if runtime.calls != 0 {
		t.Fatalf("middleman calls = %d", runtime.calls)
	}
	if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
		t.Fatalf("audit messages = %#v", messages)
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("cookies = %#v", cookies)
	}
}

func TestServeSSPBrowserRequestWithoutConsentDoesNotSetCookie(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBodyWithPlatform(t, "browser", 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})

	rr := serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://example.com"})
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("Set-Cookie = %#v, want no identity cookie without a privacy grant", cookies)
	}
}

func TestOpenRTBFromSSPIgnoresSDKCookieAndKeepsIPUAFallback(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBodyWithPlatform(t, "sdk", 1, 10, []sspAdUnitSpec{
		{Code: "debug-code", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1.25},
	})
	sspReq, err := ParseSSPRequest(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/pz", nil)
	req.RemoteAddr = "192.0.2.55:1234"
	req.Header.Set("User-Agent", "sdk-ua-test")
	req.AddCookie(&http.Cookie{Name: sspUserCookieName, Value: "valid_cookie_user_123"})

	rr := httptest.NewRecorder()
	cookieUserID := controller.resolveSSPUserCookieForPlatform(rr, req, sspReq.Platform)
	if cookieUserID != "" {
		t.Fatalf("sdk cookie resolved user = %q, want ignored", cookieUserID)
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("sdk Set-Cookie = %#v, want none", cookies)
	}

	bid, _, _, err := controller.openRTBFromSSP(req.Context(), req, sspReq, "valid_cookie_user_123")
	if err != nil {
		t.Fatal(err)
	}
	if bid.User.ID != "" || bid.User.BuyerUID != "" {
		t.Fatalf("user = %#v, want empty user for sdk cookie", bid.User)
	}
	attr, err := match.NewAttributeForImp(req.Context(), nil, bid, 0, controller.local.pubByID[1].Pub, timeNowForTest(), "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if attr.UserID == "" {
		t.Fatal("sdk attribute user should fall back to IP+UA when cookie is ignored")
	}
}

func TestOpenRTBFromSSPSDKSynthesizesAppDeviceUserAndNoCookie(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBodyWithFields(t, sspRequestBodyWithPlatform(t, "sdk", 1, 10, []sspAdUnitSpec{
		{Code: "sdk-unit", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	}), map[string]any{
		"app": map[string]any{
			"id":     "app.example.com",
			"bundle": "app.example.com",
			"domain": "app.example.com",
			"name":   "Example App",
		},
		"device": map[string]any{
			"ifa":     "ifa-sdk-123",
			"didmd5":  "did-md5",
			"dpidmd5": "dpid-md5",
		},
		"user": map[string]any{
			"id":       "user-sdk-123",
			"buyeruid": "buyer-sdk-123",
		},
	})
	sspReq, err := ParseSSPRequest(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/pz", nil)
	req.RemoteAddr = "192.0.2.55:1234"
	req.Header.Set("User-Agent", "sdk-header-agent")
	req.AddCookie(&http.Cookie{Name: sspUserCookieName, Value: "valid_cookie_user_123"})
	rr := httptest.NewRecorder()
	if got := controller.resolveSSPUserCookieForPlatform(rr, req, sspReq.Platform); got != "" {
		t.Fatalf("sdk cookie resolved user = %q, want ignored", got)
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("sdk Set-Cookie = %#v, want none", cookies)
	}

	bid, _, _, err := controller.openRTBFromSSP(req.Context(), req, sspReq, "valid_cookie_user_123")
	if err != nil {
		t.Fatal(err)
	}
	if bid.Site != nil || bid.App == nil {
		t.Fatalf("site/app = %#v/%#v, want app-only SDK request", bid.Site, bid.App)
	}
	if bid.App.ID != "app.example.com" || bid.App.Bundle != "app.example.com" || bid.App.Domain != "app.example.com" || bid.App.Name != "Example App" {
		t.Fatalf("app = %#v", bid.App)
	}
	if bid.Device.IFA != "ifa-sdk-123" || bid.Device.DIDMD5 != "did-md5" || bid.Device.DPIDMD5 != "dpid-md5" || bid.Device.UA != "sdk-header-agent" || bid.Device.IP != "192.0.2.55" {
		t.Fatalf("device = %#v", bid.Device)
	}
	if bid.User.ID != "user-sdk-123" || bid.User.BuyerUID != "buyer-sdk-123" {
		t.Fatalf("user = %#v", bid.User)
	}
	attr, err := match.NewAttributeForImp(req.Context(), nil, bid, 0, controller.local.pubByID[1].Pub, timeNowForTest(), "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if attr.UserID != "buyer-sdk-123" || attr.IFA != "ifa-sdk-123" || !attr.IsApp {
		t.Fatalf("attr identity = user %q ifa %q isApp %v", attr.UserID, attr.IFA, attr.IsApp)
	}
}

func TestServeSSPRejectsSDKAppIdentityMismatch(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBodyWithFields(t, sspRequestBodyWithPlatform(t, "sdk", 1, 10, []sspAdUnitSpec{
		{Code: "sdk-unit", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	}), map[string]any{
		"app": map[string]any{"bundle": "other.example"},
	})

	rr := serveSSPWithHeaders(t, controller, body, nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestServeSSPPolicyRejectionDoesNotSetCookieOrPublishAudits(t *testing.T) {
	controller := newLocalBidPathController(t)
	controller.audit = newAuditPublisher(nil, 10)
	body := sspRequestBodyWithPlatform(t, "browser", 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})

	rr := serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://attacker.example"})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("cookies = %#v, want none", cookies)
	}
	if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
		t.Fatalf("audit messages = %#v, want none", messages)
	}
}

func TestServeSSPRejectsDuplicateAdUnitCodeBeforeSideEffects(t *testing.T) {
	controller := newLocalBidPathController(t)
	controller.audit = newAuditPublisher(nil, 10)
	runtime := &recordingMiddlemanRuntime{}
	enableSSPMiddleman(controller, true, runtime)
	body := sspRequestBodyWithPlatform(t, "browser", 1, 10, []sspAdUnitSpec{
		{Code: "dup-code", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
		{Code: "dup-code", SlotID: 200, SizeID: match.SizeID2To1(320, 50), Banner: true, Floor: 1},
	})

	rr := serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://example.com"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if cookies := rr.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("cookies = %#v, want none", cookies)
	}
	if messages := drainAuditMessages(controller.audit); len(messages) != 0 {
		t.Fatalf("audit messages = %#v, want none", messages)
	}
	if runtime.calls != 0 {
		t.Fatalf("middleman calls = %d, want 0", runtime.calls)
	}
}

func TestServeSSPRejectsEmptyAdUnitCodesBeforeAuction(t *testing.T) {
	controller := newLocalBidPathController(t)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
		{SlotID: 200, SizeID: match.SizeID2To1(320, 50), Banner: true, Floor: 1},
	})

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
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
		{name: "direct web tag", platform: "browser", code: "pz-slot-100"},
		{name: "app-like api", platform: "sdk", code: "app-slot-100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			body := sspRequestBodyWithPlatform(t, tt.platform, 1, 10, []sspAdUnitSpec{
				{Code: tt.code, SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
			})

			headers := map[string]string{"Origin": "https://example.com"}
			if tt.platform == "sdk" {
				headers = nil
			}
			rr := serveSSPWithHeaders(t, controller, body, headers)
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

func TestPublishBidAuditsKeepsADXShapeAndMarksAttributes(t *testing.T) {
	controller := &Controller{audit: newAuditPublisher(nil, 10)}
	attr := &match.Attribute{UserID: "user"}
	if err := controller.publishBidAudits([]byte(`{"id":"adx"}`), []byte(`{"id":"rsp"}`), []bidAudit{{Attr: attr, One: match.RAdv{}, Elapsed: 3 * time.Millisecond}}); err != nil {
		t.Fatal(err)
	}

	messages := drainAuditMessages(controller.audit)
	if len(messages) != 3 {
		t.Fatalf("audit messages = %d, want 3", len(messages))
	}
	var requestAudit openrtb2.BidRequest
	if messages[0].subject != SUBJECTRequest || json.Unmarshal(messages[0].data, &requestAudit) != nil || requestAudit.ID != "adx" {
		t.Fatalf("request audit = %q/%s, want typed ADX request", messages[0].subject, messages[0].data)
	}
	var responseAudit openrtb2.BidResponse
	if messages[1].subject != SUBJECTResponse || json.Unmarshal(messages[1].data, &responseAudit) != nil || responseAudit.ID != "rsp" {
		t.Fatalf("response audit = %q/%s, want typed ADX response", messages[1].subject, messages[1].data)
	}
	var plus match.AttributePlus
	if err := json.Unmarshal(messages[2].data, &plus); err != nil {
		t.Fatal(err)
	}
	if plus.Source != "adx" || plus.Contract != "openrtb" {
		t.Fatalf("attribute source = %q/%q, want adx/openrtb", plus.Source, plus.Contract)
	}
	var raw map[string]any
	if err := json.Unmarshal(messages[2].data, &raw); err != nil {
		t.Fatal(err)
	}
	if got, ok := raw["elapsed"].(float64); !ok || got != 3 {
		t.Fatalf("attribute elapsed = %#v, want millisecond number 3", raw["elapsed"])
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

func TestServeSSPJSONResponseReturnsFillObjectsAndNative(t *testing.T) {
	controller := newLocalBidPathController(t)
	nativeContent, err := match.MarshalNativeCreativeV1(match.NativeCreativeV1{
		Version: "1", Title: "native", Description: "description", CTA: "open",
		MainImageURL: "https://cdn.example/native.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeCreativeSnapshot(t, controller.C.Spread, 10000, &match.Creative{
		CreativeName: "native", CreativeContent: nativeContent, Landing: "https://advertiser.example/one",
		SizeID: match.SizeID2To1(300, 250), MediaType: match.CreativeMediaNative,
	})
	if err := controller.ReloadLocalStaticCache(); err != nil {
		t.Fatal(err)
	}
	body := sspRequestBodyWithFields(t, sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "native-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Native: true, Floor: 1},
		{Code: "unit-two", SlotID: 200, SizeID: match.SizeID2To1(320, 50), Banner: true, Floor: 100},
	}), map[string]any{"responseFormat": "json"})

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var response []sspJSONAdUnitResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response) != 2 {
		t.Fatalf("json response len = %d, want 2", len(response))
	}
	if !response[0].Filled || response[0].AdM == "" || len(response[0].Native) == 0 || response[0].ImpressionURL == "" || response[0].ClickURL == "" {
		t.Fatalf("filled native response = %#v", response[0])
	}
	if response[0].Price <= 0 || response[0].Currency != "USD" || response[0].CampaignID != "10" || response[0].CreativeID != "10000" || response[0].AdID != "10000" || response[0].Width != 300 || response[0].Height != 250 {
		t.Fatalf("filled metadata response = %#v", response[0])
	}
	if response[1].Filled || response[1].AdM != "" || len(response[1].Native) != 0 {
		t.Fatalf("no-fill response = %#v, want filled false only", response[1])
	}
}

func TestServeSSPOpenRTBResponseReturnsBidResponseAndEmptySeatBidOnNoFill(t *testing.T) {
	tests := []struct {
		name       string
		floor      float64
		wantFilled bool
	}{
		{name: "filled", floor: 1, wantFilled: true},
		{name: "all no fill", floor: 100, wantFilled: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			body := sspRequestBodyWithFields(t, sspRequestBody(t, 1, 10, []sspAdUnitSpec{
				{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: tt.floor},
			}), map[string]any{"responseFormat": "openrtb"})

			rr := serveSSPWithHeaders(t, controller, body, map[string]string{
				"Origin":     "https://example.com",
				"User-Agent": "ssp-openrtb-test-agent",
			})
			if rr.Code != http.StatusOK {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
			}
			var response openrtb2.BidResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.ID != "ssp-test" || response.Cur != "USD" {
				t.Fatalf("response identity = %#v", response)
			}
			if !tt.wantFilled {
				if len(response.SeatBid) != 0 {
					t.Fatalf("seatbid = %#v, want empty no-fill response", response.SeatBid)
				}
				return
			}
			if len(response.SeatBid) != 1 || response.SeatBid[0].Seat != "10" || len(response.SeatBid[0].Bid) != 1 {
				t.Fatalf("seatbid = %#v", response.SeatBid)
			}
			bid := response.SeatBid[0].Bid[0]
			if bid.ImpID != "unit-one" || bid.AdM == "" || bid.CID != "10" || bid.CrID != "10000" || bid.AdID != "10000" || bid.W != 300 || bid.H != 250 {
				t.Fatalf("bid = %#v", bid)
			}
			if response.BidID == "" {
				t.Fatal("response bidid is empty")
			}
			if response.BidID == bid.ID {
				t.Fatalf("response bidid = bid id = %q, want distinct auction and concrete bid ids", response.BidID)
			}
			if _, err := UnpackBidID(response.BidID); err != nil {
				t.Fatalf("UnpackBidID(%q): %v", response.BidID, err)
			}
			if _, err := UnpackResponseBidID(bid.ID); err != nil {
				t.Fatalf("UnpackResponseBidID(%q): %v", bid.ID, err)
			}
		})
	}
}

func TestServeSSPMiddlemanFallbackRendersAllResponseFormats(t *testing.T) {
	tests := []struct {
		format string
	}{
		{format: "html"},
		{format: "json"},
		{format: "openrtb"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			runtime := &recordingMiddlemanRuntime{
				makeBids: func(bid *openrtb2.BidRequest, fallbackImps []middlemanFallbackImp) []middlemanDownstreamBid {
					if len(fallbackImps) != 1 || fmt.Sprint(fallbackImps[0].TriggerModes) != "[Fallback]" {
						t.Fatalf("fallback imps = %#v, want one Fallback imp", fallbackImps)
					}
					return []middlemanDownstreamBid{sspMiddlemanBid(0, bid.Imp[0].ID, "<div>middleman-fallback</div>", 101, fallbackImps[0].Attr)}
				},
			}
			enableSSPMiddleman(controller, false, runtime)
			body := sspRequestBodyWithFields(t, sspRequestBody(t, 1, 10, []sspAdUnitSpec{
				{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 100},
			}), map[string]any{"responseFormat": tt.format})

			rr := serveSSP(t, controller, body)
			if rr.Code != http.StatusOK {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if runtime.calls != 1 {
				t.Fatalf("middleman calls = %d, want 1", runtime.calls)
			}
			var forwarded map[string]json.RawMessage
			if err := json.Unmarshal(runtime.rawRequests[0], &forwarded); err != nil {
				t.Fatal(err)
			}
			if len(forwarded["imp"]) == 0 || len(forwarded["adUnits"]) != 0 || len(forwarded["site"]) == 0 {
				t.Fatalf("forwarded request = %s, want synthesized OpenRTB and no SSP adUnits", runtime.rawRequests[0])
			}

			switch tt.format {
			case "json":
				var response []sspJSONAdUnitResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if len(response) != 1 || !response[0].Filled || response[0].AdM != "<div>middleman-fallback</div>" || response[0].ImpressionURL != "" || response[0].ClickURL != "" {
					t.Fatalf("json response = %#v", response)
				}
			case "openrtb":
				var response openrtb2.BidResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if len(response.SeatBid) != 1 || response.SeatBid[0].Seat != "90" || len(response.SeatBid[0].Bid) != 1 || response.SeatBid[0].Bid[0].AdM != "<div>middleman-fallback</div>" {
					t.Fatalf("openrtb response = %#v", response)
				}
				if response.BidID != "mid-response" {
					t.Fatalf("response bidid = %q, want mid-response", response.BidID)
				}
			default:
				var response []string
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if len(response) != 1 || response[0] != "<div>middleman-fallback</div>" {
					t.Fatalf("html response = %#v", response)
				}
			}
		})
	}
}

func TestServeSSPMiddlemanAlwaysCompetesOnlyWhenEnabled(t *testing.T) {
	tests := []struct {
		name       string
		always     bool
		midPrice   float64
		wantCalls  int
		wantMarkup string
	}{
		{name: "always disabled keeps local", midPrice: 3.0, wantMarkup: "one.html"},
		{name: "lower middleman keeps local", always: true, midPrice: 1.9, wantCalls: 1, wantMarkup: "one.html"},
		{name: "higher middleman replaces local", always: true, midPrice: 2.5, wantCalls: 1, wantMarkup: "middleman-always"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			runtime := &recordingMiddlemanRuntime{
				makeBids: func(bid *openrtb2.BidRequest, fallbackImps []middlemanFallbackImp) []middlemanDownstreamBid {
					if len(fallbackImps) != 1 || fmt.Sprint(fallbackImps[0].TriggerModes) != "[Always]" {
						t.Fatalf("fallback imps = %#v, want one Always imp", fallbackImps)
					}
					return []middlemanDownstreamBid{sspMiddlemanBid(0, bid.Imp[0].ID, "<div>middleman-always</div>", tt.midPrice, fallbackImps[0].Attr)}
				},
			}
			enableSSPMiddleman(controller, tt.always, runtime)
			body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
				{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
			})

			rr := serveSSP(t, controller, body)
			if rr.Code != http.StatusOK {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if runtime.calls != tt.wantCalls {
				t.Fatalf("middleman calls = %d, want %d", runtime.calls, tt.wantCalls)
			}
			var response []string
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if len(response) != 1 || !strings.Contains(response[0], tt.wantMarkup) {
				t.Fatalf("html response = %#v, want markup containing %q", response, tt.wantMarkup)
			}
		})
	}
}

func TestServeSSPMiddlemanCallbackFailureFallsBackToLocalWinner(t *testing.T) {
	controller := newLocalBidPathController(t)
	runtime := &recordingMiddlemanRuntime{
		makeBids: func(bid *openrtb2.BidRequest, fallbackImps []middlemanFallbackImp) []middlemanDownstreamBid {
			return []middlemanDownstreamBid{sspMiddlemanBid(0, bid.Imp[0].ID, "<div>middleman-would-win</div>", 3.0, fallbackImps[0].Attr)}
		},
	}
	enableSSPMiddleman(controller, true, runtime)
	controller.middlemanStore = failingSetCallbackStore{memoryMiddlemanCallbackStore: newMemoryMiddlemanCallbackStore()}
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{
		{Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1},
	})

	rr := serveSSP(t, controller, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var response []string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response) != 1 || !strings.Contains(response[0], "one.html") || strings.Contains(response[0], "middleman-would-win") {
		t.Fatalf("html response = %#v, want local fallback", response)
	}
}

func TestServeSSPInvalidRequestsDoNotInvokeMiddleman(t *testing.T) {
	site, slot := directTokens(t, 1, 10, 100, match.SizeID2To1(300, 250))
	tests := []struct {
		name    string
		body    []byte
		headers map[string]string
		want    int
	}{
		{name: "malformed json", body: []byte("{"), want: http.StatusBadRequest},
		{name: "invalid token", body: []byte(`{"site":"bad","adUnits":[{"code":"x","slot":"` + slot + `","mediaTypes":{"banner":{}}}]}`), want: http.StatusBadRequest},
		{name: "unsupported media", body: []byte(`{"site":"` + site + `","adUnits":[{"code":"x","slot":"` + slot + `","mediaTypes":{"audio":{}}}]}`), want: http.StatusBadRequest},
		{name: "policy rejection", body: sspRequestBody(t, 1, 10, []sspAdUnitSpec{{Code: "x", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true}}), headers: map[string]string{"Origin": "https://attacker.example"}, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := newLocalBidPathController(t)
			runtime := &recordingMiddlemanRuntime{}
			enableSSPMiddleman(controller, true, runtime)
			headers := tt.headers
			if headers == nil {
				headers = map[string]string{"Origin": "https://example.com"}
			}

			rr := serveSSPWithHeaders(t, controller, tt.body, headers)
			if rr.Code != tt.want {
				t.Fatalf("ServeSSP status = %d, want %d: %s", rr.Code, tt.want, rr.Body.String())
			}
			if runtime.calls != 0 {
				t.Fatalf("middleman calls = %d, want 0", runtime.calls)
			}
		})
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
	controller.C.TrustedProxyCIDRs = []string{"192.0.2.0/24"}
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

func TestOpenRTBFromSSPIgnoresForwardedIPFromUntrustedRemote(t *testing.T) {
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
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	req.Header.Set("X-Real-IP", "203.0.113.10")

	bid, _, _, err := controller.openRTBFromSSP(req.Context(), req, sspReq)
	if err != nil {
		t.Fatal(err)
	}
	if bid.Device.IP != "192.0.2.55" {
		t.Fatalf("device IP = %q, want untrusted remote address", bid.Device.IP)
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

func sspRequestBody(t testing.TB, pubID, siteID uint32, specs []sspAdUnitSpec) []byte {
	t.Helper()
	return sspRequestBodyWithPlatform(t, "", pubID, siteID, specs)
}

func sspRequestBodyWithPlatform(t testing.TB, platform string, pubID, siteID uint32, specs []sspAdUnitSpec) []byte {
	t.Helper()
	if strings.EqualFold(platform, "sdk") && siteID == 10 {
		siteID = 11
	}
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
		w, h := match.SizeID1To2(spec.SizeID)
		switch {
		case spec.Video:
			media["video"] = map[string]any{"playerSize": []uint16{w, h}}
		case spec.Native:
			media["native"] = map[string]any{"image": []uint16{w, h}, "title": true}
		default:
			media["banner"] = map[string]any{"size": []uint16{w, h}}
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

func sspRequestBodyWithV2Tokens(t testing.TB, codec *acl.DirectTokenCodec, pubID, siteID uint32, specs []sspAdUnitSpec) []byte {
	t.Helper()
	site, err := codec.PackSite(pubID, siteID)
	if err != nil {
		t.Fatal(err)
	}
	units := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		slot, err := codec.PackSlot(pubID, siteID, spec.SlotID, spec.SizeID)
		if err != nil {
			t.Fatal(err)
		}
		w, h := match.SizeID1To2(spec.SizeID)
		units = append(units, map[string]any{
			"code": spec.Code, "slot": slot, "floor": spec.Floor,
			"mediaTypes": map[string]any{"banner": map[string]any{"size": []uint16{w, h}}},
		})
	}
	payload := map[string]any{"id": "ssp-v2-test", "platform": "browser", "site": site, "adUnits": units}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func directTokenV2TestCodec(t testing.TB, allowLegacy bool) *acl.DirectTokenCodec {
	t.Helper()
	codec, err := acl.NewDirectTokenCodec(acl.DirectTokenKey{
		ID: "test", Epoch: 3, Secret: bytes.Repeat([]byte{0x5a}, 32),
	}, nil, allowLegacy)
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func publisherAuthRuntimeService(t *testing.T, pubID, siteID uint64) (*publisherauth.Service, string, time.Time, func()) {
	t.Helper()
	server := miniredis.RunT(t)
	redis, err := (radix.PoolConfig{Size: 4}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		redis.Close()
		t.Fatal(err)
	}
	seed := bytes.Repeat([]byte{0x72}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicID := "102132435465768798a9bacbdcedfe0f"
	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery("SELECT c.credential_id, c.pub_id, c.site_id").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"credential_id", "pub_id", "site_id", "public_id", "public_key", "expires_at", "rotated_at", "overlap_until",
		}).AddRow(7, pubID, siteID, publicID, privateKey.Public().(ed25519.PublicKey), now.Add(time.Hour), nil, nil))
	service, err := publisherauth.NewService(publisherauth.Config{Enabled: true}.WithDefaults(), db, redis)
	if err != nil {
		redis.Close()
		db.Close()
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		redis.Close()
		db.Close()
		t.Fatal(err)
	}
	privateCredential := "w8m_pz_v1_" + publicID + "_" + base64.RawURLEncoding.EncodeToString(seed)
	var once sync.Once
	closeStores := func() {
		once.Do(func() {
			redis.Close()
			db.Close()
		})
	}
	return service, privateCredential, now, closeStores
}

func signedPublisherAuthHeaders(t testing.TB, privateCredential string, timestamp time.Time, nonceFill byte, body []byte) map[string]string {
	t.Helper()
	nonce := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{nonceFill}, 16))
	headers, err := publisherauth.SignRequest(privateCredential, timestamp, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string, len(headers))
	for name := range headers {
		out[name] = headers.Get(name)
	}
	return out
}

func tamperSSPSlotToken(t testing.TB, body []byte) []byte {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	units := payload["adUnits"].([]any)
	unit := units[0].(map[string]any)
	token := unit["slot"].(string)
	if token[len(token)-1] == 'A' {
		unit["slot"] = token[:len(token)-1] + "B"
	} else {
		unit["slot"] = token[:len(token)-1] + "A"
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func replaceSSPSlotWithLegacy(t testing.TB, body []byte, slotID, sizeID uint32) []byte {
	t.Helper()
	legacy, err := acl.PackDirectToken(slotID, sizeID)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	payload["adUnits"].([]any)[0].(map[string]any)["slot"] = legacy
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sspRequestBodyWithFields(t testing.TB, body []byte, fields map[string]any) []byte {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	for key, value := range fields {
		payload[key] = value
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func directTokens(t testing.TB, pubID, siteID, slotID, sizeID uint32) (string, string) {
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
	return serveSSPWithHeaders(t, controller, body, map[string]string{"Origin": "https://example.com"})
}

func serveSSPWithHeaders(t *testing.T, controller *Controller, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	req.RemoteAddr = "203.0.113.1:4567"
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rr := httptest.NewRecorder()
	controller.ServeSSP(rr, req)
	return rr
}

type recordingMiddlemanRuntime struct {
	calls       int
	rawRequests [][]byte
	makeBids    func(*openrtb2.BidRequest, []middlemanFallbackImp) []middlemanDownstreamBid
}

func (r *recordingMiddlemanRuntime) Fallback(_ context.Context, bid *openrtb2.BidRequest, rawRequest []byte, _ time.Time, fallbackImps []middlemanFallbackImp) ([]middlemanDownstreamBid, error) {
	r.calls++
	r.rawRequests = append(r.rawRequests, append([]byte(nil), rawRequest...))
	if r.makeBids == nil {
		return nil, nil
	}
	return r.makeBids(bid, fallbackImps), nil
}

func enableSSPMiddleman(controller *Controller, always bool, runtime *recordingMiddlemanRuntime) {
	controller.C.MiddlemanEnabled = true
	controller.C.MiddlemanAlwaysEnabled = always
	controller.C.PrivacyContextualMiddleman = true
	controller.C.MiddlemanCallbackBaseURL = controller.C.ServerURL
	controller.middlemanRuntime = runtime
	controller.middlemanStore = newMemoryMiddlemanCallbackStore()
}

func sspMiddlemanBid(impIndex int, impID, adm string, price float64, attr *match.Attribute) middlemanDownstreamBid {
	return middlemanDownstreamBid{
		ImpIndex: impIndex,
		Seat:     "90",
		Bid: openrtb2.Bid{
			ID:    "mid-bid",
			ImpID: impID,
			Price: price,
			AdM:   adm,
			CID:   "90",
			CrID:  "900",
			AdID:  "900",
			W:     300,
			H:     250,
		},
		Audit: bidAudit{
			Attr: attr,
			One: match.RAdv{
				Demand:   match.Demand{AdvID: 9, CampaignID: 90, ItemID: 901, CreativeID: 900},
				CostType: 2,
				Cost:     float32(price),
			},
		},
		ResponseBidID:      "mid-response",
		Entry:              match.MiddlemanRouteEntry{BidderID: 7, AdvID: 9, SyntheticCampaignID: 90, SyntheticItemID: 901, SyntheticCreativeID: 900},
		DownstreamBidPrice: price,
		UpstreamBidPrice:   price,
	}
}

type failingSetCallbackStore struct {
	*memoryMiddlemanCallbackStore
}

func (f failingSetCallbackStore) SetCallback(context.Context, string, middlemanCallbackContext, time.Duration) error {
	return fmt.Errorf("set callback failed")
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
	if !json.Valid(envelope.Payload) {
		t.Fatalf("payload is not JSON: %s", envelope.Payload)
	}
	if subject == SUBJECTResponse {
		for _, forbidden := range []string{"<iframe", "<img", "/imp?", "/clk?"} {
			if strings.Contains(string(envelope.Payload), forbidden) {
				t.Fatalf("response audit contains %q: %s", forbidden, envelope.Payload)
			}
		}
	}
	_ = wantPayload
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

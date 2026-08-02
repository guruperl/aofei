package dsp

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestServeBidEnforcesOpenRTB25HTTPProfile(t *testing.T) {
	controller := &Controller{}
	for _, tt := range []struct {
		name, method, contentType, version string
		want                               int
	}{
		{name: "method", method: http.MethodGet, want: http.StatusMethodNotAllowed},
		{name: "content type", method: http.MethodPost, contentType: "text/plain", want: http.StatusUnsupportedMediaType},
		{name: "version", method: http.MethodPost, contentType: "application/json", version: "2.6", want: http.StatusBadRequest},
		{name: "explicit 2.5", method: http.MethodPost, contentType: "application/openrtb+json", version: "2.5", want: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/bid/fixture", strings.NewReader("{"))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.version != "" {
				req.Header.Set("x-openrtb-version", tt.version)
			}
			response := httptest.NewRecorder()
			controller.ServeBid(response, req)
			if response.Code != tt.want {
				t.Fatalf("status = %d, want %d", response.Code, tt.want)
			}
			if got := response.Header().Get("x-openrtb-version"); got != "2.5" {
				t.Fatalf("response version = %q, want 2.5", got)
			}
		})
	}
}

func TestOpenRTBCompatibilityFixturesAreCredentialFreeAndBounded(t *testing.T) {
	directory := filepath.Join("testdata", "openrtb")
	for _, name := range []string{"request-multiimp.json", "request-native.json"} {
		raw, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		var bid openrtb2.BidRequest
		if err := json.Unmarshal(raw, &bid); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := validateBidRequest(&bid); err != nil {
			t.Fatalf("%s profile: %v", name, err)
		}
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(raw); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		reader, err := gzip.NewReader(&compressed)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := io.ReadAll(io.LimitReader(reader, maxBidRequestBodyBytes+1))
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(roundTrip, raw) {
			t.Fatalf("%s gzip round trip changed fixture", name)
		}
	}

	malformed, err := os.ReadFile(filepath.Join(directory, "malformed-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(malformed) {
		t.Fatal("malformed request fixture is valid JSON")
	}
	for _, name := range []string{
		"response-currency-eur.json", "response-native.json", "response-video.json",
		"response-unsafe-callback.json", "timeout.json",
	} {
		raw, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(raw) {
			t.Fatalf("%s is invalid JSON", name)
		}
		lower := strings.ToLower(string(raw))
		for _, forbidden := range []string{"authorization", "bearer ", "password", "credential_ref", "@w8m.com"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains forbidden credential-like text %q", name, forbidden)
			}
		}
	}
}

func i01TimeoutFixture(t *testing.T) (time.Duration, time.Duration) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "openrtb", "timeout.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		RequestTMaxMS           int `json:"request_tmax_ms"`
		SimulatedPartnerDelayMS int `json:"simulated_partner_delay_ms"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return time.Duration(fixture.RequestTMaxMS) * time.Millisecond,
		time.Duration(fixture.SimulatedPartnerDelayMS) * time.Millisecond
}

func TestValidateBidRequestOpenRTB25Profile(t *testing.T) {
	w, h := int64(300), int64(250)
	secure := int8(1)
	valid := func() *openrtb2.BidRequest {
		return &openrtb2.BidRequest{
			ID: "request-1", AT: 1, TMax: 500, Cur: []string{"USD"}, Device: &openrtb2.Device{},
			Imp: []openrtb2.Imp{{
				ID: "imp-1", Banner: &openrtb2.Banner{W: &w, H: &h},
				BidFloor: 0.25, BidFloorCur: "USD", Secure: &secure,
			}},
		}
	}
	if err := validateBidRequest(valid()); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	for name, mutate := range map[string]func(*openrtb2.BidRequest){
		"request id":        func(b *openrtb2.BidRequest) { b.ID = "" },
		"duplicate imp":     func(b *openrtb2.BidRequest) { b.Imp = append(b.Imp, b.Imp[0]) },
		"unsupported cur":   func(b *openrtb2.BidRequest) { b.Cur = []string{"EUR"} },
		"negative floor":    func(b *openrtb2.BidRequest) { b.Imp[0].BidFloor = -1 },
		"nonfinite floor":   func(b *openrtb2.BidRequest) { b.Imp[0].BidFloor = math.Inf(1) },
		"unsupported audio": func(b *openrtb2.BidRequest) { b.Imp[0].Banner = nil; b.Imp[0].Audio = &openrtb2.Audio{} },
		"excess tmax":       func(b *openrtb2.BidRequest) { b.TMax = 60_001 },
	} {
		t.Run(name, func(t *testing.T) {
			bid := valid()
			mutate(bid)
			if err := validateBidRequest(bid); err == nil {
				t.Fatal("invalid request was accepted")
			}
		})
	}
	multiFormat := valid()
	multiFormat.Imp[0].Video = &openrtb2.Video{}
	if err := validateBidRequest(multiFormat); err != nil {
		t.Fatalf("standard multi-format impression was rejected: %v", err)
	}
}

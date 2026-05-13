package dsp

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/genelet/winter/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestMiddlemanReconciledPrices(t *testing.T) {
	value := middlemanCallbackContext{
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
	}
	got := value.reconciledPrices("1.100")
	if !closeFloat(got.ChargePrice, 1.1) || !closeFloat(got.PayPrice, 0.9) {
		t.Fatalf("prices = %+v, want charge 1.1 pay 0.9", got)
	}
	got = value.reconciledPrices("")
	if !closeFloat(got.ChargePrice, 1.2) || !closeFloat(got.PayPrice, 1.0) {
		t.Fatalf("fallback prices = %+v, want charge 1.2 pay 1.0", got)
	}
	got = value.reconciledPrices("9.000")
	if !closeFloat(got.ChargePrice, 1.2) || !closeFloat(got.PayPrice, 1.0) {
		t.Fatalf("clamped prices = %+v, want charge 1.2 pay 1.0", got)
	}
}

func TestServeMiddlemanBillPublishesOnceAndForwardsPayPrice(t *testing.T) {
	var forwarded []string
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = append(forwarded, r.URL.Query().Get("auction_price"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()

	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token:              "tok",
		RequestID:          "req",
		ImpID:              "imp",
		ResponseBidID:      "resp",
		DownstreamBidID:    "dbid",
		DownstreamSeat:     "seat",
		DownstreamAdID:     "ad",
		DownstreamBURL:     downstream.URL + "/bill?auction_price=${AUCTION_PRICE}",
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
		BidderID:           7,
		RPub:               match.RPub{PubID: 1, SiteID: 2, SlotID: 3, SizeID: 4},
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}

	var published []WinLoss
	controller := &Controller{
		C: &Config{
			ServerURL:                  "http://aofei.example",
			TrackingSecret:             "test-secret",
			MiddlemanCallbackBaseURL:   "http://aofei.example",
			MiddlemanCallbackTimeoutMS: 1000,
		},
		client:         downstream.Client(),
		middlemanStore: store,
		publishWinLossFunc: func(data []byte) error {
			var wl WinLoss
			if err := json.Unmarshal(data, &wl); err != nil {
				return err
			}
			published = append(published, wl)
			return nil
		},
	}

	target := signedMiddlemanTestURL(t, controller, "/mid/bill", "tok", "1.100")
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("duplicate status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(published) != 1 || published[0].Status != StatusTrackImp {
		t.Fatalf("published = %#v, want one billable impression", published)
	}
	if !closeFloat(float64(published[0].RAdv.Cost), 1.1) || !closeFloat(published[0].Middleman.PayPrice, 0.9) {
		t.Fatalf("winloss prices = %#v middleman=%#v", published[0].RAdv, published[0].Middleman)
	}
	if len(forwarded) != 1 || forwarded[0] != "0.900" {
		t.Fatalf("forwarded prices = %#v, want one downstream pay price 0.900", forwarded)
	}
}

func closeFloat(got, want float64) bool {
	return math.Abs(got-want) < 0.000001
}

func TestServeMiddlemanBillPublishFailureAllowsRetryWithoutRefiringDownstream(t *testing.T) {
	var forwarded int
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()

	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token:              "tok",
		RequestID:          "req",
		ImpID:              "imp",
		ResponseBidID:      "resp",
		DownstreamBURL:     downstream.URL + "/bill",
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}
	publishAttempts := 0
	controller := &Controller{
		C:              &Config{ServerURL: "http://aofei.example", TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		client:         downstream.Client(),
		middlemanStore: store,
		publishWinLossFunc: func(_ []byte) error {
			publishAttempts++
			if publishAttempts == 1 {
				return errors.New("publish failed")
			}
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/bill", "tok", "1.100"), nil)
	rr := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("first status = %d, want publish failure", rr.Code)
	}
	rr = httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("retry status = %d body=%s", rr.Code, rr.Body.String())
	}
	if publishAttempts != 2 {
		t.Fatalf("publish attempts = %d, want retry", publishAttempts)
	}
	if forwarded != 1 {
		t.Fatalf("downstream forwards = %d, want one", forwarded)
	}
}

func TestPrepareMiddlemanCallbackRewritesBidAndStoresContext(t *testing.T) {
	store := newMemoryMiddlemanCallbackStore()
	controller := &Controller{
		C: &Config{
			ServerURL:                "http://aofei.example",
			TrackingSecret:           "test-secret",
			MiddlemanCallbackBaseURL: "http://aofei.example",
		},
		middlemanStore: store,
	}
	selected := &middlemanDownstreamBid{
		Bid: openRTBBidForMiddlemanTest("imp", 1.2),
		Audit: bidAudit{
			Attr: &match.Attribute{RPub: match.RPub{PubID: 1, SiteID: 2, SlotID: 3, SizeID: 4}},
			One:  match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}, CostType: 2, Cost: 1.2},
		},
		ResponseBidID:      "resp",
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		DownstreamBURL:     "http://downstream.example/bill",
		ClickRequestToken:  "rtok",
	}
	if err := controller.prepareMiddlemanCallback(context.Background(), bidRequestForMiddlemanTest("req"), selected); err != nil {
		t.Fatal(err)
	}
	if selected.Bid.NURL == "" || selected.Bid.LURL == "" || selected.Bid.BURL == "" {
		t.Fatalf("proxied callbacks were not installed: %#v", selected.Bid)
	}
	for _, raw := range []string{selected.Bid.NURL, selected.Bid.BURL, selected.Bid.LURL} {
		if !strings.Contains(raw, "auction_price=${AUCTION_PRICE}") || !strings.Contains(raw, "auction_currency=${AUCTION_CURRENCY}") {
			t.Fatalf("proxied callback did not preserve literal macros: %s", raw)
		}
	}
	u, err := url.Parse(selected.Bid.NURL)
	if err != nil {
		t.Fatal(err)
	}
	token := u.Query().Get("t")
	if token == "" {
		t.Fatalf("callback token missing from %s", selected.Bid.NURL)
	}
	stored, err := store.GetCallback(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if stored.RequestID != "req" || stored.ImpID != "imp" || stored.DownstreamBidPrice != 1.0 || stored.UpstreamBidPrice != 1.2 {
		t.Fatalf("stored context = %#v", stored)
	}
	clickToken, err := store.GetClick(context.Background(), "rtok", "imp")
	if err != nil {
		t.Fatal(err)
	}
	if clickToken != token {
		t.Fatalf("click token = %q, want %q", clickToken, token)
	}
}

func TestEncodeOpenRTBMacroQueryPreservesLiteralMacros(t *testing.T) {
	values := url.Values{}
	values.Set("auction_price", `${AUCTION_PRICE}`)
	values.Set("auction_currency", `${AUCTION_CURRENCY}`)
	values.Set("redirect", "https://example.test/click?x=1&y=2")

	got := encodeOpenRTBMacroQuery(values)
	if !strings.Contains(got, "auction_price=${AUCTION_PRICE}") {
		t.Fatalf("auction price macro was encoded: %s", got)
	}
	if !strings.Contains(got, "auction_currency=${AUCTION_CURRENCY}") {
		t.Fatalf("auction currency macro was encoded: %s", got)
	}
	if strings.Contains(got, "x=1&y=2") {
		t.Fatalf("non-macro redirect was not query-escaped: %s", got)
	}
}

func TestServeMiddlemanWinFallsBackToBillingWithoutBURL(t *testing.T) {
	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token:              "tok",
		RequestID:          "req",
		ImpID:              "imp",
		ResponseBidID:      "resp",
		DownstreamBidPrice: 2.0,
		UpstreamBidPrice:   2.5,
		MarginCPM:          0.5,
		BillOnWin:          true,
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}
	var statuses []Status
	controller := &Controller{
		C:              &Config{ServerURL: "http://aofei.example", TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		middlemanStore: store,
		publishWinLossFunc: func(data []byte) error {
			var wl WinLoss
			if err := json.Unmarshal(data, &wl); err != nil {
				return err
			}
			statuses = append(statuses, wl.Status)
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/win", "tok", "2.400"), nil)
	rr := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(statuses) != 2 || statuses[0] != StatusWin || statuses[1] != StatusTrackImp {
		t.Fatalf("statuses = %#v, want win then billable impression", statuses)
	}
}

func TestServeMiddlemanWinDoesNotForwardDuplicateNotifications(t *testing.T) {
	var forwarded int
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()

	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token:              "tok",
		RequestID:          "req",
		ImpID:              "imp",
		ResponseBidID:      "resp",
		DownstreamNURL:     downstream.URL + "/win",
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}
	var forwards []string
	controller := &Controller{
		C:              &Config{ServerURL: "http://aofei.example", TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		client:         downstream.Client(),
		middlemanStore: store,
		publishWinLossFunc: func(data []byte) error {
			var wl WinLoss
			if err := json.Unmarshal(data, &wl); err != nil {
				return err
			}
			forwards = append(forwards, wl.Middleman.ForwardStatus)
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/win", "tok", "1.100"), nil)
	rr := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("duplicate status = %d body=%s", rr.Code, rr.Body.String())
	}
	if forwarded != 1 {
		t.Fatalf("downstream forwards = %d, want one", forwarded)
	}
	if len(forwards) != 2 || forwards[0] != "ok" || forwards[1] != "duplicate" {
		t.Fatalf("forward statuses = %#v, want ok then duplicate", forwards)
	}
}

func TestServeMiddlemanWinRetriesOnlyRetryableDownstreamFailure(t *testing.T) {
	for _, tt := range []struct {
		name        string
		statusCode  int
		wantForward int
	}{
		{"http400", http.StatusBadRequest, 1},
		{"http429", http.StatusTooManyRequests, 2},
		{"http500", http.StatusInternalServerError, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var forwarded int
			downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				forwarded++
				w.WriteHeader(tt.statusCode)
			}))
			defer downstream.Close()

			store := newMemoryMiddlemanCallbackStore()
			value := middlemanCallbackContext{
				Token:              "tok",
				RequestID:          "req",
				ImpID:              "imp",
				ResponseBidID:      "resp",
				DownstreamNURL:     downstream.URL + "/win",
				DownstreamBidPrice: 1.0,
				UpstreamBidPrice:   1.2,
				MarginCPM:          0.2,
				RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
			}
			if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
				t.Fatal(err)
			}
			controller := &Controller{
				C: &Config{
					ServerURL:                "http://aofei.example",
					TrackingSecret:           "test-secret",
					MiddlemanCallbackBaseURL: "http://aofei.example",
				},
				client:         downstream.Client(),
				middlemanStore: store,
				publishWinLossFunc: func(_ []byte) error {
					return nil
				},
			}
			req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/win", "tok", "1.100"), nil)
			for i := 0; i < 2; i++ {
				rr := httptest.NewRecorder()
				controller.ServeMiddlemanCallback(rr, req)
				if rr.Code != http.StatusNoContent {
					t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
				}
			}
			if forwarded != tt.wantForward {
				t.Fatalf("forwards = %d, want %d", forwarded, tt.wantForward)
			}
		})
	}
}

func openRTBBidForMiddlemanTest(impID string, price float64) openrtb2.Bid {
	return openrtb2.Bid{ID: "bid", ImpID: impID, Price: price}
}

func bidRequestForMiddlemanTest(id string) *openrtb2.BidRequest {
	return &openrtb2.BidRequest{ID: id}
}

func TestServeMiddlemanClickUsesCooperativeMapping(t *testing.T) {
	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token:              "tok",
		RequestID:          "req",
		ImpID:              "imp",
		ResponseBidID:      "resp",
		DownstreamBidPrice: 1,
		UpstreamBidPrice:   1.1,
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.SetClick(context.Background(), "rtok", "imp", "tok", time.Hour); err != nil {
		t.Fatal(err)
	}
	var published []WinLoss
	controller := &Controller{
		C:              &Config{ServerURL: "http://aofei.example", TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		middlemanStore: store,
		publishWinLossFunc: func(data []byte) error {
			var wl WinLoss
			if err := json.Unmarshal(data, &wl); err != nil {
				return err
			}
			published = append(published, wl)
			return nil
		},
	}
	clickURL, err := controller.middlemanClickProxyURL("rtok", "imp")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, clickURL, nil)
	rr := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(published) != 1 || published[0].Status != StatusTrackClk {
		t.Fatalf("published = %#v, want one click", published)
	}
}

func signedMiddlemanTestURL(t *testing.T, controller *Controller, path, token, price string) string {
	t.Helper()
	raw, err := controller.middlemanProxyURL(path, token)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("auction_price", price)
	q.Set("auction_currency", "USD")
	u.RawQuery = q.Encode()
	return u.String()
}

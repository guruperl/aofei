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
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestMiddlemanCallbackTTLFallbackCoversAcceptedSignatureLifetime(t *testing.T) {
	want := defaultTrackingSignatureTTL + maxTrackingSignatureFutureSkew
	if got := (*Controller)(nil).middlemanCallbackTTL(); got != want {
		t.Fatalf("nil-controller callback TTL = %s, want %s", got, want)
	}
	if got := (&Controller{C: &Config{}}).middlemanCallbackTTL(); got != want {
		t.Fatalf("zero-config callback TTL = %s, want %s", got, want)
	}
}

func TestServeMiddlemanLookupClassifiesDependencyAndMissingState(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		lookupFail string
	}{
		{name: "status", path: "/mid/win", lookupFail: "callback"},
		{name: "click", path: "/mid/click", lookupFail: "click"},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := newMemoryMiddlemanCallbackStore()
			store := &failingMiddlemanLookupStore{memoryMiddlemanCallbackStore: base}
			controller := &Controller{
				C:              &Config{TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
				middlemanStore: store,
			}
			var target string
			if test.lookupFail == "callback" {
				store.callbackErr = errors.New("callback Redis unavailable")
				target = signedMiddlemanTestURL(t, controller, test.path, "tok", "1.000")
			} else {
				store.clickErr = errors.New("click Redis unavailable")
				var err error
				target, err = controller.middlemanClickProxyURL("rtok", "imp")
				if err != nil {
					t.Fatal(err)
				}
			}
			rr := httptest.NewRecorder()
			controller.ServeMiddlemanCallback(rr, httptest.NewRequest(http.MethodGet, target, nil))
			if rr.Code != http.StatusServiceUnavailable {
				t.Fatalf("dependency status = %d body=%s, want 503", rr.Code, rr.Body.String())
			}

			store.callbackErr = nil
			store.clickErr = nil
			rr = httptest.NewRecorder()
			controller.ServeMiddlemanCallback(rr, httptest.NewRequest(http.MethodGet, target, nil))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("missing-state status = %d body=%s, want 400", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestMiddlemanReconciledPricesUseServerOwnedPrice(t *testing.T) {
	value := middlemanCallbackContext{
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
	}
	got := value.reconciledPrices()
	if !closeFloat(got.ChargePrice, 1.2) || !closeFloat(got.PayPrice, 1.0) {
		t.Fatalf("prices = %+v, want charge 1.2 pay 1.0", got)
	}
}

func TestMiddlemanExactCallbackRejectsTamperedPriceIdentity(t *testing.T) {
	value := middlemanCallbackContext{
		AccountingVersion: accounting.ExactMoneyContract,
		DownstreamBidCPM:  1_000_000,
		UpstreamBidCPM:    1_200_000,
		MarginCPMExact:    200_001,
	}
	if _, err := value.reconciledPricesExact(); err == nil {
		t.Fatal("tampered exact callback price identity was accepted")
	}
}

func TestRedisMiddlemanStoreSeparatesForwardAndPublishState(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	store := redisMiddlemanCallbackStore{redis: client}
	ctx := context.Background()
	first, err := store.SetNotifyOnce(ctx, "tok", "win", "notify-owner", time.Minute)
	if err != nil || !first {
		t.Fatalf("initial forward claim = %v, %v", first, err)
	}
	state, err := store.GetNotify(ctx, "tok", "win")
	if err != nil || !state.Processing {
		t.Fatalf("processing state = %+v, %v", state, err)
	}
	completed := middlemanForwardState{Status: "ok", HTTPStatus: http.StatusNoContent}
	if err := store.CompleteNotify(ctx, "tok", "win", "notify-owner", completed, time.Hour); err != nil {
		t.Fatal(err)
	}
	first, err = store.SetNotifyOnce(ctx, "tok", "win", "other-notify-owner", time.Minute)
	if err != nil || first {
		t.Fatalf("completed forward claim = %v, %v; want duplicate", first, err)
	}
	state, err = store.GetNotify(ctx, "tok", "win")
	if err != nil || state.Processing || state.Status != "ok" || state.HTTPStatus != http.StatusNoContent {
		t.Fatalf("completed state = %+v, %v", state, err)
	}
	publishFirst, err := store.SetPublishOnce(ctx, "tok", "win", "publish-owner", time.Minute)
	if err != nil || !publishFirst {
		t.Fatalf("initial publish claim = %v, %v", publishFirst, err)
	}
	publishFirst, err = store.SetPublishOnce(ctx, "tok", "win", "other-publish-owner", time.Minute)
	if err != nil || publishFirst {
		t.Fatalf("duplicate publish claim = %v, %v", publishFirst, err)
	}
	if err := store.ClearPublish(ctx, "tok", "win", "publish-owner"); err != nil {
		t.Fatal(err)
	}
	publishFirst, err = store.SetPublishOnce(ctx, "tok", "win", "replacement-owner", time.Minute)
	if err != nil || !publishFirst {
		t.Fatalf("released publish claim = %v, %v", publishFirst, err)
	}
	if err := store.CompletePublish(ctx, "tok", "win", "replacement-owner", time.Hour); err != nil {
		t.Fatal(err)
	}

	publishFirst, err = store.SetPublishOnce(ctx, "crash", "win", "expired-owner", time.Second)
	if err != nil || !publishFirst {
		t.Fatalf("crash claim = %v, %v", publishFirst, err)
	}
	server.FastForward(2 * time.Second)
	publishFirst, err = store.SetPublishOnce(ctx, "crash", "win", "new-owner", time.Minute)
	if err != nil || !publishFirst {
		t.Fatalf("claim after processing lease expiry = %v, %v", publishFirst, err)
	}
	if err := store.ClearPublish(ctx, "crash", "win", "expired-owner"); err == nil {
		t.Fatal("expired owner cleared a replacement claim")
	}
	if err := store.CompletePublish(ctx, "crash", "win", "new-owner", time.Hour); err != nil {
		t.Fatal(err)
	}
}

func TestMiddlemanProcessingLeaseCoversConfiguredForwardTimeout(t *testing.T) {
	controller := &Controller{C: &Config{MiddlemanCallbackTimeoutMS: 60_000}}
	if got := controller.middlemanProcessingTTL(); got < 65*time.Second {
		t.Fatalf("processing lease = %s, want at least callback timeout plus cleanup margin", got)
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
		DownstreamBURL:     safeTestOrigin + "/bill?auction_price=${AUCTION_PRICE}",
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
		client:         safeTestClient(downstream),
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
	if !closeFloat(float64(published[0].RAdv.Cost), 1.2) || !closeFloat(published[0].Middleman.PayPrice, 1.0) {
		t.Fatalf("winloss prices = %#v middleman=%#v", published[0].RAdv, published[0].Middleman)
	}
	if len(forwarded) != 1 || forwarded[0] != "1.000" {
		t.Fatalf("forwarded prices = %#v, want one downstream pay price 1.000", forwarded)
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
		DownstreamBURL:     safeTestOrigin + "/bill",
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
		client:         safeTestClient(downstream),
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
	if rr.Code != http.StatusServiceUnavailable {
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
	if stored.AccountingVersion != accounting.ExactMoneyContract || stored.DownstreamBidCPM != 1_000_000 || stored.UpstreamBidCPM != 1_200_000 || stored.MarginCPMExact != 200_000 {
		t.Fatalf("stored exact price identity = %#v", stored)
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
		DownstreamNURL:     safeTestOrigin + "/win",
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
		client:         safeTestClient(downstream),
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
	if len(forwards) != 1 || forwards[0] != "ok" {
		t.Fatalf("forward statuses = %#v, want only first win published", forwards)
	}
}

func TestServeMiddlemanStatusPublishFailureRetriesLocallyWithoutRefiringDownstream(t *testing.T) {
	for _, test := range []struct {
		path   string
		status Status
	}{
		{path: "/mid/win", status: StatusWin},
		{path: "/mid/loss", status: StatusLoss},
	} {
		t.Run(test.path, func(t *testing.T) {
			retryableBefore := expvarMapInt64(metricMiddlemanCallbackOutcomes, "local_publish_retryable")
			releasesBefore := expvarMapInt64(metricMiddlemanCallbackOutcomes, "claim_released")
			duplicatesBefore := expvarMapInt64(metricMiddlemanCallbackOutcomes, "publish_duplicate")
			forwarded := 0
			downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				forwarded++
				w.WriteHeader(http.StatusNoContent)
			}))
			defer downstream.Close()

			store := newMemoryMiddlemanCallbackStore()
			value := middlemanCallbackContext{
				Token: "tok", RequestID: "req", ImpID: "imp", ResponseBidID: "resp",
				DownstreamNURL: safeTestOrigin + "/win", DownstreamLURL: safeTestOrigin + "/loss",
				DownstreamBidPrice: 1, UpstreamBidPrice: 1.1,
				RAdv: match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
			}
			if err := store.SetCallback(context.Background(), value.Token, value, time.Hour); err != nil {
				t.Fatal(err)
			}
			publishAttempts := 0
			controller := &Controller{
				C:      &Config{TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
				client: safeTestClient(downstream), middlemanStore: store,
				publishWinLossFunc: func(data []byte) error {
					publishAttempts++
					if publishAttempts == 1 {
						return errors.New("publish failed")
					}
					var wl WinLoss
					if err := json.Unmarshal(data, &wl); err != nil {
						return err
					}
					if wl.Status != test.status || wl.Middleman.ForwardStatus != "ok" {
						t.Fatalf("published callback = %+v", wl)
					}
					return nil
				},
			}
			req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, test.path, value.Token, "1.000"), nil)
			first := httptest.NewRecorder()
			controller.ServeMiddlemanCallback(first, req)
			if first.Code != http.StatusServiceUnavailable {
				t.Fatalf("first status = %d, want publish failure", first.Code)
			}
			for i := 0; i < 2; i++ {
				retry := httptest.NewRecorder()
				controller.ServeMiddlemanCallback(retry, req)
				if retry.Code != http.StatusNoContent {
					t.Fatalf("retry status = %d body=%s", retry.Code, retry.Body.String())
				}
			}
			if forwarded != 1 {
				t.Fatalf("downstream forwards = %d, want one", forwarded)
			}
			if publishAttempts != 2 {
				t.Fatalf("local publish attempts = %d, want failed attempt plus successful retry", publishAttempts)
			}
			if got := expvarMapInt64(metricMiddlemanCallbackOutcomes, "local_publish_retryable") - retryableBefore; got != 1 {
				t.Fatalf("retryable callback metric delta = %d, want 1", got)
			}
			if got := expvarMapInt64(metricMiddlemanCallbackOutcomes, "claim_released") - releasesBefore; got != 1 {
				t.Fatalf("callback claim-release metric delta = %d, want 1", got)
			}
			if got := expvarMapInt64(metricMiddlemanCallbackOutcomes, "publish_duplicate") - duplicatesBefore; got != 1 {
				t.Fatalf("callback duplicate metric delta = %d, want 1", got)
			}
		})
	}
}

func TestMiddlemanCallbackOutcomeMetricRejectsDynamicKeys(t *testing.T) {
	before := expvarMapInt64(metricMiddlemanCallbackOutcomes, "other")
	recordMiddlemanCallbackOutcome("raw-callback-token")
	if got := expvarMapInt64(metricMiddlemanCallbackOutcomes, "other") - before; got != 1 {
		t.Fatalf("other callback metric delta = %d, want 1", got)
	}
	if metricMiddlemanCallbackOutcomes.Get("raw-callback-token") != nil {
		t.Fatal("dynamic callback metric key was published")
	}
}

func TestServeMiddlemanWinSuppressesConcurrentForwardAndPublication(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var forwarded atomic.Int32
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if forwarded.Add(1) == 1 {
			close(started)
		}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()
	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token: "tok", RequestID: "req", ImpID: "imp", ResponseBidID: "resp", DownstreamNURL: safeTestOrigin,
		DownstreamBidPrice: 1, UpstreamBidPrice: 1.1,
		RAdv: match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), value.Token, value, time.Hour); err != nil {
		t.Fatal(err)
	}
	var published atomic.Int32
	controller := &Controller{
		C:      &Config{TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		client: safeTestClient(downstream), middlemanStore: store,
		publishWinLossFunc: func([]byte) error { published.Add(1); return nil },
	}
	target := signedMiddlemanTestURL(t, controller, "/mid/win", value.Token, "1.000")
	ownerDone := make(chan int, 1)
	go func() {
		rr := httptest.NewRecorder()
		controller.ServeMiddlemanCallback(rr, httptest.NewRequest(http.MethodGet, target, nil))
		ownerDone <- rr.Code
	}()
	<-started
	duplicate := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(duplicate, httptest.NewRequest(http.MethodGet, target, nil))
	if duplicate.Code != http.StatusNoContent || published.Load() != 0 {
		t.Fatalf("in-flight duplicate status=%d published=%d", duplicate.Code, published.Load())
	}
	close(release)
	if code := <-ownerDone; code != http.StatusNoContent {
		t.Fatalf("owner status = %d", code)
	}
	completed := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(completed, httptest.NewRequest(http.MethodGet, target, nil))
	if completed.Code != http.StatusNoContent || forwarded.Load() != 1 || published.Load() != 1 {
		t.Fatalf("completed duplicate status=%d forwarded=%d published=%d", completed.Code, forwarded.Load(), published.Load())
	}
}

func TestServeMiddlemanRejectsMissingTimestampSignature(t *testing.T) {
	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token:              "tok",
		RequestID:          "req",
		ImpID:              "imp",
		ResponseBidID:      "resp",
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}
	controller := &Controller{
		C:              &Config{ServerURL: "http://aofei.example", TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		middlemanStore: store,
	}
	args := url.Values{"t": []string{"tok"}}
	args.Set("sig", signTrackingValues("test-secret", "/mid/win", middlemanSignableValues("/mid/win", args)))
	req := httptest.NewRequest(http.MethodGet, "/mid/win?"+args.Encode(), nil)
	rr := httptest.NewRecorder()

	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want bad request", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "timestamp missing") {
		t.Fatalf("body = %q, want timestamp missing error", rr.Body.String())
	}
}

func TestServeMiddlemanRejectsExpiredSignature(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ServerURL:                   "http://aofei.example",
			TrackingSecret:              "test-secret",
			MiddlemanCallbackBaseURL:    "http://aofei.example",
			TrackingSignatureTTLSeconds: 1,
		},
	}
	raw, err := controller.middlemanProxyURL("/mid/win", "tok")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set(trackingSignatureTimestampParam, "1")
	q.Set(trackingSignatureParam, signTrackingValues("test-secret", "/mid/win", middlemanSignableValues("/mid/win", q)))
	u.RawQuery = q.Encode()
	req := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
	rr := httptest.NewRecorder()

	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want bad request", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "expired") {
		t.Fatalf("body = %q, want expired error", rr.Body.String())
	}
}

func TestServeMiddlemanSSRFIsInvalidForwardTarget(t *testing.T) {
	forwarded := 0
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		DownstreamNURL:     "http://127.0.0.1/win",
		DownstreamBidPrice: 1.0,
		UpstreamBidPrice:   1.2,
		MarginCPM:          0.2,
		RAdv:               match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), "tok", value, time.Hour); err != nil {
		t.Fatal(err)
	}
	var published []WinLoss
	controller := &Controller{
		C:                &Config{ServerURL: "http://aofei.example", TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		client:           safeTestClient(downstream),
		callbackURLGuard: func(context.Context, string) error { return nil },
		middlemanStore:   store,
		publishWinLossFunc: func(data []byte) error {
			var wl WinLoss
			if err := json.Unmarshal(data, &wl); err != nil {
				return err
			}
			published = append(published, wl)
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/win", "tok", "1.100"), nil)
	rr := httptest.NewRecorder()

	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(published) != 1 || published[0].Middleman.ForwardStatus != "invalid_url" {
		t.Fatalf("published = %#v, want invalid_url forward status", published)
	}
	if forwarded != 0 {
		t.Fatalf("injected client received %d private-target requests, want zero", forwarded)
	}
}

func TestServeMiddlemanWinRetriesOnlyRetryableDownstreamFailure(t *testing.T) {
	for _, tt := range []struct {
		name        string
		statusCode  int
		wantStatus  int
		wantForward int
	}{
		{"http400", http.StatusBadRequest, http.StatusNoContent, 1},
		{"http429", http.StatusTooManyRequests, http.StatusServiceUnavailable, 2},
		{"http500", http.StatusInternalServerError, http.StatusServiceUnavailable, 2},
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
				DownstreamNURL:     safeTestOrigin + "/win",
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
				client:         safeTestClient(downstream),
				middlemanStore: store,
				publishWinLossFunc: func(_ []byte) error {
					return nil
				},
			}
			req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/win", "tok", "1.100"), nil)
			for i := 0; i < 2; i++ {
				rr := httptest.NewRecorder()
				controller.ServeMiddlemanCallback(rr, req)
				if rr.Code != tt.wantStatus {
					t.Fatalf("status = %d body=%s, want %d", rr.Code, rr.Body.String(), tt.wantStatus)
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
	rr = httptest.NewRecorder()
	controller.ServeMiddlemanCallback(rr, req)
	if rr.Code != http.StatusNoContent || len(published) != 1 {
		t.Fatalf("duplicate status = %d published=%d, want no duplicate click", rr.Code, len(published))
	}
}

func TestServeMiddlemanClickPublishFailureAllowsRetry(t *testing.T) {
	store := newMemoryMiddlemanCallbackStore()
	value := middlemanCallbackContext{
		Token: "tok", RequestID: "req", ImpID: "imp", ResponseBidID: "resp",
		DownstreamBidPrice: 1, UpstreamBidPrice: 1.1,
		RAdv: match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), value.Token, value, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.SetClick(context.Background(), "rtok", "imp", value.Token, time.Hour); err != nil {
		t.Fatal(err)
	}
	publishAttempts := 0
	controller := &Controller{
		C: &Config{TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"}, middlemanStore: store,
		publishWinLossFunc: func([]byte) error {
			publishAttempts++
			if publishAttempts == 1 {
				return errors.New("publish failed")
			}
			return nil
		},
	}
	clickURL, err := controller.middlemanClickProxyURL("rtok", "imp")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, clickURL, nil)
	first := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(first, req)
	if first.Code != http.StatusServiceUnavailable {
		t.Fatalf("first status = %d, want publish failure", first.Code)
	}
	retry := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(retry, req)
	if retry.Code != http.StatusNoContent || publishAttempts != 2 {
		t.Fatalf("retry status = %d attempts=%d", retry.Code, publishAttempts)
	}
}

type failCompleteNotifyStore struct {
	*memoryMiddlemanCallbackStore
	failures int
}

type failingMiddlemanLookupStore struct {
	*memoryMiddlemanCallbackStore
	callbackErr error
	clickErr    error
}

func (s *failingMiddlemanLookupStore) GetCallback(ctx context.Context, token string) (middlemanCallbackContext, error) {
	if s.callbackErr != nil {
		return middlemanCallbackContext{}, s.callbackErr
	}
	return s.memoryMiddlemanCallbackStore.GetCallback(ctx, token)
}

func (s *failingMiddlemanLookupStore) GetClick(ctx context.Context, requestToken, impID string) (string, error) {
	if s.clickErr != nil {
		return "", s.clickErr
	}
	return s.memoryMiddlemanCallbackStore.GetClick(ctx, requestToken, impID)
}

type contextCheckingClearStore struct {
	*memoryMiddlemanCallbackStore
	sawCanceledContext bool
}

func (s *contextCheckingClearStore) ClearPublish(ctx context.Context, token, source, owner string) error {
	s.sawCanceledContext = ctx.Err() != nil
	return s.memoryMiddlemanCallbackStore.ClearPublish(ctx, token, source, owner)
}

func TestMiddlemanPublishFailureCleanupDetachesFromRequestCancellation(t *testing.T) {
	store := &contextCheckingClearStore{memoryMiddlemanCallbackStore: newMemoryMiddlemanCallbackStore()}
	ctx, cancel := context.WithCancel(context.Background())
	publishAttempts := 0
	controller := &Controller{
		middlemanStore: store,
		publishWinLossFunc: func([]byte) error {
			publishAttempts++
			cancel()
			return errors.New("publish failed")
		},
	}
	value := middlemanCallbackContext{
		Token: "tok", RAdv: match.RAdv{},
		DownstreamBidPrice: 1, UpstreamBidPrice: 1,
	}
	if err := controller.publishMiddlemanEventOnce(ctx, StatusTrackClk, value, "click", value.reconciledPrices(), "none", 0); err == nil {
		t.Fatal("publication failure succeeded")
	}
	if store.sawCanceledContext {
		t.Fatal("claim cleanup inherited request cancellation")
	}
	controller.publishWinLossFunc = func([]byte) error { publishAttempts++; return nil }
	if err := controller.publishMiddlemanEventOnce(context.Background(), StatusTrackClk, value, "click", value.reconciledPrices(), "none", 0); err != nil {
		t.Fatal(err)
	}
	if publishAttempts != 2 {
		t.Fatalf("publication attempts = %d, want failed attempt plus retry", publishAttempts)
	}
}

func (s *failCompleteNotifyStore) CompleteNotify(ctx context.Context, token, source, owner string, state middlemanForwardState, ttl time.Duration) error {
	if s.failures > 0 {
		s.failures--
		return errors.New("notify completion failed")
	}
	return s.memoryMiddlemanCallbackStore.CompleteNotify(ctx, token, source, owner, state, ttl)
}

func TestMiddlemanPostForwardStateFailureRetainsAtLeastOnceBoundary(t *testing.T) {
	forwarded := 0
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		forwarded++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer downstream.Close()
	store := &failCompleteNotifyStore{memoryMiddlemanCallbackStore: newMemoryMiddlemanCallbackStore(), failures: 1}
	value := middlemanCallbackContext{
		Token: "tok", RequestID: "req", ImpID: "imp", ResponseBidID: "resp", DownstreamNURL: safeTestOrigin,
		DownstreamBidPrice: 1, UpstreamBidPrice: 1.1,
		RAdv: match.RAdv{Demand: match.Demand{AdvID: 8, CampaignID: 101, ItemID: 102, CreativeID: 103}},
	}
	if err := store.SetCallback(context.Background(), value.Token, value, time.Hour); err != nil {
		t.Fatal(err)
	}
	published := 0
	controller := &Controller{
		C:      &Config{TrackingSecret: "test-secret", MiddlemanCallbackBaseURL: "http://aofei.example"},
		client: safeTestClient(downstream), middlemanStore: store,
		publishWinLossFunc: func([]byte) error { published++; return nil },
	}
	req := httptest.NewRequest(http.MethodGet, signedMiddlemanTestURL(t, controller, "/mid/win", value.Token, "1.000"), nil)
	first := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(first, req)
	if first.Code != http.StatusServiceUnavailable || published != 0 {
		t.Fatalf("first callback status=%d published=%d", first.Code, published)
	}
	retry := httptest.NewRecorder()
	controller.ServeMiddlemanCallback(retry, req)
	if retry.Code != http.StatusNoContent || published != 1 || forwarded != 2 {
		t.Fatalf("retry status=%d published=%d forwarded=%d; want at-least-once downstream retry", retry.Code, published, forwarded)
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

package dsp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestMiddlemanRoutesRefreshesOnceForConcurrentRequests(t *testing.T) {
	cache := &match.MiddlemanRouteCache{Version: match.MiddlemanRouteCacheVersion}
	started := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	controller := &Controller{
		C: &Config{MiddlemanRouteCacheTTLMS: 5000, MiddlemanTimeoutMS: 1000},
		middlemanRouteLoad: func(context.Context) (*match.MiddlemanRouteCache, error) {
			if loads.Add(1) == 1 {
				close(started)
			}
			<-release
			return cache, nil
		},
	}
	const callers = 16
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := controller.middlemanRoutes(context.Background())
			if err == nil && got != cache {
				err = errors.New("unexpected route cache pointer")
			}
			errs <- err
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("route cache loads = %d, want 1", got)
	}
}

func TestMiddlemanRoutesCachesRefreshErrorWithoutUsingStaleRoutes(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	loads := 0
	wantErr := errors.New("redis unavailable")
	controller := &Controller{
		C: &Config{MiddlemanRouteCacheTTLMS: 5000},
		middlemanRouteNow: func() time.Time {
			return now
		},
		middlemanRouteLoad: func(context.Context) (*match.MiddlemanRouteCache, error) {
			loads++
			if loads == 1 {
				return &match.MiddlemanRouteCache{Version: match.MiddlemanRouteCacheVersion}, nil
			}
			return nil, wantErr
		},
	}
	if _, err := controller.middlemanRoutes(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(6 * time.Second)
	if cache, err := controller.middlemanRoutes(context.Background()); !errors.Is(err, wantErr) || cache != nil {
		t.Fatalf("expired refresh = cache %#v, err %v", cache, err)
	}
	now = now.Add(time.Second)
	if cache, err := controller.middlemanRoutes(context.Background()); !errors.Is(err, wantErr) || cache != nil {
		t.Fatalf("cached refresh error = cache %#v, err %v", cache, err)
	}
	if loads != 2 {
		t.Fatalf("route cache loads = %d, want 2", loads)
	}
}

func TestMiddlemanRoutesCanceledCallerStillPopulatesSharedRefresh(t *testing.T) {
	cache := &match.MiddlemanRouteCache{Version: match.MiddlemanRouteCacheVersion}
	started := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	controller := &Controller{
		C: &Config{MiddlemanRouteCacheTTLMS: 5000, MiddlemanTimeoutMS: 1000},
		middlemanRouteLoad: func(ctx context.Context) (*match.MiddlemanRouteCache, error) {
			if loads.Add(1) == 1 {
				close(started)
			}
			<-release
			return cache, ctx.Err()
		},
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := controller.middlemanRoutes(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled caller error = %v", err)
	}
	<-started
	waiterResult := make(chan error, 1)
	go func() {
		got, err := controller.middlemanRoutes(context.Background())
		if err == nil && got != cache {
			err = errors.New("unexpected route cache pointer")
		}
		waiterResult <- err
	}()
	close(release)
	if err := <-waiterResult; err != nil {
		t.Fatalf("shared refresh waiter failed: %v", err)
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("route cache loads = %d, want 1", got)
	}
}

func TestMiddlemanRoutesInitiatorCancellationDoesNotFailWaiter(t *testing.T) {
	started := make(chan struct{})
	waiting := make(chan struct{})
	release := make(chan struct{})
	cache := &match.MiddlemanRouteCache{Version: match.MiddlemanRouteCacheVersion}
	var loads atomic.Int32
	controller := &Controller{
		C: &Config{MiddlemanRouteCacheTTLMS: 5000, MiddlemanTimeoutMS: 1000},
		middlemanRouteLoad: func(ctx context.Context) (*match.MiddlemanRouteCache, error) {
			loads.Add(1)
			close(started)
			<-release
			return cache, ctx.Err()
		},
		middlemanRouteWaitHook: func() {
			close(waiting)
		},
	}
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leaderErr := make(chan error, 1)
	go func() {
		_, err := controller.middlemanRoutes(leaderCtx)
		leaderErr <- err
	}()
	<-started

	waiterResult := make(chan error, 1)
	go func() {
		got, err := controller.middlemanRoutes(context.Background())
		if err == nil && got != cache {
			err = errors.New("unexpected route cache pointer")
		}
		waiterResult <- err
	}()
	<-waiting
	cancelLeader()

	if err := <-leaderErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader error = %v, want context canceled", err)
	}
	close(release)
	if err := <-waiterResult; err != nil {
		t.Fatalf("waiter inherited initiator cancellation: %v", err)
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("route cache loads = %d, want 1", got)
	}
}

func TestMiddlemanRoutesCachesSharedLoadTimeoutWithoutStaleRoutes(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	loads := 0
	controller := &Controller{
		C: &Config{MiddlemanRouteCacheTTLMS: 5000, MiddlemanTimeoutMS: 20},
		middlemanRouteNow: func() time.Time {
			return now
		},
		middlemanRouteLoad: func(ctx context.Context) (*match.MiddlemanRouteCache, error) {
			loads++
			if loads == 1 {
				return &match.MiddlemanRouteCache{Version: match.MiddlemanRouteCacheVersion}, nil
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	if _, err := controller.middlemanRoutes(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(6 * time.Second)
	if cache, err := controller.middlemanRoutes(context.Background()); !errors.Is(err, context.DeadlineExceeded) || cache != nil {
		t.Fatalf("timed-out refresh = cache %#v, err %v", cache, err)
	}
	now = now.Add(time.Second)
	if cache, err := controller.middlemanRoutes(context.Background()); !errors.Is(err, context.DeadlineExceeded) || cache != nil {
		t.Fatalf("cached timeout = cache %#v, err %v", cache, err)
	}
	if loads != 2 {
		t.Fatalf("route cache loads = %d, want 2", loads)
	}
}

func TestMiddlemanCandidatesDedupAndCap(t *testing.T) {
	slotType, slotID := match.EntityPointer(32, 30)
	siteType, siteID := match.EntityPointer(31, 20)
	cache := &match.MiddlemanRouteCache{
		Version: match.MiddlemanRouteCacheVersion,
		Entries: []match.MiddlemanRouteEntry{
			{TargetID: 3, RouteBidderID: 3, BidderID: 3, RouteBidderPriority: 30, TargetPriority: 1},
			{TargetID: 2, RouteBidderID: 2, BidderID: 2, RouteBidderPriority: 20, TargetPriority: 1},
			{TargetID: 4, RouteBidderID: 4, BidderID: 2, RouteBidderPriority: 99, TargetPriority: 1, EntityTypeID: slotType, EntityID: slotID},
			{TargetID: 1, RouteBidderID: 1, BidderID: 1, RouteBidderPriority: 10, TargetPriority: 1, EntityTypeID: siteType, EntityID: siteID},
		},
	}
	attr := &match.Attribute{
		RPub: match.RPub{PubID: 10, SiteID: 20, SlotID: 30, SizeID: 4194368},
		ACL:  &acl.ACL{SiteType: acl.SiteTypeWeb},
	}

	got := middlemanCandidatesForImp(cache, middlemanFallbackImp{Index: 0, Attr: attr}, 2)
	if len(got) != 2 {
		t.Fatalf("len(candidates) = %d, want 2", len(got))
	}
	if got[0].Entry.BidderID != 1 || got[1].Entry.BidderID != 2 {
		t.Fatalf("candidate bidders = %d,%d; want 1,2", got[0].Entry.BidderID, got[1].Entry.BidderID)
	}
	if got[1].Entry.RouteBidderID != 2 {
		t.Fatalf("dedupe chose route bidder %d, want lower priority route 2", got[1].Entry.RouteBidderID)
	}
}

func TestMiddlemanCandidatesTriggerModes(t *testing.T) {
	cache := &match.MiddlemanRouteCache{
		Version: match.MiddlemanRouteCacheVersion,
		Entries: []match.MiddlemanRouteEntry{
			{TargetID: 1, RouteBidderID: 1, BidderID: 1, TriggerMode: "Fallback", RouteBidderPriority: 10, TargetPriority: 1},
			{TargetID: 2, RouteBidderID: 2, BidderID: 2, TriggerMode: "Always", RouteBidderPriority: 20, TargetPriority: 1},
		},
	}
	attr := &match.Attribute{RPub: match.RPub{PubID: 10, SiteID: 20, SlotID: 30, SizeID: 4194368}, ACL: &acl.ACL{SiteType: acl.SiteTypeWeb}}

	fallbackOnly := middlemanCandidatesForImp(cache, middlemanFallbackImp{Index: 0, Attr: attr}, 5)
	if len(fallbackOnly) != 1 || fallbackOnly[0].Entry.BidderID != 1 {
		t.Fatalf("fallback candidates = %#v, want only fallback bidder", fallbackOnly)
	}

	alwaysOnly := middlemanCandidatesForImp(cache, middlemanFallbackImp{Index: 0, Attr: attr, TriggerModes: []string{"Always"}}, 5)
	if len(alwaysOnly) != 1 || alwaysOnly[0].Entry.BidderID != 2 {
		t.Fatalf("always candidates = %#v, want only always bidder", alwaysOnly)
	}

	both := middlemanCandidatesForImp(cache, middlemanFallbackImp{Index: 0, Attr: attr, TriggerModes: []string{"Fallback", "Always"}}, 5)
	if len(both) != 2 {
		t.Fatalf("both candidates = %#v, want both trigger modes", both)
	}
}

func TestChooseMiddlemanWinnerByEffectiveCPM(t *testing.T) {
	local := bidWinner{ImpIndex: 0, EffectiveCPM: 2.0, Comparable: true, Local: true}
	low := middlemanDownstreamBid{ImpIndex: 0, Seat: "mid", Bid: openrtb2.Bid{ImpID: "imp", Price: 1.9}}
	if _, replace := chooseMiddlemanWinner(local, true, low); replace {
		t.Fatal("lower middleman bid replaced local winner")
	}

	high := middlemanDownstreamBid{ImpIndex: 0, Seat: "mid", Bid: openrtb2.Bid{ImpID: "imp", Price: 2.1}}
	winner, replace := chooseMiddlemanWinner(local, true, high)
	if !replace || winner.Local || winner.Bid.Price != 2.1 {
		t.Fatalf("winner = %#v replace=%v, want middleman", winner, replace)
	}

	unsafeLocal := bidWinner{ImpIndex: 0, Local: true}
	if _, replace := chooseMiddlemanWinner(unsafeLocal, true, high); replace {
		t.Fatal("middleman replaced unsafe local winner")
	}
}

func TestMaterializeBidWinnersSupportsMixedLocalAndMiddleman(t *testing.T) {
	store := newMemoryMiddlemanCallbackStore()
	controller := &Controller{
		C: &Config{
			ServerURL:                "http://aofei.example",
			TrackingSecret:           "test-secret",
			MiddlemanCallbackBaseURL: "http://aofei.example",
		},
		middlemanStore: store,
	}
	bid := &openrtb2.BidRequest{
		ID: "req",
		Imp: []openrtb2.Imp{
			{ID: "local-imp"},
			{ID: "mid-imp"},
		},
	}
	local := bidWinner{
		ImpIndex:      0,
		Seat:          "local-seat",
		Bid:           openrtb2.Bid{ID: "local-bid", ImpID: "local-imp", Price: 1.2},
		Audit:         bidAudit{Attr: &match.Attribute{}, One: match.RAdv{Demand: match.Demand{CampaignID: 10}}},
		ResponseBidID: "local-response",
		Comparable:    true,
		Local:         true,
	}
	mid := middlemanDownstreamBid{
		ImpIndex:           1,
		Seat:               "mid-seat",
		Bid:                openrtb2.Bid{ID: "mid-bid", ImpID: "mid-imp", Price: 2.2},
		Audit:              bidAudit{Attr: &match.Attribute{}, One: match.RAdv{Demand: match.Demand{CampaignID: 20}}},
		ResponseBidID:      "mid-response",
		Entry:              match.MiddlemanRouteEntry{BidderID: 7},
		DownstreamBidPrice: 2.0,
		UpstreamBidPrice:   2.2,
	}
	midWinner, replace := chooseMiddlemanWinner(bidWinner{}, false, mid)
	if !replace {
		t.Fatal("middleman should win empty impression")
	}
	seatOrder, seatBids, audits, responseBidID, materialized := controller.materializeBidWinners(
		context.Background(),
		bid,
		map[int]bidWinner{0: local, 1: midWinner},
		map[int]bidWinner{0: local},
		nil,
	)
	if responseBidID != "local-response" {
		t.Fatalf("response bid id = %q, want first local response", responseBidID)
	}
	if len(seatOrder) != 2 || seatOrder[0] != "local-seat" || seatOrder[1] != "mid-seat" {
		t.Fatalf("seat order = %#v", seatOrder)
	}
	if len(seatBids["local-seat"]) != 1 || len(seatBids["mid-seat"]) != 1 {
		t.Fatalf("seat bids = %#v", seatBids)
	}
	if seatBids["mid-seat"][0].NURL == "" || seatBids["mid-seat"][0].LURL == "" {
		t.Fatalf("middleman callbacks were not proxied: %#v", seatBids["mid-seat"][0])
	}
	if len(audits) != 2 || audits[0].One.CampaignID != 10 || audits[1].One.CampaignID != 20 {
		t.Fatalf("audits = %#v", audits)
	}
	if len(materialized) != 2 || !materialized[0].Local || materialized[1].Local {
		t.Fatalf("materialized winners = %#v", materialized)
	}
}

func TestMiddlemanRequestExtAndPreservesImps(t *testing.T) {
	raw := []byte(`{"id":"req-1","imp":[{"id":"imp-1"},{"id":"imp-2"}],"ext":{"request_domain":"old.example","kept":true},"vendor_unknown":{"x":1}}`)
	body, err := middlemanRequestBodyForAssignment(raw, "aofei.example")
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatal(err)
	}
	var got openrtb2.BidRequest
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Imp) != 2 || got.Imp[0].ID != "imp-1" || got.Imp[1].ID != "imp-2" {
		t.Fatalf("forwarded imps = %#v, want full original imp list", got.Imp)
	}
	var ext map[string]any
	if err := json.Unmarshal(root["ext"], &ext); err != nil {
		t.Fatal(err)
	}
	if ext["request_domain"] != "aofei.example" || ext["kept"] != true {
		t.Fatalf("ext = %#v", ext)
	}
	if _, ok := root["vendor_unknown"]; !ok {
		t.Fatalf("unknown top-level field was not preserved: %s", body)
	}
}

func TestMiddlemanCredentialHeaders(t *testing.T) {
	t.Setenv("BIDDER_HEADERS", `{"Authorization":"Bearer test","X-Bidder":"one"}`)
	headers, err := middlemanCredentialHeaders("BIDDER_HEADERS")
	if err != nil {
		t.Fatal(err)
	}
	if headers.Get("Authorization") != "Bearer test" || headers.Get("X-Bidder") != "one" {
		t.Fatalf("headers = %#v", headers)
	}

	t.Setenv("BAD_BIDDER_HEADERS", `{"Host":"evil.example"}`)
	if _, err := middlemanCredentialHeaders("BAD_BIDDER_HEADERS"); err == nil {
		t.Fatalf("blocked header should fail")
	}
}

func TestCallMiddlemanBidderNormalizesResponse(t *testing.T) {
	t.Setenv("BIDDER_HEADERS", `{"Authorization":"Bearer test"}`)
	var received openrtb2.BidRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Errorf("Authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("x-openrtb-version") != "2.5" {
			t.Errorf("x-openrtb-version = %q", r.Header.Get("x-openrtb-version"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(openrtb2.BidResponse{
			ID:    "req-1",
			BidID: "resp-1",
			Cur:   "USD",
			SeatBid: []openrtb2.SeatBid{{
				Seat: "downstream",
				Bid: []openrtb2.Bid{{
					ID:    "bid-1",
					ImpID: "imp-2",
					Price: 2.0,
					AdM:   "<img>",
					CID:   "downstream-campaign",
					CrID:  "downstream-creative",
					AdID:  "downstream-ad",
					NURL:  "http://downstream.example/win?auction_price=${AUCTION_PRICE}",
					BURL:  "http://downstream.example/bill?auction_price=${AUCTION_PRICE}",
					LURL:  "http://downstream.example/loss?auction_price=${AUCTION_PRICE}",
				}},
			}},
		})
	}))
	defer server.Close()

	entry := match.MiddlemanRouteEntry{
		BidderID:            7,
		AdvID:               8,
		EndpointURL:         server.URL,
		OpenRTBVersion:      "2.5",
		CredentialRef:       "BIDDER_HEADERS",
		GroupTimeoutMS:      100,
		BidderTimeoutMS:     100,
		GroupMarginPct:      0.10,
		GroupMinMarginCPM:   0.50,
		SyntheticCampaignID: 101,
		SyntheticItemID:     102,
		SyntheticCreativeID: 103,
	}
	bid := &openrtb2.BidRequest{
		ID:   "req-1",
		TMax: 100,
		Ext:  json.RawMessage(`{"request_domain":"old.example"}`),
		Imp: []openrtb2.Imp{
			{ID: "imp-1", BidFloor: 0.1, BidFloorCur: "USD"},
			{ID: "imp-2", BidFloor: 2.4, BidFloorCur: "USD"},
		},
	}
	assignment := middlemanAssignment{
		Entry: entry,
		EntriesByID: map[string]match.MiddlemanRouteEntry{
			"imp-2": entry,
		},
		AttrsByID: map[string]*match.Attribute{
			"imp-2": {RPub: match.RPub{PubID: 1, SiteID: 2, SlotID: 3, SizeID: 4}, ACL: &acl.ACL{}},
		},
	}
	controller := &Controller{
		C: &Config{
			ServerURL:                "http://aofei.example",
			TrackingSecret:           "test-secret",
			MiddlemanExchangeDomain:  "aofei.example",
			MiddlemanTimeoutMS:       100,
			MiddlemanCallbackBaseURL: "http://aofei.example",
		},
		client:           server.Client(),
		callbackURLGuard: func(context.Context, string) error { return nil },
	}

	raw, err := json.Marshal(bid)
	if err != nil {
		t.Fatal(err)
	}
	got, err := controller.callMiddlemanBidder(context.Background(), server.Client(), bid, raw, time.Now(), assignment)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len(bids) = %d, want 1", len(got))
	}
	if len(received.Imp) != 2 || received.Imp[0].ID != "imp-1" || received.Imp[1].ID != "imp-2" {
		t.Fatalf("received imps = %#v", received.Imp)
	}
	var ext map[string]any
	if err := json.Unmarshal(received.Ext, &ext); err != nil {
		t.Fatal(err)
	}
	if ext["request_domain"] != "aofei.example" {
		t.Fatalf("request_domain = %#v", ext["request_domain"])
	}
	mid, ok := ext["aofei_middleman"].(map[string]any)
	if !ok {
		t.Fatalf("aofei_middleman ext missing: %#v", ext)
	}
	clicks, ok := mid["click_notify_urls"].(map[string]any)
	if !ok || clicks["imp-2"] == "" {
		t.Fatalf("click notify URLs missing: %#v", mid)
	}
	if got[0].Bid.Price != 2.5 {
		t.Fatalf("price = %f, want 2.5", got[0].Bid.Price)
	}
	if got[0].Seat != "101" || got[0].Bid.CID != "101" || got[0].Bid.CrID != "103" || got[0].Bid.AdID != "103" {
		t.Fatalf("synthetic ids not applied: seat=%s bid=%#v", got[0].Seat, got[0].Bid)
	}
	if got[0].Audit.One.ItemID != 102 || got[0].Audit.One.CostType != 2 || got[0].ResponseBidID != "resp-1" {
		t.Fatalf("audit/response id = %#v %q", got[0].Audit.One, got[0].ResponseBidID)
	}
	if got[0].DownstreamBidPrice != 2.0 || got[0].UpstreamBidPrice != 2.5 || got[0].DownstreamAdID != "downstream-ad" {
		t.Fatalf("downstream accounting = %#v", got[0])
	}
}

func TestCallMiddlemanBidderRejectsUnsafeEndpointWhenClientMissing(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ServerURL:               "http://aofei.example",
			MiddlemanExchangeDomain: "aofei.example",
			MiddlemanTimeoutMS:      100,
		},
	}
	bid := &openrtb2.BidRequest{
		ID:   "req-1",
		TMax: 100,
		Imp:  []openrtb2.Imp{{ID: "imp-1", BidFloorCur: "USD"}},
	}
	raw, err := json.Marshal(bid)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.callMiddlemanBidder(context.Background(), nil, bid, raw, time.Now(), middlemanAssignment{
		Entry: match.MiddlemanRouteEntry{
			EndpointURL:         "http://127.0.0.1:1/bid",
			GroupTimeoutMS:      100,
			BidderTimeoutMS:     100,
			SyntheticCampaignID: 101,
			SyntheticItemID:     102,
			SyntheticCreativeID: 103,
		},
		EntriesByID: map[string]match.MiddlemanRouteEntry{"imp-1": {}},
		AttrsByID:   map[string]*match.Attribute{"imp-1": {ACL: &acl.ACL{}}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe callback host") {
		t.Fatalf("callMiddlemanBidder error = %v, want unsafe endpoint rejection", err)
	}
}

func TestCallMiddlemanBidderRejectsUnsafeEndpointWithCustomClient(t *testing.T) {
	controller := &Controller{
		C: &Config{
			ServerURL:               "http://aofei.example",
			MiddlemanExchangeDomain: "aofei.example",
			MiddlemanTimeoutMS:      100,
		},
	}
	bid := &openrtb2.BidRequest{
		ID:   "req-1",
		TMax: 100,
		Imp:  []openrtb2.Imp{{ID: "imp-1", BidFloorCur: "USD"}},
	}
	raw, err := json.Marshal(bid)
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.callMiddlemanBidder(context.Background(), http.DefaultClient, bid, raw, time.Now(), middlemanAssignment{
		Entry: match.MiddlemanRouteEntry{
			EndpointURL:         "http://127.0.0.1:1/bid",
			GroupTimeoutMS:      100,
			BidderTimeoutMS:     100,
			SyntheticCampaignID: 101,
			SyntheticItemID:     102,
			SyntheticCreativeID: 103,
		},
		EntriesByID: map[string]match.MiddlemanRouteEntry{"imp-1": {}},
		AttrsByID:   map[string]*match.Attribute{"imp-1": {ACL: &acl.ACL{}}},
	})
	if err == nil || !strings.Contains(err.Error(), "unsafe callback host") {
		t.Fatalf("callMiddlemanBidder error = %v, want unsafe endpoint rejection", err)
	}
}

func TestMiddlemanAssignmentTimeout(t *testing.T) {
	entry := match.MiddlemanRouteEntry{GroupTimeoutMS: 90, BidderTimeoutMS: 80, RouteTimeoutMS: match.Uint16Pointer(70)}
	got, ok := middlemanAssignmentTimeout(&openrtb2.BidRequest{TMax: 60}, middlemanAssignment{
		EntriesByID: map[string]match.MiddlemanRouteEntry{"imp": entry},
	}, 100, 0)
	if !ok || got != 60*time.Millisecond {
		t.Fatalf("timeout = %s, want 60ms", got)
	}

	got, ok = middlemanAssignmentTimeout(&openrtb2.BidRequest{TMax: 60}, middlemanAssignment{
		EntriesByID: map[string]match.MiddlemanRouteEntry{"imp": entry},
	}, 100, 25*time.Millisecond)
	if !ok || got != 35*time.Millisecond {
		t.Fatalf("remaining timeout = %s, want 35ms", got)
	}

	_, ok = middlemanAssignmentTimeout(&openrtb2.BidRequest{TMax: 60}, middlemanAssignment{
		EntriesByID: map[string]match.MiddlemanRouteEntry{"imp": entry},
	}, 100, 60*time.Millisecond)
	if ok {
		t.Fatalf("exhausted tmax should not allow fanout")
	}
}

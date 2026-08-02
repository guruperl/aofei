package dsp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestAuctionRejectsInvalidHigherCreativeBeforeDeliveryReservation(t *testing.T) {
	controller := newLocalBidPathController(t)
	server := attachAuctionRedis(t, controller)
	now := time.Now()
	high := auctionDeliveryRAdv(t, now, 2, 20, 2000, 99999, 3, 91)
	low := auctionDeliveryRAdv(t, now, 1, 10, 1000, 10000, 2, 92)
	sizeID := match.SizeID2To1(300, 250)
	controller.local.mu.Lock()
	controller.local.radvs[sizeID][100] = match.RAdvs{high, low}
	controller.local.mu.Unlock()

	rr := serveSmokeBid(t, controller, "pub.example", marshalBidRequest(t, localBidRequest("USD", "USD")))
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte(`"price":2`)) {
		t.Fatalf("response = %d %s, want lower valid CPM winner", rr.Code, rr.Body.String())
	}
	if server.Exists(deliveryTotalKey(91)) {
		t.Fatal("invalid higher-price creative acquired a delivery reservation")
	}
	if got := server.HGet(deliveryTotalKey(92), "used_imp"); got != "1" {
		t.Fatalf("valid fallback used_imp = %q, want 1", got)
	}
}

func TestServeBidWriteFailureReleasesDeliveryReservation(t *testing.T) {
	controller := newLocalBidPathController(t)
	server := attachAuctionRedis(t, controller)
	installAuctionDelivery(t, controller, 93)
	req := httptest.NewRequest(http.MethodPost, "/bid/pub.example", bytes.NewReader(marshalBidRequest(t, localBidRequest("USD", "USD"))))
	req.SetPathValue("domain", "pub.example")
	w := &failedBidWriter{header: make(http.Header)}

	controller.ServeBid(w, req)

	if w.status != http.StatusOK || w.writes != 1 {
		t.Fatalf("writer status/writes = %d/%d, want 200/1", w.status, w.writes)
	}
	if got := server.HGet(deliveryTotalKey(93), "used_imp"); got != "0" {
		t.Fatalf("used_imp after failed response = %q, want 0", got)
	}
}

func TestServeSSPWriteFailureReleasesDeliveryReservation(t *testing.T) {
	controller := newLocalBidPathController(t)
	server := attachAuctionRedis(t, controller)
	installAuctionDelivery(t, controller, 94)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{{
		Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1,
	}})
	req := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	req.Header.Set("Origin", "https://example.com")
	w := &failedBidWriter{header: make(http.Header)}

	controller.ServeSSP(w, req)

	if w.status != http.StatusOK || w.writes != 1 {
		t.Fatalf("writer status/writes = %d/%d, want 200/1", w.status, w.writes)
	}
	if got := server.HGet(deliveryTotalKey(94), "used_imp"); got != "0" {
		t.Fatalf("used_imp after failed SSP response = %q, want 0", got)
	}
}

func TestHigherMiddlemanWinnerReleasesLocalDeliveryReservation(t *testing.T) {
	controller := newLocalBidPathController(t)
	server := attachAuctionRedis(t, controller)
	installAuctionDelivery(t, controller, 95)
	runtime := &recordingMiddlemanRuntime{
		makeBids: func(bid *openrtb2.BidRequest, fallbackImps []middlemanFallbackImp) []middlemanDownstreamBid {
			return []middlemanDownstreamBid{sspMiddlemanBid(0, bid.Imp[0].ID, "<div>middleman-winner</div>", 3, fallbackImps[0].Attr)}
		},
	}
	enableSSPMiddleman(controller, true, runtime)
	body := sspRequestBody(t, 1, 10, []sspAdUnitSpec{{
		Code: "unit-one", SlotID: 100, SizeID: match.SizeID2To1(300, 250), Banner: true, Floor: 1,
	}})

	rr := serveSSP(t, controller, body)

	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("middleman-winner")) {
		t.Fatalf("response = %d %s, want middleman winner", rr.Code, rr.Body.String())
	}
	if got := server.HGet(deliveryTotalKey(95), "used_imp"); got != "0" {
		t.Fatalf("used_imp after middleman replacement = %q, want 0", got)
	}
}

func attachAuctionRedis(t *testing.T, controller *Controller) *miniredis.Miniredis {
	t.Helper()
	server := miniredis.RunT(t)
	client, err := (radix.PoolConfig{Size: 2}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	controller.Redis = client
	return server
}

func installAuctionDelivery(t *testing.T, controller *Controller, balanceID uint32) {
	t.Helper()
	sizeID := match.SizeID2To1(300, 250)
	block := auctionDeliveryRAdv(t, time.Now(), 1, 10, 1000, 10000, 2, balanceID)
	controller.local.mu.Lock()
	controller.local.radvs[sizeID][100] = match.RAdvs{block}
	controller.local.mu.Unlock()
}

func auctionDeliveryRAdv(t *testing.T, now time.Time, advID, campaignID, itemID, creativeID uint32, cpm float32, balanceID uint32) match.RAdv {
	t.Helper()
	delivery := match.Delivery{
		GeneratedAtUnix: now.Unix(),
		ItemTotal:       match.DeliveryBalance{ID: balanceID, LimitImp: 10},
	}
	if err := delivery.SetTimezone("UTC"); err != nil {
		t.Fatal(err)
	}
	return match.RAdv{
		Demand:   match.Demand{AdvID: advID, CampaignID: campaignID, ItemID: itemID, CreativeID: creativeID},
		Weight:   1,
		CostType: match.CostTypeCPM,
		Cost:     cpm,
		Delivery: delivery,
	}
}

type failedBidWriter struct {
	header http.Header
	status int
	writes int
}

func (w *failedBidWriter) Header() http.Header { return w.header }

func (w *failedBidWriter) WriteHeader(status int) { w.status = status }

func (w *failedBidWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("client connection closed")
}

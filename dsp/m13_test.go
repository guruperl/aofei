package dsp

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

func TestServeBidSupportsMultipleImpressions(t *testing.T) {
	controller := newLocalBidPathController(t)

	body := marshalBidRequest(t, localBidRequest("USD", "USD"))
	rr := serveSmokeBid(t, controller, "pub.example", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeBid status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.SeatBid) != 1 {
		t.Fatalf("seat bids = %+v, want one seat", response.SeatBid)
	}
	bids := response.SeatBid[0].Bid
	if len(bids) != 2 {
		t.Fatalf("bid count = %d, want 2: %+v", len(bids), bids)
	}
	if bids[0].ImpID != "imp-1" || bids[1].ImpID != "imp-2" {
		t.Fatalf("imp ids = %q, %q; want imp-1, imp-2", bids[0].ImpID, bids[1].ImpID)
	}
	if bids[0].Price != 2 || bids[1].Price != 3 {
		t.Fatalf("prices = %f, %f; want eCPM prices", bids[0].Price, bids[1].Price)
	}
	if bids[0].ID == bids[1].ID {
		t.Fatalf("bid IDs are not unique: %q", bids[0].ID)
	}
}

func TestServeBidSkipsUnsupportedCurrencyImpression(t *testing.T) {
	controller := newLocalBidPathController(t)

	body := marshalBidRequest(t, localBidRequest("USD", "EUR"))
	rr := serveSmokeBid(t, controller, "pub.example", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("ServeBid status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.SeatBid) != 1 || len(response.SeatBid[0].Bid) != 1 {
		t.Fatalf("response shape = %+v, want one served bid", response.SeatBid)
	}
	if got := response.SeatBid[0].Bid[0].ImpID; got != "imp-1" {
		t.Fatalf("served imp = %q, want imp-1", got)
	}
}

func newLocalBidPathController(t testing.TB) *Controller {
	t.Helper()
	top := t.TempDir()
	sizeOne := match.SizeID2To1(300, 250)
	sizeTwo := match.SizeID2To1(320, 50)

	pub := &acl.Pub{
		AccountingVersion: accounting.ExactMoneyContract,
		PubID:             1,
		Active:            true,
		DefaultWebSiteID:  10,
		DefaultWebSlotID:  100,
		Sites:             map[string]uint32{"example.com": 10, "app.example.com": 11},
		SiteTypes:         map[uint32]acl.SiteType{10: acl.SiteTypeWeb, 11: acl.SiteTypeAPP},
		Slots: map[uint32]map[string]uint32{
			10: map[string]uint32{"slot-one": 100, "slot-two": 200},
			11: map[string]uint32{"slot-one": 100, "slot-two": 200},
		},
		SlotSizes: map[uint32]map[uint32]uint32{
			10: map[uint32]uint32{100: sizeOne, 200: sizeTwo},
			11: map[uint32]uint32{100: sizeOne, 200: sizeTwo},
		},
		SlotFloors: map[uint32]map[uint32]float64{
			10: map[uint32]float64{100: 0, 200: 0},
			11: map[uint32]float64{100: 0, 200: 0},
		},
		SlotFloorCPMs: map[uint32]map[uint32]accounting.CPM{
			10: {100: 0, 200: 0},
			11: {100: 0, 200: 0},
		},
	}
	writePubSnapshot(t, top, "pub.example", pub)
	writeRAdvsSnapshot(t, top, sizeOne, 100, match.RAdvs{{Demand: match.Demand{AdvID: 1, CampaignID: 10, ItemID: 1000, CreativeID: 10000}, Weight: 1, CostType: match.CostTypeCPM, Cost: 2, CostCPM: 2_000_000}})
	writeRAdvsSnapshot(t, top, sizeTwo, 200, match.RAdvs{{Demand: match.Demand{AdvID: 1, CampaignID: 10, ItemID: 2000, CreativeID: 20000}, Weight: 1, CostType: match.CostTypeCPM, Cost: 3, CostCPM: 3_000_000}})
	writeCreativeSnapshot(t, top, 10000, &match.Creative{CreativeName: "one", CreativeContent: "https://cdn.example/one.html?click={CLICK_URL}", Landing: "https://advertiser.example/one", SizeID: sizeOne, MediaType: match.CreativeMediaBanner, MIME: "text/html"})
	writeCreativeSnapshot(t, top, 20000, &match.Creative{CreativeName: "two", CreativeContent: "https://cdn.example/two.html", Landing: "https://advertiser.example/two", SizeID: sizeTwo, MediaType: match.CreativeMediaBanner, MIME: "text/html"})

	controller := &Controller{
		C:      &Config{Spread: top, IsLocal: true, ServerURL: "https://dsp.example", TrackingSecret: "test-secret"},
		Logger: zap.NewNop(),
	}
	if err := controller.ReloadLocalStaticCache(); err != nil {
		t.Fatal(err)
	}
	return controller
}

func localBidRequest(curOne, curTwo string) *openrtb2.BidRequest {
	w1, h1 := int64(300), int64(250)
	w2, h2 := int64(320), int64(50)
	return &openrtb2.BidRequest{
		ID:     "req-1",
		Device: &openrtb2.Device{IP: "203.0.113.1", UA: "test-agent"},
		Site:   &openrtb2.Site{Domain: "example.com"},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", TagID: "slot-one", Banner: &openrtb2.Banner{W: &w1, H: &h1}, BidFloor: 1, BidFloorCur: curOne},
			{ID: "imp-2", TagID: "slot-two", Banner: &openrtb2.Banner{W: &w2, H: &h2}, BidFloor: 1, BidFloorCur: curTwo},
		},
	}
}

func marshalBidRequest(t testing.TB, bid *openrtb2.BidRequest) []byte {
	t.Helper()
	data, err := json.Marshal(bid)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writePubSnapshot(t testing.TB, top, name string, pub *acl.Pub) {
	t.Helper()
	path := filepath.Join(top, acl.HashNamePubmap, name)
	mkdirParent(t, path)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pub.PackIO(f); err != nil {
		t.Fatal(err)
	}
}

func writeRAdvsSnapshot(t testing.TB, top string, sizeID, slotID uint32, radvs match.RAdvs) {
	t.Helper()
	data, err := radvs.Pack()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(top, match.HashIONameRAdvs(sizeID), strconvUint(slotID))
	mkdirParent(t, path)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeCreativeSnapshot(t testing.TB, top string, creativeID uint32, creative *match.Creative) {
	t.Helper()
	path := filepath.Join(top, match.HashNameCreative, strconvUint(creativeID))
	mkdirParent(t, path)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := creative.PackIO(f); err != nil {
		t.Fatal(err)
	}
}

func mkdirParent(t testing.TB, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
}

func strconvUint(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

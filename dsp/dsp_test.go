package dsp

import (
	"net/url"
	"testing"
	"time"

	"github.com/genelet/winter/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// TestBidID creats a new bidID and compares it to the original Bid.
func TestBidID(t *testing.T) {
	user := "user"
	when := time.Now()
	bid := bidID{
		When:   when.UnixNano(),
		UserID: user,
	}
	encoded := bid.String()
	bid2, err := UnpackBidID(encoded)
	if err != nil {
		t.Error(err)
	}
	if bid.When != bid2.When {
		t.Errorf("When %d != %d", bid.When, bid2.When)
	}
	if bid.UserID != bid2.UserID || bid.UserID != user {
		t.Errorf("UserID %s != %s", bid.UserID, bid2.UserID)
	}
}

// TestresponseBidIDPack tests the Pack and Unpack functions of responseBidID.
func TestResponseBidIDPack(t *testing.T) {
	when := time.Now()
	cid := uint32(4)
	bid := &responseBidID{
		When:       when.UnixNano(),
		CreativeID: cid,
		ImpIndex:   2,
	}
	packed := bid.String()
	bid2, err := UnpackResponseBidID(packed)
	if err != nil {
		t.Error(err)
	}
	if bid.When != bid2.When {
		t.Errorf("When %d != %d", bid.When, bid2.When)
	}
	if bid.CreativeID != bid2.CreativeID {
		t.Errorf("CreativeID %d != %d", bid.CreativeID, bid2.CreativeID)
	}
	if bid.ImpIndex != bid2.ImpIndex {
		t.Errorf("ImpIndex %d != %d", bid.ImpIndex, bid2.ImpIndex)
	}
}

func TestResponseBidIDPackLegacy(t *testing.T) {
	when := time.Now()
	packed := (&responseBidID{When: when.UnixNano(), CreativeID: 4}).String()
	legacy := packed[:16] + "4"

	bid, err := UnpackResponseBidID(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if bid.CreativeID != 4 || bid.ImpIndex != 0 {
		t.Fatalf("legacy unpack = %+v, want creative 4 and imp index 0", bid)
	}
}

func TestWinLossUsesSelectedBidPrice(t *testing.T) {
	w, h := int64(300), int64(250)
	bid := &openrtb2.BidRequest{
		ID:     "request-1",
		Device: &openrtb2.Device{},
		Imp:    []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{W: &w, H: &h}}},
	}
	attr := &match.Attribute{
		When: time.Now(),
		RPub: match.RPub{PubID: 1, SiteID: 2, SlotID: 3, SizeID: match.SizeID2To1(300, 250)},
	}
	one := match.RAdv{
		Demand:   match.Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4},
		CostType: 3,
		Cost:     0.02,
	}
	creative := &match.Creative{
		CreativeContent: "https://cdn.example/ad.html",
		SizeID:          match.SizeID2To1(300, 250),
	}
	dsp := NewDSPForImp(bid, 0, attr, one, nil, creative, nil, 2.0, "https://dsp.example")

	winloss := dsp.WinLoss(StatusBid)
	responseBid, err := dsp.NewBid(winloss)
	if err != nil {
		t.Fatal(err)
	}
	if responseBid.Price != 2.0 {
		t.Fatalf("response price = %f, want selected bid price", responseBid.Price)
	}
	if winloss.RAdv.Cost != 2.0 {
		t.Fatalf("winloss cost = %f, want selected bid price", winloss.RAdv.Cost)
	}
	tracker, err := url.Parse(winloss.ImpURL())
	if err != nil {
		t.Fatal(err)
	}
	if got := tracker.Query().Get("auction_price"); got != "2.000000" {
		t.Fatalf("tracker auction_price = %q, want selected bid price", got)
	}
	if got := winloss.Macro()[`${AUCTION_PRICE}`]; got != "2.000" {
		t.Fatalf("macro auction price = %q, want selected bid price", got)
	}
}

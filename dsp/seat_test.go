package dsp

import (
	"testing"
	"time"
)

// TestSeatBidBidPack tests the Pack and Unpack functions of SeatBidBid.
func TestSeatBidBidPack(t *testing.T) {
	when := time.Now().UnixNano()
	cid := uint32(4)
	bid := &SeatBidBid{when, cid}
	packed := bid.Pack()
	bid2, err := UnpackSeatBidBid(packed)
	if err != nil {
		t.Error(err)
	}
	if bid.When != bid2.When {
		t.Errorf("When %d != %d", bid.When, bid2.When)
	}
	if bid.CreativeID != bid2.CreativeID {
		t.Errorf("CreativeID %d != %d", bid.CreativeID, bid2.CreativeID)
	}
}

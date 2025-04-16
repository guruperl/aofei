package dsp

import (
	"testing"
	"time"
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
}

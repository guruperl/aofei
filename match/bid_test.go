package match

import (
	"testing"
	"time"
)

// TestBidID creats a new BidID and compares it to the original Bid.
func TestBidID(t *testing.T) {
	adv := RAdv{
		ItemID:     123,
		CreativeID: 123,
	}
	when := time.Now()
	bid := NewBid(when, adv)
	encoded, err := bid.BidID()
	if err != nil {
		t.Error(err)
	}
	t.Errorf("encoded: %s", encoded)
	bid2, err := UnpackBidID(encoded)
	if err != nil {
		t.Error(err)
	}
	if bid.ItemID != bid2.ItemID {
		t.Errorf("ItemID %d != %d", bid.ItemID, bid2.ItemID)
	}
	if bid.CreativeID != bid2.CreativeID {
		t.Errorf("CreativeID %d != %d", bid.CreativeID, bid2.CreativeID)
	}
	if bid.When != bid2.When {
		t.Errorf("When %d != %d", bid.When, bid2.When)
	}
}

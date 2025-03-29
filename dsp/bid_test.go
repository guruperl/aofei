package match

import (
	"testing"
	"time"
)

// TestBidID creats a new BidID and compares it to the original Bid.
func TestBidID(t *testing.T) {
	user := "user"
	when := time.Now()
	bid := NewBid(when, user)
	encoded := bid.BidID()
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

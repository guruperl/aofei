package dsp

import (
	"fmt"
	"time"
)

type Bid struct {
	When   int64
	UserID string
}

// BidID packs the Bid to string.
func (self Bid) BidID() string {
	return fmt.Sprintf("%16x%s", self.When, self.UserID)
}

// UnpackBidID unpacks the string from Bid.
func UnpackBidID(data string) (Bid, error) {
	var when int64
	var userID string
	_, err := fmt.Sscanf(data, "%16x%s", &when, &userID)
	return Bid{
		When:   when,
		UserID: userID,
	}, err
}

func NewBid(when time.Time, userID string) Bid {
	return Bid{
		When:   when.UnixNano(),
		UserID: userID,
	}
}

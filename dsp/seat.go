package dsp

import (
	"fmt"
	"time"
)

type SeatBidBid struct {
	When       int64
	CreativeID uint32
}

// Pack returns the packed string of SeatBidBid.
func (self SeatBidBid) Pack() string {
	return fmt.Sprintf("%16x%d", self.When, self.CreativeID)
}

// UnpackSeatBidBid unpacks the SeatBidBid from the packed string.
func UnpackSeatBidBid(data string) (SeatBidBid, error) {
	var seatBid SeatBidBid
	_, err := fmt.Sscanf(data, "%16x%d", &seatBid.When, &seatBid.CreativeID)
	return seatBid, err
}

func NewSeatBidBid(when time.Time, creativeID uint32) SeatBidBid {
	return SeatBidBid{
		When:       when.UnixNano(),
		CreativeID: creativeID,
	}
}

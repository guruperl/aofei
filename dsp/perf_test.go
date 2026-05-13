package dsp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func BenchmarkBidResponseMarshal(b *testing.B) {
	response := &openrtb2.BidResponse{
		ID:    "auction",
		BidID: "bid",
		Cur:   "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "seat",
			Bid: []openrtb2.Bid{{
				ID:    "bid-1",
				ImpID: "imp-1",
				Price: 1.25,
				AdM:   `{"native":{"assets":[]}}`,
				NURL:  "https://dsp.example/win",
				LURL:  "https://dsp.example/loss",
			}},
		}},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(response); err != nil {
			b.Fatal(err)
		}
	}
}

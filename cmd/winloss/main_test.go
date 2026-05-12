package main

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestExtractBidResponseTrackersValidatesShape(t *testing.T) {
	tests := []struct {
		name string
		rsp  *openrtb2.BidResponse
		want string
	}{
		{
			name: "no seat bids",
			rsp:  &openrtb2.BidResponse{},
			want: "no seat bids",
		},
		{
			name: "missing bid",
			rsp: &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{
				{},
			}},
			want: "no bids",
		},
		{
			name: "malformed native markup",
			rsp:  responseWithADM("{bad json"),
			want: "unpack native markup",
		},
		{
			name: "missing impression tracker",
			rsp:  responseWithADM(`{"native":{"link":{"clicktrackers":["http://example.test/click"]}}}`),
			want: "missing impression trackers",
		},
		{
			name: "missing click tracker",
			rsp:  responseWithADM(`{"native":{"imptrackers":["http://example.test/imp"],"link":{}}}`),
			want: "missing click trackers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractBidResponseTrackers(tt.rsp)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("extractBidResponseTrackers error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestExtractBidResponseTrackersValidResponse(t *testing.T) {
	trackers, err := extractBidResponseTrackers(responseWithADM(`{"native":{"imptrackers":["http://example.test/imp"],"link":{"clicktrackers":["http://example.test/click"]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if trackers.nurl.String() == "" || trackers.lurl.String() == "" || trackers.imp.String() == "" || trackers.click.String() == "" {
		t.Fatalf("expected all tracker URLs, got %#v", trackers)
	}
}

func responseWithADM(adm string) *openrtb2.BidResponse {
	return &openrtb2.BidResponse{
		ID:    "auction",
		BidID: "bid-response",
		Cur:   "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "seat",
			Bid: []openrtb2.Bid{{
				ID:    "bid",
				ImpID: "imp",
				AdID:  "ad",
				Price: 1.23,
				NURL:  "http://example.test/win?auction_id=${AUCTION_ID}&auction_bid_id=${AUCTION_BID_ID}&auction_imp_id=${AUCTION_IMP_ID}&auction_price=${AUCTION_PRICE}&auction_currency=${AUCTION_CURRENCY}",
				LURL:  "http://example.test/loss?auction_id=${AUCTION_ID}&auction_bid_id=${AUCTION_BID_ID}&auction_imp_id=${AUCTION_IMP_ID}&auction_price=${AUCTION_PRICE}&auction_currency=${AUCTION_CURRENCY}",
				AdM:   adm,
			}},
		}},
	}
}

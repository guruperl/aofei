// Package dsp implements a demand-side platform.
package dsp

import (
	"fmt"
	"net/url"
	"time"

	"github.com/genelet/winter/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type Status uint8

const (
	StatusBid Status = iota
	StatusWin
	StatusLoss
	StatusTrackImp
	StatusTrackClk
)

// WinLoss is a win/loss notification received after bidding.
// ${AUCTION_ID} ID of the bid request; from BidRequest.id attribute.
// ${AUCTION_BID_ID} ID of the bid; from BidResponse.bidid attribute.
// ${AUCTION_IMP_ID} ID of the impression just won; from imp.id attribute.
// ${AUCTION_SEAT_ID} ID of the bidder seat;for whom the bid was made.
// ${AUCTION_AD_ID} ID of the ad markup the bidder wishes to serve; from bid.adid attribute.
type WinLoss struct {
	Status       `json:"status,omitempty"`
	Current      time.Time `json:"current,omitempty"`
	match.RPub   `json:"rpub,omitempty"`
	match.RAdv   `json:"radv,omitempty"`
	BothCap      *match.BothCap `json:"-,omitempty"`
	Seat         string         `json:"seat,omitempty"`
	AuctionID    string         `json:"auction_id,omitempty"`
	AuctionBidID string         `json:"auction_bid_id,omitempty"`
	AuctionImpID string         `json:"auction_imp_id,omitempty"`
	AuctionAdID  string         `json:"auction_ad_id,omitempty"`
	serverURL    string
}

// NewWinLoss creates a new WinLoss instance from the current time, status, rpub, radv, and bothcap.
func NewWinLoss(
	how Status,
	current time.Time,
	rpub match.RPub,
	radv match.RAdv,
	bothcap *match.BothCap,
	seat, auctionID, auctionBidID, auctionImpID, auctionAdID, serverURL string,
) *WinLoss {
	return &WinLoss{
		Status:       how,
		Current:      current,
		RPub:         rpub,
		RAdv:         radv,
		BothCap:      bothcap,
		Seat:         seat,
		AuctionID:    auctionID,
		AuctionBidID: auctionBidID,
		AuctionImpID: auctionImpID,
		AuctionAdID:  auctionAdID,
		serverURL:    serverURL,
	}
}

// Macro returns the replacement macro hash
func (self *WinLoss) Macro() map[string]string {
	return map[string]string{
		`${AUCTION_ID}`:       self.AuctionID,
		`${AUCTION_BID_ID}`:   self.AuctionBidID,
		`${AUCTION_IMP_ID}`:   self.AuctionImpID,
		`${AUCTION_AD_ID}`:    self.AuctionAdID,
		`${AUCTION_SEAT_ID}`:  self.Seat,
		`${AUCTION_PRICE}`:    fmt.Sprintf("%.3f", self.RAdv.Cost),
		`${AUCTION_CURRENCY}`: "USD",
	}
}

// NURL returns the notification URL for the win/loss notification.
func (self *WinLoss) NURL() string {
	return self.serverURL + "/win?" + self.PackURLString()
}

// LURL returns the loss URL for the win/loss notification.
func (self *WinLoss) LURL() string {
	return self.serverURL + "/loss?" + self.PackURLString()
}

// ImpURL returns the impression URL for the win/loss notification.
func (self *WinLoss) ImpURL() string {
	return self.serverURL + "/imp?" + self.PackURLString(true)
}

// ClkURL returns the click URL for the win/loss notification.
func (self *WinLoss) ClkURL() string {
	return self.serverURL + "/clk?" + self.PackURLString(true)
}

// PackURLString returns the URL query string of the win/loss notification.
func (self *WinLoss) PackURLString(tracking ...bool) string {
	status := self.Status
	args := url.Values{}
	// seatid and adid are not used in the URL
	if status == StatusTrackClk || status == StatusTrackImp || (len(tracking) > 0 && tracking[0]) {
		args.Set("auction_id", self.AuctionID)
		args.Set("auction_bid_id", self.AuctionBidID)
		args.Set("auction_imp_id", self.AuctionImpID)
		args.Set("auction_price", fmt.Sprintf("%f", self.Cost))
		args.Set("auction_currency", "USD")
		if self.RAdv.Cap.CapNumber > 0 || self.RAdv.Cap.ClickNumber > 0 {
			cap, _ := self.RAdv.Cap.PackString()
			args.Set("cap", cap)
		}
	} else {
		args.Set("auction_id", `${AUCTION_ID}`)
		args.Set("auction_bid_id", `${AUCTION_BID_ID}`)
		args.Set("auction_imp_id", `${AUCTION_IMP_ID}`)
		args.Set("auction_price", `${AUCTION_PRICE}`)
		args.Set("auction_currency", `${AUCTION_CURRENCY}`)
	}
	demand, _ := self.RAdv.Demand.PackString()
	args.Set("demand", demand)
	supply, _ := self.RPub.PackString()
	args.Set("supply", supply)

	return args.Encode()
}

// UnpackURLString returns the WinLoss instance from the URL query string.
func UnpackURLString(urlString string, bidResponse ...*openrtb2.BidResponse) (*url.URL, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	if len(bidResponse) == 0 {
		u.Scheme = "http"
		return u, nil
	}

	auctionID := bidResponse[0].ID
	auctionBidID := bidResponse[0].BidID
	auctionAdID := bidResponse[0].SeatBid[0].Bid[0].AdID
	auctionImpID := bidResponse[0].SeatBid[0].Bid[0].ImpID
	auctionPrice := bidResponse[0].SeatBid[0].Bid[0].Price
	auctionCurrency := bidResponse[0].Cur
	auctionSeatID := bidResponse[0].SeatBid[0].Seat
	args := u.Query()
	for k, v := range args {
		switch v[0] {
		case `${AUCTION_ID}`:
			args.Set(k, auctionID)
		case `${AUCTION_BID_ID}`:
			args.Set(k, auctionBidID)
		case `${AUCTION_IMP_ID}`:
			args.Set(k, auctionImpID)
		case `${AUCTION_SEAT_ID}`:
			args.Set(k, auctionSeatID)
		case `${AUCTION_AD_ID}`:
			args.Set(k, auctionAdID)
		case `${AUCTION_PRICE}`:
			args.Set(k, fmt.Sprintf("%.3f", auctionPrice))
		case `${AUCTION_CURRENCY}`:
			args.Set(k, auctionCurrency)
		default:
		}
	}
	u.RawQuery = args.Encode()
	u.Scheme = "http"
	return u, nil
}

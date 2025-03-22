// Package dsp implements a demand-side platform.
package dsp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/genelet/winter/match"
	"github.com/mediocregopher/radix/v4"
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
	Current      time.Time `json:"current,omitempty"`
	Status       `json:"status,omitempty"`
	match.RPub   `json:"rpub,omitempty"`
	match.RAdv   `json:"radv,omitempty"`
	BothCap      *match.BothCap `json:"-,omitempty"`
	AuctionID    string         `json:"auction_id,omitempty"`
	AuctionBidID string         `json:"auction_bid_id,omitempty"`
	AuctionImpID string         `json:"auction_imp_id,omitempty"`
}

// NewWinLoss creates a new WinLoss instance from the current time, status, rpub, radv, and bothcap.
func NewWinLoss(current time.Time, how Status, rpub match.RPub, radv match.RAdv, bothcap *match.BothCap, auctionID, auctionBidID, auctionImpID string) *WinLoss {
	return &WinLoss{
		Current:      current,
		Status:       how,
		RPub:         rpub,
		RAdv:         radv,
		BothCap:      bothcap,
		AuctionID:    auctionID,
		AuctionBidID: auctionBidID,
		AuctionImpID: auctionImpID,
	}
}

func (self *Controller) serveStatus(ctx context.Context, status Status, current time.Time, args url.Values) error {
	var err error
	wl := &WinLoss{
		Current:      current,
		Status:       status,
		AuctionID:    args.Get("auction_id"),
		AuctionBidID: args.Get("auction_bid_id"),
		AuctionImpID: args.Get("auction_imp_id"),
	}

	demand := args.Get("demand")
	supply := args.Get("supply")
	if demand != "" {
		wl.RAdv.Demand, err = match.UnpackDemandString(supply)
		if err != nil {
			return err
		}
	}
	if supply != "" {
		wl.RPub, err = match.UnpackRPubString(demand)
		if err != nil {
			return err
		}
	}

	price, err := strconv.ParseFloat(args.Get("auction_price"), 64)
	if err != nil {
		return err
	}
	wl.RAdv.Cost = float32(price)
	if v := args.Get("auction_currency"); v == "USD" {
		wl.RAdv.CostType = 1
	}

	switch status {
	case StatusTrackClk, StatusTrackImp:
		u := args.Get("cap")
		v := args.Get("bothcap")
		if u == "" {
			break
		}

		cap, err := match.UnpackCapString(u)
		if err != nil {
			return err
		}
		wl.RAdv.Cap = cap

		bid, err := match.UnpackBidID(wl.AuctionBidID)
		if err != nil {
			return err
		}

		var both match.BothCap
		if v != "" {
			both, err = match.UnpackBothCapString(v)
			if err != nil {
				return err
			}
		}
		both.Refresh(current, wl.RAdv, status == StatusTrackImp, status == StatusTrackClk)
		bs, err := both.Pack()
		if err != nil {
			return err
		}

		err = self.Redis.Do(ctx, radix.Cmd(nil, "HSET", match.HashNameBothCap(bid.UserID), fmt.Sprintf("%d", wl.RAdv.ItemID), string(bs)))
		if err != nil {
			return err
		}
	default:
	}

	bs, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	return self.Nc.Publish(SUBJECTWinLoss, bs)
}

// PackURLString returns the URL query string of the win/loss notification.
func (self *WinLoss) PackURLString(tracking ...bool) string {
	status := self.Status
	args := url.Values{}
	if status == StatusTrackClk || status == StatusTrackImp || (len(tracking) > 0 && tracking[0]) {
		args.Set("auction_id", self.AuctionID)
		args.Set("auction_bid_id", self.AuctionBidID)
		args.Set("auction_imp_id", self.AuctionImpID)
		args.Set("auction_price", fmt.Sprintf("%f", self.Cost))
		args.Set("auction_currency", "USD")
		cap, _ := self.RAdv.Cap.PackString()
		args.Set("cap", cap)
		if self.BothCap != nil {
			bothcap, _ := self.BothCap.PackString()
			args.Set("bothcap", bothcap)
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
	auctionImpID := bidResponse[0].SeatBid[0].Bid[0].ImpID
	auctionPrice := bidResponse[0].SeatBid[0].Bid[0].Price
	auctionCurrency := bidResponse[0].Cur
	args := u.Query()
	for k, v := range args {
		switch v[0] {
		case `${AUCTION_ID}`:
			args.Set(k, auctionID)
		case `${AUCTION_BID_ID}`:
			args.Set(k, auctionBidID)
		case `${AUCTION_IMP_ID}`:
			args.Set(k, auctionImpID)
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

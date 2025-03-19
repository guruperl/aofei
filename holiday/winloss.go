package holiday

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/genelet/winter/match"
	"github.com/mediocregopher/radix/v4"
)

type Status struct {
	Win      bool
	Tracking bool
	Click    bool
}

// NewStatus reports the status of the win, tracking, and click.
func NewStatus(win, tracking, click bool) Status {
	return Status{win, tracking, click}
}

// Pack serializes the status into an int8 integer.
func (self Status) Pack() int8 {
	var status int8
	if self.Win {
		status |= 1
	}
	if self.Tracking {
		status |= 2
	}
	if self.Click {
		status |= 4
	}
	return status
}

// UnpackStatus deserializes the status from an int8 integer.
func UnpackStatus(status int8) Status {
	win := status&1 != 0
	tracking := status&2 != 0
	click := status&4 != 0
	return NewStatus(win, tracking, click)
}

// WinLoss is a win/loss notification received after bidding.
// ${AUCTION_ID} ID of the bid request; from BidRequest.id attribute.
// ${AUCTION_BID_ID} ID of the bid; from BidResponse.bidid attribute.
// ${AUCTION_IMP_ID} ID of the impression just won; from imp.id attribute.
// ${AUCTION_SEAT_ID} ID of the bidder seat;for whom the bid was made.
// ${AUCTION_AD_ID} ID of the ad markup the bidder wishes to serve; from bid.adid attribute.
type WinLoss struct {
	Current      time.Time `json:"current,omitempty"`
	Status       int8      `json:"status,omitempty"`
	match.RPub   `json:"rpub,omitempty"`
	match.RAdv   `json:"radv,omitempty"`
	BothCap      match.BothCap `json:"bothcap,omitempty"`
	AuctionID    string        `json:"auction_id,omitempty"`
	AuctionBidID string        `json:"auction_bid_id,omitempty"`
	AuctionImpID string        `json:"auction_imp_id,omitempty"`
}

func (self *Controller) serveStatus(ctx context.Context, win, tracking, click bool, current time.Time, args url.Values) error {
	var err error
	status := NewStatus(win, tracking, click)

	wl := &WinLoss{
		Current:      current,
		Status:       status.Pack(),
		AuctionID:    args.Get("auction_id"),
		AuctionBidID: args.Get("auction_bid_id"),
		AuctionImpID: args.Get("auction_imp_id"),
	}

	demand := args.Get("demand")
	supply := args.Get("supply")
	if demand == "" {
		wl.RAdv.Demand, err = match.UnpackDemandString(supply)
		if err != nil {
			return err
		}
	}
	if supply == "" {
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

	var found bool
	if u := args.Get("cap"); u != "" {
		cap, err := match.UnpackCapString(u)
		if err != nil {
			return err
		}
		found = true
		wl.RAdv.Cap = cap
	}

	if v := args.Get("bothcap"); v != "" && found {
		both, err := match.UnpackBothCapString(v)
		if err != nil {
			return err
		}

		bid, err := match.UnpackBidID(wl.AuctionBidID)
		if err != nil {
			return err
		}

		both.Refresh(current, wl.RAdv, !status.Win && status.Tracking && !status.Click, !status.Win && status.Tracking && status.Click)
		bs, err := both.Pack()
		if err != nil {
			return err
		}
		err = self.Redis.Do(ctx, radix.Cmd(nil, "HSET", match.HashNameBothCap(bid.UserID), fmt.Sprintf("%d", wl.RAdv.ItemID), string(bs)))
		if err != nil {
			return err
		}
	}

	bs, err := json.Marshal(wl)
	if err != nil {
		return err
	}
	return self.Nc.Publish(SUBJECTAttributeWinLoss, bs)
}

// GetURLString returns the URL query string of the win/loss notification.
func (self *WinLoss) GetURLString() string {
	args := url.Values{}
	args.Set("auction_id", self.AuctionID)
	args.Set("auction_bid_id", self.AuctionBidID)
	args.Set("auction_imp_id", self.AuctionImpID)
	demand, _ := self.Demand.PackString()
	args.Set("demand", demand)
	supply, _ := self.RPub.PackString()
	args.Set("supply", supply)
	args.Set("auction_price", fmt.Sprintf("%f", self.Cost))
	args.Set("auction_currency", "USD")
	cap, _ := self.RAdv.Cap.PackString()
	args.Set("cap", cap)
	bothcap, _ := self.BothCap.PackString()
	args.Set("bothcap", bothcap)
	return args.Encode()
}

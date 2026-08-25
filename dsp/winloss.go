// Package dsp implements a demand-side platform.
package dsp

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/match"
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
	Status              `json:"status,omitempty"`
	Current             time.Time `json:"current,omitempty"`
	match.RPub          `json:"rpub,omitempty"`
	match.RAdv          `json:"radv,omitempty"`
	BothCap             *match.BothCap        `json:"-"`
	Seat                string                `json:"seat,omitempty"`
	AuctionID           string                `json:"auction_id,omitempty"`
	AuctionBidID        string                `json:"auction_bid_id,omitempty"`
	AuctionImpID        string                `json:"auction_imp_id,omitempty"`
	AuctionAdID         string                `json:"auction_ad_id,omitempty"`
	Middleman           *MiddlemanWinLossMeta `json:"middleman,omitempty"`
	Reporting           *ReportingDimensions  `json:"reporting,omitempty"`
	DeliveryReservation string                `json:"-"`
	serverURL           string
	trackingSecret      string
	actionTokenTTL      time.Duration
	actionToken         string
}

// MiddlemanWinLossMeta carries charge/pay audit details for proxied middleman
// callbacks without changing the existing ledger aggregation contract.
type MiddlemanWinLossMeta struct {
	AccountingVersion  string         `json:"accounting_version,omitempty"`
	BidderID           uint32         `json:"bidder_id,omitempty"`
	GroupID            uint32         `json:"group_id,omitempty"`
	RouteBidderID      uint32         `json:"route_bidder_id,omitempty"`
	TargetID           uint32         `json:"target_id,omitempty"`
	TriggerMode        string         `json:"trigger_mode,omitempty"`
	Source             string         `json:"source,omitempty"`
	ForwardStatus      string         `json:"forward_status,omitempty"`
	ForwardHTTPStatus  int            `json:"forward_http_status,omitempty"`
	DownstreamBidPrice float64        `json:"downstream_bid_price,omitempty"`
	UpstreamBidPrice   float64        `json:"upstream_bid_price,omitempty"`
	ChargePrice        float64        `json:"charge_price,omitempty"`
	PayPrice           float64        `json:"pay_price,omitempty"`
	ChargeCPM          accounting.CPM `json:"charge_cpm_micros,omitempty"`
	PayCPM             accounting.CPM `json:"pay_cpm_micros,omitempty"`
	MarginCPM          float64        `json:"margin_cpm,omitempty"`
	MarginCPMExact     accounting.CPM `json:"margin_cpm_micros,omitempty"`
	Currency           string         `json:"currency,omitempty"`
}

// ReportingDimensions contains only bounded, coarse classifications approved
// for R02 aggregation. It never carries an IP, user agent, IFA, cookie,
// consent string, or raw geographic label.
type ReportingDimensions struct {
	CountryID         uint32 `json:"country_id,omitempty"`
	StateID           uint32 `json:"state_id,omitempty"`
	DeviceOS          uint8  `json:"device_os,omitempty"`
	DeviceType        uint8  `json:"device_type,omitempty"`
	Environment       string `json:"environment,omitempty"`
	IntegrationMode   string `json:"integration_mode,omitempty"`
	MediaIntent       string `json:"media_intent,omitempty"`
	Placement         string `json:"placement,omitempty"`
	RenderContext     string `json:"render_context,omitempty"`
	RefreshMode       string `json:"refresh_mode,omitempty"`
	RefreshSeconds    uint16 `json:"refresh_seconds,omitempty"`
	AdDensity         string `json:"ad_density,omitempty"`
	TrafficQuality    string `json:"traffic_quality,omitempty"`
	SourceQuality     string `json:"source_quality,omitempty"`
	ManagementControl string `json:"management_control,omitempty"`
	SellerType        string `json:"seller_type,omitempty"`
	SellerID          string `json:"seller_id,omitempty"`
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

func (self *WinLoss) WithTrackingSecret(secret string) *WinLoss {
	self.trackingSecret = secret
	return self
}

func (self *WinLoss) withDeliveryReservation(token string) *WinLoss {
	self.DeliveryReservation = token
	return self
}

func (self *WinLoss) withActionTokenTTL(ttl time.Duration) *WinLoss {
	self.actionTokenTTL = ttl
	return self
}

func (self *WinLoss) withReportingDimensions(dimensions *ReportingDimensions) *WinLoss {
	self.Reporting = dimensions
	return self
}

// Macro returns the replacement macro hash
func (self *WinLoss) Macro() map[string]string {
	price := fmt.Sprintf("%.3f", self.RAdv.Cost)
	if cpm, ok := self.RAdv.ExactCPM(); ok {
		price = cpm.String()
	}
	return map[string]string{
		`${AUCTION_ID}`:       self.AuctionID,
		`${AUCTION_BID_ID}`:   self.AuctionBidID,
		`${AUCTION_IMP_ID}`:   self.AuctionImpID,
		`${AUCTION_AD_ID}`:    self.AuctionAdID,
		`${AUCTION_SEAT_ID}`:  self.Seat,
		`${AUCTION_PRICE}`:    price,
		`${AUCTION_CURRENCY}`: "USD",
	}
}

// NURL returns the notification URL for the win/loss notification.
func (self *WinLoss) NURL() string {
	return self.trackerURL("/win", false, "")
}

// LURL returns the loss URL for the win/loss notification.
func (self *WinLoss) LURL() string {
	return self.trackerURL("/loss", false, "")
}

// ImpURL returns the impression URL for the win/loss notification.
func (self *WinLoss) ImpURL() string {
	return self.trackerURL("/imp", true, "")
}

// ClkURL returns the click URL for the win/loss notification.
func (self *WinLoss) ClkURL() string {
	return self.trackerURL("/clk", true, "")
}

// ClkRedirectURL returns a click URL that records the click and then redirects
// to the already-rendered advertiser landing URL.
func (self *WinLoss) ClkRedirectURL(landing string) string {
	url, _ := self.clkRedirectURLWithError(landing)
	return url
}

func (self *WinLoss) clkRedirectURLWithError(landing string) (string, error) {
	if landing == "" {
		return self.trackerURLWithError("/clk", true, "")
	}
	if self.actionToken == "" {
		self.actionToken, _ = newActionToken(self.trackingSecret, self, self.actionTokenTTL, self.Current)
	}
	landing = appendActionToken(landing, self.actionToken)
	return self.trackerURLWithError("/clk", true, landing)
}

func (self *WinLoss) trackerURL(path string, tracking bool, redirect string) string {
	url, _ := self.trackerURLWithError(path, tracking, redirect)
	return url
}

// trackerURLWithError builds a signed tracker URL. Invalid packed tracking
// state (an invalid frequency-cap configuration) yields no URL and an error so
// materialization can abort instead of emitting an unsigned or cap-less tracker.
func (self *WinLoss) trackerURLWithError(path string, tracking bool, redirect string) (string, error) {
	args, err := self.packURLValues(tracking)
	if err != nil {
		return "", err
	}
	if redirect != "" {
		args.Set("redirect", redirect)
	}
	addTrackingSignature(self.trackingSecret, path, args)
	return self.serverURL + path + "?" + encodeOpenRTBMacroQuery(args), nil
}

// PackURLString returns the URL query string of the win/loss notification.
func (self *WinLoss) PackURLString(tracking ...bool) string {
	useTracking := len(tracking) > 0 && tracking[0]
	args, err := self.packURLValues(useTracking)
	if err != nil {
		return ""
	}
	return encodeOpenRTBMacroQuery(args)
}

func (self *WinLoss) packURLValues(tracking bool) (url.Values, error) {
	status := self.Status
	args := url.Values{}
	// seatid and adid are not used in the URL
	if status == StatusTrackClk || status == StatusTrackImp || tracking {
		args.Set("auction_id", self.AuctionID)
		args.Set("auction_bid_id", self.AuctionBidID)
		args.Set("auction_imp_id", self.AuctionImpID)
		cpm, ok := self.RAdv.ExactCPM()
		if !ok {
			return nil, fmt.Errorf("tracking price has no exact USD CPM")
		}
		args.Set("auction_price", cpm.String())
		args.Set("auction_currency", "USD")
		if self.RAdv.Cap.CapNumber > 0 || self.RAdv.Cap.CapThrottle > 0 || self.RAdv.Cap.ClickNumber > 0 {
			cap, err := self.RAdv.Cap.PackString()
			if err != nil {
				return nil, fmt.Errorf("pack frequency cap for tracking: %w", err)
			}
			args.Set("cap", cap)
		}
	} else {
		args.Set("auction_id", `${AUCTION_ID}`)
		args.Set("auction_bid_id", `${AUCTION_BID_ID}`)
		args.Set("auction_imp_id", `${AUCTION_IMP_ID}`)
		args.Set("auction_price", `${AUCTION_PRICE}`)
		args.Set("auction_currency", `${AUCTION_CURRENCY}`)
	}
	demand, err := self.RAdv.Demand.PackString()
	if err != nil {
		return nil, fmt.Errorf("pack demand for tracking: %w", err)
	}
	args.Set("demand", demand)
	supply, err := self.RPub.PackString()
	if err != nil {
		return nil, fmt.Errorf("pack supply for tracking: %w", err)
	}
	args.Set("supply", supply)
	if self.DeliveryReservation != "" {
		args.Set("delivery_reservation", self.DeliveryReservation)
	}
	if self.Reporting != nil {
		args.Set("report_country_id", strconv.FormatUint(uint64(self.Reporting.CountryID), 10))
		args.Set("report_state_id", strconv.FormatUint(uint64(self.Reporting.StateID), 10))
		args.Set("report_device_os", strconv.FormatUint(uint64(self.Reporting.DeviceOS), 10))
		args.Set("report_device_type", strconv.FormatUint(uint64(self.Reporting.DeviceType), 10))
		args.Set("report_environment", self.Reporting.Environment)
		args.Set("report_integration", self.Reporting.IntegrationMode)
		args.Set("report_media_intent", self.Reporting.MediaIntent)
		args.Set("report_placement", self.Reporting.Placement)
		args.Set("report_render_context", self.Reporting.RenderContext)
		args.Set("report_refresh_mode", self.Reporting.RefreshMode)
		args.Set("report_refresh_seconds", strconv.FormatUint(uint64(self.Reporting.RefreshSeconds), 10))
		args.Set("report_ad_density", self.Reporting.AdDensity)
		args.Set("report_traffic_quality", self.Reporting.TrafficQuality)
		args.Set("report_source_quality", self.Reporting.SourceQuality)
		args.Set("report_management", self.Reporting.ManagementControl)
		args.Set("report_seller_type", self.Reporting.SellerType)
		args.Set("report_seller_id", self.Reporting.SellerID)
	}

	return args, nil
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
	if bidResponse[0] == nil {
		return nil, errors.New("bid response is nil")
	}
	if len(bidResponse[0].SeatBid) == 0 {
		return nil, errors.New("bid response has no seat bids")
	}
	if len(bidResponse[0].SeatBid[0].Bid) == 0 {
		return nil, errors.New("bid response has no bids")
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

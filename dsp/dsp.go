package dsp

import (
	"fmt"
	"strconv"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type DSP struct {
	bid                 *openrtb2.BidRequest
	impIndex            int
	attribute           *match.Attribute
	one                 match.RAdv
	bothcap             *match.BothCap
	creative            *match.Creative
	audience            *match.Audience
	bidCPM              accounting.CPM
	serverURL           string
	trackingSecret      string
	actionTokenTTL      time.Duration
	deliveryReservation string
}

// NewDSP creates a new DSP instance.
func NewDSP(
	bid *openrtb2.BidRequest,
	attribute *match.Attribute,
	one match.RAdv,
	bothcap *match.BothCap,
	creative *match.Creative,
	audience *match.Audience,
	serverURL string,
) *DSP {
	price, _ := one.ExactCPM()
	return newDSPForImpExact(bid, 0, attribute, one, bothcap, creative, audience, price, serverURL)
}

// NewDSPForImp creates a DSP instance for one impression in a request.
func NewDSPForImp(
	bid *openrtb2.BidRequest,
	impIndex int,
	attribute *match.Attribute,
	one match.RAdv,
	bothcap *match.BothCap,
	creative *match.Creative,
	audience *match.Audience,
	bidPrice float32,
	serverURL string,
) *DSP {
	cpm, _ := accounting.ParseCPM(strconv.FormatFloat(float64(bidPrice), 'f', 6, 32))
	return newDSPForImpExact(bid, impIndex, attribute, one, bothcap, creative, audience, cpm, serverURL)
}

func newDSPForImpExact(
	bid *openrtb2.BidRequest,
	impIndex int,
	attribute *match.Attribute,
	one match.RAdv,
	bothcap *match.BothCap,
	creative *match.Creative,
	audience *match.Audience,
	bidCPM accounting.CPM,
	serverURL string,
) *DSP {
	return &DSP{
		bid:       bid,
		impIndex:  impIndex,
		attribute: attribute,
		one:       one,
		creative:  creative,
		audience:  audience,
		bidCPM:    bidCPM,
		serverURL: serverURL,
		bothcap:   bothcap,
	}
}

func (self *DSP) WithTrackingSecret(secret string) *DSP {
	self.trackingSecret = secret
	return self
}

func (self *DSP) withDeliveryReservation(token string) *DSP {
	self.deliveryReservation = token
	return self
}

func (self *DSP) withActionTokenTTL(ttl time.Duration) *DSP {
	self.actionTokenTTL = ttl
	return self
}

// impID returns the impID of the bid.
func (self *DSP) impID() string {
	if self == nil || self.bid == nil || self.impIndex < 0 || self.impIndex >= len(self.bid.Imp) {
		return ""
	}
	return self.bid.Imp[self.impIndex].ID
}

// adID returns the AdID of the bid.
func (self *DSP) adID() string {
	return fmt.Sprintf("%d", self.one.CreativeID)
}

// seat returns the seat ID of the bid.
func (self *DSP) seat() string {
	return fmt.Sprintf("%d", self.one.CampaignID)
}

// bidID returns the bidID of the response bid.
func (self *DSP) bidID() string {
	return bidID{
		When:   self.attribute.When.UnixNano(),
		UserID: self.attribute.UserID,
	}.String()
}

// rspndBidID returns the response bid ID of the response bid.
func (self *DSP) rspndBidID() string {
	return responseBidID{
		When:       self.attribute.When.UnixNano(),
		CreativeID: self.one.CreativeID,
		ImpIndex:   uint32(self.impIndex),
	}.String()
}

// bidID is the BidID in the openrtb2.BidResponse.
type bidID struct {
	When   int64
	UserID string
}

// String packs the bidID to string.
func (self bidID) String() string {
	return fmt.Sprintf("%16x%s", self.When, self.UserID)
}

// UnpackBidID unpacks the string from Bid.
func UnpackBidID(data string) (bidID, error) {
	if len(data) < 16 {
		return bidID{}, fmt.Errorf("invalid bid ID length")
	}
	when, err := strconv.ParseInt(data[:16], 16, 64)
	if err != nil {
		return bidID{}, err
	}
	return bidID{
		When:   when,
		UserID: data[16:],
	}, nil
}

// responseBidID is the ID of openrtb2.Bid
type responseBidID struct {
	When       int64
	CreativeID uint32
	ImpIndex   uint32
}

// Pack returns the packed string of responseBidID.
func (self responseBidID) String() string {
	return fmt.Sprintf("%16x%d:%d", self.When, self.CreativeID, self.ImpIndex)
}

// UnpackResponseBidID unpacks the responseBidID from the packed string.
func UnpackResponseBidID(data string) (responseBidID, error) {
	var seatBid responseBidID
	_, err := fmt.Sscanf(data, "%16x%d:%d", &seatBid.When, &seatBid.CreativeID, &seatBid.ImpIndex)
	if err != nil {
		_, err = fmt.Sscanf(data, "%16x%d", &seatBid.When, &seatBid.CreativeID)
	}
	return seatBid, err
}

// WinLoss returns the WinLoss instance.
func (self *DSP) WinLoss(StatusBid Status) *WinLoss {
	return NewWinLoss(
		StatusBid,
		self.attribute.When,
		self.attribute.RPub,
		self.billableRAdv(),
		self.bothcap,
		self.seat(),
		self.bid.ID,
		self.bidID(),
		self.impID(),
		self.adID(),
		self.serverURL,
	).WithTrackingSecret(self.trackingSecret).
		withDeliveryReservation(self.deliveryReservation).
		withActionTokenTTL(self.actionTokenTTL).
		withReportingDimensions(reportingDimensionsFromAttribute(self.attribute))
}

func reportingDimensionsFromAttribute(attribute *match.Attribute) *ReportingDimensions {
	dimensions := &ReportingDimensions{
		Environment:       "Unknown",
		IntegrationMode:   "Unknown",
		MediaIntent:       "Unknown",
		Placement:         "Unknown",
		RenderContext:     "Unknown",
		RefreshMode:       "Unknown",
		AdDensity:         "Unknown",
		TrafficQuality:    "Unknown",
		SourceQuality:     "Unknown",
		ManagementControl: "Unknown",
		SellerType:        "Unknown",
	}
	if attribute == nil {
		return dimensions
	}
	if attribute.Geo != nil {
		dimensions.CountryID = attribute.Geo.CountryID
		dimensions.StateID = attribute.Geo.StateID
	}
	if attribute.PzUa != nil {
		dimensions.DeviceOS = uint8(attribute.PzUa.OS)
		dimensions.DeviceType = uint8(attribute.PzUa.Device)
	}
	dimensions.Environment = attribute.Supply.Site.Normalize().Environment
	dimensions.IntegrationMode = attribute.Supply.Site.Normalize().IntegrationMode
	slot := attribute.Supply.Slot.Normalize()
	dimensions.MediaIntent = slot.MediaIntent
	dimensions.Placement = slot.Placement
	dimensions.RenderContext = slot.RenderContext
	dimensions.RefreshMode = slot.RefreshMode
	dimensions.RefreshSeconds = slot.RefreshSeconds
	dimensions.AdDensity = slot.AdDensity
	dimensions.TrafficQuality = slot.TrafficQuality
	dimensions.SourceQuality = slot.SourceQuality
	dimensions.ManagementControl = slot.ManagementControl
	if attribute.Supply.Seller.Authorized {
		dimensions.SellerType = attribute.Supply.Seller.Type
		dimensions.SellerID = attribute.Supply.Seller.ID
	}
	return dimensions
}

func (self *DSP) billableRAdv() match.RAdv {
	one := self.one
	one.Cost = self.bidCPM.Float32()
	if self.one.CostCPM != 0 {
		one.CostCPM = self.bidCPM
	} else {
		// Preserve v1/v2 cache provenance. The bounded compatibility adapter
		// remains authoritative for that drain fact; it must not be relabeled v3.
		one.CostCPM = 0
	}
	one.CostType = match.CostTypeCPM
	return one
}

// NewBid returns the SeatBid for the bid response.
func (self *DSP) NewBid(winloss *WinLoss) (openrtb2.Bid, error) {
	if self.bidCPM <= 0 || self.bidCPM > accounting.MaxCPM {
		return openrtb2.Bid{}, fmt.Errorf("selected bid has no exact USD CPM")
	}
	macroStandard := winloss.Macro()
	macroCustom := self.Macro()
	landing, err := self.creative.LandingURL(macroStandard, macroCustom)
	if err != nil {
		return openrtb2.Bid{}, err
	}
	impURL, err := winloss.trackerURLWithError("/imp", true, "")
	if err != nil {
		return openrtb2.Bid{}, err
	}
	clkRedirect, err := winloss.clkRedirectURLWithError(landing)
	if err != nil {
		return openrtb2.Bid{}, err
	}
	adm, err := self.creative.AdM(self.attribute, impURL, clkRedirect, macroStandard, macroCustom)
	if err != nil {
		return openrtb2.Bid{}, err
	}
	nurl, err := winloss.trackerURLWithError("/win", false, "")
	if err != nil {
		return openrtb2.Bid{}, err
	}
	lurl, err := winloss.trackerURLWithError("/loss", false, "")
	if err != nil {
		return openrtb2.Bid{}, err
	}
	w, h := match.SizeID1To2(self.creative.SizeID)
	one := self.one
	var bundle string
	var categories []string
	if self.audience != nil {
		bundle = self.audience.CampaignForeignID
		categories = self.audience.Categories
	}
	rspnsBid := openrtb2.Bid{
		ID:    self.rspndBidID(),
		ImpID: self.impID(),
		Price: self.bidCPM.Float64(),
		NURL:  nurl,
		LURL:  lurl,
		AdM:   adm,
		AdID:  self.adID(),
		//ADomain: []string{self.audience.AdvDomain},
		Bundle: bundle,
		CID:    fmt.Sprintf("%d", one.CampaignID),
		CrID:   fmt.Sprintf("%d", one.CreativeID),
		Cat:    categories,
		W:      int64(w),
		H:      int64(h),
	}

	return rspnsBid, nil
}

// Macro returns the replacement macro hash
func (self *DSP) Macro() map[string]string {
	bid := self.bid
	one := self.one
	attribute := self.attribute
	var device *openrtb2.Device
	if bid != nil {
		device = bid.Device
	}
	if device == nil {
		device = &openrtb2.Device{}
	}
	geo := device.Geo
	if geo == nil {
		geo = &openrtb2.Geo{}
	}
	var app *openrtb2.App
	if bid != nil {
		app = bid.App
	}
	if app == nil {
		app = &openrtb2.App{}
	}
	var imp openrtb2.Imp
	if bid != nil && self.impIndex >= 0 && self.impIndex < len(bid.Imp) {
		imp = bid.Imp[self.impIndex]
	}
	var rpub match.RPub
	var nativeFormat *match.NativeFormat
	var pubName string
	if attribute != nil {
		rpub = attribute.RPub
		nativeFormat = attribute.NativeFormat
		if attribute.ACL != nil {
			pubName = attribute.ACL.PubStr
		}
	}
	w, h := match.SizeID1To2(rpub.SizeID)
	var campaignForeignID string
	if self.audience != nil {
		campaignForeignID = self.audience.CampaignForeignID
	}
	var creativeName string
	if self.creative != nil {
		creativeName = self.creative.CreativeName
	}

	return map[string]string{
		`{DSP_IP}`:              "",
		`{DSP_COUNTRY}`:         geo.Country,
		`{DSP_REGION}`:          geo.Region,
		`{DSP_CITY}`:            "",
		`{DSP_CARRIER}`:         "",
		`{DSP_CONNECTION_TYPE}`: fmt.Sprintf("%v", device.ConnectionType),
		`{DSP_USER_AGENT}`:      "",
		`{DSP_OS}`:              device.OS,
		`{DSP_OS_VERSION}`:      "",
		`{DSP_DEVICE_TYPE}`:     fmt.Sprintf("%v", device.DeviceType),
		`{DSP_DEVICE_BRAND}`:    "",
		`{DSP_DEVICE_MODEL}`:    "",
		`{DSP_DEVICE_LANGUAGE}`: device.Language,
		`{DSP_GAID}`:            "",
		`{DSP_IDFA}`:            "",
		`{DSP_DEVICE_ID}`:       "",
		`{DSP_DEVICE_ID_MD5}`:   "",
		`{DSP_DEVICE_ID_SHA1}`:  "",
		`{DSP_BUNDLE}`:          app.Bundle,
		`{DSP_TAGID}`:           imp.TagID,
		`{DSP_AD_FORMAT}`:       fmt.Sprintf("%v", nativeFormat),
		`{DSP_AD_SIZE}`:         fmt.Sprintf("%dx%d", w, h),
		`{DSP_CAMPAIGN_ID}`:     fmt.Sprintf("%d", one.CampaignID),
		`{DSP_ADV_CAMPAIGN}`:    campaignForeignID,
		`{DSP_AD_GROUP_ID}`:     fmt.Sprintf("%d", one.ItemID),
		`{DSP_CREATIVE_ID}`:     fmt.Sprintf("%d", one.CreativeID),
		`{DSP_CREATIVE_NAME}`:   creativeName,
		`{DSP_PUBLISHER_ID}`:    fmt.Sprintf("%v", rpub.PubID),
		`{DSP_PUBLISHER_NAME}`:  pubName,
	}
}

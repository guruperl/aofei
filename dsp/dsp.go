package dsp

import (
	"fmt"

	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type DSP struct {
	bid            *openrtb2.BidRequest
	impIndex       int
	attribute      *match.Attribute
	one            match.RAdv
	bothcap        *match.BothCap
	creative       *match.Creative
	audience       *match.Audience
	bidPrice       float32
	serverURL      string
	trackingSecret string
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
	price, _ := one.ECPM()
	return NewDSPForImp(bid, 0, attribute, one, bothcap, creative, audience, price, serverURL)
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
	return &DSP{
		bid:       bid,
		impIndex:  impIndex,
		attribute: attribute,
		one:       one,
		creative:  creative,
		audience:  audience,
		bidPrice:  bidPrice,
		serverURL: serverURL,
		bothcap:   bothcap,
	}
}

func (self *DSP) WithTrackingSecret(secret string) *DSP {
	self.trackingSecret = secret
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
	var when int64
	var userID string
	_, err := fmt.Sscanf(data, "%16x%s", &when, &userID)
	return bidID{
		When:   when,
		UserID: userID,
	}, err
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
	).WithTrackingSecret(self.trackingSecret)
}

func (self *DSP) billableRAdv() match.RAdv {
	one := self.one
	one.Cost = self.bidPrice
	one.CostType = 2
	return one
}

// NewBid returns the SeatBid for the bid response.
func (self *DSP) NewBid(winloss *WinLoss) (openrtb2.Bid, error) {
	macroStandard := winloss.Macro()
	macroCustom := self.Macro()
	landing, err := self.creative.LandingURL(macroStandard, macroCustom)
	if err != nil {
		return openrtb2.Bid{}, err
	}

	adm, err := self.creative.AdM(self.attribute, winloss.ImpURL(), winloss.ClkRedirectURL(landing), macroStandard, macroCustom)
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
		Price: float64(self.bidPrice),
		NURL:  winloss.NURL(),
		LURL:  winloss.LURL(),
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

	gaid := device.DPIDMD5
	if gaid == "" {
		gaid = device.DPIDSHA1
	}
	did := device.DIDMD5
	if did == "" {
		did = device.DIDSHA1
	}
	return map[string]string{
		`{DSP_IP}`:              device.IP,
		`{DSP_COUNTRY}`:         geo.Country,
		`{DSP_REGION}`:          geo.Region,
		`{DSP_CITY}`:            geo.City,
		`{DSP_CARRIER}`:         device.Carrier,
		`{DSP_CONNECTION_TYPE}`: fmt.Sprintf("%v", device.ConnectionType),
		`{DSP_USER_AGENT}`:      device.UA,
		`{DSP_OS}`:              device.OS,
		`{DSP_OS_VERSION}`:      device.OSV,
		`{DSP_DEVICE_TYPE}`:     fmt.Sprintf("%v", device.DeviceType),
		`{DSP_DEVICE_BRAND}`:    device.Make,
		`{DSP_DEVICE_MODEL}`:    device.Model,
		`{DSP_DEVICE_LANGUAGE}`: device.Language,
		`{DSP_GAID}`:            gaid,
		`{DSP_IDFA}`:            device.IFA,
		`{DSP_DEVICE_ID}`:       did,
		`{DSP_DEVICE_ID_MD5}`:   device.MACMD5,
		`{DSP_DEVICE_ID_SHA1}`:  device.MACSHA1,
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

package dsp

import (
	"fmt"

	"github.com/genelet/winter/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type DSP struct {
	bid       *openrtb2.BidRequest
	attribute *match.Attribute
	one       match.RAdv
	bothcap   *match.BothCap
	creative  *match.Creative
	audience  *match.Audience
	serverURL string
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
	return &DSP{
		bid:       bid,
		attribute: attribute,
		one:       one,
		creative:  creative,
		audience:  audience,
		serverURL: serverURL,
		bothcap:   bothcap,
	}
}

// impID returns the impID of the bid.
func (self *DSP) impID() string {
	return self.bid.Imp[0].ID
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
}

// Pack returns the packed string of responseBidID.
func (self responseBidID) String() string {
	return fmt.Sprintf("%16x%d", self.When, self.CreativeID)
}

// UnpackResponseBidID unpacks the responseBidID from the packed string.
func UnpackResponseBidID(data string) (responseBidID, error) {
	var seatBid responseBidID
	_, err := fmt.Sscanf(data, "%16x%d", &seatBid.When, &seatBid.CreativeID)
	return seatBid, err
}

// WinLoss returns the WinLoss instance.
func (self *DSP) WinLoss(StatusBid Status) *WinLoss {
	return NewWinLoss(
		StatusBid,
		self.attribute.When,
		self.attribute.RPub,
		self.one,
		self.bothcap,
		self.seat(),
		self.bid.ID,
		self.bidID(),
		self.impID(),
		self.adID(),
		self.serverURL,
	)
}

// NewBid returns the SeatBid for the bid response.
func (self *DSP) NewBid(winloss *WinLoss) (openrtb2.Bid, error) {
	macroStandard := winloss.Macro()
	macroCustom := self.Macro()

	adm, err := self.creative.AdM(self.attribute, winloss.ImpURL(), winloss.ClkURL(), macroStandard, macroCustom)
	if err != nil {
		return openrtb2.Bid{}, err
	}
	w, h := match.SizeID1To2(self.creative.SizeID)
	one := self.one
	rspnsBid := openrtb2.Bid{
		ID:    self.rspndBidID(),
		ImpID: self.impID(),
		Price: float64(one.Cost),
		NURL:  winloss.NURL(),
		LURL:  winloss.LURL(),
		AdM:   adm,
		AdID:  self.adID(),
		//ADomain: []string{self.audience.AdvDomain},
		Bundle: self.audience.CampaignForeignID,
		CID:    fmt.Sprintf("%d", one.CampaignID),
		CrID:   fmt.Sprintf("%d", one.CreativeID),
		Cat:    self.audience.Categories,
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
	device := bid.Device
	w, h := match.SizeID1To2(attribute.RPub.SizeID)

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
		`{DSP_COUNTRY}`:         device.Geo.Country,
		`{DSP_REGION}`:          device.Geo.Region,
		`{DSP_CITY}`:            device.Geo.City,
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
		`{DSP_BUNDLE}`:          bid.App.Bundle,
		`{DSP_TAGID}`:           bid.Imp[0].TagID,
		`{DSP_AD_FORMAT}`:       fmt.Sprintf("%v", attribute.NativeFormat),
		`{DSP_AD_SIZE}`:         fmt.Sprintf("%dx%d", w, h),
		`{DSP_CAMPAIGN_ID}`:     fmt.Sprintf("%d", one.CampaignID),
		`{DSP_ADV_CAMPAIGN}`:    self.audience.CampaignForeignID,
		`{DSP_AD_GROUP_ID}`:     fmt.Sprintf("%d", one.ItemID),
		`{DSP_CREATIVE_ID}`:     fmt.Sprintf("%d", one.CreativeID),
		`{DSP_CREATIVE_NAME}`:   self.creative.CreativeName,
		`{DSP_PUBLISHER_ID}`:    fmt.Sprintf("%v", attribute.RPub.PubID),
		`{DSP_PUBLISHER_NAME}`:  attribute.ACL.PubStr,
	}
}

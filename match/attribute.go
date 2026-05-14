package match

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/advice"
	"github.com/guruperl/aofei/demo"
	"github.com/guruperl/aofei/dh"
	"github.com/guruperl/aofei/maxmind"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type Attribute struct {
	RPub
	*NativeFormat `json:"-"`
	IsApp         bool      `json:"is_app"`
	IsVideo       bool      `json:"is_video"`
	When          time.Time `json:"when"`
	IFA           string    `json:"ifa,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	*demo.Demo
	*maxmind.Geo
	*advice.PzUa
	*dh.DH `json:"-"`
	*acl.ACL
}

type AttributePlus struct {
	Attribute
	RAdv
	Elapsed  int64  `json:"elapsed"`
	Source   string `json:"source,omitempty"`
	Contract string `json:"contract,omitempty"`
}

// NewAttribute creates a new Attribute from a bid request.
func NewAttribute(ctx context.Context, ipSearch *maxmind.IPSearch, bidRequest *openrtb2.BidRequest, pubObj *acl.Pub, when time.Time, pubStr string) (*Attribute, error) {
	return NewAttributeForImp(ctx, ipSearch, bidRequest, 0, pubObj, when, pubStr)
}

// NewAttributeForImp creates a new Attribute from one impression in a bid request.
func NewAttributeForImp(ctx context.Context, ipSearch *maxmind.IPSearch, bidRequest *openrtb2.BidRequest, impIndex int, pubObj *acl.Pub, when time.Time, pubStr string) (*Attribute, error) {
	if bidRequest == nil {
		return nil, fmt.Errorf("bid request is nil")
	}
	if len(bidRequest.Imp) == 0 {
		return nil, fmt.Errorf("bid request has no impressions")
	}
	if impIndex < 0 || impIndex >= len(bidRequest.Imp) {
		return nil, fmt.Errorf("impression index %d out of range", impIndex)
	}
	device := bidRequest.Device
	if device == nil {
		return nil, fmt.Errorf("bid request has no device")
	}
	if pubObj == nil {
		return nil, fmt.Errorf("publisher is nil")
	}

	attr := &Attribute{
		When:    when,
		IsVideo: bidRequest.Imp[impIndex].Video != nil,
		IsApp:   bidRequest.App != nil,
	}

	var err error
	attr.IFA = getIFA(device)
	attr.UserID = attr.IFA
	if bidRequest.User != nil {
		switch {
		case bidRequest.User.BuyerUID != "":
			attr.UserID = bidRequest.User.BuyerUID
		case bidRequest.User.ID != "":
			attr.UserID = bidRequest.User.ID
		case attr.IFA != "":
			attr.UserID = attr.IFA
		}
	}

	attr.Demo = demo.NewOpenRTBDemo(bidRequest)
	if ipSearch != nil {
		attr.Geo, err = ipSearch.NewOpenRTBGeo(device)
		if err != nil {
			return nil, err
		}
	} else {
		attr.Geo = &maxmind.Geo{}
	}

	attr.DH = dh.NewDHFromMinutes(when, attr.Geo.Location.UTCOffset)
	attr.PzUa = advice.NewOpenRTBPzUa(device)
	attr.ACL = acl.NewOpenRTBACLForImp(bidRequest, impIndex, pubStr)
	d1, d2, d3 := pubObj.GetRPub(attr.ACL.SiteStr, attr.ACL.SlotStr, attr.IsApp)
	attr.RPub = RPub{PubID: d1, SiteID: d2, SlotID: d3}
	attr.RPub.SizeID, attr.NativeFormat, err = getSizeIDNativeForImp(&bidRequest.Imp[impIndex])
	if err != nil {
		return nil, err
	}
	return attr, nil
}

// getIFA returns the IFA from the device.
func getIFA(device *openrtb2.Device) string {
	ifa := device.IFA
	if ifa == "" {
		switch {
		case device.DIDSHA1 != "":
			ifa = device.DIDSHA1
		case device.DIDMD5 != "":
			ifa = device.DIDMD5
		case device.DPIDSHA1 != "":
			ifa = device.DPIDSHA1
		case device.DPIDMD5 != "":
			ifa = device.DPIDMD5
		case device.MACSHA1 != "":
			ifa = device.MACSHA1
		case device.MACMD5 != "":
			ifa = device.MACMD5
		case device.UA != "" && device.IP != "":
			v := md5.Sum([]byte(device.IP + "." + device.UA))
			ifa = hex.EncodeToString(v[:])
		default:
		}
	}
	return ifa
}

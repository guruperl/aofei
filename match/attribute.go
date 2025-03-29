package match

import (
	"context"
	"crypto/md5"
	"time"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dh"
	"github.com/genelet/winter/maxmind"
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
	Elapsed time.Duration `json:"elapsed"`
}

// NewAttribute creates a new Attribute from a bid request.
func NewAttribute(ctx context.Context, ipSearch *maxmind.IPSearch, bidRequest *openrtb2.BidRequest, pubObj *acl.Pub, when time.Time, pubStr string) (*Attribute, error) {
	device := bidRequest.Device
	if device == nil {
		return nil, nil
	}

	attr := &Attribute{
		When:    when,
		IsVideo: bidRequest.Imp[0].Video != nil,
		IsApp:   bidRequest.App != nil,
	}

	var err error
	attr.IFA = getIFA(device)

	if bidRequest.User != nil && (bidRequest.User.BuyerUID != "" || bidRequest.User.ID != "") {
		attr.UserID = bidRequest.User.BuyerUID
		if attr.UserID == "" {
			attr.UserID = bidRequest.User.ID
		}
	}

	attr.Demo = demo.NewOpenRTBDemo(bidRequest)
	attr.Geo, err = ipSearch.NewOpenRTBGeo(device)
	if err != nil {
		return nil, err
	}

	attr.DH = dh.NewDH(when, uint8(attr.Geo.Location.UTCOffset))
	attr.PzUa = advice.NewOpenRTBPzUa(device)
	attr.ACL = acl.NewOpenRTBACL(bidRequest, pubStr)
	d1, d2, d3 := pubObj.GetRPub(attr.ACL.SiteStr, attr.ACL.SlotStr, attr.IsApp)
	attr.RPub = RPub{PubID: d1, SiteID: d2, SlotID: d3}
	attr.RPub.SizeID, attr.NativeFormat, err = getSizeIDNative(bidRequest)
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
		case device.MACSHA1 != "":
			ifa = device.MACSHA1
		case device.MACMD5 != "":
			ifa = device.MACMD5
		case device.UA != "" && device.IP != "":
			v := md5.Sum([]byte(device.IP + "." + device.UA))
			ifa = string(v[:])
		default:
		}
	}
	return ifa
}

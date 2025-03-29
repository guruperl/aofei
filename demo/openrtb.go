package demo

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// NewDemo creates a new Demo from a bid request.
func NewOpenRTBDemo(bidRequest *openrtb2.BidRequest) *Demo {
	var demo *Demo
	user := bidRequest.User
	if user != nil {
		demo = NewDemo(user.Gender, uint32(user.Yob), getLangs(bidRequest))
	} else {
		demo = NewDemo("", 0, getLangs(bidRequest))
	}

	return demo
}

// getLangs returns the WLangs object from the bid request and device.
func getLangs(bidRequest *openrtb2.BidRequest) []string {
	var langs []string
	if bidRequest.WLang != nil {
		langs = bidRequest.WLang
	} else if bidRequest.Device != nil && bidRequest.Device.Language != "" || bidRequest.Device.LangB != "" {
		if bidRequest.Device.Language != "" {
			langs = append(langs, bidRequest.Device.Language)
		}
		if bidRequest.Device.LangB != "" {
			langs = append(langs, bidRequest.Device.LangB)
		}
	}

	return langs
}

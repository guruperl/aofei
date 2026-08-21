package demo

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// NewDemo creates a new Demo from a bid request.
func NewOpenRTBDemo(bidRequest *openrtb2.BidRequest) *Demo {
	if bidRequest == nil {
		return NewDemo("", 0, nil)
	}
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
	if bidRequest == nil {
		return nil
	}
	var langs []string
	if bidRequest.WLang != nil {
		langs = bidRequest.WLang
	} else if device := bidRequest.Device; device != nil && (device.Language != "" || device.LangB != "") {
		if device.Language != "" {
			langs = append(langs, device.Language)
		}
		if device.LangB != "" {
			langs = append(langs, device.LangB)
		}
	}

	return langs
}

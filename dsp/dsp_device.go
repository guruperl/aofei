package dsp

import (
	"github.com/genelet/winter/ipsearch"
	"github.com/genelet/winter/uadevice"

	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

func convertGeoIPUA(bidRequest *openrtb2.BidRequest) (uint32, *uadevice.PzUa, *ipsearch.PzGeo, error) {
	var ip32 uint32
	var ua *uadevice.PzUa
	var geo *ipsearch.PzGeo
	var err error

	if bidRequest.Device != nil {
		if bidRequest.Device.IP != "" {
			ip32, err = ipsearch.IPToLong(bidRequest.Device.IP)
			if err != nil {
				return 0, nil, nil, err
			}
		}

		if bidRequest.Device.UA != "" {
			ua = uadevice.GetPzUa(bidRequest.Device.UA)
		}
	}

	if bidRequest.User != nil {
		if bidRequest.User.Geo != nil {
			geo = ipsearch.NewPzGeo(bidRequest.User.Geo.Lat, bidRequest.User.Geo.Lon)
		}
	}

	return ip32, ua, geo, nil
}

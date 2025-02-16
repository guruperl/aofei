package dsp

import (
	"github.com/genelet/winter/ipsearch"

	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

func convertGeo(geo *openrtb2.Geo) (*ipsearch.PzGeo, error) {
	if geo == nil {
		return nil, nil
	}

	pzGeo := new(ipsearch.PzGeo)
	if geo.Lat != nil && geo.Lon != nil {
		pzGeo.Lat = *geo.Lat
		pzGeo.Lon = *geo.Lon
		pzGeo.Type = geo.Type
		pzGeo.Accuracy = geo.Accuracy
		pzGeo.LastFix = geo.LastFix
	}
	pzGeo.Metro = geo.Metro
	pzGeo.City = geo.City
	pzGeo.State = geo.Region
	pzGeo.Country = geo.Country
	pzGeo.Zip = geo.ZIP
}

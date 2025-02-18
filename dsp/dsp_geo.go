package dsp

import (
	ipsearch "github.com/genelet/winter/maxmind"
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
	}
	pzGeo.SetType(geo.Type)
	pzGeo.SetAccuracy(geo.Accuracy)
	pzGeo.SetLastFix(geo.LastFix)
	pzGeo.Metro = geo.Metro
	pzGeo.City = geo.City
	pzGeo.State = geo.Region
	pzGeo.Country = geo.Country
	pzGeo.Zip = geo.ZIP

	return pzGeo, nil
}

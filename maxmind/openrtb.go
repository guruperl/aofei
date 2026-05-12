package maxmind

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// NewOpenRTBGeo returns the Geo object from the device.
func (self *IPSearch) NewOpenRTBGeo(device *openrtb2.Device) (*Geo, error) {
	pzg := new(Geo)
	if device.ConnectionType != nil {
		pzg.ConnectionType = *device.ConnectionType
	}
	if geo := device.Geo; geo != nil {
		if geo.Lat != nil && geo.Lon != nil {
			pzg.Location.Lat = *geo.Lat
			pzg.Location.Lon = *geo.Lon
		}
		pzg.Location.Type = geo.Type
		pzg.Location.Accuracy = geo.Accuracy
		pzg.Location.LastFix = geo.LastFix
		pzg.Location.IPService = geo.IPService
		if geo.UTCOffset != 0 {
			pzg.Location.UTCOffset = geo.UTCOffset
		}
		if geo.Country != "" {
			pzg.CountryID = self.CountryMap[geo.Country]
			if geo.Region != "" {
				pzg.StateID = self.StateMap[pzg.CountryID][geo.Region]
			}
		}
		if geo.Metro != "" {
			pzg.DmaID = MetroMap[geo.Metro]
		}
	}

	if device.IP != "" && needsIPGeo(pzg) {
		mm, err := self.CreatePzGeo(device.IP)
		if err != nil {
			return nil, err
		}
		if pzg.CountryID == 0 {
			pzg.CountryID = mm.Geo.CountryID
		}
		if pzg.StateID == 0 {
			pzg.StateID = mm.Geo.StateID
		}
		if pzg.CityID == 0 {
			pzg.CityID = mm.Geo.CityID
		}
		if pzg.DmaID == 0 {
			pzg.DmaID = mm.Geo.DmaID
		}
		if pzg.Location.UTCOffset == 0 {
			pzg.Location.UTCOffset = mm.Geo.Location.UTCOffset
		}
	}
	return pzg, nil
}

func needsIPGeo(pzg *Geo) bool {
	return pzg.CountryID == 0 || pzg.StateID == 0 || pzg.CityID == 0 || pzg.DmaID == 0
}

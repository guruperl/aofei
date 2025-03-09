package holiday

import (
	ipsearch "github.com/genelet/winter/maxmind"
	adcom1 "github.com/mxmCherry/openrtb/adcom1"
)

type Geo struct {
	DeviceGeo *adcom1.Geo
	Ips       *ipsearch.IpSearch
	IPString  string
}

func (self *Geo) IP32() uint32 {
	return Ip32Uint(self.IPString)
}

func (self *Geo) GetTags() *Tags {
	if self.DeviceGeo != nil {
		//	return self.GetTagsFromIncoming()
	}

	if IsPrivateIP(self.IPString) {
		return nil
	}

	pzgeo := self.Ips.CreatePzGeo(self.IPString)
	if pzgeo.StateId == 0 {
		return nil
	}
	ref := map[uint32][]uint32{
		1101: []uint32{uint32(pzgeo.ContinentId)},
		1102: []uint32{uint32(pzgeo.CountryId)},
		1103: []uint32{uint32(pzgeo.StateId)},
		1105: []uint32{uint32(pzgeo.DmaId)},
		1104: []uint32{uint32(pzgeo.CityId)},
		1106: []uint32{uint32(pzgeo.ZipId)},
		1107: []uint32{uint32(pzgeo.IspId)}}
	return &Tags{TagHashArray: ref}
}

/*
type LocationType int8
const (
    LocationGPS          LocationType = 1 // GPS/Location Services
    LocationIP           LocationType = 2 // IP Address
    LocationUserProvided LocationType = 3 // User Provided (e.g., registration data)
)

type IPLocationService int8
const (
    LocationServiceIP2Location IPLocationService = 1 // ip2location
    LocationServiceNeustar     IPLocationService = 2 // Neustar (Quova)
    LocationServiceMaxMind     IPLocationService = 3 // MaxMind
    LocationServiceNetAcuity   IPLocationService = 4 // NetAcuity (Digital Element)
)

type Geo struct {
    Type LocationType `json:"type,omitempty"`
    Lat float64 `json:"lat,omitempty"`
    Lon float64 `json:"lon,omitempty"`
    //   Estimated location accuracy in meters; recommended when lat/lon
    Accur int64 `json:"accur,omitempty"`
    //   Number of seconds since this geolocation fix was established.
    LastFix int64 `json:"lastfix,omitempty"`
    IPServ IPLocationService `json:"ipserv,omitempty"`
    //   Country code using ISO-3166-1-alpha-2.
    Country string `json:"country,omitempty"`
    //   Region code using ISO-3166-2; 2-letter state code if USA.
    Region string `json:"region,omitempty"`
    //   Regional marketing areas such as Nielsen's DMA codes
    Metro string `json:"metro,omitempty"`
    //   City using United Nations Code for Trade & Transport Locations “UN/LOCODE” with the space between country and city suppressed (e.g., Boston MA, USA = “USBOS”).
    City string `json:"city,omitempty"`
    //   ZIP or postal code.
    ZIP string `json:"zip,omitempty"`
    //   Local time as the number +/- of minutes from UTC.
    UTCOffset int64 `json:"utcoffset,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}
*/

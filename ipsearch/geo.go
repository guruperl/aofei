package ipsearch

import "github.com/prebid/openrtb/v20/adcom1"

type Location struct {
	Lat       float64
	Lon       float64
	Type      adcom1.LocationType
	Accuracy  int64
	LastFix   int64
	IPService adcom1.IPLocationService
}

// Geo 33 bytes
// 1+2+2+2+4+2+4+8+8 = 33
type Geo struct {
	ContinentID uint8
	CountryID   uint16
	StateID     uint16
	DmaID       uint16
	CityID      uint32
	IspID       uint16
	ZipID       uint32
	Location
}

type ipIndex struct {
	StartIP     uint32
	EndIP       uint32
	LocalOffset uint32
	LocalLength uint32
	Geo
	LocalString []byte
}

type PzGeo struct {
	Geo
	Continent string
	Country   string
	State     string
	Metro     string
	City      string
	Zip       string
	Isp       string
}

package maxmind

import (
	"github.com/prebid/openrtb/v20/adcom1"
)

type Location struct {
	Lat        float64
	Lon        float64
	aType      adcom1.LocationType
	aAccuracy  int64
	aLastFix   int64
	aIPService adcom1.IPLocationService
}

// Geo 33 bytes
// 1+4+4+4+4+2+4+location (8+8?) = 39
type Geo struct {
	ContinentID uint8
	CountryID   uint32
	StateCode   uint32
	DmaID       uint32
	CityID      uint32
	IspID       uint16
	ZipID       uint32
	Location
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

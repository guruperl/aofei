package maxmind

import (
	"github.com/prebid/openrtb/v20/adcom1"
)

type Location struct {
	ConnectionType adcom1.ConnectionType    `json:"connection_type,omitempty"`
	Lat            float64                  `json:"lat,omitempty"`
	Lon            float64                  `json:"lon,omitempty"`
	UTCOffset      int64                    `json:"utc_offset,omitempty"`
	Type           adcom1.LocationType      `json:"type,omitempty"`
	Accuracy       int64                    `json:"accuracy,omitempty"`
	LastFix        int64                    `json:"last_fix,omitempty"`
	IPService      adcom1.IPLocationService `json:"ip_service,omitempty"`
}

// Geo 33 bytes
// 1+4+4+4+4+2+4+location (8+8?) = 39
type Geo struct {
	ContinentID uint8  `json:"-"`
	CountryID   uint32 `json:"country_id,omitempty"`
	StateID     uint32 `json:"state_id,omitempty"`
	DmaID       uint32 `json:"dma_id,omitempty"`
	CityID      uint32 `json:"city_id,omitempty"`
	IspID       uint16 `json:"isp_id,omitempty"`
	ZipID       uint32 `json:"zip_id,omitempty"`
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

var MetroMap = map[string]uint32{
	"New York":         501,
	"Los Angeles":      502,
	"Chicago":          503,
	"Philadelphia":     504,
	"Dallas-Ft. Worth": 505,
	"San Francisco":    506,
	"Boston":           507,
	"Washington, DC":   508,
	"Atlanta":          509,
	"Houston":          510,
	"Detroit":          511,
	"Phoenix":          512,
	"Seattle":          513,
	"Minneapolis":      514,
	"San Diego":        515,
	"St. Louis":        516,
	"Tampa":            517,
	"Baltimore":        518,
	"Denver":           519,
	"Pittsburgh":       520,
}

var MetroMapRev = map[uint32]string{
	501: "New York",
	502: "Los Angeles",
	503: "Chicago",
	504: "Philadelphia",
	505: "Dallas-Ft. Worth",
	506: "San Francisco",
	507: "Boston",
	508: "Washington, DC",
	509: "Atlanta",
	510: "Houston",
	511: "Detroit",
	512: "Phoenix",
	513: "Seattle",
	514: "Minneapolis",
	515: "San Diego",
	516: "St. Louis",
	517: "Tampa",
	518: "Baltimore",
	519: "Denver",
	520: "Pittsburgh",
}

package maxmind

import (
	"github.com/prebid/openrtb/v20/adcom1"
)

type Location struct {
	ConnectionType adcom1.ConnectionType
	Lat            float64
	Lon            float64
	UTCOffset      int64
	aType          adcom1.LocationType
	aAccuracy      int64
	aLastFix       int64
	aIPService     adcom1.IPLocationService
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

func (self *PzGeo) SetType(atype adcom1.LocationType) {
	self.aType = atype
}

func (self *PzGeo) SetAccuracy(aAccuracy int64) {
	self.aAccuracy = aAccuracy
}

func (self *PzGeo) SetLastFix(aLastFix int64) {
	self.aLastFix = aLastFix
}

var CountryMap = map[string]uint32{
	"USA": 840,
	"CAN": 124,
	"MEX": 484,
	"GBR": 826,
	"FRA": 250,
	"DEU": 276,
	"JPN": 392,
	"CHN": 156,
	"IND": 356,
	"BRA": 76,
	"RUS": 643,
	"AUS": 36,
	"ITA": 380,
	"ESP": 724,
	"KOR": 410,
	"SAU": 682,
	"ZAF": 710,
	"NGA": 566,
	"ARG": 32,
	"COL": 170,
	"EGY": 818,
	"THA": 764,
	"VNM": 704,
	"IDN": 360,
	"IRN": 364,
	"TUR": 792,
	"PAK": 586,
	"PHL": 608,
	"UKR": 804,
	"POL": 616,
}

var CountryMapRev = map[uint32]string{
	840: "USA",
	124: "CAN",
	484: "MEX",
	826: "GBR",
	250: "FRA",
	276: "DEU",
	392: "JPN",
	156: "CHN",
	356: "IND",
	76:  "BRA",
	643: "RUS",
	36:  "AUS",
	380: "ITA",
	724: "ESP",
	410: "KOR",
	682: "SAU",
	710: "ZAF",
	566: "NGA",
	32:  "ARG",
	170: "COL",
	818: "EGY",
	764: "THA",
	704: "VNM",
	360: "IDN",
	364: "IRN",
	792: "TUR",
	586: "PAK",
	608: "PHL",
	804: "UKR",
	616: "POL",
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

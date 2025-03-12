package maxmind

import (
	"github.com/genelet/winter/pzutil"
	"github.com/prebid/openrtb/v20/adcom1"
)

type GeoAudience struct {
	GeoCountries    []uint32
	GeoStates       []uint32
	GeoDmas         []uint32
	GeoCitys        []uint32
	GeoIsps         []uint32
	GeoZipcodes     []uint32
	GeoLons         []uint32
	GeoLats         []uint32
	ConnectionTypes []uint32
}

func (self *GeoAudience) Has(geo *Geo) bool {
	if self.GeoCountries != nil && (geo == nil || geo.CountryID == 0 || !pzutil.GrepUint32(self.GeoCountries, uint32(geo.CountryID))) {
		return false
	}
	if self.GeoStates != nil && (geo == nil || geo.StateID == 0 || !pzutil.GrepUint32(self.GeoStates, uint32(geo.StateID))) {
		return false
	}
	if self.GeoDmas != nil && (geo == nil || geo.DmaID == 0 || !pzutil.GrepUint32(self.GeoDmas, uint32(geo.DmaID))) {
		return false
	}
	if self.GeoCitys != nil && (geo == nil || geo.CityID == 0 || !pzutil.GrepUint32(self.GeoCitys, uint32(geo.CityID))) {
		return false
	}
	if self.GeoIsps != nil && (geo == nil || geo.IspID == 0 || !pzutil.GrepUint32(self.GeoIsps, uint32(geo.IspID))) {
		return false
	}
	if self.GeoZipcodes != nil && (geo == nil || geo.ZipID == 0 || !pzutil.GrepUint32(self.GeoZipcodes, uint32(geo.ZipID))) {
		return false
	}
	if self.GeoLons != nil && (geo == nil || geo.Lon == 0 || !pzutil.GrepUint32(self.GeoLons, uint32(geo.Lon))) {
		return false
	}
	if self.GeoLats != nil && (geo == nil || geo.Lat == 0 || !pzutil.GrepUint32(self.GeoLats, uint32(geo.Lat))) {
		return false
	}
	if self.ConnectionTypes != nil && (geo == nil || geo.ConnectionType == adcom1.ConnectionUnknown || !pzutil.GrepUint32(self.ConnectionTypes, uint32(geo.ConnectionType))) {
		return false
	}

	return true
}

/*
	func GeoAudienceFromArgs(ARGS url.Values) *GeoAudience {
		g := func(_ url.Values, name string, which *[]uint32) {
			values := ARGS[name]
			if len(values) > 0 {
				for _, value := range values {
					v, err := strconv.ParseUint(value, 10, 32)
					if err == nil && v > 0 {
						*which = append(*which, uint32(v))
					}
				}
			}
		}

		aud := new(GeoAudience)

		g(ARGS, "state", &aud.GeoStates)
		g(ARGS, "dma", &aud.GeoDmas)
		g(ARGS, "city", &aud.GeoCitys)
		g(ARGS, "isp", &aud.GeoIsps)

		return aud
	}

	func (self *GeoAudience) ToArgs(ARGS url.Values) {
		g := func(args url.Values, name string, values []uint32) {
			if len(values) > 0 {
				for _, value := range values {
					args.Add(name, strconv.FormatUint(uint64(value), 10))
				}
			}
		}

		g(ARGS, "state", self.GeoStates)
		g(ARGS, "dma", self.GeoDmas)
		g(ARGS, "city", self.GeoCitys)
		g(ARGS, "isp", self.GeoIsps)
	}
*/

func (self *GeoAudience) DBFillGeoAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "country":
		self.GeoCountries = append(self.GeoCountries, valueID)
		return 1
	case "state":
		self.GeoStates = append(self.GeoStates, valueID)
		return 1
	case "dma":
		self.GeoDmas = append(self.GeoDmas, valueID)
		return 1
	case "city":
		self.GeoCitys = append(self.GeoCitys, valueID)
		return 1
	case "isp":
		self.GeoIsps = append(self.GeoIsps, valueID)
		return 1
	case "zip":
		self.GeoZipcodes = append(self.GeoZipcodes, valueID)
		return 1
	case "lon":
		self.GeoLons = append(self.GeoLons, valueID)
		return 1
	case "lat":
		self.GeoLats = append(self.GeoLats, valueID)
		return 1
	case "bandwidth":
		self.ConnectionTypes = append(self.ConnectionTypes, valueID)
		return 1
	default:
	}
	return 0
}

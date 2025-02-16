package maxmind

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/pzutil"
)

type GeoAudience struct {
	GeoStates []uint32
	GeoDmas   []uint32
	GeoCitys  []uint32
	GeoIsps   []uint32
}

func (self *GeoAudience) MatchGeo(geo Geo) bool {
	if !pzutil.GrepUint32(self.GeoStates, uint32(geo.StateCode)) {
		return false
	}
	if !pzutil.GrepUint32(self.GeoDmas, uint32(geo.DmaID)) {
		return false
	}
	if !pzutil.GrepUint32(self.GeoCitys, uint32(geo.CityID)) {
		return false
	}
	if !pzutil.GrepUint32(self.GeoIsps, uint32(geo.IspID)) {
		return false
	}

	return true
}

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

func (self *GeoAudience) DBFillGeoAudience(attrname string, valueID uint32) {
	switch attrname {
	case "state":
		self.GeoStates = append(self.GeoStates, valueID)
	case "dma":
		self.GeoDmas = append(self.GeoDmas, valueID)
	case "city":
		self.GeoCitys = append(self.GeoCitys, valueID)
	case "isp":
		self.GeoIsps = append(self.GeoIsps, valueID)
	default:
	}
}

func (self *GeoAudience) DBLineGeoAudience(attrname string, valueids string) {
	if valueids == "" {
		return
	}
	for _, id := range strings.Split(valueids, ",") {
		if valueID, err := strconv.ParseUint(id, 10, 32); err == nil {
			self.DBFillGeoAudience(attrname, uint32(valueID))
		}
	}
}

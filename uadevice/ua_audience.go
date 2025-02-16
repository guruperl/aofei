package uadevice

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/pzutil"
)

type UaAudience struct {
	UaBVersion  uint32
	UaOVersion  uint32
	UaBrowsers  []uint32
	UaOSs       []uint32
	UaPlatforms []uint32
	UaDevices   []uint32
}

func (self *UaAudience) MatchUa(ua PzUa) bool {
	if self.UaBVersion > ua.BVersion {
		return false
	}
	if self.UaOVersion > ua.OVersion {
		return false
	}
	if !pzutil.GrepUint32(self.UaBrowsers, ua.Browser) {
		return false
	}
	if !pzutil.GrepUint32(self.UaOSs, ua.OS) {
		return false
	}
	if !pzutil.GrepUint32(self.UaPlatforms, ua.Platform) {
		return false
	}
	if !pzutil.GrepUint32(self.UaDevices, ua.Device) {
		return false
	}

	return true
}

func UaAudienceFromArgs(ARGS url.Values) *UaAudience {
	f := func(_ url.Values, name string, which *uint32) {
		value := ARGS.Get(name)
		if value != "" {
			v, err := strconv.ParseUint(value, 10, 32)
			if err == nil {
				*which = uint32(v)
			}
		}
	}

	g := func(_ url.Values, name string, which *[]uint32) {
		values := ARGS[name]
		if values != nil && len(values) > 0 {
			for _, value := range values {
				v, err := strconv.ParseUint(value, 10, 32)
				if err == nil && v > 0 {
					*which = append(*which, uint32(v))
				}
			}
		}
	}

	aud := new(UaAudience)

	f(ARGS, "bversion", &aud.UaBVersion)
	f(ARGS, "oversion", &aud.UaOVersion)

	g(ARGS, "browser", &aud.UaBrowsers)
	g(ARGS, "os", &aud.UaOSs)
	g(ARGS, "platform", &aud.UaPlatforms)
	g(ARGS, "device", &aud.UaDevices)

	return aud
}

func (self *UaAudience) ToArgs(ARGS url.Values) {
	f := func(args url.Values, name string, value uint32) {
		if value > 0 {
			args.Add(name, strconv.FormatUint(uint64(value), 10))
		}
	}

	g := func(args url.Values, name string, values []uint32) {
		if values != nil && len(values) > 0 {
			for _, value := range values {
				args.Add(name, strconv.FormatUint(uint64(value), 10))
			}
		}
	}

	f(ARGS, "bversion", self.UaBVersion)
	f(ARGS, "oversion", self.UaOVersion)

	g(ARGS, "browser", self.UaBrowsers)
	g(ARGS, "os", self.UaOSs)
	g(ARGS, "platform", self.UaPlatforms)
	g(ARGS, "device", self.UaDevices)
}

func (aud *UaAudience) DBFillUaAudience(attrname string, value_id uint32) {
	switch attrname {
	case "bversion":
		aud.UaBVersion = value_id
	case "oversion":
		aud.UaOVersion = value_id
	case "browser":
		aud.UaBrowsers = append(aud.UaBrowsers, value_id)
	case "os":
		aud.UaOSs = append(aud.UaOSs, value_id)
	case "platform":
		aud.UaPlatforms = append(aud.UaPlatforms, value_id)
	case "device":
		aud.UaDevices = append(aud.UaDevices, value_id)

	default:
	}
}

func (aud *UaAudience) DbLineUaAudience(attrname string, value_ids string) {
	if value_ids == "" {
		return
	}
	for _, id := range strings.Split(value_ids, ",") {
		if value_id, err := strconv.ParseUint(id, 10, 32); err == nil {
			aud.DBFillUaAudience(attrname, uint32(value_id))
		}
	}
}

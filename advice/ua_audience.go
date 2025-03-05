package advice

import (
	"net/url"
	"strconv"
)

type UaAudience struct {
	UaBrowsers  uint32
	UaOSs       uint32
	UaOVersions uint32
	UaPlatforms uint32
	UaDevices   uint32
}

func NewUaAudience(oss []int, ovs []int, makers []int, dts []int) *UaAudience {
	aud := new(UaAudience)
	for _, os := range oss {
		aud.UaOSs |= 1 << uint(os)
	}
	for _, ov := range ovs {
		aud.UaOVersions |= 1 << uint(ov)
	}
	for _, maker := range makers {
		if maker >= 32 {
			aud.UaBrowsers |= 1 << uint(maker-32)
		} else {
			aud.UaPlatforms |= 1 << uint(maker)
		}
	}
	for _, dt := range dts {
		aud.UaDevices |= 1 << uint(dt)
	}
	return aud
}

func (self *UaAudience) Tmpls() map[string]map[int][]interface{} {
	pzuas := make(map[string]map[int][]interface{})
	ref := make(map[string]map[int]bool)
	if self.UaOSs != 0 {
		ref["os"] = make(map[int]bool)
		for _, item := range uint32OSs(self.UaOSs) {
			ref["os"][int(item)] = true
		}
	}
	if self.UaOVersions != 0 {
		ref["oversion"] = make(map[int]bool)
		for _, item := range uint32OVersions(self.UaOVersions) {
			ref["oversion"][int(item)] = true
		}
	}
	if self.UaPlatforms != 0 || self.UaBrowsers != 0 {
		ref["platform"] = make(map[int]bool)
		for _, item := range uint32Platforms(self.UaPlatforms, self.UaBrowsers) {
			ref["platform"][int(item)] = true
		}
	}
	if self.UaDevices != 0 {
		ref["device"] = make(map[int]bool)
		for _, item := range uint32Devices(self.UaDevices) {
			ref["device"][int(item)] = true
		}
	}
	for attrname, val := range uaNames() {
		item := make(map[int][]interface{})
		for valueID, name := range val {
			item[int(valueID)] = []interface{}{name, ref[attrname][int(valueID)]}
		}
		pzuas[attrname] = item
	}
	return pzuas
}

func uint32OSs(x uint32) []DeviceOS {
	var os []DeviceOS
	for i := 0; i < 32; i++ {
		if x&(1<<uint(i)) > 0 {
			os = append(os, DeviceOS(i))
		}
	}
	return os
}

func uint32OVersions(x uint32) []DeviceOSV {
	var ovs []DeviceOSV
	for i := 0; i < 32; i++ {
		if x&(1<<uint(i)) > 0 {
			ovs = append(ovs, DeviceOSV(i))
		}
	}
	return ovs
}

func uint32Platforms(x, y uint32) []DeviceMake {
	var makers []DeviceMake
	for i := 0; i < 32; i++ {
		if x&(1<<uint(i)) > 0 {
			makers = append(makers, DeviceMake(i))
		}
	}
	for i := 0; i < 32; i++ {
		if y&(1<<uint(i)) > 0 {
			makers = append(makers, DeviceMake(i+32))
		}
	}
	return makers
}

func uint32Devices(x uint32) []DeviceType {
	var dts []DeviceType
	for i := 0; i < 32; i++ {
		if x&(1<<uint(i)) > 0 {
			dts = append(dts, DeviceType(i))
		}
	}
	return dts
}

func (self *UaAudience) MatchUa(ua *PzUa) bool {
	if ua.OS != OSUnknown && (self.UaOSs == 0 || (self.UaOSs&uint32(ua.OS)) == 0) {
		return false
	}
	if ua.OVersion != OVersionUnknown && (self.UaOVersions == 0 || (self.UaOVersions&uint32(ua.OVersion)) == 0) {
		return false
	}
	if ua.Platform >= 32 {
		if self.UaBrowsers == 0 || (self.UaBrowsers&uint32(ua.Platform-32)) == 0 {
			return false
		}
	} else if ua.Platform != MakerUnknown {
		if self.UaPlatforms == 0 || (self.UaPlatforms&uint32(ua.Platform)) == 0 {
			return false
		}
	}
	if ua.Device != TypeUnknown && (self.UaDevices == 0 || (self.UaDevices&uint32(ua.Device)) == 0) {
		return false
	}
	return true
}

func UaAudienceFromArgs(ARGS url.Values) (*UaAudience, error) {
	pars := make(map[string][]int)
	for _, item := range []string{"os", "oversion", "platform", "device"} {
		if values, ok := ARGS[item]; ok {
			for _, value := range values {
				if value != "" {
					v, err := strconv.ParseInt(value, 10, 32)
					if err != nil {
						return nil, err
					}
					pars[item] = append(pars[item], int(v))
				}
			}
		}
	}
	return NewUaAudience(pars["os"], pars["oversion"], pars["platform"], pars["device"]), nil
}

func (self *UaAudience) ToArgs(ARGS url.Values) {
	if self.UaOSs != 0 {
		ARGS.Del("os")
		for _, item := range uint32OSs(self.UaOSs) {
			ARGS.Add("os", strconv.FormatUint(uint64(item), 10))
		}
	}
	if self.UaOVersions != 0 {
		ARGS.Del("oversion")
		for _, item := range uint32OVersions(self.UaOVersions) {
			ARGS.Add("oversion", strconv.FormatUint(uint64(item), 10))
		}
	}
	if self.UaPlatforms != 0 || self.UaBrowsers != 0 {
		ARGS.Del("platform")
		for _, item := range uint32Platforms(self.UaPlatforms, self.UaBrowsers) {
			ARGS.Add("platform", strconv.FormatUint(uint64(item), 10))
		}
	}
	if self.UaDevices != 0 {
		ARGS.Del("device")
		for _, item := range uint32Devices(self.UaDevices) {
			ARGS.Add("device", strconv.FormatUint(uint64(item), 10))
		}
	}
}

func ResetArgs(ARGS url.Values) error {
	aud, err := UaAudienceFromArgs(ARGS)
	if err != nil {
		return err
	}
	if aud == nil {
		return nil
	}

	if aud.UaOSs != 0 {
		ARGS.Set("os", strconv.FormatUint(uint64(aud.UaOSs), 10))
	}
	if aud.UaOVersions != 0 {
		ARGS.Set("oversion", strconv.FormatUint(uint64(aud.UaOVersions), 10))
	}
	if aud.UaPlatforms != 0 {
		ARGS.Set("platform", strconv.FormatUint(uint64(aud.UaPlatforms), 10))
	}
	if aud.UaBrowsers != 0 {
		ARGS.Set("browser", strconv.FormatUint(uint64(aud.UaBrowsers), 10))
	}
	if aud.UaDevices != 0 {
		ARGS.Set("device", strconv.FormatUint(uint64(aud.UaDevices), 10))
	}
	return nil
}

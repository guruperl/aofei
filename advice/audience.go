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

func newUaAudience(oss []uint32, ovs []uint32, platforms []uint32, devices []uint32) *UaAudience {
	aud := new(UaAudience)
	for _, os := range oss {
		aud.UaOSs += (1 << os)
	}
	for _, ov := range ovs {
		aud.UaOVersions += (1 << ov)
	}
	for _, platform := range platforms {
		if platform >= 32 {
			aud.UaBrowsers += (1 << (platform - 32))
		} else {
			aud.UaPlatforms += (1 << platform)
		}
	}
	for _, device := range devices {
		aud.UaDevices += (1 << device)
	}
	return aud
}

// hasOS returns true if the UaAudience has the given OS.
func (self *UaAudience) hasOS(os DeviceOS, display ...bool) bool {
	if (display == nil || !display[0]) && self.UaOSs == 0 {
		return true
	}
	if os == OSUnknown {
		return false
	}
	return self.UaOSs&(1<<uint32(os)) > 0
}

// hasOV returns true if the UaAudience has the given OS version.
func (self *UaAudience) hasOV(ov DeviceOSV, display ...bool) bool {
	if (display == nil || !display[0]) && self.UaOVersions == 0 {
		return true
	}
	if ov == OVersionUnknown {
		return false
	}
	return self.UaOVersions&(1<<uint32(ov)) > 0
}

// hasPlatform returns true if the UaAudience has the given platform.
func (self *UaAudience) hasPlatform(platform DeviceMake, display ...bool) bool {
	if uint32(platform) >= 32 {
		if (display == nil || !display[0]) && self.UaBrowsers == 0 {
			return true
		}
		return self.UaBrowsers&(1<<uint32(platform-32)) > 0
	}
	if (display == nil || !display[0]) && self.UaPlatforms == 0 {
		return true
	}
	if platform == MakerUnknown {
		return false
	}
	return self.UaPlatforms&(1<<uint32(platform)) > 0
}

// hasDevice returns true if the UaAudience has the given device.
func (self *UaAudience) hasDevice(device DeviceType, display ...bool) bool {
	if (display == nil || !display[0]) && self.UaDevices == 0 {
		return true
	}
	if device == TypeUnknown {
		return false
	}
	return self.UaDevices&(1<<uint32(device)) > 0
}

// Has returns true if the UaAudience has the given user agent.
func (self *UaAudience) Has(ua *PzUa) bool {
	// this logic is not supposed to happen, but just in case
	if self == nil {
		return true
	}

	if !self.hasOS(ua.OS) {
		return false
	}
	if !self.hasOV(ua.OVersion) {
		return false
	}
	if !self.hasDevice(ua.Device) {
		return false
	}
	if !self.hasPlatform(ua.Platform) {
		return false
	}

	return true
}

// UAResetArgs resets the ARGS to the values in the UaAudience, ready to be inserted or updated in the database.
func UAResetArgs(ARGS url.Values) error {
	pars := make(map[string][]uint32)
	for _, item := range []string{"os", "oversion", "platform", "device"} {
		if values, ok := ARGS[item]; ok {
			for _, value := range values {
				if value == "0" {
					pars[item] = nil
					break
				}
				if value != "" {
					v, err := strconv.ParseInt(value, 10, 32)
					if err != nil {
						return err
					}
					pars[item] = append(pars[item], uint32(v))
				}
			}
		}
	}
	if len(pars) == 0 {
		return nil
	}

	aud := newUaAudience(pars["os"], pars["oversion"], pars["platform"], pars["device"])

	ARGS.Del("os")
	ARGS.Del("oversion")
	ARGS.Del("platform")
	ARGS.Del("browser")
	ARGS.Del("device")
	if aud.UaOSs != 0 {
		ARGS.Set("os", strconv.FormatInt(int64(aud.UaOSs), 10))
	}
	if aud.UaOVersions != 0 {
		ARGS.Set("oversion", strconv.FormatInt(int64(aud.UaOVersions), 10))
	}
	if aud.UaPlatforms != 0 {
		ARGS.Set("platform", strconv.FormatInt(int64(aud.UaPlatforms), 10))
	}
	if aud.UaBrowsers != 0 {
		ARGS.Set("browser", strconv.FormatInt(int64(aud.UaBrowsers), 10))
	}
	if aud.UaDevices != 0 {
		ARGS.Set("device", strconv.FormatInt(int64(aud.UaDevices), 10))
	}
	return nil
}

// Tmpls returns the UaAudience in a map of attribute name to valueID, ready to use on web page.
func (self *UaAudience) Tmpls() map[string]map[int][]interface{} {
	pzuas := make(map[string]map[int][]interface{})
	for attrname, val := range uaNames() {
		item := make(map[int][]interface{})
		for valueID, name := range val {
			switch attrname {
			case "os":
				item[int(valueID)] = []interface{}{name, self.hasOS(DeviceOS(valueID), true)}
			case "oversion":
				item[int(valueID)] = []interface{}{name, self.hasOV(DeviceOSV(valueID), true)}
			case "platform":
				item[int(valueID)] = []interface{}{name, self.hasPlatform(DeviceMake(valueID), true)}
			case "device":
				item[int(valueID)] = []interface{}{name, self.hasDevice(DeviceType(valueID), true)}
			}
		}
		pzuas[attrname] = item
	}
	return pzuas
}

// DBFillUaAudience fills the UaAudience with the given attribute name and valueID, derived from the database.
func (self *UaAudience) DBFillUaAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "os":
		self.UaOSs = valueID
		return 1
	case "oversion":
		self.UaOVersions = valueID
		return 1
	case "platform":
		self.UaPlatforms = valueID
		return 1
	case "browser":
		self.UaBrowsers = valueID
		return 1
	case "device":
		self.UaDevices = valueID
		return 1
	default:
	}

	return 0
}

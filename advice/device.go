// Package advice handles device and user agent
package advice

import (
	"github.com/mileusna/useragent"
)

type PzUa struct {
	OS       DeviceOS
	OVersion DeviceOSV
	Platform DeviceMake
	Device   DeviceType
}

func GetPzUa(raw string) *PzUa {
	if raw == "" {
		return nil
	}
	ua := useragent.Parse(raw)
	var maker DeviceMake
	if ua.Device != "" {
		maker = ParseMaker(ua.Device)
	}
	if maker == MakerUnknown {
		maker = ParseMaker(ua.Name)
	}
	os := ParseOS(ua.OS)
	ov := ParseOVersion(ua.OSVersion)
	dt := ParseType(ua, os)
	return &PzUa{Platform: maker, OS: os, OVersion: ov, Device: dt}
}

func (self *PzUa) Pack() uint32 {
	return uint32(self.OS)<<24 + uint32(self.OVersion)<<16 + uint32(self.Platform)<<8 + uint32(self.Device)
}

func UnpackPzUa(n uint32) *PzUa {
	return &PzUa{Platform: DeviceMake(n >> 8 & 0xff), OS: DeviceOS(n >> 24 & 0xff), OVersion: DeviceOSV(n >> 16 & 0xff), Device: DeviceType(n & 0xff)}
}

func UaAttrs() map[string]string {
	return map[string]string{"oversion": "OS Version", "os": "OS", "platform": "Platform", "device": "Device type"}
}

func uaNames() map[string]map[uint32]string {
	ovs := make(map[uint32]string)
	oss := make(map[uint32]string)
	platforms := make(map[uint32]string)
	devices := make(map[uint32]string)

	for i := range int8(OVersion20) + 1 {
		k := DeviceOSV(i)
		ovs[uint32(i)] = k.String()
	}
	for i := range int8(OSWindowsPhone) + 1 {
		k := DeviceOS(i)
		oss[uint32(i)] = k.String()
	}
	for i := range int(MakerBLACKBERRY) + 1 {
		k := DeviceMake(i)
		platforms[uint32(i)] = k.String()
	}
	for i := range int(TypeTablet) + 1 {
		k := DeviceType(i)
		devices[uint32(i)] = k.String()
	}
	return map[string]map[uint32]string{"oversion": ovs, "os": oss, "platform": platforms, "device": devices}
}

// Package advice handles device and user agent
package advice

import (
	"github.com/mileusna/useragent"
)

/*
type UA uint32;
browser: 0
version: 0
os: 2^8
version: 2^8
platform: 2^8
device: 2^8
*/

type PzUa struct {
	Browser  uint32
	BVersion uint32
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
	osvs := make(map[uint32]string)
	oss := make(map[uint32]string)
	platforms := make(map[uint32]string)
	devices := make(map[uint32]string)

	for i := range 20 {
		k := DeviceOSV(i)
		osvs[uint32(i)] = k.String()
	}
	for i := range 24 {
		k := DeviceOS(i)
		oss[uint32(i)] = k.String()
	}
	for i := range 47 {
		k := DeviceMake(i)
		platforms[uint32(i)] = k.String()
	}
	for i := range 7 {
		k := DeviceType(i)
		devices[uint32(i)] = k.String()
	}
	return map[string]map[uint32]string{"oversion": osvs, "os": oss, "platform": platforms, "device": devices}
}

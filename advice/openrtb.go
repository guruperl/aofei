package advice

import (
	"github.com/prebid/openrtb/v20/openrtb2"
)

// NewOpenRTBPzUa returns the PzUa object from the openrtb device.
func NewOpenRTBPzUa(device *openrtb2.Device) *PzUa {
	pzua := new(PzUa)
	if device.DeviceType != 0 {
		pzua.Device = DeviceTypeType(device.DeviceType)
	}
	if device.Make != "" {
		pzua.Platform = ParseMaker(device.Make)
	}
	// device.Model?
	if device.OS != "" {
		pzua.OS = ParseOS(device.OS)
	}
	if device.OSV != "" {
		pzua.OVersion = ParseOVersion(device.OSV)
	}

	if device.UA != "" && (pzua.OS == 0 || pzua.OVersion == 0 || pzua.Platform == 0 || pzua.Device == 0) {
		mm := GetPzUa(device.UA)
		if pzua.OS == 0 {
			pzua.OS = mm.OS
		}
		if pzua.OVersion == 0 {
			pzua.OVersion = mm.OVersion
		}
		if pzua.Platform == 0 {
			pzua.Platform = mm.Platform
		}
		if pzua.Device == 0 {
			pzua.Device = mm.Device
		}
	}
	return pzua
}

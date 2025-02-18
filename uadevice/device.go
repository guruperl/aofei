package uadevice

import (
	"net/url"
	"strconv"

	"github.com/avct/uasurfer"
	"github.com/genelet/winter/pzutil"
)

/*
type UA uint32;
browser: 2^6
version: 2^7
os: 2^5
version: 2^6
platform: 2^4
device: 2^4
*/
type PzUa struct {
	Browser  uint32
	BVersion uint32
	OS       uint32
	OVersion uint32
	Platform uint32
	Device   uint32
}

func GetPzUa(raw string) *PzUa {
	if raw == "" {
		return nil
	}
	ua := uasurfer.Parse(raw)

	b := uint32(ua.Browser.Name)
	bv := uint32(ua.Browser.Version.Major)
	o := uint32(ua.OS.Name)
	ov := uint32(ua.OS.Version.Major)
	p := uint32(ua.OS.Platform)
	d := uint32(ua.DeviceType)

	return &PzUa{b, bv, o, ov, p, d}
}

func CreateTwoUa(raw string) *PzUa {
	if raw == "" {
		return nil
	}
	ua := uasurfer.Parse(raw)

	b := uint32(ua.Browser.Name)
	bv := uint32(ua.Browser.Version.Major)
	o := uint32(ua.OS.Name)
	ov := uint32(ua.OS.Version.Major)
	p := uint32(ua.OS.Platform)
	d := uint32(ua.DeviceType)

	return &PzUa{b, bv, o, ov, p, d}
}

func (self *PzUa) Pack() uint32 {
	if self.Browser >= 64 {
		self.Browser = 0
	}
	if self.BVersion >= 128 {
		self.BVersion = 127
	}
	if self.OS >= 32 {
		self.OS = 0
	}
	if self.OVersion >= 64 {
		self.OVersion = 63
	}
	if self.Platform >= 16 {
		self.Platform = 0
	}
	if self.Device >= 16 {
		self.Device = 0
	}

	return uint32((self.Browser&63)<<26) +
		uint32((self.BVersion&127)<<19) +
		uint32((self.OS&31)<<14) +
		uint32((self.OVersion&63)<<8) +
		uint32((self.Platform&15)<<4) +
		uint32(self.Device&15)
}

func UaAttrs() map[string]string {
	return map[string]string{"browser": "Browser", "os": "Operation System", "platform": "Platform", "device": "Device"}
}

func UaNames() map[string]map[uint32]string {
	browsers := make(map[uint32]string)
	oss := make(map[uint32]string)
	platforms := make(map[uint32]string)
	devices := make(map[uint32]string)

	for i := 0; i < 29; i++ {
		k := uint32(i)
		if pzutil.GrepUint32([]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 15, 16, 17}, k) {
			browsers[k] = uasurfer.BrowserName(i).String()
		}
	}
	for i := 0; i < 15; i++ {
		k := uint32(i)
		if pzutil.GrepUint32([]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}, k) {
			oss[k] = uasurfer.OSName(i).String()
		}
	}
	for i := 0; i < 13; i++ {
		k := uint32(i)
		if pzutil.GrepUint32([]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, k) {
			platforms[k] = uasurfer.Platform(i).String()
		}
	}
	for i := 0; i < 7; i++ {
		k := uint32(i)
		if pzutil.GrepUint32([]uint32{1, 2, 3, 4, 5, 6}, k) {
			devices[k] = uasurfer.DeviceType(i).String()
		}
	}

	return map[string]map[uint32]string{"browser": browsers, "os": oss, "platform": platforms, "device": devices}
}

func CreatePzUa(browser, bversion, os, oversion, platform, device string) *PzUa {
	browsers := make([]string, 0)
	oss := make([]string, 0)
	platforms := make([]string, 0)
	devices := make([]string, 0)
	for i := 0; i < 29; i++ {
		browsers = append(browsers, uasurfer.BrowserName(i).String())
	}
	for i := 0; i < 15; i++ {
		oss = append(oss, uasurfer.OSName(i).String())
	}
	for i := 0; i < 13; i++ {
		platforms = append(platforms, uasurfer.Platform(i).String())
	}
	for i := 0; i < 7; i++ {
		devices = append(devices, uasurfer.DeviceType(i).String())
	}

	b := pzutil.IndexString(browsers, browser)
	if b < 0 {
		b = 0
	}
	bv, err := strconv.Atoi(bversion)
	if err != nil {
		bv = 0
	}
	o := pzutil.IndexString(oss, os)
	if o < 0 {
		o = 0
	}
	ov, err := strconv.Atoi(oversion)
	if err != nil {
		ov = 0
	}
	p := pzutil.IndexString(platforms, platform)
	if p < 0 {
		p = 0
	}
	d := pzutil.IndexString(devices, device)
	if d < 0 {
		d = 0
	}

	return &PzUa{uint32(b), uint32(bv), uint32(o), uint32(ov), uint32(p), uint32(d)}
}

func UnpackPzUa(ua uint32) *PzUa {
	return &PzUa{UnpackPzUaBrowser(ua), UnpackPzUaBVersion(ua), UnpackPzUaOs(ua), UnpackPzUaOVersion(ua), UnpackPzUaPlatform(ua), UnpackPzUaDevice(ua)}
}
func UnpackPzUaBrowser(ua uint32) uint32 {
	return (ua >> 26) & 63
}
func UnpackPzUaBVersion(ua uint32) uint32 {
	return (ua >> 19) & 127 // 1<<7
}
func UnpackPzUaOs(ua uint32) uint32 {
	return (ua >> 14) & 31 // 1<<5
}
func UnpackPzUaOVersion(ua uint32) uint32 {
	return (ua >> 8) & 63
}
func UnpackPzUaPlatform(ua uint32) uint32 {
	return (ua >> 4) & 15
}
func UnpackPzUaDevice(ua uint32) uint32 {
	return (ua >> 0) & 15
}

func MatchBrowser(uas []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(uas, UnpackPzUaBrowser), attrvalue)
}
func MatchBVersion(uas []uint32, attrvalue uint32) bool {
	for _, v := range pzutil.MapUint32(uas, UnpackPzUaBVersion) {
		if attrvalue >= v {
			return true
		}
	}
	return false
}
func MatchOs(uas []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(uas, UnpackPzUaOs), attrvalue)
}
func MatchOVersion(uas []uint32, attrvalue uint32) bool {
	for _, v := range pzutil.MapUint32(uas, UnpackPzUaOVersion) {
		if attrvalue >= v {
			return true
		}
	}
	return false
}
func MatchPlatform(uas []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(uas, UnpackPzUaPlatform), attrvalue)
}
func MatchDevice(uas []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(uas, UnpackPzUaDevice), attrvalue)
}

func SetArgsPzuas(ARGS url.Values, pzua, browser, bversion, os, oversion, platform, device string) {
	devices, ok := ARGS[device]
	if !ok {
		devices = make([]string, 0)
	}
	browsers, ok := ARGS[browser]
	if !ok {
		browsers = make([]string, 0)
	}
	oss, ok := ARGS[os]
	if !ok {
		oss = make([]string, 0)
	}
	platforms, ok := ARGS[platform]
	if !ok {
		platforms = make([]string, 0)
	}

	for _, device := range devices {
		for _, browser := range browsers {
			for _, os := range oss {
				for _, platform := range platforms {
					ua := CreatePzUa(browser, ARGS.Get(bversion), os, ARGS.Get(oversion), platform, device)
					ARGS.Add(pzua, strconv.FormatUint(uint64(ua.Pack()), 10))
				}
			}
		}
	}
}

func GetPzUaParameters(other map[string]interface{}) {
	browsers := make(map[int]string)
	oss := make(map[int]string)
	platforms := make(map[int]string)
	devices := make(map[int]string)
	for i := 1; i < 29; i++ {
		browsers[i] = uasurfer.BrowserName(i).String()
	}
	for i := 1; i < 15; i++ {
		oss[i] = uasurfer.OSName(i).String()
	}
	for i := 1; i < 13; i++ {
		platforms[i] = uasurfer.Platform(i).String()
	}
	for i := 1; i < 6; i++ {
		devices[i] = uasurfer.DeviceType(i).String()
	}

	other["browser"] = browsers
	other["os"] = oss
	other["platform"] = platforms
	other["device"] = devices
}

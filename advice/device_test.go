package advice

import (
	"testing"
)

func TestDevice(t *testing.T) {
	bs := []uint32{0, 1, 11, 21, 31, 41, 51, 61}
	bvs := []uint32{0, 12, 52, 102}
	os := []uint32{0, 3, 13, 23}
	ovs := []uint32{0, 4, 14, 24, 34, 44, 54}
	ps := []uint32{0, 5, 15}
	ds := []uint32{0, 6, 7, 8, 9}
	uas := make([]uint32, 0)
	for _, b := range bs {
		for _, bv := range bvs {
			for _, o := range os {
				for _, ov := range ovs {
					for _, p := range ps {
						for _, d := range ds {
							ua := &PzUa{b, bv, DeviceOS(o), DeviceOSV(ov), DeviceMake(p), DeviceType(d)}
							packed := ua.Pack()
							uas = append(uas, packed)
							ua1 := UnpackPzUa(packed)
							if ua1.Browser != ua.Browser ||
								ua1.BVersion != ua.BVersion ||
								ua1.OS != ua.OS ||
								ua1.OVersion != ua.OVersion ||
								ua1.Platform != ua.Platform ||
								ua1.Device != ua.Device {
								t.Errorf("%v %v", ua1, ua)
							}
						}
					}
				}
			}
		}
	}
	for _, ua := range uas {
		ua1 := UnpackPzUa(ua)
		if ua1.Pack() != ua {
			t.Errorf("%v %v", ua1, ua)
		}
	}

	ua := GetPzUa("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36")
	if ua.Browser != 1 ||
		ua.BVersion != 59 ||
		ua.OS != 2 ||
		ua.OVersion != 10 ||
		ua.Platform != 1 ||
		ua.Device != 1 {
		t.Errorf("%v", ua)
	}

	//if self.Browser  >= 64  { self.Browse   = 0; }
	//if self.BVersion >= 128 { self.BVersion = 127; }
	//if self.OS       >= 32  { self.OS       = 0; }
	//if self.OVersion >= 64  { self.OVersion = 63; }
	//if self.Platform >= 16  { self.Platform = 0; }
	//if self.Device   >= 16  { self.Device   = 0; }
	ua.Browser = 65
	ua.BVersion = 128
	ua.OS = 40
	ua.OVersion = 70
	ua.Platform = 20
	ua.Device = 16
	packed := ua.Pack()
	ua = UnpackPzUa(packed)
	if ua.Browser != 0 ||
		ua.BVersion != 127 ||
		ua.OS != 0 ||
		ua.OVersion != 63 ||
		ua.Platform != 0 ||
		ua.Device != 0 {
		t.Errorf("%v", ua)
	}

	//t.Errorf("%v", GetPzUaParameters())
}

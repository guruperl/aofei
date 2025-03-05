package advice

import (
	"testing"
)

func TestUAName(t *testing.T) {
	names := uaNames()
	for k, v := range names {
		switch k {
		case "os":
			for key, value := range v {
				if key == 1 && value != "Android" {
					t.Errorf("%v %v", key, value)
				}
			}
		case "oversion":
			for key, value := range v {
				if key == 1 && value != "1" {
					t.Errorf("%v %v", key, value)
				}
			}
		case "platform":
			for key, value := range v {
				if key == 1 && value != "OPPO" {
					t.Errorf("%v %v", key, value)
				}
			}
		case "device":
			for key, value := range v {
				if key == 1 && value != "Connected Device" {
					t.Errorf("%v %v", key, value)
				}
			}
		}
	}
}

func TestDevice(t *testing.T) {
	os := []uint32{0, 3}
	ovs := []uint32{0, 4, 14}
	ps := []uint32{0, 5, 15}
	ds := []uint32{0, 1, 2}
	uas := make([]uint32, 0)
	for _, o := range os {
		for _, ov := range ovs {
			for _, p := range ps {
				for _, d := range ds {
					ua := &PzUa{OS: DeviceOS(o), OVersion: DeviceOSV(ov), Platform: DeviceMake(p), Device: DeviceType(d)}
					packed := ua.Pack()
					uas = append(uas, packed)
					ua1 := UnpackPzUa(packed)
					if ua1.OS != ua.OS ||
						ua1.OVersion != ua.OVersion ||
						ua1.Platform != ua.Platform ||
						ua1.Device != ua.Device {
						t.Errorf("%v %v", ua1, ua)
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
	if ua.OS != OSWindows ||
		ua.OVersion != OVersion10 ||
		ua.Platform != MakerUnknown ||
		ua.Device != TypePersonalComputer {
		t.Errorf("%v", ua)
	}
}

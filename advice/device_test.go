package advice

import (
	"testing"
)

func TestUAName(t *testing.T) {
	for k, v := range uaNames() {
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
	ua := GetPzUa("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36")
	if ua.OS != OSWindows ||
		ua.OVersion != OVersion10 ||
		ua.Platform != MakerUnknown ||
		ua.Device != TypePersonalComputer {
		t.Errorf("%v", ua)
	}
}

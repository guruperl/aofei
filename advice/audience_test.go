package advice

import (
	"net/url"
	"testing"
)

func TestUaAudienceArgs(t *testing.T) {
	ARGS := url.Values{}
	ARGS.Add("os", "1")
	ARGS.Add("os", "2")
	ARGS.Add("oversion", "1")
	ARGS.Add("oversion", "2")
	ARGS.Add("platform", "1")
	ARGS.Add("platform", "2")
	ARGS.Add("device", "1")
	ARGS.Add("device", "2")

	if ARGS["os"][0] != "1" || ARGS["os"][1] != "2" {
		t.Errorf("os: %v", ARGS["os"])
	}
	if ARGS["oversion"][0] != "1" || ARGS["oversion"][1] != "2" {
		t.Errorf("oversion: %v", ARGS["oversion"])
	}
	if ARGS["platform"][0] != "1" || ARGS["platform"][1] != "2" {
		t.Errorf("platform: %v", ARGS["platform"])
	}
	if ARGS["device"][0] != "1" || ARGS["device"][1] != "2" {
		t.Errorf("device: %v", ARGS["device"])
	}

	ARGS.Add("platform", "30")
	ARGS.Add("platform", "31")
	ARGS.Add("platform", "32")
	ARGS.Add("platform", "33")
	ARGS.Add("platform", "34")
	err := UAResetArgs(ARGS)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if ARGS.Get("os") != "6" {
		t.Errorf("os: %v", ARGS["os"])
	}
	if ARGS.Get("oversion") != "6" {
		t.Errorf("oversion: %v", ARGS["oversion"])
	}
	if ARGS.Get("device") != "6" {
		t.Errorf("device: %v", ARGS["device"])
	}
	if ARGS.Get("platform") != "3221225478" {
		t.Errorf("platform: %v", ARGS["platform"])
	}
}

func TestUaAudienceTmpls(t *testing.T) {
	aud := newUaAudience([]uint32{1, 2}, []uint32{1, 2}, []uint32{1, 2, 30, 31, 32, 33, 34}, []uint32{1, 2})
	tmpl := aud.Tmpls()
	for k, v := range tmpl["platform"] {
		switch k {
		case 1, 2, 30, 31, 32, 33, 34:
			if v[1].(bool) == false {
				t.Errorf("Tmpl %v, %v ", k, v)
			}
		default:
			if v[1].(bool) == true {
				t.Errorf("Tmpl %v, %v ", k, v)
			}
		}
	}
}

func TestUaAudienceReset(t *testing.T) {
	ARGS := url.Values{}
	ARGS.Add("os", "1")
	ARGS.Add("os", "2")
	ARGS.Add("oversion", "1")
	ARGS.Add("oversion", "2")
	ARGS.Add("platform", "1")
	ARGS.Add("platform", "2")
	ARGS.Add("platform", "30")
	ARGS.Add("platform", "31")
	ARGS.Add("platform", "32")
	ARGS.Add("platform", "33")
	ARGS.Add("platform", "34")
	ARGS.Add("device", "1")
	ARGS.Add("device", "2")

	err := UAResetArgs(ARGS)
	if err != nil {
		t.Fatal(err)
	}
	if ARGS["os"][0] != "6" ||
		ARGS["oversion"][0] != "6" ||
		ARGS["device"][0] != "6" ||
		ARGS["platform"][0] != "3221225478" ||
		ARGS["browser"][0] != "7" {
		t.Errorf("ARGS: %v", ARGS)
	}
}

func TestUaAudienceMatch(t *testing.T) {
	aud := newUaAudience([]uint32{1, 2}, []uint32{1, 2}, []uint32{1, 2, 30, 31, 32, 33, 34}, []uint32{1, 2})

	ua := &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(1), Device: DeviceType(1)}
	if !aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(1), Device: DeviceType(3)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(30), Device: DeviceType(1)}
	if !aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(30), Device: DeviceType(3)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(31), Device: DeviceType(1)}
	if !aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(31), Device: DeviceType(3)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(32), Device: DeviceType(1)}
	if !aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(32), Device: DeviceType(3)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(33), Device: DeviceType(1)}
	if !aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(33), Device: DeviceType(3)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(34), Device: DeviceType(1)}
	if !aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(34), Device: DeviceType(3)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(35), Device: DeviceType(1)}
	if aud.Has(ua) {
		t.Errorf("Match: %v", ua)
	}
}

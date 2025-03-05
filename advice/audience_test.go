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
	aud, err := UaAudienceFromArgs(ARGS)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if aud.UaOSs != 6 {
		t.Errorf("UaOSs: %v", aud.UaOSs)
	}
	if aud.UaOVersions != 6 {
		t.Errorf("UaOVersions: %v", aud.UaOVersions)
	}
	if aud.UaPlatforms != 6 {
		t.Errorf("UaPlatforms: %v", aud.UaPlatforms)
	}
	if aud.UaDevices != 6 {
		t.Errorf("UaDevices: %v", aud.UaDevices)
	}
	args := url.Values{}
	aud.ToArgs(args)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if args["os"][0] != "1" || args["os"][1] != "2" {
		t.Errorf("os: %v", args["os"])
	}
	if args["oversion"][0] != "1" || args["oversion"][1] != "2" {
		t.Errorf("oversion: %v", args["oversion"])
	}
	if args["platform"][0] != "1" || args["platform"][1] != "2" {
		t.Errorf("platform: %v", args["platform"])
	}
	if args["device"][0] != "1" || args["device"][1] != "2" {
		t.Errorf("device: %v", args["device"])
	}

	ARGS.Add("platform", "30")
	ARGS.Add("platform", "31")
	ARGS.Add("platform", "32")
	ARGS.Add("platform", "33")
	ARGS.Add("platform", "34")
	aud, err = UaAudienceFromArgs(ARGS)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	args = url.Values{}
	aud.ToArgs(args)
	if args["platform"][0] != "1" || args["platform"][1] != "2" || args["platform"][2] != "30" || args["platform"][3] != "31" || args["platform"][4] != "32" {
		t.Errorf("platform: %v", args["platform"])
	}
}

func TestUaAudienceTmpls(t *testing.T) {
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
	aud, err := UaAudienceFromArgs(ARGS)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
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

	ResetArgs(ARGS)
	if ARGS["os"][0] != "6" ||
		ARGS["oversion"][0] != "6" ||
		ARGS["device"][0] != "6" ||
		ARGS["platform"][0] != "3221225478" ||
		ARGS["browser"][0] != "7" {
		t.Errorf("ARGS: %v", ARGS)
	}
}

func TestUaAudienceMatch(t *testing.T) {
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
	aud, err := UaAudienceFromArgs(ARGS)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	ua := &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(1), Device: DeviceType(1)}
	if !aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(1), Device: DeviceType(3)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(30), Device: DeviceType(1)}
	if !aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(30), Device: DeviceType(3)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(31), Device: DeviceType(1)}
	if !aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(31), Device: DeviceType(3)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(32), Device: DeviceType(1)}
	if !aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(32), Device: DeviceType(3)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(33), Device: DeviceType(1)}
	if !aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(33), Device: DeviceType(3)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(34), Device: DeviceType(1)}
	if !aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(34), Device: DeviceType(3)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
	ua = &PzUa{OS: DeviceOS(1), OVersion: DeviceOSV(1), Platform: DeviceMake(35), Device: DeviceType(1)}
	if aud.MatchUa(ua) {
		t.Errorf("Match: %v", ua)
	}
}

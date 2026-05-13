package dsp

import (
	"testing"

	"github.com/guruperl/aofei/acl"
)

const currentPzdesignSSPSample = `{
  "site": "AAAACAH774AAA",
  "platform": "browser",
  "adUnits": [{
    "code": "pz-image-1234",
    "slot": "AAAACAAUAMAAA",
    "mediaTypes": {
      "banner": {
        "size": [300, 250]
      }
    }
	},{
	    "code": "pz-video-5678",
	    "slot": "CYBQAAH677776",
	    "mediaTypes": {
	      "video": {
        "context": "instream",
        "playerSize": [640, 480]
      }
    }
  },{
    "code": "pz-html-9012",
    "slot": "CUBQAAH774AAA",
    "mediaTypes": {
      "native": {
        "image": [150, 50],
        "title": true,
        "sponsoredBy": true,
        "body": true,
        "icon": [50, 50]
      }
    }
  }]
}`

const legacyHolidaySSPSample = `{
  "site": {"ID": "AAAACAH774AAA"},
  "platform": "browser",
  "adUnits": [{
    "config_id": "pz-image-1234",
    "code": "AAAACAAUAMAAA",
    "floor": 1.2,
    "banner": {
      "size": [300, 250]
    }
  },{
    "config_id": "pz-video-5678",
    "code": "AAAACAH677776",
    "floor": 1.2,
    "video": {
      "context": "instream",
      "playerSize": [640, 480]
    }
  },{
    "config_id": "pz-html-9012",
    "code": "CUBQAAH774AAA",
    "floor": 1.2,
    "native": {
      "image": [150, 50],
      "title": true,
      "sponsoredBy": true,
      "body": true,
      "icon": [50, 50]
    }
  }]
}`

func TestParseSSPRequestCurrentPzdesignSample(t *testing.T) {
	req, err := ParseSSPRequest([]byte(currentPzdesignSSPSample))
	if err != nil {
		t.Fatal(err)
	}
	if req.Platform != "browser" || string(req.Site) != "AAAACAH774AAA" {
		t.Fatalf("request identity = platform %q site %q", req.Platform, req.Site)
	}
	if len(req.AdUnits) != 3 {
		t.Fatalf("adUnits len = %d, want 3", len(req.AdUnits))
	}
	if req.AdUnits[0].Code != "pz-image-1234" || req.AdUnits[0].legacySlotToken() != "AAAACAAUAMAAA" {
		t.Fatalf("first adUnit = %#v", req.AdUnits[0])
	}
	if got := req.AdUnits[0].EffectiveMediaTypes().Banner.Size; len(got) != 2 || got[0] != 300 || got[1] != 250 {
		t.Fatalf("banner size = %#v", got)
	}
}

func TestParseSSPRequestSDKOpenRTBLikeFields(t *testing.T) {
	req, err := ParseSSPRequest([]byte(`{
		"id": "sdk-req",
		"platform": "sdk",
		"responseFormat": "openrtb",
		"site": "AAAACAH774AAA",
		"app": {"bundle": "example.com"},
		"device": {"ifa": "ifa-123", "ua": "sdk-agent"},
		"user": {"id": "user-123", "buyeruid": "buyer-123"},
		"adUnits": [{
			"code": "sdk-unit",
			"slot": "AAAACAAUAMAAA",
			"mediaTypes": {"banner": {"size": [300, 250]}}
		}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if req.Platform != "sdk" || req.ResponseFormat != "openrtb" {
		t.Fatalf("request contract = %#v", req)
	}
	if req.App == nil || req.App.Bundle != "example.com" {
		t.Fatalf("app = %#v", req.App)
	}
	if req.Device == nil || req.Device.IFA != "ifa-123" || req.Device.UA != "sdk-agent" {
		t.Fatalf("device = %#v", req.Device)
	}
	if req.User == nil || req.User.ID != "user-123" || req.User.BuyerUID != "buyer-123" {
		t.Fatalf("user = %#v", req.User)
	}
}

func TestParseSSPRequestLegacyHolidaySample(t *testing.T) {
	req, err := ParseSSPRequest([]byte(legacyHolidaySSPSample))
	if err != nil {
		t.Fatal(err)
	}
	if string(req.Site) != "AAAACAH774AAA" {
		t.Fatalf("site = %q, want legacy object ID", req.Site)
	}
	if req.AdUnits[1].legacySlotToken() != "AAAACAH677776" {
		t.Fatalf("legacy code slot token = %q", req.AdUnits[1].legacySlotToken())
	}
	if media := req.AdUnits[1].EffectiveMediaTypes(); media.Video == nil || media.Video.Context != "instream" {
		t.Fatalf("legacy video media = %#v", media.Video)
	}
}

func TestSSPValidateSupplyAgainstDirectPubCache(t *testing.T) {
	req, err := ParseSSPRequest([]byte(currentPzdesignSSPSample))
	if err != nil {
		t.Fatal(err)
	}
	pub := directSSPTestPub()
	units, err := req.ValidateSupply(pub)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 3 {
		t.Fatalf("validated units = %d, want 3", len(units))
	}
	got := units[2]
	if got.Code != "pz-html-9012" || got.SiteStr != "example.com" || got.SlotStr != "native-slot" {
		t.Fatalf("third validated unit = %#v", got)
	}
	if got.RPub.PubID != 65536 || got.RPub.SiteID != 65535 || got.RPub.SlotID != 789 || got.RPub.SizeID != 65535 {
		t.Fatalf("third RPub = %#v", got.RPub)
	}
}

func TestSSPValidateSupplyRejectsTamperedSiteSlotPair(t *testing.T) {
	req, err := ParseSSPRequest([]byte(currentPzdesignSSPSample))
	if err != nil {
		t.Fatal(err)
	}
	pub := directSSPTestPub()
	delete(pub.Slots[65535], 65536)
	if _, err := req.ValidateSupply(pub); err == nil {
		t.Fatal("expected tampered site/slot pair to fail")
	}
}

func TestSSPValidateSupplyDoesNotTrustCodeAsSlot(t *testing.T) {
	req, err := ParseSSPRequest([]byte(legacyHolidaySSPSample))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := req.ValidateSupply(directSSPTestPub()); err == nil {
		t.Fatal("expected legacy code-as-slot request to fail v1 supply validation")
	}
}

func directSSPTestPub() *acl.DirectPub {
	pub := &acl.Pub{
		PubID:  65536,
		Active: true,
		Sites:  map[string]uint32{"example.com": 65535},
		Slots: map[uint32]map[string]uint32{
			65535: map[string]uint32{
				"banner-slot": 65536,
				"video-slot":  790,
				"native-slot": 789,
			},
		},
		SlotSizes: map[uint32]map[uint32]uint32{
			65535: map[uint32]uint32{
				65536: 788,
				790:   4294967294,
				789:   65535,
			},
		},
	}
	return acl.NewDirectPub("pub.example", pub)
}

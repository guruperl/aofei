package holiday

import (
	"testing"
)

var ADs string = `{
"site": {"ID":"AAAACAH774AAA"},
"platform": "browser",
"adUnits" : [
	{
    	"config_id": "pz-image-1234",
        "code": "AAAACAAUAMAAA",
		"floor":1.2,
        "banner": {
            "size": [ 300, 250 ]
         }
    },{
         "config_id": "pz-video-5678",
         "code": "AAAACAH677776",
		 "floor":1.2,
         "video": {
             "context": "instream",
             "playerSize": [640, 480]
		}
    },{
         "config_id": "pz-html-9012",
         "code": "CUBQAAH774AAA",
		 "floor":1.2,
         "native": {
             "image": [150, 50],
             "title": true,
             "sponsoredBy": true,
             "body": true,
             "icon": [50, 50]
        }
    }]
}`

/*
func TestAdunit(t *testing.T) {
	incoming, err := NewIncoming([]byte(ADs))
	if err != nil {
		t.Fatal(err)
	}
	adunits := incoming.Adunits
	native := adunits[2]
	if !native.MatchMime(uint32(1)) {
		t.Errorf("%v", native.MatchMime(uint32(1)))}
	if !native.MatchMime(uint32(2)) {
		t.Errorf("%v", native.MatchMime(uint32(2)))}
	if !native.MatchMime(uint32(3)) {
		t.Errorf("%v", native.MatchMime(uint32(3)))}
	if  native.MatchMime(uint32(4)) {
		t.Errorf("%v", native.MatchMime(uint32(4)))}
	if !native.MatchMime(uint32(5)) {
		t.Errorf("%v", native.MatchMime(uint32(5)))}
	if !native.MatchMime(uint32(6)) {
		t.Errorf("%v", native.MatchMime(uint32(6)))}
	if !native.MatchMime(uint32(7)) {
		t.Errorf("%v", native.MatchMime(uint32(7)))}
	if !native.MatchMime(uint32(8)) {
		t.Errorf("%v", native.MatchMime(uint32(8)))}
	if incoming.PType != 1 {
		t.Errorf("%v", incoming.PType)
	}
}
*/

func TestIncoming(t *testing.T) {
	incoming, err := NewIncoming([]byte(ADs))
	if err != nil {
		t.Fatal(err)
	}
	adunits := incoming.Adunits
	if incoming.Platform!="browser" ||
		incoming.Site.ID !="AAAACAH774AAA" {
		t.Errorf("%#v", incoming.Platform)
		t.Errorf("%#v", incoming.Site.ID)
	}

	if adunits[0].ConfigId != "pz-image-1234" ||
		adunits[1].ConfigId != "pz-video-5678" ||
		adunits[2].ConfigId != "pz-html-9012" {
		t.Errorf("%#v", adunits[0])
		t.Errorf("%#v", adunits[1])
		t.Errorf("%#v", adunits[2])
		t.Errorf("%v", adunits[0])
		t.Errorf("%v", adunits[1])
		t.Errorf("%v", adunits[2])
	}
	x := adunits[1]
	if x.PubId != 65536 ||
		x.SiteId != 65535 ||
		x.SlotId != 65536 ||
		x.SizeId != 4294967294 ||
		x.Video.Context != "instream" ||
		x.Video.PlayerSize[0] != 640 ||
		x.Video.PlayerSize[1] != 480 {
		t.Errorf("%v", x)
	}
}

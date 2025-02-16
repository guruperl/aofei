package match

import (
	"encoding/json"
	"testing"
)

var ADs string = `{
"site": "AAAACAH774AAA",
"platform": "browser",
"adUnits" : [{
        "code": "pz-image-1234",
        "slot": "AAAACAAUAMAAA",
        "mediaTypes": {
            "banner": {
                "size": [ 300, 250 ]
            }
        }
    },{
        "code": "pz-video-5678",
        "slot": "AAAACAH677776",
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

func TestIncoming(t *testing.T) {
	incoming := &Incoming{}
	err := json.Unmarshal([]byte(ADs), incoming)
	if err != nil {
		t.Fatal(err)
	}
	imps, err := incoming.Unpack()
	if err != nil {
		t.Fatal(err)
	}
	if incoming.Platform != "browser" ||
		incoming.Site != "AAAACAH774AAA" ||
		(incoming.AdUnits)[0].Code != "pz-image-1234" ||
		(incoming.AdUnits)[1].Code != "pz-video-5678" ||
		(incoming.AdUnits)[2].Code != "pz-html-9012" {
		t.Errorf("%#v", incoming)
	}
	imp := imps[1]
	x := imp.RPub
	y := imp.Video
	if x.PubID != 65536 ||
		x.SiteID != 65535 ||
		x.SlotID != 65536 ||
		x.SizeID != 4294967294 ||
		y.Context != "instream" ||
		y.PlayerSize[0] != 640 ||
		y.PlayerSize[1] != 480 {
		t.Errorf("%v", x)
		t.Errorf("%v", y)
	}
}

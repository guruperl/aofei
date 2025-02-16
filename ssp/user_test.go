package ssp

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/ipsearch"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/pzutil"
	// "github.com/nats-io/nats.go"
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

func TestUserGet(t *testing.T) {
	c := pzutil.NewConfig("../conf/pzadx.conf")
	ips, _ := ipsearch.LoadIPData("../conf/qq-pz.dat")
	uastr := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	current := time.Now()

	sizeid := pzutil.GetSizeID(100, 200)
	rpub, _ := match.RPub{
		PubID:  1,
		SiteID: 2,
		SlotID: 3,
		SizeID: sizeid,
	}.Pack()
	r, _ := http.NewRequest("GET", "http://example.com/pz/"+rpub+".html?yob=1960&gender=F", bytes.NewBuffer([]byte("")))
	r.Header.Set("User-Agent", uastr)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "210.51.200.123:80"

	//status, incoming, adImp, clk, bid, err := match.GetPathIds(r, c)
	match.GetPathIds(r)

	user, err := CreateUser(r, c, ips, current)
	if err != nil {
		t.Fatal(err)
	}

	ua1 := user.PzUa
	if ua1.Browser != 1 ||
		ua1.BVersion != 59 ||
		ua1.OS != 2 ||
		ua1.OVersion != 10 ||
		ua1.Platform != 1 ||
		ua1.Device != 1 {
		t.Errorf("%v", ua1)
	}
	t.Errorf("%#v", user.PzGeo)
	geo := user.PzGeo
	if geo.ContinentID != 3 ||
		geo.CountryID != 48 ||
		geo.StateID != 620 ||
		geo.DmaID != 141 ||
		geo.CityID != 0 ||
		geo.IspID != 6 {
		t.Errorf("%v", geo)
	}

	ip32 := pzutil.IP2Uint(net.ParseIP("210.51.200.123"))
	if user.Visitor.Pid.StartIP != ip32 {
		t.Errorf("%v", user.Visitor)
		t.Errorf("%d %d", user.Visitor.Pid.StartIP, ip32)
	}

	dmo := demo.CreateArgsDemo(r.Form, "gender", "yob", "married", "income", "child", "household", "ethnicity", "education", "occupation")
	if dmo.Yob != 1960 || dmo.Gender != 2 {
		t.Errorf("%#v\n", dmo)
	}
}

func TestUserPost(t *testing.T) {
	c := pzutil.NewConfig("../conf/pzadx.conf")
	ips, _ := ipsearch.LoadIPData("../conf/qq-pz.dat")
	uastr := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	current := time.Now()

	r, _ := http.NewRequest("POST", "http://example.com/pz?yob=1960&gender=F", bytes.NewBuffer([]byte(ADs)))
	r.Header.Set("User-Agent", uastr)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "210.51.200.123:80"

	status, incoming, adImps, _, _, err := match.GetPathIds(r)
	if err != nil {
		t.Fatal(err)
	}

	user, err := CreateUser(r, c, ips, current)
	if err != nil {
		t.Fatal(err)
	}

	ua1 := user.PzUa
	if ua1.Browser != 1 ||
		ua1.BVersion != 59 || //"log"
		ua1.OS != 2 ||
		ua1.OVersion != 10 ||
		ua1.Platform != 1 ||
		ua1.Device != 1 {
		t.Errorf("%v", ua1)
	}

	geo := user.Geo
	if geo.ContinentID != 3 ||
		geo.CountryID != 48 ||
		geo.StateID != 620 ||
		geo.DmaID != 141 ||
		geo.CityID != 0 ||
		geo.IspID != 6 {
		t.Errorf("%v", geo)
	}

	ip32 := pzutil.IP2Uint(net.ParseIP("210.51.200.123"))
	if user.Visitor.Pid.StartIP != ip32 {
		t.Errorf("%v", user.Visitor)
		t.Errorf("%d %d", user.Visitor.Pid.StartIP, ip32)
	}

	dmo := demo.CreateArgsDemo(r.Form, "gender", "yob", "married", "income", "child", "household", "ethnicity", "education", "occupation")
	if dmo.Yob != 1960 || dmo.Gender != 2 {
		t.Errorf("%v", dmo)
	}

	if status.Mime != pzutil.JSON {
		t.Errorf("%#v", status)
	}
	if incoming.Platform != "browser" {
		t.Errorf("%#v", *incoming)
	}
	if len(adImps) != 3 {
		t.Errorf("%#v", adImps)
	}
	imp := user.ToImp(status, adImps[0].RPub)
	wins := make([]match.Win, 3)
	for i, adImp := range adImps {
		i32 := uint32(i)
		wins[i] = match.Win{
			SlotID: adImp.RPub.SlotID,
			RAdv: match.RAdv{
				AdvID:      i32,
				CampaignID: i32,
				ItemID:     i32,
				CreativeID: i32, //"log"
				Price:      float32(i) * 1.0,
			},
		}
	}
	record := match.Record{
		Imp:  imp,
		Wins: wins,
	}
	//"log"
	imp1 := imp
	imp1.IP32 += 1
	imp1.PzUa32 += 1
	imp1.PubID += 1
	imp1.SiteID += 1
	win1 := match.Win{
		SlotID: 100,
		RAdv: match.RAdv{
			AdvID:      11,
			CampaignID: 22,
			ItemID:     33,
			CreativeID: 44,
			Price:      55.5,
		},
	}
	record1 := match.Record{
		Imp:  imp1,
		Wins: []match.Win{win1},
	}

	imp2 := imp
	imp2.IP32 += 2
	imp2.PzUa32 += 2
	imp2.PubID += 2
	imp2.SiteID += 2
	win2 := match.Win{
		SlotID: 200,
		RAdv: match.RAdv{
			AdvID:      66,
			CampaignID: 77,
			ItemID:     88,
			CreativeID: 99,
			Price:      10.5,
		},
	}
	record2 := match.Record{
		Imp:  imp2,
		Wins: []match.Win{win2},
	}

	fn := "tmp.bin"
	f, err := os.Create(fn)
	if err != nil {
		t.Fatal(err)
	}
	if err := record.PackIO(f); err != nil {
		t.Fatal(err)
	}
	if err := record1.PackIO(f); err != nil {
		t.Fatal(err)
	} //"log"
	if err := record2.PackIO(f); err != nil {
		t.Fatal(err)
	}
	f.Close()

	g, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	gecord, err := match.UnpackRecordIO(g)
	if err != nil {
		t.Fatal(err)
	}
	gecord1, err := match.UnpackRecordIO(g)
	if err != nil {
		t.Fatal(err)
	}
	gecord2, err := match.UnpackRecordIO(g)
	if err != nil {
		t.Fatal(err)
	}
	g.Close()

	if record.Imp != gecord.Imp ||
		record1.Imp != gecord1.Imp ||
		record2.Imp != gecord2.Imp {
		t.Errorf("%#v", record)
		t.Errorf("%#v", *gecord)
	}
}

func getUserSample() *User {
	current := time.Now()
	fcap2 := match.NewFcap(current.Add(-40 * time.Minute))
	fcap22 := match.NewFcap(current.Add(-60 * time.Minute))
	user := new(User)
	user.FullTime = current
	icaps := make(map[uint32]match.Fcap)
	icaps[uint32(2)] = fcap2
	icaps[uint32(22)] = fcap22
	ccaps := make(map[uint32]match.Fcap)
	ccaps[uint32(2)] = fcap2
	ccaps[uint32(22)] = fcap22
	user.ICaps = icaps
	user.CCaps = ccaps
	return user
}

func TestWeight(t *testing.T) {
	weights := match.GetWeightSamples()
	w1 := weights[0]
	w11 := weights[1]

	user := getUserSample()
	a1, a2 := user.Check(w1)
	b1, b2 := user.Check(w11)
	if a1 != 1 || a2 != 1 || b1 != 1 || b2 != 1 { // both ok, add one more count
		t.Errorf("%d %d", a1, a2)
		t.Errorf("%d %d", b1, b2)
	}

	icaps := user.ICaps
	if _, ok := icaps[2]; !ok {
		t.Errorf("2 should exists %v", icaps)
	}
	if _, ok := icaps[22]; ok { // because expired
		t.Errorf("22 should NOT exists %v", icaps)
	}

	user = getUserSample()
	icaps = user.ICaps
	icaps[2] = match.RefreshFcap(icaps[2], time.Now())
	icaps[2] = match.RefreshFcap(icaps[2], time.Now())
	icaps[22] = match.RefreshFcap(icaps[22], time.Now())
	icaps[22] = match.RefreshFcap(icaps[22], time.Now())
	ccaps := user.CCaps
	ccaps[2] = match.RefreshFcap(ccaps[2], time.Now())
	ccaps[2] = match.RefreshFcap(ccaps[2], time.Now())
	ccaps[22] = match.RefreshFcap(ccaps[22], time.Now())
	ccaps[22] = match.RefreshFcap(ccaps[22], time.Now())
	a1, a2 = user.Check(w1)
	b1, b2 = user.Check(w11)
	if a1 != -1 || a2 != -1 || b1 != 1 || b2 != 1 {
		t.Errorf("%d %d", a1, a2)
		t.Errorf("%d %d", b1, b2)
	}

	user = getUserSample()
	w1.CapThrottle = 45
	a1, a2 = user.Check(w1)
	if a1 != -1 || a2 != 1 {
		t.Errorf("%d %d", a1, a2)
	}
}

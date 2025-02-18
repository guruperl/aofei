package ssp

import (
	"net"
	"testing"
	"time"

	ipsearch "github.com/genelet/winter/maxmind"
	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/uadevice"
)

func TestVisitor(t *testing.T) {
	current := time.Now()
	ua := uadevice.GetPzUa("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36")

	ipstr := "210.51.200.123"
	ip := net.ParseIP(ipstr)
	ip32 := pzutil.IP2Uint(ip)

	p, _ := ipsearch.LoadIPData("../conf/qq-pz.dat")
	geo, err := p.CreatePzGeo(ipstr)
	if err != nil {
		t.Fatal(err)
	}

	v := NewVisitor(current, ip32, ua, geo)
	packed, err := v.Pack()
	if err != nil {
		t.Fatal(err)
	}
	v1, err := UnpackVisitor(packed)
	if err != nil {
		t.Fatal(err)
	}

	if v1.Pid.StartIP != ip32 || v1.Pid.StartUa != ua.Pack() ||
		v1.lastIP != ip32 || v1.Geo != v.Geo {
		t.Errorf("%v %v", v, v1)
	}
	if v1.PzGeo.Zip != "420100" {
		t.Errorf("%v", v1.Geo)
	}

	visitor, needCookie, isNew, err := RetrieveVisitor("", p, ua, current, ipstr, ip32)
	geo = visitor.PzGeo
	if err != nil {
		t.Fatal(err)
	}
	if needCookie == false || isNew == false {
		t.Errorf("%v %v", visitor, geo)
	}
	pid := visitor.Pid
	g := visitor.Geo
	if pid.StartIP != 3526609019 || visitor.lastIP != 3526609019 ||
		g.ContinentID != 3 ||
		g.CountryID != 48 ||
		g.DmaID != 141 ||
		g.CityID != 0 ||
		g.IspID != 6 ||
		g.ZipID != 420100 {
		t.Errorf("%v %v", visitor, g)
	}
}

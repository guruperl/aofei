package maxmind

import (
	"testing"
	//"pzutil"
	//"database/sql"
	//_ "github.com/go-sql-driver/mysql"
)

func TestIpsearch(t *testing.T) {
	//p, err := LoadIPData("../conf/qqzeng-ip-utf8.dat");
	p, err := LoadIPData("../etc/GeoLite2-City.mmdb")
	if err != nil {
		t.Fatal(err)
	}

	//ip := "210.51.200.123"
	ip := "99.228.161.54"
	/*
		ipstr := p.GetSimple(ip)
		if ipstr != `亚洲|中国|湖北|武汉||联通|420100|China|CN|114.298572|30.584355` {
			t.Errorf("%s", ipstr)
		}
	*/

	pzgeo, err := p.CreatePzGeo(ip)
	if err != nil {
		t.Fatal(err)
	}
	if pzgeo.Continent != `亚洲` || pzgeo.Metro != `武汉` || pzgeo.Isp != `联通` {
		t.Errorf("%v", pzgeo)
	}
}

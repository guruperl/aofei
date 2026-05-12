package ipsearch

import (
	"os"
	"testing"
	//"pzutil"
	//"database/sql"
	//_ "github.com/go-sql-driver/mysql"
)

/**
 * @author xiao.luo
 * @description This is the unit test for IpSearch
 */

/*
// make a PZ ip dat file
func TestDatabase(t *testing.T) {
	c := pzutil.NewConfig("../../backup/gotest.conf")
	db, err := sql.Open(c.Db[0], c.Db[1])
	defer db.Close();
	if err != nil { t.Fatal(err); }
	err = DatabaseToDat(db, "../../etc/qq-pz.dat");
	if err != nil { t.Fatal(err); }
}
*/

func TestIpsearch(t *testing.T) {
	const dataFile = "../../etc/qq-pz.dat"
	if _, err := os.Stat(dataFile); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("requires local legacy qq-pz.dat asset at %s", dataFile)
		}
		t.Fatalf("stat %s: %v", dataFile, err)
	}
	//p, err := LoadIpData("../../etc/qqzeng-ip-utf8.dat");
	p, err := LoadIPData(dataFile)
	if err != nil {
		t.Fatal(err)
	}

	ip := "210.51.200.123"

	/*
		ipstr := p.GetSimple(ip)
		if ipstr != `亚洲|中国|湖北|武汉||联通|420100|China|CN|114.298572|30.584355` {
			t.Errorf("%s", ipstr)
		}
	*/

	ipfull, err := p.getIPIndex(ip)
	if err != nil {
		t.Fatal(err)
	}
	if ipfull.StartIP != 3526608896 {
		t.Errorf("%d", ipfull.StartIP)
	}
	if ipfull.Geo.ContinentID != 3 {
		t.Errorf("%d", ipfull.Geo.ContinentID)
	}
	if ipfull.Geo.CountryID != 48 {
		t.Errorf("%d", ipfull.Geo.CountryID)
	}
	if ipfull.Geo.StateID != 620 {
		t.Errorf("%d", ipfull.Geo.StateID)
	}
	if ipfull.Geo.DmaID != 141 {
		t.Errorf("%d", ipfull.Geo.DmaID)
	}
	if ipfull.Geo.CityID != 0 {
		t.Errorf("%d", ipfull.Geo.CityID)
	}
	if ipfull.Geo.IspID != 6 {
		t.Errorf("%d", ipfull.Geo.IspID)
	}
	if ipfull.Geo.ZipID != 420100 {
		t.Errorf("%d", ipfull.ZipID)
	}
	if ipfull.Geo.Lat != 30.584355 || ipfull.Geo.Lon != 114.298572 {
		t.Errorf("%f %f", ipfull.Geo.Lat, ipfull.Geo.Lon)
	}
	if string(ipfull.LocalString) != `亚洲|中国|湖北|武汉||420100|联通` {
		t.Errorf("%s", ipfull.LocalString)
	}

	pzgeo, err := p.CreatePzGeo(ip)
	if err != nil {
		t.Fatal(err)
	}
	if pzgeo.Continent != `亚洲` || pzgeo.Metro != `武汉` || pzgeo.Isp != `联通` {
		t.Errorf("%v", pzgeo)
	}
}

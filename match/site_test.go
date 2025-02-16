package match

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/genelet/winter/pzutil"
	_ "github.com/go-sql-driver/mysql"
)

func TestSite(t *testing.T) {
	site := GetSiteSample()
	ids := site.ChannelIds
	if (site.SiteName != "site name") || (ids[2] != uint16(55)) {
		t.Errorf("%v", site)
	}

	c := pzutil.NewConfig("../conf/gotest.conf")
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	site, err = DBGetSite(db, uint32(4))
	if err != nil {
		t.Fatal(err)
	}
	if uint32(4) != site.SiteID {
		t.Errorf("%v", site)
	}
}

func TestDomain(t *testing.T) {
	site := GetSiteSample()
	for _, d := range []string{"aaa.com", "aaa.com/abc", "www.aaa.com", "abc.aaa.com", "aaa.com/abc", "www.bb.com/cc", "bb.com/cc", "bb.com/cc/dd"} {
		u, e := url.Parse("http://" + d)
		if e != nil {
			t.Error(e)
		}
		if !site.DomainMatch(u) {
			t.Errorf("%v: %#v", u, site.Referers)
		}
	}

	for _, d := range []string{"aa.com", "aa.com/cc", "aa.com/abc", "www.bb.com/cd1", "bb.com/dc1"} {
		u, e := url.Parse("http://" + d)
		if e != nil {
			t.Error(e)
		}
		if site.DomainMatch(u) {
			t.Errorf("%v: %#v", u, site.Referers)
		}
	}
}

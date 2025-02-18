package match

import (
	"database/sql"
	"net/url"
	"strconv"
	"testing"

	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dmp"
	ipsearch "github.com/genelet/winter/maxmind"
	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/uadevice"
	_ "github.com/go-sql-driver/mysql"
)

func GetDmpSample() *dmp.Dmp {
	return &dmp.Dmp{
		Sex:       1,
		Byear:     2,
		Bmonth:    3,
		Horoscope: 4,
		Zodiac:    5,
		Vip:       1,
		Browser:   1,
		Gps:       1,
		Bplace:    9,
		Living:    10,
		Brand:     11,
		Screen:    2,
		Time:      13,
		Price:     4, Plan: 5,
		Group:    6,
		Hold:     1,
		Games:    []uint32{1, 2},
		Shops:    []uint32{2, 3},
		Finances: []uint32{4, 5},
		Grocerys: []uint32{6, 7},
		Medias:   []uint32{8, 9},
		Healths:  []uint32{1, 2},
		Learns:   []uint32{3, 4},
		Travels:  []uint32{1, 3},
		Socials:  []uint32{2, 4},
		Cars:     []uint32{1},
		Foods:    []uint32{2},
		Photos:   []uint32{1, 4},
		Works:    []uint32{2, 3},
		Books:    []uint32{3},
		Reports:  []uint32{4},
		Others:   []uint32{1},
	}
}

func GetAudienceSample() *Audience {
	audience := &Audience{
		DmpAudience: dmp.DmpAudience{
			Sex:        1,
			Byears:     []uint32{2, 1},
			Bmonths:    []uint32{3, 1},
			Horoscopes: []uint32{4, 1},
			Zodiacs:    []uint32{5, 1},
			Vip:        1,
			Browser:    1,
			Gps:        1,
			Bplaces:    []uint32{9},
			Livings:    []uint32{10},
			Brands:     []uint32{11, 1},
			Screens:    []uint32{2, 1},
			Times:      []uint32{3, 1},
			Prices:     []uint32{4, 1},
			Plans:      []uint32{5, 1},
			Groups:     []uint32{6, 1},
			Holds:      []uint32{1, 2},
			Games:      []uint32{1, 2},
			Shops:      []uint32{2, 3},
			Finances:   []uint32{4, 5},
			Grocerys:   []uint32{6, 7},
			Medias:     []uint32{8, 9},
			Healths:    []uint32{1, 2},
			Learns:     []uint32{3, 4},
			Travels:    []uint32{1, 3},
			Socials:    []uint32{2, 4},
			Cars:       []uint32{1},
			Foods:      []uint32{2},
			Photos:     []uint32{1, 4},
			Works:      []uint32{2, 3},
			Books:      []uint32{3},
			Reports:    []uint32{4},
			Others:     []uint32{1},
		},
		GeoAudience: ipsearch.GeoAudience{
			GeoStates: []uint32{3, 2},
			GeoDmas:   []uint32{4, 2},
			GeoCitys:  []uint32{6, 77},
			GeoIsps:   []uint32{5, 2},
		},
		DemoAudience: demo.DemoAudience{},
		UaAudience: uadevice.UaAudience{
			UaBVersion:  1,
			UaOVersion:  1,
			UaBrowsers:  []uint32{2, 1},
			UaOSs:       []uint32{3, 1},
			UaPlatforms: []uint32{4, 1},
			UaDevices:   []uint32{5, 1},
		},
		WeekDays:  1111,
		WeekHours: 2222,
	}
	return audience
}

func TestAudience(t *testing.T) {
	dmmp := GetDmpSample()
	aud := GetAudienceSample()
	if aud.Sex != 1 ||
		aud.Bplaces[0] != 9 ||
		aud.Brands[0] != 11 ||
		aud.Holds[0] != 1 ||
		aud.Games[0] != 1 ||
		aud.Medias[0] != 8 ||
		aud.Medias[1] != 9 ||
		aud.Healths[0] != 1 ||
		aud.Healths[1] != 2 ||
		aud.Learns[0] != 3 ||
		aud.Learns[1] != 4 ||
		aud.Reports[0] != 4 ||
		aud.Others[0] != 1 {
		t.Errorf("%v", aud)
	}

	bs, err := aud.Pack()
	if err != nil {
		t.Fatal(err)
	}
	//t.Errorf("%d", len(bs))
	unp, err := UnpackAudience(bs)
	if err != nil {
		t.Fatal(err)
	}
	if unp.Sex != 1 ||
		unp.Bplaces[0] != 9 ||
		unp.Brands[0] != 11 ||
		unp.Holds[0] != 1 ||
		unp.Games[0] != 1 ||
		unp.Medias[0] != 8 ||
		unp.Medias[1] != 9 ||
		unp.Healths[0] != 1 ||
		unp.Healths[1] != 2 ||
		unp.Learns[0] != 3 ||
		unp.Learns[1] != 4 ||
		unp.Reports[0] != 4 ||
		unp.Others[0] != 1 {
		t.Errorf("%v\n%v", aud, unp)
	}

	if !aud.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", aud, dmmp)
	}

	unp.Others[0] = 5
	if unp.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", unp, dmmp)
	}
	unp.Others[0] = 1
	unp.Reports[0] = 5
	if unp.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", unp, dmmp)
	}
	unp.Reports[0] = 5
	unp.Learns[0] = 4
	if unp.MatchDmp(dmmp) {
		t.Errorf("%v\n%v", unp, dmmp)
	}

	ARGS := url.Values{}
	campaignid := uint32(25)
	ARGS.Set("campaign_id", strconv.FormatUint(uint64(campaignid), 10))
	aud.ToArgs(ARGS)
	if ARGS.Get("Sex") != "1" ||
		ARGS.Get("Bplace") != "9" ||
		ARGS.Get("Brand") != "11" ||
		ARGS.Get("Hold") != "1" ||
		ARGS.Get("Game") != "1" ||
		ARGS.Get("Media") != "8" ||
		ARGS.Get("Health") != "1" ||
		ARGS.Get("Learn") != "3" ||
		ARGS.Get("Report") != "4" ||
		ARGS.Get("Other") != "1" {
		t.Errorf("%v\n%v", aud, ARGS)
	}

	aud0 := AudienceFromArgs(ARGS)
	if aud0.Sex != 1 ||
		aud0.Bplaces[0] != 9 ||
		aud0.Brands[0] != 11 ||
		aud0.Holds[0] != 1 ||
		aud0.Games[0] != 1 ||
		aud0.Medias[0] != 8 ||
		aud0.Medias[1] != 9 ||
		aud0.Healths[0] != 1 ||
		aud0.Healths[1] != 2 ||
		aud0.Learns[0] != 3 ||
		aud0.Learns[1] != 4 ||
		aud0.Reports[0] != 4 ||
		aud0.Others[0] != 1 {
		t.Errorf("%v\n%v", ARGS, aud0)
	}

	c := pzutil.NewConfig("../conf/pzadx.conf")
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = DBInsertAudience(db, ARGS)
	if err != nil {
		t.Errorf("%v", err)
	}

	unp, err = DBGetAudience(db, campaignid)
	if err != nil {
		t.Errorf("%v", err)
	}
	if unp.Sex != 1 ||
		unp.Bplaces[0] != 9 ||
		unp.Brands[0] != 11 ||
		unp.Holds[0] != 1 ||
		unp.Games[0] != 1 ||
		unp.Medias[0] != 8 ||
		unp.Medias[1] != 9 ||
		unp.Healths[0] != 1 ||
		unp.Healths[1] != 2 ||
		unp.Learns[0] != 3 ||
		unp.Learns[1] != 4 ||
		unp.Reports[0] != 4 ||
		unp.Others[0] != 1 {
		t.Errorf("%v\n%v", aud, unp)
	}
}

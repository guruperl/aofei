package holiday

import (
	"fmt"
	"testing"
	"time"

	ipsearch "github.com/genelet/winter/maxmind"
	adcom1 "github.com/mxmCherry/openrtb/adcom1"
)

func TestUser(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil {
		t.Fatal(err)
	}

	user := new(User)
	user.Gender = "M"
	user.YOB = 1990
	user.Data = []adcom1.Data{
		{ID: "telecom",
			Segment: []adcom1.Segment{
				{Value: "xxxx,yyyy,0102,0100,0201,0601,0611,0900,0901,0902"},
			},
		},
		{ID: "aofei",
			Segment: []adcom1.Segment{
				{Value: "xxxx,yyyy,06,0701,001001001,130200,130102,140105"},
			},
		},
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	dev := &Device{UA: ua}

	p, err := ipsearch.LoadIpData("../../conf/qq-pz.dat")
	if err != nil {
		t.Fatal(err)
	}
	ip := "210.51.200.123"
	geo := &Geo{Ips: p, IPString: ip}

	telecomMap, err := NewTagMapTelecom(c.Telecom)
	if err != nil {
		t.Fatal(err)
	}
	aofeiMap, err := NewTagMapAofei(c.Aofei)
	if err != nil {
		t.Fatal(err)
	}
	refMap := map[string]*TagMap{"telecom": telecomMap, "aofei": aofeiMap}

	user.UpdateTags(dev, geo, time.Now(), refMap)
	/*
		t.Errorf("%v", user.Tags["telecom"].TagHashArray)
		t.Errorf("%v", user.Tags["aofei"].TagHashArray)
		t.Errorf("%v", user.Tags["device"].TagHashArray)
		t.Errorf("%v", user.Tags["geo"].TagHashArray)
		t.Errorf("%v", user.Tags["demo"].TagHashArray)
		t.Errorf("%v", user.Tags["dayhour"].TagHashArray)

		map[1:[0 2] 2:[1] 6:[265 521 262 6 4358 9]]
		map[1401:[0] 1403:[130200 130000 130102 130100 140105 140100 140000] 1413:[1393 1392 293]]
		map[1201:[1] 1202:[59] 1203:[2] 1204:[10] 1205:[1] 1206:[1]]
		map[1101:[3] 1102:[48] 1103:[620] 1104:[0] 1105:[141] 1106:[420100] 1107:[6]]
		map[1301:[1] 1302:[53]]
		map[1001:[18268] 1002:[438449] 1003:[2] 1004:[9]]

		t.Errorf("%v", user.MergeTags())
		&{map[1:[2 0] 2:[1] 6:[265 521 262 6 4358 9] 1001:[18268] 1002:[438449] 1003:[2] 1004:[9] 1101:[3] 1102:[48] 1103:[620] 1104:[0] 1105:[141] 1106:[420100] 1107:[6] 1201:[1] 1202:[59] 1203:[2] 1204:[10] 1205:[1] 1206:[1] 1301:[1] 1302:[53] 1401:[0] 1403:[130100 140105 140100 140000 130200 130000 130102] 1413:[293 1393 1392]]}
	*/

	for k, v := range user.MergeTags().TagHashArray {
		if IndexUint32([]uint32{1, 2, 6, 1001, 1002, 1003, 1004, 1101, 1102, 1103, 1104, 1105, 1106, 1107, 1201, 1202, 1203, 1204, 1205, 1206, 1301, 1302, 1401, 1403, 1413}, k) < 0 {
			t.Errorf("%d=>%v", k, v)
		}
	}

	top10 := user.Top10Tags()
	_, ok0 := top10["tag0"]
	_, ok9 := top10["tag9"]
	if !ok0 || !ok9 {
		for k := 0; k < 9; k++ {
			v := top10[fmt.Sprintf("tag%d", k)].(uint32)
			s1, s2 := GetSizes(v)
			t.Errorf("%d=>%d", s1, s2)
		}
	}
}

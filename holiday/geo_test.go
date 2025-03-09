package holiday

import (
	"testing"

	ipsearch "github.com/genelet/winter/maxmind"
)

func TestGeo(t *testing.T) {
	p, err := ipsearch.LoadIpData("../../conf/qq-pz.dat")
	if err != nil {
		t.Fatal(err)
	}
	ip := "210.51.200.123"
	attrs_obj := NewAttrsFromNames([]string{"continent", "country", "state", "city", "dma", "zip", "isp", "bandwidth", "areacode"})
	geo := &Geo{Ips: p, IPString: ip}
	tags := geo.GetTags()
	for k, item := range tags.TagHashArray {
		if k == 1101 && item[0] != 3 {
			// 1105=>&{武汉  dma 141}
			// 1104=>&{  city 0}
			// 1106=>&{420100  zip 420100}
			// 1107=>&{联通  isp 6}
			// 1101=>&{亚洲  continent 3}
			// 1102=>&{中国  country 48}
			// 1103=>&{湖北  state 620}
			t.Errorf("%d=>%v", k, item[0])
		}
	}

	lists := []map[string]interface{}{
		{"attrname_id": int32(1101), "value_id": int32(3)},
		{"attrname_id": int32(1102), "value_id": int32(48)},
		{"attrname_id": int32(1103), "value_id": int32(620)},
		{"attrname_id": int32(1105), "value_id": int32(141)},
		{"attrname_id": int32(1106), "value_id": int32(420100)},
		{"attrname_id": int32(1107), "value_id": int32(6)},
	}
	audiences := GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
	if len(audiences.Audiences) != 6 {
		t.Errorf("%v", audiences)
	}
	if audiences.Match(tags) == false {
		t.Errorf("%v", audiences.Match(tags))
	}

	for i := 0; i < 6; i++ {
		lists = []map[string]interface{}{
			{"attrname_id": int32(1101), "value_id": int32(3)},
			{"attrname_id": int32(1102), "value_id": int32(48)},
			{"attrname_id": int32(1103), "value_id": int32(620)},
			{"attrname_id": int32(1105), "value_id": int32(141)},
			{"attrname_id": int32(1106), "value_id": int32(420100)},
			{"attrname_id": int32(1107), "value_id": int32(6)},
		}
		lists[i]["value_id"] = 1 + lists[i]["value_id"].(int32) // not match
		audiences = GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
		if len(audiences.Audiences) != 6 {
			t.Errorf("%v", audiences)
		}
		if audiences.Match(tags) == true {
			t.Errorf("%v", audiences.Match(tags))
		}
	}
}

package holiday

import (
	"testing"
)

func TestDevice(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36"
	attrs_obj := NewAttrsFromNames([]string{"pzua", "browser", "bversion", "os", "oversion", "platform", "device"})
	dev := &Device{UA:ua}

	tags := dev.GetTags()
	for k, item := range tags.TagHashArray {
		if k==1201 && item[0] != 1 {
//1202=>&{59  bversion 59}
//1203=>&{OSWindows  os 2}
//1204=>&{10  oversion 10}
//1205=>&{PlatformWindows  platform 1}
//1206=>&{DeviceComputer  device 1}
//1201=>&{BrowserChrome  browser 1}
			t.Errorf("%d=>%v", k, item[0])
		}
	}

    lists := []map[string]interface{}{
{"attrname_id":int32(1201), "value_id":int32(1)},
{"attrname_id":int32(1202), "value_id":int32(59)},
{"attrname_id":int32(1203), "value_id":int32(2)},
{"attrname_id":int32(1204), "value_id":int32(10)},
{"attrname_id":int32(1205), "value_id":int32(1)},
{"attrname_id":int32(1206), "value_id":int32(1)},
}
    audiences := GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
    if len(audiences.Audiences) != 6 {
       t.Errorf("%v", audiences)
    }
    if audiences.Match(tags) == false {
       t.Errorf("%v", audiences.Match(tags))
    }

	for i:=0; i<6; i++ {
	    lists = []map[string]interface{}{
{"attrname_id":int32(1201), "value_id":int32(1)},
{"attrname_id":int32(1202), "value_id":int32(59)},
{"attrname_id":int32(1203), "value_id":int32(2)},
{"attrname_id":int32(1204), "value_id":int32(10)},
{"attrname_id":int32(1205), "value_id":int32(1)},
{"attrname_id":int32(1206), "value_id":int32(1)},
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

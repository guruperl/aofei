package holiday

import (
	"testing"
	"time"
)

func TestDayhour(t *testing.T) {
	dh := &Dayhour{time.Now()}
	tags := dh.GetTags()

	attrs_obj := NewAttrsFromNames([]string{"fullday", "fullhour", "weekday", "weekhour"})

    lists := []map[string]interface{}{
{"attrname_id":int32(1001), "value_id":int32(tags.TagHashArray[1001][0])},
{"attrname_id":int32(1002), "value_id":int32(tags.TagHashArray[1002][0])},
{"attrname_id":int32(1003), "value_id":int32(tags.TagHashArray[1003][0])},
{"attrname_id":int32(1004), "value_id":int32(tags.TagHashArray[1004][0])},
}
    audiences := GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
    if len(audiences.Audiences) != 4 {
       t.Errorf("%v", audiences)
    }
    if audiences.Match(tags) == false {
       t.Errorf("%v", audiences.Match(tags))
    }

	for i:=0; i<4; i++ {
	    lists = []map[string]interface{}{
{"attrname_id":int32(1001), "value_id":int32(tags.TagHashArray[1001][0])},
{"attrname_id":int32(1002), "value_id":int32(tags.TagHashArray[1002][0])},
{"attrname_id":int32(1003), "value_id":int32(tags.TagHashArray[1003][0])},
{"attrname_id":int32(1004), "value_id":int32(tags.TagHashArray[1004][0])},
}
		lists[i]["value_id"] = 1 + lists[i]["value_id"].(int32) // not match
	    audiences = GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
		if len(audiences.Audiences) != 4 {
			t.Errorf("%v", audiences)
		}
		if audiences.Match(tags) == true {
			t.Errorf("%v", audiences.Match(tags))
		}
	}
}

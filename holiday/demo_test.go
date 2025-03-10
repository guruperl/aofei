package holiday

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestDemo(t *testing.T) {
	user := &openrtb2.User{Yob: 1973, Gender: "M"}
	tags := getDemoTags(user)

	attrs_obj := NewAttrsFromNames([]string{"gender", "yob", "married", "income", "child", "household", "ethnicity", "education", "occupation"})

	lists := []map[string]interface{}{
		{"attrname_id": int32(1301), "value_id": int32(1)},
		{"attrname_id": int32(1302), "value_id": int32(36)},
	}
	audiences := GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
	if len(audiences.Audiences) != 2 {
		t.Errorf("%v", audiences)
	}
	if audiences.Match(tags) == false {
		t.Errorf("%v", audiences.Match(tags))
	}

	for i := 0; i < 2; i++ {
		lists = []map[string]interface{}{
			{"attrname_id": int32(1301), "value_id": int32(1)},
			{"attrname_id": int32(1302), "value_id": int32(36)},
		}
		lists[i]["value_id"] = 1 + lists[i]["value_id"].(int32) // not match
		audiences = GetTagAudiencesFromTao(attrs_obj.GetAttrIDs(), lists)
		if len(audiences.Audiences) != 2 {
			t.Errorf("%v", audiences)
		}
		if audiences.Match(tags) == true {
			t.Errorf("%v", audiences.Match(tags))
		}
	}
}

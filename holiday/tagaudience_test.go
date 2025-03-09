package holiday

import (
	"testing"
)

func TestAofeiTagAudiences(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil { t.Fatal(err) }
	tagMap, err := NewTagMapAofei(c.Aofei)
	if err != nil { t.Fatal(err) }
	str := "xxxx,yyyy,06,0701,001001001,130200,130102,140105"
	tags_obj := tagMap.GetTagsFromCodes(str)

	// all targetings with one tag in the above tags should match
	for _, value_id := range []uint32{140105,140100,140000,130200,130000,130102,130100} {
		lists := []map[string]interface{}{
{"attrname_id":int32(1403), "value_id":int32(value_id)},
{"attrname_id":int32(1403), "value_id":int32(130999)},
{"attrname_id":int32(9999), "value_id":int32(140105)},
}
		audiences := GetTagAudiencesFromTao(tagMap.GetAttrIDs(), lists)
		if len(audiences.Audiences)>1 { // only 1403! 9999 is not Aofei's attr
			t.Errorf("%v", audiences)
		}
		items := audiences.Audiences[1403]
		if len(items) != 2 || items[0] != value_id || items[1] != 130999 {
			t.Errorf("%v", items)
		}
		if audiences.Match(tags_obj) == false {
			t.Errorf("%v", audiences.Match(tags_obj))
		}
	}

	// failed matches
	for _, value_id := range []uint32{149105,149100,149000,139200,139000,139102,139100} {
		lists := []map[string]interface{}{
{"attrname_id":int32(1403), "value_id":int32(value_id)},
{"attrname_id":int32(1403), "value_id":int32(130999)},
{"attrname_id":int32(9999), "value_id":int32(140105)},
}
		audiences := GetTagAudiencesFromTao(tagMap.GetAttrIDs(),lists)
		if len(audiences.Audiences)>1 { // only 1403! 9999 is not Aofei's attr
			t.Errorf("%v", audiences)
		}
		items := audiences.Audiences[1403]
		if len(items) != 2 || items[0] != value_id || items[1] != 130999 {
			t.Errorf("%v", items)
		}
		if audiences.Match(tags_obj) {
			t.Errorf("%v", audiences.Match(tags_obj))
		}
	}
}

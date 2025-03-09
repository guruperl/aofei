package holiday

import (
//	"log"
)

type TagAudiences struct {
	Audiences map[uint32][]uint32 `json:"audiences"`
}

// see summer/targetname/component.json
func GetTagAudiencesFromTao(attrIDs []uint32, lists []map[string]interface{}) *TagAudiences {
	auds := make(map[uint32][]uint32)
	n := 0
	for _, item := range lists {
		attrID := uint32(item["attrname_id"].(int32))
		// only targetings belong to the provider's lists are compared
		if attrIDs != nil && IndexUint32(attrIDs, attrID) < 0 { continue }
		if item["value_id"]==nil { continue }
		value_id := uint32(item["value_id"].(int32))
		// value_id =0 means no targeting ?
		// if value_id==0 { continue } ?
		if _, ok := auds[attrID]; !ok {
			auds[attrID] = make([]uint32, 0)
			n++
		}
		auds[attrID] = append(auds[attrID], value_id)
	}
	if n==0 { return nil }
	return &TagAudiences{auds}
}

func (self *TagAudiences) Match(tags *Tags) bool {
    if tags==nil { return false }
    for attrID, audience := range self.Audiences {
        tag, ok := tags.TagHashArray[attrID]
        if !ok { return false }
        if GrepOrN(tag, audience)==false { return false }
    }
    return true
}

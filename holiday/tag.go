package holiday

import (
//"log"
	"strings"
)

type Tag struct {
    ID      string `json:"code"`	// unique ID string
    Name    string `json:"name,omitignore"`	// any name eg UTF8, doesn't matter
    Parent  string `json:"p_code"`	// ID's parent ID
    Val     uint32 `json:"val"`     // local ID's value under the parent
}

type TagMap struct {
	Attrs                    `json:"attrs"`
	TagRef   map[string]*Tag `json:"tag_ref"`
}

// Tags has key in attrID and value the Val of tags
type Tags struct {
	TagHashArray  map[uint32][]uint32 `json:"tags"`
}

// GetTags generates a new Tags, using a comma separated string
func (self *TagMap) GetTagsFromCodes(str string) *Tags {
	if str == "" { return nil }
    codes := strings.SplitN(str, ",", -1)

	ref  := make(map[uint32]map[string]*Tag)

	n:= 0
    for _, code := range codes { // check each input tag
        tags := self.allParents(code)
        if tags==nil || len(tags)==0 { continue }
		n++
        category := tags[len(tags)-1]
		attrID := self.GetAttrID(category.Parent)
		if _, ok := ref[attrID]; !ok { ref[attrID] = make(map[string]*Tag) }
		for _, tag := range tags {
			ref[attrID][tag.ID] = tag
		}
    }
	if n==0 { return nil }

    hash := make(map[uint32][]uint32)
	for attrID, tags := range ref {
		if _, ok := hash[attrID]; !ok { hash[attrID] = make([]uint32, 0) }
		for _, tag := range tags {
			hash[attrID] = append(hash[attrID], tag.Val)
		}
	}
    return &Tags{hash}
}

func (self *TagMap) allParents(code string) []*Tag {
    tags := make([]*Tag, 0)
    current, ok := self.TagRef[code]
    if !ok { return nil }
    tags = append(tags, current)
	k := 0
    for {
		if k > 5 { break } // 5 level of tree is limited
        if new_current, ok := self.TagRef[current.Parent]; ok {
            if new_current.ID==current.ID {
                break
            }
            tags = append(tags, new_current)
            current = new_current
        } else {
            break
        }
		k++
    }
    return tags
}

package holiday

type Tag struct {
	ID     string `json:"code"`            // unique ID string
	Name   string `json:"name,omitignore"` // any name eg UTF8, doesn't matter
	Parent *Tag   `json:"parent"`          // ID's parent ID
	Val    uint32 `json:"val"`             // local ID's value under the parent
}

func (self *Tag) Parents(n int) []*Tag {
	if n == 0 {
		return nil
	}
	parent := self.Parent
	if parent == nil {
		return nil
	}
	return append(parent.Parents(n-1), parent)
}

type TagMap map[string]*Tag

/*

func NewTagHashArray(codes []string) TagHashArray {
	ref := make(map[uint32]map[string]*Tag)

	n := 0
	for _, code := range codes { // check each input tag
		tags := self.allParents(code)
		if tags == nil || len(tags) == 0 {
			continue
		}
		n++
		category := tags[len(tags)-1]
		attrID := self.GetAttrID(category.Parent)
		if _, ok := ref[attrID]; !ok {
			ref[attrID] = make(map[string]*Tag)
		}
		for _, tag := range tags {
			ref[attrID][tag.ID] = tag
		}
	}
	if n == 0 {
		return nil
	}

	hash := make(map[uint32][]uint32)
	for attrID, tags := range ref {
		if _, ok := hash[attrID]; !ok {
			hash[attrID] = make([]uint32, 0)
		}
		for _, tag := range tags {
			hash[attrID] = append(hash[attrID], tag.Val)
		}
	}
	return &Tags{hash}
}
*/

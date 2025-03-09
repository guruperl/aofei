package holiday

import (
	"io/ioutil"
	"encoding/json"
	"strings"
)

func NewTagMapAofei(filename string) (*TagMap, error) {
	if filename=="" { return nil, nil }
    content, err := ioutil.ReadFile(filename)
    if err != nil { return nil, err }
    parsed := make([]*Tag, 0)
    err = json.Unmarshal(content, &parsed)
    if err != nil { return nil, err }

	Marker := "001"

	TagRef := map[string]*Tag{}
	attrs := make([]uint32,0)
	for _, tag := range parsed {
		tag.Name = strings.TrimSpace(tag.Name)
		if tag.Parent==Marker {
			tag.Val += 1400
			attrs = append(attrs, tag.Val)
			continue // top attrs should NOT in reference
		}
		TagRef[tag.ID] = tag
	}
	return &TagMap{Attrs{attrs}, TagRef}, nil
}

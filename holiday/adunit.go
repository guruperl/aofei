package holiday

import (
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type Adunit struct {
	RPub
	SizeID       uint32
	ConfigId     string           `json:"config_id"`
	Code         string           `json:"code"`
	Floor        float32          `json:"flr,omitempty"` // incoming if not exists
	Banner       *openrtb2.Banner `json:"banner,omitempty"`
	Video        *openrtb2.Video  `json:"video,omitempty"`
	ParsedNative *NativeType      `json:"parsed_native,omitempty"`
}

func (self *Adunit) SetSizeID() {
	var w, h uint16
	if banner := self.Banner; banner != nil {
		w = uint16(*banner.W)
		h = uint16(*banner.H)
	} else if native := self.ParsedNative; native != nil {
		w = uint16(64)
		h = uint16(64)
		if native.Assets != nil && len(native.Assets) > 0 && native.Assets[0].Image != nil {
			w, h = native.Assets[0].Image.Sizes()
		}
	} else if self.Video != nil {
		w = uint16(*self.Video.W)
		h = uint16(*self.Video.H)
	}
	self.SizeID = getSizeID(w, h)
}

func (self *Adunit) GetItemsFromTao(when time.Time, lists []map[string]interface{}) []*Item {
	if lists == nil || len(lists) == 0 {
		return nil
	}
	items := make([]*Item, 0)
	for _, hash := range lists {
		item := itemFromTao(hash)
		if self.Floor > 0.0 && self.Floor > item.Eprice() {
			continue
		}
		if hash["endx"] != nil && uint32(hash["endx"].(int32)) < uint32(when.Unix()) {
			continue
		}
		items = append(items, item)
	}
	return items
}

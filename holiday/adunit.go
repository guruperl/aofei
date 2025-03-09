package holiday

import (
//"log"
//	"encoding/json"
	"time"
)

type RPub struct {
    PubId    uint32
    SiteId   uint32
    SlotId   uint32
}

func (self RPub) ToArgs() map[string]interface{} {
	return map[string]interface{}{
		"pub_id" : self.PubId,
		"site_id": self.SiteId,
		"slot_id": self.SlotId,
	}
}

type Adunit struct {
	RPub
	SizeId	uint32
	ConfigId	string	`json:"config_id"`
	Code	string		`json:"code"`
	Floor   float32     `json:"flr,omitempty"`	// incoming if not exists
	Banner	*BannerType	`json:"banner,omitempty"`
	Video	*VideoType	`json:"video,omitempty"`
	Native	*NativeType	`json:"native,omitempty"`
}

type BannerType struct {
	Size []uint16		`json:"size"`
}

type VideoType struct {
	Context string		`json:"context,omitempty"`
	PlayerSize []uint16	`json:"playerSize,omitempty"`
	Mimes []string		`json:"mimes,omitempty"`
}

type NativeType struct {
	Image   []uint16	`json:"image,omitempty"`
	Icon    []uint16	`json:"icon,omitempty"`
	Title   bool		`json:"title,omitempty"`
	SponsoredBy bool	`json:"sponsoredBy,omitempty"`
	Body    bool		`json:"body,omitempty"`
}

func (self *Adunit)MatchMime(mime8 uint32) bool {
	// build adslot's mime set
	mime := uint32(0)
	switch {
	case self.Native != nil:
		mime = 1<<MimeUnknown + 1<<JsonMime + 1<<H5Mime + 0<<JSMime +
			1<<ImageMime + 1<<VideoMime
	case self.Banner != nil:
		mime = 1<<MimeUnknown + 0<<JsonMime + 0<<H5Mime + 0<<JSMime +
			1<<ImageMime + 0<<VideoMime
    case self.Video != nil:
		mime = 0<<MimeUnknown + 0<<JsonMime + 0<<H5Mime + 0<<JSMime +
			0<<ImageMime + 1<<VideoMime
	default:
	}
	// in mysql (from summer), the index 0 is researved for empty string
	// so the real string starts at 1
	return 1 == (mime >> (mime8-1)) & 1
}

func (self *Adunit)NativeSizeId() uint32 {
    sizes := []uint16{0,0}
    if self.Banner!=nil {
        sizes = self.Banner.Size
    } else if self.Native!=nil {
        sizes = self.Native.Image
    } else if self.Video!=nil {
        sizes = self.Video.PlayerSize
    }
	size_id := GetSizeId(sizes[0], sizes[1])
	if size_id == 0 { return self.SizeId }
    return size_id
}

func (self *Adunit) GetItemsFromTao(when time.Time, lists []map[string]interface{}) []*Item {
    if lists==nil || len(lists)==0 { return nil }
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

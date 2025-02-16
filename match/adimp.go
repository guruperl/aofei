package match

import "github.com/genelet/winter/pzutil"

type BannerType struct {
	Size []uint16
}

type VideoType struct {
	Context    string
	PlayerSize []uint16
}

type NativeImg struct {
	Type int    `json:"type"`
	WMin uint16 `json:"wmin"`
	HMin uint16 `json:"hmin"`
}

type AssetType struct {
	ID       int                    `json:"id"`
	Required int                    `json:"required"`
	Title    map[string]interface{} `json:"title"`
	Image    *NativeImg             `json:"img"`
}

type NativeNative struct {
	Ver    string       `json:"ver"`
	Assets []*AssetType `json:"assets"`
}

type NativeType struct {
	Request     string
	Ver         string
	Native      *NativeNative
	Image       []uint16
	Icon        []uint16
	Title       bool
	SponsoredBy bool
	Body        bool
}

type AdImp struct {
	RPub
	Native *NativeType
	Banner *BannerType
	Video  *VideoType
}

func (self *AdImp) MatchMime(m8 uint8) bool {
	switch m8 {
	case uint8(0), uint8(1):
		return self.Native != nil
	case uint8(2):
		return self.Banner != nil
	case uint8(3):
		return self.Video != nil
	default:
	}
	return false
}

func (self *AdImp) GetSizeID() uint32 {
	var sizes []uint16
	if self.Banner != nil {
		sizes = self.Banner.Size
	} else if self.Native != nil {
		sizes = self.Native.Image
	} else if self.Video != nil {
		sizes = self.Video.PlayerSize
	} else {
		sizes = []uint16{0, 0}
	}
	if sizes == nil {
		sizes = []uint16{0, 0}
	}
	return pzutil.GetSizeID(sizes[0], sizes[1])
}

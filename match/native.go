package match

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type AssetImg struct {
	Type  int                    `json:"type,omitempty"`
	URL   string                 `json:"url,omitempty"`
	W     uint16                 `json:"w,omitempty"`
	H     uint16                 `json:"h,omitempty"`
	WMin  uint16                 `json:"wmin,omitempty"`
	HMin  uint16                 `json:"hmin,omitempty"`
	Mimes []string               `json:"mimes,omitempty"`
	Ext   map[string]interface{} `json:"ext,omitempty"`
}

type AssetType struct {
	ID       int                    `json:"id"`
	Required int                    `json:"required,omitempty"`
	Title    *adcom1.TitleAsset     `json:"title,omitempty"`
	Img      *AssetImg              `json:"img,omitempty"`
	Image    *adcom1.ImageAsset     `json:"image,omitempty"`
	Video    *adcom1.VideoAsset     `json:"video,omitempty"`
	Data     *adcom1.DataAsset      `json:"data,omitempty"`
	Ext      map[string]interface{} `json:"ext,omitempty"`
}

type NativeLink struct {
	URL           string   `json:"url,omitempty"`
	Clicktrackers []string `json:"clicktrackers,omitempty"`
	Fallback      string   `json:"fallback,omitempty"`
}

type NativeType struct {
	Ver         string                 `json:"ver"`
	Assets      []*AssetType           `json:"assets"`
	Link        *NativeLink            `json:"link,omitempty"`
	ImpTrackers []string               `json:"imptrackers,omitempty"`
	Ext         map[string]interface{} `json:"ext,omitempty"`
}

// GetSizes return the width and height of the native type
func (self *NativeType) GetSizes() (uint16, uint16) {
	for _, asset := range self.Assets {
		if img := asset.Img; img != nil {
			if img.W > 0 && img.H > 0 {
				return img.W, img.H
			}
			return img.WMin, img.HMin
		} else if asset.Image != nil {
			return uint16(asset.Image.W), uint16(asset.Image.H)
		}
	}
	return 0, 0
}

func NewNativeType(native *openrtb2.Native) (*NativeType, error) {
	if native != nil && native.Request != "" {
		return requestStringToNativeType([]byte(native.Request))
	}
	return nil, nil
}

func requestStringToNativeType(bs []byte) (*NativeType, error) {
	x := map[string]*NativeType{}
	if err := json.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	return x["native"], nil
}

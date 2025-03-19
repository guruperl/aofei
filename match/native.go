package match

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// AssetFormat is the struct for the asset format
// note that adcom1 has no required field
type AssetFormat struct {
	adcom1.AssetFormat
	Required int `json:"required,omitempty"`
}

// NativeFormat is the struct for the native format
// note that adcom1 has no required ver
type NativeFormat struct {
	Ver    string                 `json:"ver"`
	Assets []*AssetFormat         `json:"assets"`
	Ext    map[string]interface{} `json:"ext,omitempty"`
}

// GetSizes return the width and height of the native format
func (self *NativeFormat) GetSizes() (uint16, uint16) {
	for _, asset := range self.Assets {
		if img := asset.Img; img != nil {
			if img.W > 0 && img.H > 0 {
				return uint16(img.W), uint16(img.H)
			}
			return uint16(img.WMin), uint16(img.HMin)
		} else if asset.Video != nil {
			return uint16(asset.Video.W), uint16(asset.Video.H)
		}
	}
	return 0, 0
}

func requestStringToNativeFormat(bs []byte) (*NativeFormat, error) {
	x := map[string]*NativeFormat{}
	if err := json.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	return x["native"], nil
}

// NewNativeFormat creates a new NativeFormat from penrtb2 native
func NewNativeFormat(native *openrtb2.Native) (*NativeFormat, error) {
	if native != nil && native.Request != "" {
		return requestStringToNativeFormat([]byte(native.Request))
	}
	return nil, nil
}

// Asset is the simplest struct for the asset
// note that adcom1 has no required Img nor Required
type Asset struct {
	ID       int8               `json:"id"`
	Required int8               `json:"required,omitempty"`
	Title    *adcom1.TitleAsset `json:"title,omitempty"`
	Img      *adcom1.ImageAsset `json:"img,omitempty"`
	Video    *adcom1.VideoAsset `json:"video,omitempty"`
}

// LinkAsset is the struct for the link asset
// note that adcom1 Link is very different
type LinkAsset struct {
	URL           string   `json:"url,omitempty"`
	Clicktrackers []string `json:"clicktrackers,omitempty"`
	Fallback      string   `json:"fallback,omitempty"`
}

// Native is the struct for the native
// note that adcom1 has no ImpTrackers nor Ver, and Link is different
type Native struct {
	Ver         string                 `json:"ver"`
	Assets      []Asset                `json:"assets"`
	Link        *LinkAsset             `json:"link,omitempty"`
	ImpTrackers []string               `json:"imptrackers,omitempty"`
	Ext         map[string]interface{} `json:"ext,omitempty"`
}

// AdM returns the adm string
func (self *Native) AdM(trackers ...string) (string, error) {
	if len(trackers) > 0 {
		self.ImpTrackers = []string{trackers[0]}
		if len(trackers) > 1 {
			self.Link.Clicktrackers = trackers[1:]
		}
	}
	x := map[string]*Native{"native": self}
	bs, err := json.Marshal(x)
	return string(bs), err
}

func requestStringToNative(bs []byte) (*Native, error) {
	x := map[string]*Native{}
	if err := json.Unmarshal(bs, &x); err != nil {
		return nil, err
	}
	return x["native"], nil
}

// DefaultImgNative creates a default image native
func DefaultImgNative(url string, title string, w, h uint16) *Native {
	return &Native{
		Ver: "1.1",
		Assets: []Asset{
			{
				Required: 1,
				ID:       1,
				Title: &adcom1.TitleAsset{
					Text: title,
				},
			}, {
				Required: 1,
				ID:       2,
				Img: &adcom1.ImageAsset{
					Type: 1,
					URL:  url,
					W:    int64(w),
					H:    int64(h),
				},
			},
		},
	}
}

// DefaultVideoNative creates a default video native from cohntent only
func DefaultVideoNative(adm string) *Native {
	return &Native{
		Ver: "1.1",
		Assets: []Asset{
			{
				Required: 1,
				ID:       1,
				Video: &adcom1.VideoAsset{
					AdM: adm,
				},
			},
		},
	}
}

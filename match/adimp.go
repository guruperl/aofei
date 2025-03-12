package match

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

type AdImp struct {
	RPub
	imp    openrtb2.Imp
	Native *NativeType
	Banner *openrtb2.Banner
	Video  *openrtb2.Video
}

func NewAdImp(bidRequest *openrtb2.BidRequest) ([]*AdImp, error) {
	if bidRequest == nil {
		return nil, nil
	}

	rpub := getRPub(bidRequest)

	var adImp []*AdImp
	var err error
	for _, imp := range bidRequest.Imp {
		item := &AdImp{
			RPub: rpub,
			imp:  imp,
		}
		if imp.Banner != nil {
			item.Banner = imp.Banner
		} else if imp.Video != nil {
			item.Video = imp.Video
		} else if imp.Native != nil {
			native, err := parseNative(imp.Native)
			if err != nil {
				return nil, err
			}
			item.Native = native
		}
		item.RPub.SizeID = item.GetSizeID()
		adImp = append(adImp, item)
	}
	return adImp, err
}

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
	ID       int                `json:"id"`
	Required int                `json:"required"`
	Title    adcom1.TitleAsset  `json:"title"`
	Img      *NativeImg         `json:"img"`
	Image    *adcom1.ImageAsset `json:"image"`
	Video    *adcom1.VideoAsset `json:"video"`
}

type NativeNative struct {
	Ver    string       `json:"ver"`
	Assets []*AssetType `json:"assets"`
}

type NativeType struct {
	Request string
	Ver     string
	Native  *NativeNative
}

func (self *AdImp) GetSizeID() uint32 {
	var sizes []uint16
	if self.Banner != nil && self.Banner.W != nil && self.Banner.H != nil {
		sizes = []uint16{uint16(*self.Banner.W), uint16(*self.Banner.H)}
	} else if self.Native != nil {
		assets := self.Native.Native.Assets
		for _, asset := range assets {
			if asset.Img != nil {
				sizes = []uint16{asset.Img.WMin, asset.Img.HMin}
				break
			} else if asset.Image != nil {
				sizes = []uint16{uint16(asset.Image.W), uint16(asset.Image.H)}
				break
			}
		}
	} else if self.Video != nil && self.Video.W != nil && self.Video.H != nil {
		sizes = []uint16{uint16(*self.Video.W), uint16(*self.Video.H)}
	} else {
		sizes = []uint16{0, 0}
	}
	if sizes == nil {
		sizes = []uint16{0, 0}
	}
	return getSizeID(sizes[0], sizes[1])
}

func encodeDurFloors(floors []openrtb2.DurFloors) string {
	if floors == nil {
		return ""
	}

	var arrs []string
	for _, floor := range floors {
		arrs = append(arrs, fmt.Sprintf("%d/%d/%f", floor.MinDur, floor.MaxDur, floor.BidFloor))
	}
	return strings.Join(arrs, ",")
}

func decodeDurFloors(str string) []openrtb2.DurFloors {
	if str == "" {
		return nil
	}

	var floors []openrtb2.DurFloors
	arrs := strings.Split(str, ",")
	for _, item := range arrs {
		var minDur, maxDur int
		var bidFloor float64
		fmt.Sscanf(item, "%d/%d/%f", &minDur, &maxDur, &bidFloor)
		floors = append(floors, openrtb2.DurFloors{MinDur: int64(minDur), MaxDur: int64(maxDur), BidFloor: bidFloor})
	}
	return floors
}

func parseNative(native *openrtb2.Native) (*NativeType, error) {
	if native == nil {
		return nil, nil
	}
	output := &NativeType{
		Request: native.Request,
		Ver:     native.Ver,
	}
	if native.Request != "" {
		x := make(map[string]*NativeNative)
		if err := json.Unmarshal([]byte(native.Request), &x); err != nil {
			return nil, err
		}
		output.Native = x["native"]
	}

	return output, nil
}

func getSizeID(w, h uint16) uint32 {
	return (uint32(w) << 16) | uint32(h)
}

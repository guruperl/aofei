package dsp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/genelet/winter/match"
	"github.com/genelet/winter/pzutil"

	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

func convertBanner(banner *openrtb2.Banner) *match.BannerType {
	if banner == nil {
		return nil
	}
	w := uint16(*banner.W)
	h := uint16(*banner.H)
	return &match.BannerType{Size: []uint16{w, h}}
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

func convertVideo(video *openrtb2.Video) *match.VideoType {
	if video == nil {
		return nil
	}
	if video.W != nil && video.H != nil {
		return &match.VideoType{
			Context:    encodeDurFloors(video.DurFloors),
			PlayerSize: []uint16{uint16(*video.W), uint16(*video.H)},
		}
	}
	return nil
}

func convertNative(native *openrtb2.Native) (*match.NativeType, error) {
	if native == nil {
		return nil, nil
	}
	output := &match.NativeType{
		Request: native.Request,
		Ver:     native.Ver,
	}
	if native.Request != "" {
		x := new(match.NativeNative)
		if err := json.Unmarshal([]byte(native.Request), x); err != nil {
			return nil, err
		}
		output.Native = x
	}

	return output, nil
}

func convertAdImp(bidRequest *openrtb2.BidRequest) ([]*match.AdImp, error) {
	if bidRequest == nil {
		return nil, nil
	}

	rpub := convertRPub(bidRequest)

	var adImp []*match.AdImp
	var err error
	for _, imp := range bidRequest.Imp {
		if imp.Banner != nil {
			banner := convertBanner(imp.Banner)
			rpub.SizeID = pzutil.GetSizeID(banner.Size[0], banner.Size[1])
			adImp = append(adImp, &match.AdImp{
				RPub:   rpub,
				Banner: banner,
			})
		} else if imp.Video != nil {
			video := convertVideo(imp.Video)
			rpub.SizeID = pzutil.GetSizeID(video.PlayerSize[0], video.PlayerSize[1])
			adImp = append(adImp, &match.AdImp{
				RPub:  rpub,
				Video: video,
			})
		} else if imp.Native != nil {
			native, err := convertNative(imp.Native)
			if err != nil {
				return nil, err
			}
			w := uint16(64)
			h := uint16(64)
			if native.Native.Assets != nil && len(native.Native.Assets) > 0 && native.Native.Assets[0].Image != nil {
				w = uint16(native.Native.Assets[0].Image.WMin)
				h = uint16(native.Native.Assets[0].Image.HMin)
			}
			rpub.SizeID = pzutil.GetSizeID(w, h)
			adImp = append(adImp, &match.AdImp{
				RPub:   rpub,
				Native: native,
			})
		}
	}
	return adImp, err
}

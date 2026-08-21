package match

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
)

const maxSizeDimension = int64(1<<16 - 1)

// SizeID2To1 converts two uint16 values to one uint32 value.
func SizeID2To1(w, h uint16) uint32 {
	return uint32(w)<<16 | uint32(h)
}

// SizeID1To2 converts one uint32 value to two uint16 values.
func SizeID1To2(sizeID uint32) (uint16, uint16) {
	return uint16(sizeID >> 16), uint16(sizeID & 0xffff)
}

// getSizeID returns the size ID from the bid request for banner, video, and native ads.
func getSizeIDNative(bidRequest *openrtb2.BidRequest) (uint32, *NativeFormat, error) {
	return getSizeIDNativeForImp(&bidRequest.Imp[0])
}

// getSizeIDNativeForImp returns the size ID for one impression.
func getSizeIDNativeForImp(imp *openrtb2.Imp) (uint32, *NativeFormat, error) {
	if imp == nil {
		return 0, nil, nil
	}
	if native := imp.Native; native != nil {
		nt, err := NewNativeFormat(native)
		if err != nil {
			return 0, nil, err
		}
		if nt != nil {
			w, h, err := nt.validatedSizes()
			if err != nil {
				return 0, nil, err
			}
			if w != 0 && h != 0 {
				return SizeID2To1(w, h), nt, nil
			}
		}
	}
	if video := imp.Video; video != nil {
		if video.W != nil && video.H != nil {
			w, h, err := validatedSizePair("video", *video.W, *video.H)
			if err != nil {
				return 0, nil, err
			}
			return SizeID2To1(w, h), nil, nil
		}
	}
	if banner := imp.Banner; banner != nil {
		if banner.W != nil && banner.H != nil {
			w, h, err := validatedSizePair("banner", *banner.W, *banner.H)
			if err != nil {
				return 0, nil, err
			}
			return SizeID2To1(w, h), nil, nil
		}
	}
	return 0, nil, nil
}

func validatedSizePair(kind string, width, height int64) (uint16, uint16, error) {
	if width <= 0 || height <= 0 || width > maxSizeDimension || height > maxSizeDimension {
		return 0, 0, fmt.Errorf("%s dimensions %dx%d are outside the supported range 1..%d", kind, width, height, maxSizeDimension)
	}
	return uint16(width), uint16(height), nil
}

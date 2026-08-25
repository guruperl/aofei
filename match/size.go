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
			if *video.W == 0 && *video.H == 0 {
				return 0, nil, nil
			}
			w, h, err := validatedSizePair("video", *video.W, *video.H)
			if err != nil {
				return 0, nil, err
			}
			return SizeID2To1(w, h), nil, nil
		}
	}
	if banner := imp.Banner; banner != nil {
		if banner.W != nil && banner.H != nil {
			if *banner.W == 0 && *banner.H == 0 {
				if w, h, ok := firstExactBannerFormat(banner.Format); ok {
					return SizeID2To1(w, h), nil, nil
				}
				return 0, nil, nil
			}
			w, h, err := validatedSizePair("banner", *banner.W, *banner.H)
			if err != nil {
				return 0, nil, err
			}
			return SizeID2To1(w, h), nil, nil
		}
	}
	return 0, nil, nil
}

// firstExactBannerFormat returns the first banner format entry with an exact,
// representable width and height. Ratio-only or oversized entries are skipped.
func firstExactBannerFormat(formats []openrtb2.Format) (uint16, uint16, bool) {
	for _, format := range formats {
		if format.W <= 0 || format.H <= 0 || format.W > maxSizeDimension || format.H > maxSizeDimension {
			continue
		}
		return uint16(format.W), uint16(format.H), true
	}
	return 0, 0, false
}

func validatedSizePair(kind string, width, height int64) (uint16, uint16, error) {
	if width <= 0 || height <= 0 || width > maxSizeDimension || height > maxSizeDimension {
		return 0, 0, fmt.Errorf("%s dimensions %dx%d are outside the supported range 1..%d", kind, width, height, maxSizeDimension)
	}
	return uint16(width), uint16(height), nil
}

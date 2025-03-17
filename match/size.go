package match

import "github.com/prebid/openrtb/v20/openrtb2"

// SizeID2To1 converts two uint16 values to one uint32 value.
func SizeID2To1(w, h uint16) uint32 {
	return uint32(w)<<16 | uint32(h)
}

// SizeID1To2 converts one uint32 value to two uint16 values.
func SizeID1To2(sizeID uint32) (uint16, uint16) {
	return uint16(sizeID >> 16), uint16(sizeID & 0xffff)
}

// getSizeID returns the size ID from the bid request for banner, video, and native ads.
func getSizeIDNative(bidRequest *openrtb2.BidRequest) (uint32, *NativeType, error) {
	if banner := bidRequest.Imp[0].Banner; banner != nil {
		if banner.W != nil && banner.H != nil {
			return SizeID2To1(uint16(*banner.W), uint16(*banner.H)), nil, nil
		}
	}
	if video := bidRequest.Imp[0].Video; video != nil {
		if video.W != nil && video.H != nil {
			return SizeID2To1(uint16(*video.W), uint16(*video.H)), nil, nil
		}
	}
	if native := bidRequest.Imp[0].Native; native != nil {
		nt, err := NewNativeType(native)
		if err != nil {
			return 0, nil, err
		}
		return SizeID2To1(nt.GetSizes()), nt, nil
	}
	return 0, nil, nil
}

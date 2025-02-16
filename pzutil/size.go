package pzutil

type Size struct {
	W       uint16
	H       uint16
	Display string
	Click   string
}

func GetSizeID(w, h uint16) uint32 {
	return (uint32(w) << 16) | uint32(h)
}

func GetSizes(sizeID uint32) (uint16, uint16) {
	return uint16(sizeID >> 16), uint16(0xFFFF & sizeID)
}

/*
func PackTwo(x, y uint32) string {
    return strconv.FormatUint((uint64(x)<<32) | uint64(y), 16)
}

func UnpackTwo(z string) (uint32, uint32) {
    result, _ := strconv.ParseUint(z, 16, 64)
    return uint32(result>>32), uint32(0xFFFFFFFF & result)
}
*/

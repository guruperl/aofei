package ipsearch

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

func IPToLong(ip string) (uint32, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || !addr.Is4() {
		return 0, fmt.Errorf("invalid IP address: %s", ip)
	}
	quads := addr.As4()
	return binary.BigEndian.Uint32(quads[:]), nil
}

/*
func StateCodeToUint32(stateCode string) uint32 {
	if stateCode == "" {
		return 0
	}
	switch len(stateCode) {
	case 3:
		stateCode += " "
	case 2:
		stateCode += "  "
	case 1:
		stateCode += "   "
	}
	bs := []byte(stateCode)
	return bytesToLong(bs[0], bs[1], bs[2], bs[3])
}

func Uint32ToStateCode(ip uint32) string {
	bs := get4(ip)
	return strings.TrimSpace(string(bs[:]))
}
*/
// 字节转整形
// binary.LittleEndian.Uint32([]byte) => uint32
func bytesToLong(a, b, c, d byte) uint32 {
	a1 := uint32(a)
	b1 := uint32(b)
	c1 := uint32(c)
	d1 := uint32(d)
	return (a1 & 0xFF) | ((b1 << 8) & 0xFF00) | ((c1 << 16) & 0xFF0000) | ((d1 << 24) & 0xFF000000)
}

func get4(ip uint32) []byte {
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, ip)
	return b4
}

func get3(ip uint32) []byte {
	b4 := get4(ip)
	b := make([]byte, 3)
	b[0] = b4[0]
	b[1] = b4[1]
	b[2] = b4[2]
	return b
}

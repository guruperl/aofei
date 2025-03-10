package ipsearch

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

func IPToLong(ip string) (uint32, error) {
	quads := strings.Split(ip, ".")
	if len(quads) != 4 {
		return 0, fmt.Errorf("invalid IP address: %s", ip)
	}
	var result uint32 = 0
	a, err := strconv.ParseUint(quads[3], 10, 32)
	if err != nil {
		return 0, err
	}
	result += uint32(a)
	b, err := strconv.ParseUint(quads[2], 10, 32)
	if err != nil {
		return 0, err
	}
	result += uint32(b) << 8
	c, err := strconv.ParseUint(quads[1], 10, 32)
	if err != nil {
		return 0, err
	}
	result += uint32(c) << 16
	d, err := strconv.ParseUint(quads[0], 10, 32)
	if err != nil {
		return 0, err
	}
	result += uint32(d) << 24
	return result, nil
}

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

// 字节转整形
// binary.LittleEndian.Uint32([]byte) => uint32
func bytesToLong(a, b, c, d byte) uint32 {
	a1 := uint32(a)
	b1 := uint32(b)
	c1 := uint32(c)
	d1 := uint32(d)
	return (a1 & 0xFF) | ((b1 << 8) & 0xFF00) | ((c1 << 16) & 0xFF0000) | ((d1 << 24) & 0xFF000000)
}

func bytesToLong3(a, b, c byte) uint32 {
	a1 := uint32(a)
	b1 := uint32(b)
	c1 := uint32(c)
	return (a1 & 0xFF) | ((b1 << 8) & 0xFF00) | ((c1 << 16) & 0xFF0000)

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

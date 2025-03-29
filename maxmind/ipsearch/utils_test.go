package ipsearch

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestUtils(t *testing.T) {
	ipstr := "123.234.56.78"

	// using my genelet ip 2 int way, bigEndian
	ip, err := IPToLong(ipstr)
	if err != nil {
		t.Fatalf("%s", err)
	}
	bs := [4]byte{byte(ip >> 24), byte(ip >> 16), byte(ip >> 8), byte(ip)}
	outstr := fmt.Sprintf("%d.%d.%d.%d", bs[0], bs[1], bs[2], bs[3])
	if ipstr != outstr {
		t.Errorf("%s %s", ipstr, outstr)
	}

	ip1 := bytesToLong(bs[3], bs[2], bs[1], bs[0])
	if ip1 != ip {
		t.Errorf("%d %d", ip, ip1)
	}

	// using GO libaray, littleEndian
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, ip)
	ip2 := bytesToLong(b4[0], b4[1], b4[2], b4[3])
	if ip2 != ip {
		t.Errorf("%d %d", ip, ip2)
	}

	a4 := get4(ip)
	ip3 := bytesToLong(a4[0], a4[1], a4[2], a4[3])
	ip4 := binary.LittleEndian.Uint32(a4)
	if ip3 != ip {
		t.Errorf("%d %d", ip, ip3)
	}
	if ip4 != ip {
		t.Errorf("%d %d", ip, ip4)
	}
}

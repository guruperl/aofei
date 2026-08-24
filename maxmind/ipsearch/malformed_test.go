package ipsearch

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestParseIPDataValidatesAndFindsFirstIndex(t *testing.T) {
	data := validDatFixture(t)
	search, err := parseIPData(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := search.Get("1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "continent|country|state|metro|city|zip|isp" {
		t.Fatalf("Get() = %q", got)
	}
	index, err := search.getIPIndex("1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if index == nil || index.StartIP != mustIPLong(t, "1.2.3.0") || index.EndIP != mustIPLong(t, "1.2.3.255") {
		t.Fatalf("first index = %+v", index)
	}
	missing, err := search.Get("1.2.4.1")
	if err != nil {
		t.Fatal(err)
	}
	if missing != "" {
		t.Fatalf("missing lookup = %q", missing)
	}
	if _, err := search.Get("300.1.1.1"); err == nil {
		t.Fatal("invalid IPv4 lookup error = nil")
	}
}

func TestParseIPDataRejectsMalformedOffsets(t *testing.T) {
	valid := validDatFixture(t)
	firstOffset := binary.LittleEndian.Uint32(valid[0:4])
	prefixStart := binary.LittleEndian.Uint32(valid[8:12])

	tests := []struct {
		name string
		data func() []byte
		want string
	}{
		{name: "truncated header", data: func() []byte { return []byte{1, 2, 3} }, want: "header is truncated"},
		{name: "first offset outside file", data: mutateDat(valid, func(data []byte) {
			binary.LittleEndian.PutUint32(data[0:4], uint32(len(data)+1))
		}), want: "first index offset"},
		{name: "index end mismatch", data: mutateDat(valid, func(data []byte) {
			binary.LittleEndian.PutUint32(data[4:8], firstOffset+1)
		}), want: "index end offset"},
		{name: "prefix end outside file", data: mutateDat(valid, func(data []byte) {
			binary.LittleEndian.PutUint32(data[12:16], uint32(len(data)))
		}), want: "prefix region ends"},
		{name: "short location", data: mutateDat(valid, func(data []byte) {
			data[firstOffset+11] = byte(datGeoSize - 1)
		}), want: "shorter than geo record"},
		{name: "location outside data", data: mutateDat(valid, func(data []byte) {
			data[firstOffset+8] = 0
			data[firstOffset+9] = 0
			data[firstOffset+10] = 0
		}), want: "outside the data region"},
		{name: "descending IP range", data: mutateDat(valid, func(data []byte) {
			binary.LittleEndian.PutUint32(data[firstOffset:firstOffset+4], 100)
			binary.LittleEndian.PutUint32(data[firstOffset+4:firstOffset+8], 99)
		}), want: "descending IP range"},
		{name: "prefix index outside range", data: mutateDat(valid, func(data []byte) {
			binary.LittleEndian.PutUint32(data[prefixStart+5:prefixStart+9], 1)
		}), want: "invalid index range"},
		{name: "trailing bytes", data: func() []byte { return append(append([]byte(nil), valid...), 0) }, want: "file length"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseIPData(test.data()); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parseIPData() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGetPrefixIndexRejectsShortBuffer(t *testing.T) {
	if got := GetPrefixIndex([]byte{1, 2, 3}, 0); got != nil {
		t.Fatalf("GetPrefixIndex() = %+v, want nil", got)
	}
	if got := GetPrefixIndex(make([]byte, 9), ^uint32(0)); got != nil {
		t.Fatalf("GetPrefixIndex(overflow) = %+v, want nil", got)
	}
}

func TestIPToLongRejectsNonIPv4AndOversizedOctets(t *testing.T) {
	for _, ip := range []string{"", "1.2.3", "300.1.1.1", "::1"} {
		if _, err := IPToLong(ip); err == nil {
			t.Errorf("IPToLong(%q) error = nil", ip)
		}
	}
}

func FuzzParseIPDataDoesNotPanic(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte{1, 2, 3})
	f.Add(validDatFixture(f))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseIPData(data)
	})
}

type fataler interface {
	Helper()
	Fatal(...any)
}

func validDatFixture(t fataler) []byte {
	t.Helper()
	start := mustIPLong(t, "1.2.3.0")
	end := mustIPLong(t, "1.2.3.255")
	location := []byte("continent|country|state|metro|city|zip|isp")
	length := datGeoSize + uint32(len(location))
	index := &ipIndex{
		StartIP:     start,
		EndIP:       end,
		LocalOffset: datHeaderSize,
		LocalLength: length,
		Geo: Geo{
			ContinentID: 1,
			CountryID:   2,
			StateID:     3,
			DmaID:       4,
			CityID:      5,
			IspID:       6,
			ZipID:       7,
			Lat:         8.5,
			Lon:         9.5,
		},
		LocalString: location,
	}
	firstOffset := datHeaderSize + length
	var out bytes.Buffer
	if err := writeDat(&out, []*ipIndex{index}, []uint32{1}, map[uint32]*prefixIndex{
		1: {StartIndex: 0, EndIndex: 0},
	}, firstOffset); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func mustIPLong(t fataler, ip string) uint32 {
	t.Helper()
	value, err := IPToLong(ip)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mutateDat(data []byte, mutate func([]byte)) func() []byte {
	return func() []byte {
		copy := append([]byte(nil), data...)
		mutate(copy)
		return copy
	}
}

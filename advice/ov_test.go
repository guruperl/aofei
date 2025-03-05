package advice

import (
	"testing"
)

func TestOVParse(t *testing.T) {
	tests := []struct {
		in  string
		out DeviceOSV
	}{
		{"", OVersionUnknown},
		{"1", OVersion1},
		{"2", OVersion2},
		{"3", OVersion3},
		{"4", OVersion4},
		{"5", OVersion5},
		{"6", OVersion6},
		{"7", OVersion7},
		{"8", OVersion8},
		{"9", OVersion9},
		{"10", OVersion10},
		{"11", OVersion11},
		{"12", OVersion12},
		{"13", OVersion13},
		{"14", OVersion14},
		{"15", OVersion15},
		{"16", OVersion16},
		{"17", OVersion17},
		{"18", OVersion18},
		{"19", OVersion19},
		{"20", OVersion20},
	}
	for _, tt := range tests {
		if got := ParseOVersion(tt.in); got != tt.out {
			t.Errorf("ParseOS(%q) = %v; want %v", tt.in, got, tt.out)
		}
	}
}

func TestOVString(t *testing.T) {
	tests := []struct {
		in  DeviceOSV
		out string
	}{
		{OVersionUnknown, "Unknown"},
		{OVersion1, "1"},
		{OVersion2, "2"},
		{OVersion3, "3"},
		{OVersion4, "4"},
		{OVersion5, "5"},
		{OVersion6, "6"},
		{OVersion7, "7"},
		{OVersion8, "8"},
		{OVersion9, "9"},
		{OVersion10, "10"},
		{OVersion11, "11"},
		{OVersion12, "12"},
		{OVersion13, "13"},
		{OVersion14, "14"},
		{OVersion15, "15"},
		{OVersion16, "16"},
		{OVersion17, "17"},
		{OVersion18, "18"},
		{OVersion19, "19"},
		{OVersion20, "20"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.out {
			t.Errorf("%v.String() = %q; want %q", tt.in, got, tt.out)
		}
	}
}

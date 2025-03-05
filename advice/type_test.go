package advice

import (
	"testing"

	"github.com/mileusna/useragent"
)

func TestTypeParse(t *testing.T) {
	tests := []struct {
		in  string
		out DeviceType
	}{
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8", TypePersonalComputer},
		{"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36", TypePersonalComputer},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1", TypeMobileTablet},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) FxiOS/8.1.1b4948 Mobile/14F89 Safari/603.2.4", TypeMobileTablet},
		{"Mozilla/5.0 (iPad; CPU OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1", TypeMobileTablet},
		{"Mozilla/5.0 (Android 4.3; Mobile; rv:54.0) Gecko/54.0 Firefox/54.0", TypeMobileTablet},
		{"Mozilla/5.0 (Linux; Android 4.3; GT-I9300 Build/JSS15J) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.91 Mobile Safari/537.36 OPR/42.9.2246.119956", TypeMobileTablet},
		{"Opera/9.80 (Android; Opera Mini/28.0.2254/66.318; U; en) Presto/2.12.423 Version/12.16", TypeMobileTablet},
	}
	for _, tt := range tests {
		ua := useragent.Parse(tt.in)
		if got := ParseType(ua, DeviceOS(0)); got != tt.out {
			t.Errorf("ParseType(%q) = %v; want %v", tt.in, got, tt.out)
		}
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		in  DeviceType
		out string
	}{
		{TypeConnectedDevice, "Connected Device"},
		{TypeConnectedTV, "Connected TV"},
		{TypeMobileTablet, "Mobile/Tablet"},
		{TypePersonalComputer, "Personal Computer"},
		{TypePhone, "Phone"},
		{TypeSetTopBox, "Set Top Box"},
		{TypeTablet, "Tablet"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.out {
			t.Errorf("DeviceType(%q).String() = %v; want %v", tt.in, got, tt.out)
		}
	}
}

package advice

import (
	"testing"
)

func TestMakerParse(t *testing.T) {
	tests := []struct {
		in  string
		out DeviceMake
	}{
		{"OPPO", MakerOPPO},
		{"SAMSUNG", MakerSAMSUNG},
		{"XIAOMI", MakerXIAOMI},
		{"VIVO", MakerVIVO},
		{"REALME", MakerREALME},
		{"HUAWEI", MakerHUAWEI},
		{"GENERIC", MakerGENERIC},
	}
	for _, tt := range tests {
		if got := ParseMaker(tt.in); got != tt.out {
			t.Errorf("ParseMaker(%q) = %v; want %v", tt.in, got, tt.out)
		}
	}
}

func TestMakerString(t *testing.T) {
	tests := []struct {
		in  DeviceMake
		out string
	}{
		{MakerOPPO, "OPPO"},
		{MakerSAMSUNG, "SAMSUNG"},
		{MakerXIAOMI, "XIAOMI"},
		{MakerVIVO, "VIVO"},
		{MakerREALME, "REALME"},
		{MakerHUAWEI, "HUAWEI"},
		{MakerGENERIC, "GENERIC"},
		{MakerMOTOROLA, "MOTOROLA"},
		{MakerINFINIX, "INFINIX"},
		{MakerASUS, "ASUS"},
		{MakerTECNO, "TECNO"},
		{MakerAPPLE, "APPLE"},
		{MakerLG, "LG"},
		{MakerOPPO, "OPPO"},
		{MakerSAMSUNG, "SAMSUNG"},
		{MakerXIAOMI, "XIAOMI"},
		{MakerVIVO, "VIVO"},
		{MakerREALME, "REALME"},
		{MakerHUAWEI, "HUAWEI"},
		{MakerGENERIC, "GENERIC"},
		{MakerMOTOROLA, "MOTOROLA"},
		{MakerINFINIX, "INFINIX"},
		{MakerASUS, "ASUS"},
		{MakerTECNO, "TECNO"},
		{MakerAPPLE, "APPLE"},
		{MakerLG, "LG"},
		{MakerOPPO, "OPPO"},
		{MakerSAMSUNG, "SAMSUNG"},
		{MakerXIAOMI, "XIAOMI"},
		{MakerVIVO, "VIVO"},
		{MakerREALME, "REALME"},
		{MakerHUAWEI, "HUAWEI"},
		{MakerGENERIC, "GENERIC"},
		{MakerMOTOROLA, "MOTOROLA"},
		{MakerINFINIX, "INFINIX"},
		{MakerASUS, "ASUS"},
		{MakerTECNO, "TECNO"},
		{MakerAPPLE, "APPLE"},
		{MakerLG, "LG"},
		{MakerOPPO, "OPPO"},
		{MakerSAMSUNG, "SAMSUNG"},
		{MakerXIAOMI, "XIAOMI"},
		{MakerVIVO, "VIVO"},
		{MakerREALME, "REALME"},
		{MakerHUAWEI, "HUAWEI"},
		{MakerGENERIC, "GENERIC"},
		{MakerMOTOROLA, "MOTOROLA"},
		{MakerINFINIX, "INFINIX"},
		{MakerASUS, "ASUS"},
		{MakerTECNO, "TECNO"},
		{MakerAPPLE, "APPLE"},
		{MakerLG, "LG"},
		{MakerOPPO, "OPPO"},
		{MakerSAMSUNG, "SAMSUNG"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.out {
			t.Errorf("DeviceMake(%q).String() = %v; want %v", tt.in, got, tt.out)
		}
	}
}

// Package advice handles device and user agent
package advice

import (
	"strings"
)

type DeviceMake int8

const (
	MakerUnknown DeviceMake = iota
	MakerOPPO
	MakerSAMSUNG
	MakerXIAOMI
	MakerVIVO
	MakerREALME
	MakerHUAWEI
	MakerGENERIC
	MakerMOTOROLA
	MakerINFINIX
	MakerASUS
	MakerTECNO
	MakerAPPLE
	MakerLG
	MakerLENOVO
	MakerVSMART
	MakerZTE
	MakerLGE
	MakerHMD
	MakerSONY
	MakerADVAN
	MakerITEL
	MakerTCL
	MakerQMOBILE
	MakerNOKIA
	MakerMEIZU
	MakerEVERCOSS
	MakerLAVA
	MakerALPS
	MakerANDROID
	MakerBLU
	MakerALCATEL
	MakerHAIER
	MakerONEPLUS
	MakerWIKO
	MakerHISENSE
	MakerSHARP
	MakerHOTWAV
	MakerLANIX
	MakerHYUNDAI
	MakerHTC
	MakerFORTUNE
	MakerLIBRE
	MakerMITO
	MakerGOOGLE
	MakerELEVATE
	MakerREDMI
	MakerBLACKBERRY
)

func (self DeviceMake) String() string {
	switch self {
	case MakerOPPO:
		return "OPPO"
	case MakerSAMSUNG:
		return "SAMSUNG"
	case MakerXIAOMI:
		return "XIAOMI"
	case MakerVIVO:
		return "VIVO"
	case MakerREALME:
		return "REALME"
	case MakerHUAWEI:
		return "HUAWEI"
	case MakerGENERIC:
		return "GENERIC"
	case MakerMOTOROLA:
		return "MOTOROLA"
	case MakerINFINIX:
		return "INFINIX"
	case MakerASUS:
		return "ASUS"
	case MakerTECNO:
		return "TECNO"
	case MakerAPPLE:
		return "APPLE"
	case MakerLG:
		return "LG"
	case MakerLENOVO:
		return "LENOVO"
	case MakerVSMART:
		return "VSMART"
	case MakerZTE:
		return "ZTE"
	case MakerLGE:
		return "LGE"
	case MakerHMD:
		return "HMD"
	case MakerSONY:
		return "SONY"
	case MakerADVAN:
		return "ADVAN"
	case MakerITEL:
		return "ITEL"
	case MakerTCL:
		return "TCL"
	case MakerQMOBILE:
		return "QMOBILE"
	case MakerNOKIA:
		return "NOKIA"
	case MakerMEIZU:
		return "MEIZU"
	case MakerEVERCOSS:
		return "EVERCOSS"
	case MakerLAVA:
		return "LAVA"
	case MakerALPS:
		return "ALPS"
	case MakerANDROID:
		return "ANDROID"
	case MakerBLU:
		return "BLU"
	case MakerALCATEL:
		return "ALCATEL"
	case MakerHAIER:
		return "HAIER"
	case MakerONEPLUS:
		return "ONEPLUS"
	case MakerWIKO:
		return "WIKO"
	case MakerHISENSE:
		return "HISENSE"
	case MakerSHARP:
		return "SHARP"
	case MakerHOTWAV:
		return "HOTWAV"
	case MakerLANIX:
		return "LANIX"
	case MakerHYUNDAI:
		return "HYUNDAI"
	case MakerHTC:
		return "HTC"
	case MakerFORTUNE:
		return "FORTUNE"
	case MakerLIBRE:
		return "LIBRE"
	case MakerMITO:
		return "MITO"
	case MakerGOOGLE:
		return "GOOGLE"
	case MakerELEVATE:
		return "ELEVATE"
	case MakerREDMI:
		return "REDMI"
	case MakerBLACKBERRY:
		return "BLACKBERRY"
	default:
		return "All"
	}
}

func ParseMaker(name string) DeviceMake {
	switch {
	case strings.Contains(name, "OPPO"):
		return MakerOPPO
	case strings.Contains(name, "SAMSUNG"):
		return MakerSAMSUNG
	case strings.Contains(name, "XIAOMI"):
		return MakerXIAOMI
	case strings.Contains(name, "VIVO"):
		return MakerVIVO
	case strings.Contains(name, "REALME"):
		return MakerREALME
	case strings.Contains(name, "HUAWEI"):
		return MakerHUAWEI
	case strings.Contains(name, "GENERIC"):
		return MakerGENERIC
	case strings.Contains(name, "MOTOROLA"):
		return MakerMOTOROLA
	case strings.Contains(name, "INFINIX"):
		return MakerINFINIX
	case strings.Contains(name, "ASUS"):
		return MakerASUS
	case strings.Contains(name, "TECNO"):
		return MakerTECNO
	case strings.Contains(name, "APPLE"):
		return MakerAPPLE
	case strings.Contains(name, "LG"):
		return MakerLG
	case strings.Contains(name, "LENOVO"):
		return MakerLENOVO
	case strings.Contains(name, "VSMART"):
		return MakerVSMART
	case strings.Contains(name, "ZTE"):
		return MakerZTE
	case strings.Contains(name, "LGE"):
		return MakerLGE
	case strings.Contains(name, "HMD"):
		return MakerHMD
	case strings.Contains(name, "SONY"):
		return MakerSONY
	case strings.Contains(name, "ADVAN"):
		return MakerADVAN
	case strings.Contains(name, "ITEL"):
		return MakerITEL
	case strings.Contains(name, "TCL"):
		return MakerTCL
	case strings.Contains(name, "QMOBILE"):
		return MakerQMOBILE
	case strings.Contains(name, "NOKIA"):
		return MakerNOKIA
	case strings.Contains(name, "MEIZU"):
		return MakerMEIZU
	case strings.Contains(name, "EVERCOSS"):
		return MakerEVERCOSS
	case strings.Contains(name, "LAVA"):
		return MakerLAVA
	case strings.Contains(name, "ALPS"):
		return MakerALPS
	case strings.Contains(name, "ANDROID"):
		return MakerANDROID
	case strings.Contains(name, "BLU"):
		return MakerBLU
	case strings.Contains(name, "ALCATEL"):
		return MakerALCATEL
	case strings.Contains(name, "HAIER"):
		return MakerHAIER
	case strings.Contains(name, "ONEPLUS"):
		return MakerONEPLUS
	case strings.Contains(name, "WIKO"):
		return MakerWIKO
	case strings.Contains(name, "HISENSE"):
		return MakerHISENSE
	case strings.Contains(name, "SHARP"):
		return MakerSHARP
	case strings.Contains(name, "HOTWAV"):
		return MakerHOTWAV
	case strings.Contains(name, "LANIX"):
		return MakerLANIX
	case strings.Contains(name, "HYUNDAI"):
		return MakerHYUNDAI
	case strings.Contains(name, "HTC"):
		return MakerHTC
	case strings.Contains(name, "FORTUNE"):
		return MakerFORTUNE
	case strings.Contains(name, "LIBRE"):
		return MakerLIBRE
	case strings.Contains(name, "MITO"):
		return MakerMITO
	case strings.Contains(name, "GOOGLE"):
		return MakerGOOGLE
	case strings.Contains(name, "ELEVATE"):
		return MakerELEVATE
	case strings.Contains(name, "REDMI"):
		return MakerREDMI
	case strings.Contains(name, "BLACKBERRY"):
		return MakerBLACKBERRY
	default:
		return MakerUnknown
	}
}

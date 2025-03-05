package advice

import "strings"

type DeviceOSV int8

const (
	OVersionUnknown DeviceOSV = iota
	OVersion1
	OVersion2
	OVersion3
	OVersion4
	OVersion5
	OVersion6
	OVersion7
	OVersion8
	OVersion9
	OVersion10
	OVersion11
	OVersion12
	OVersion13
	OVersion14
	OVersion15
	OVersion16
	OVersion17
	OVersion18
	OVersion19
	OVersion20
)

func (self DeviceOSV) String() string {
	switch self {
	case OVersion1:
		return "1"
	case OVersion2:
		return "2"
	case OVersion3:
		return "3"
	case OVersion4:
		return "4"
	case OVersion5:
		return "5"
	case OVersion6:
		return "6"
	case OVersion7:
		return "7"
	case OVersion8:
		return "8"
	case OVersion9:
		return "9"
	case OVersion10:
		return "10"
	case OVersion11:
		return "11"
	case OVersion12:
		return "12"
	case OVersion13:
		return "13"
	case OVersion14:
		return "14"
	case OVersion15:
		return "15"
	case OVersion16:
		return "16"
	case OVersion17:
		return "17"
	case OVersion18:
		return "18"
	case OVersion19:
		return "19"
	case OVersion20:
		return "20"
	}
	return "Unknown"
}

func ParseOVersion(full string) DeviceOSV {
	s := strings.Split(full, ".")[0]
	switch s {
	case "1":
		return OVersion1
	case "2":
		return OVersion2
	case "3":
		return OVersion3
	case "4":
		return OVersion4
	case "5":
		return OVersion5
	case "6":
		return OVersion6
	case "7":
		return OVersion7
	case "8":
		return OVersion8
	case "9":
		return OVersion9
	case "10":
		return OVersion10
	case "11":
		return OVersion11
	case "12":
		return OVersion12
	case "13":
		return OVersion13
	case "14":
		return OVersion14
	case "15":
		return OVersion15
	case "16":
		return OVersion16
	case "17":
		return OVersion17
	case "18":
		return OVersion18
	case "19":
		return OVersion19
	case "20":
		return OVersion20
	}
	return OVersionUnknown
}

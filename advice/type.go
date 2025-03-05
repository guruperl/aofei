package advice

import (
	"github.com/mileusna/useragent"
	"github.com/prebid/openrtb/v20/adcom1"
)

type DeviceType int8

const (
	TypeUnknown DeviceType = iota
	TypeConnectedDevice
	TypeConnectedTV
	TypeMobileTablet
	TypePersonalComputer
	TypePhone
	TypeSetTopBox
	TypeTablet
)

func (self DeviceType) String() string {
	switch self {
	case TypeConnectedDevice:
		return "Connected Device"
	case TypeConnectedTV:
		return "Connected TV"
	case TypeMobileTablet:
		return "Mobile/Tablet"
	case TypePersonalComputer:
		return "Personal Computer"
	case TypePhone:
		return "Phone"
	case TypeSetTopBox:
		return "Set Top Box"
	case TypeTablet:
		return "Tablet"
	}
	return "Unknown"
}

func DeviceTypeType(dt adcom1.DeviceType) DeviceType {
	switch dt {
	case adcom1.DeviceMobile:
		return TypeMobileTablet
	case adcom1.DevicePC:
		return TypePersonalComputer
	case adcom1.DeviceTV:
		return TypeConnectedTV
	case adcom1.DevicePhone:
		return TypePhone
	case adcom1.DeviceTablet:
		return TypeTablet
	case adcom1.DeviceConnected:
		return TypeConnectedDevice
	case adcom1.DeviceSetTopBox:
		return TypeSetTopBox
	default:
	}
	return TypeUnknown
}

func ParseType(ua useragent.UserAgent, os DeviceOS) DeviceType {
	if ua.Mobile || ua.Tablet {
		return TypeMobileTablet
	} else if ua.Desktop {
		return TypePersonalComputer
	}

	switch os {
	case OSAppleTV:
		return TypeConnectedTV
	case OSAndroid:
		return TypePhone
	case OSIOS:
		return TypePhone
	case OSWindows:
		return TypePersonalComputer
	case OSWindowsPhone:
		return TypePhone
	case OSChromeOS:
		return TypePersonalComputer
	case OSWebOS:
		return TypeConnectedTV
	case OSWatchOS:
		return TypePhone
	case OSUnix:
		return TypePersonalComputer
	case OSUbuntu:
		return TypePersonalComputer
	case OSTizen:
		return TypeConnectedTV
	case OSSymbian:
		return TypePhone
	case OSNetBSD:
		return TypePersonalComputer
	case OSMorphOS:
		return TypePersonalComputer
	case OSMeeGo:
		return TypePhone
	case OSMacOS:
		return TypePersonalComputer
	case OSLinux:
		return TypePersonalComputer
	case OSDarwin:
		return TypePersonalComputer
	case OSBlackBerry:
		return TypePhone
	case OSBada:
		return TypePhone
	case OSAsha:
		return TypePhone
	default:
	}
	return TypeUnknown
}

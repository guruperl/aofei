// Package advice handles device and user agent
package advice

import (
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
)

type DeviceOS int8

const (
	OSUnknown DeviceOS = iota
	OSAndroid
	OSAppleTV
	OSAsha
	OSBada
	OSBlackBerry
	OSChromeOS
	OSDarwin
	OSFedora
	OSIOS
	OSLinux
	OSMacOS
	OSMeeGo
	OSMorphOS
	OSNetBSD
	OSOS2
	OSSeries
	OSSymbian
	OSTizen
	OSUbuntu
	OSUnix
	OSWatchOS
	OSWebOS
	OSWindows
	OSWindowsPhone
)

func (self DeviceOS) String() string {
	switch self {
	case OSUnknown:
		return "Unknown"
	case OSAndroid:
		return "Android"
	case OSAppleTV:
		return "AppleTV"
	case OSAsha:
		return "Asha"
	case OSBada:
		return "Bada"
	case OSBlackBerry:
		return "BlackBerry"
	case OSChromeOS:
		return "ChromeOS"
	case OSDarwin:
		return "Darwin"
	case OSFedora:
		return "Fedora"
	case OSIOS:
		return "iOS"
	case OSLinux:
		return "Linux"
	case OSMacOS:
		return "MacOS"
	case OSMeeGo:
		return "MeeGo"
	case OSMorphOS:
		return "MorphOS"
	case OSNetBSD:
		return "NetBSD"
	case OSOS2:
		return "OS2"
	case OSSeries:
		return "Series"
	case OSSymbian:
		return "Symbian"
	case OSTizen:
		return "Tizen"
	case OSUbuntu:
		return "Ubuntu"
	case OSUnix:
		return "Unix"
	case OSWatchOS:
		return "WatchOS"
	case OSWebOS:
		return "webOS"
	case OSWindows:
		return "Windows"
	case OSWindowsPhone:
		return "Windows Phone"
	}
	return "Unknown"
}

func ParseOS(name string) DeviceOS {
	switch strings.ToLower(name) {
	case "android":
		return OSAndroid
	case "appletv", "apple tv":
		return OSAppleTV
	case "asha":
		return OSAsha
	case "bada":
		return OSBada
	case "blackberry":
		return OSBlackBerry
	case "chromeos", "chrome os":
		return OSChromeOS
	case "darwin":
		return OSDarwin
	case "fedora":
		return OSFedora
	case "ios":
		return OSIOS
	case "linux":
		return OSLinux
	case "macos":
		return OSMacOS
	case "meego":
		return OSMeeGo
	case "morphos":
		return OSMorphOS
	case "netbsd":
		return OSNetBSD
	case "os2":
		return OSOS2
	case "series":
		return OSSeries
	case "symbian":
		return OSSymbian
	case "tizen":
		return OSTizen
	case "ubuntu":
		return OSUbuntu
	case "unix":
		return OSUnix
	case "watchos", "watch os":
		return OSWatchOS
	case "webos", "web os":
		return OSWebOS
	case "windows":
		return OSWindows
	case "windows phone":
		return OSWindowsPhone
	}
	return OSUnknown
}

func OperatingSystemOS(os adcom1.OperatingSystem) DeviceOS {
	switch os {
	case adcom1.OSAndroid:
		return OSAndroid
	case adcom1.OSAppleTV:
		return OSAppleTV
	case adcom1.OSAsha:
		return OSAsha
	case adcom1.OSBada:
		return OSBada
	case adcom1.OSBlackBerry:
		return OSBlackBerry
	case adcom1.OSChromeOS:
		return OSChromeOS
	case adcom1.OSDarwin:
		return OSDarwin
	case adcom1.OSIOS:
		return OSIOS
	case adcom1.OSLinux:
		return OSLinux
	case adcom1.OSMacOS:
		return OSMacOS
	case adcom1.OSMeeGo:
		return OSMeeGo
	case adcom1.OSMorphOS:
		return OSMorphOS
	case adcom1.OSNetBSD:
		return OSNetBSD
	case adcom1.OSSymbian:
		return OSSymbian
	case adcom1.OSTizen:
		return OSTizen
	case adcom1.OSWatchOS:
		return OSWatchOS
	case adcom1.OSWebOS:
		return OSWebOS
	case adcom1.OSWindows:
		return OSWindows
	default:
		return OSUnknown
	}
}

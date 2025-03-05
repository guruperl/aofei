package advice

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
)

func TestOSParse(t *testing.T) {
	tests := []struct {
		in  string
		out DeviceOS
	}{
		{"Android", OSAndroid},
		{"AppleTV", OSAppleTV},
		{"Asha", OSAsha},
		{"Bada", OSBada},
		{"BlackBerry", OSBlackBerry},
		{"ChromeOS", OSChromeOS},
		{"Darwin", OSDarwin},
		{"Fedora", OSFedora},
		{"iOS", OSIOS},
		{"Linux", OSLinux},
		{"MacOS", OSMacOS},
		{"MeeGo", OSMeeGo},
		{"MorphOS", OSMorphOS},
		{"NetBSD", OSNetBSD},
		{"OS2", OSOS2},
		{"Series", OSSeries},
		{"Symbian", OSSymbian},
		{"Tizen", OSTizen},
		{"Ubuntu", OSUbuntu},
		{"Unix", OSUnix},
		{"WatchOS", OSWatchOS},
		{"WebOS", OSWebOS},
		{"Windows", OSWindows},
		{"WindowsPhone", OSWindowsPhone},
	}
	for _, tt := range tests {
		if got := ParseOS(tt.in); got != tt.out {
			t.Errorf("ParseOS(%q) = %v; want %v", tt.in, got, tt.out)
		}
	}
}

func TestOSString(t *testing.T) {
	tests := []struct {
		in  DeviceOS
		out string
	}{
		{OSUnknown, "Unknown"},
		{OSAndroid, "Android"},
		{OSAppleTV, "AppleTV"},
		{OSAsha, "Asha"},
		{OSBada, "Bada"},
		{OSBlackBerry, "BlackBerry"},
		{OSChromeOS, "ChromeOS"},
		{OSDarwin, "Darwin"},
		{OSFedora, "Fedora"},
		{OSIOS, "iOS"},
		{OSLinux, "Linux"},
		{OSMacOS, "MacOS"},
		{OSMeeGo, "MeeGo"},
		{OSMorphOS, "MorphOS"},
		{OSNetBSD, "NetBSD"},
		{OSOS2, "OS2"},
		{OSSeries, "Series"},
		{OSSymbian, "Symbian"},
		{OSTizen, "Tizen"},
		{OSUbuntu, "Ubuntu"},
		{OSUnix, "Unix"},
		{OSWatchOS, "WatchOS"},
		{OSWebOS, "webOS"},
		{OSWindows, "Windows"},
		{OSWindowsPhone, "WindowsPhone"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.out {
			t.Errorf("OS(%v).String() = %v; want %v", tt.in, got, tt.out)
		}
	}
}

func TestOSOperatingSystem(t *testing.T) {
	tests := []struct {
		in  adcom1.OperatingSystem
		out DeviceOS
	}{
		{adcom1.OSAndroid, OSAndroid},
		{adcom1.OSAppleTV, OSAppleTV},
		{adcom1.OSAsha, OSAsha},
		{adcom1.OSBada, OSBada},
		{adcom1.OSBlackBerry, OSBlackBerry},
		{adcom1.OSChromeOS, OSChromeOS},
		{adcom1.OSDarwin, OSDarwin},
		{adcom1.OSIOS, OSIOS},
		{adcom1.OSLinux, OSLinux},
		{adcom1.OSMacOS, OSMacOS},
		{adcom1.OSMeeGo, OSMeeGo},
		{adcom1.OSMorphOS, OSMorphOS},
		{adcom1.OSNetBSD, OSNetBSD},
		{adcom1.OSSymbian, OSSymbian},
		{adcom1.OSTizen, OSTizen},
		{adcom1.OSWatchOS, OSWatchOS},
		{adcom1.OSWebOS, OSWebOS},
		{adcom1.OSWindows, OSWindows},
	}
	for _, tt := range tests {
		if got := OperatingSystemOS(tt.in); got != tt.out {
			t.Errorf("IsOperationSystem(%v) = %v; want %v", tt.in, got, tt.out)
		}
	}
}

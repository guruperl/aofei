package holiday

import (
	uadevice "github.com/genelet/winter/advice"
	adcom1 "github.com/mxmCherry/openrtb/adcom1"
)

type Device struct {
	IncomingDevice *adcom1.Device
	UA             string
	PzUA32         uint32
}

func (self *Device) GetTags() *Tags {
	if self.IncomingDevice != nil {
		//  return self.GetTagsFromIncoming()
	}

	pzua := uadevice.GetPzUa(self.UA)
	ref := map[uint32][]uint32{
		1201: []uint32{uint32(pzua.Browser)},
		1202: []uint32{uint32(pzua.BVersion)},
		1203: []uint32{uint32(pzua.OS)},
		1204: []uint32{uint32(pzua.OVersion)},
		1205: []uint32{uint32(pzua.Platform)},
		1206: []uint32{uint32(pzua.Device)}}
	return &Tags{TagHashArray: ref}
}

/*
type DeviceType int8
const (
    DeviceMobile    DeviceType = 1 // Mobile/Tablet - General
    DevicePC        DeviceType = 2 // Personal Computer
    DeviceTV        DeviceType = 3 // Connected TV
    DevicePhone     DeviceType = 4 // Phone
    DeviceTablet    DeviceType = 5 // Tablet
    DeviceConnected DeviceType = 6 // Connected Device
    DeviceSetTopBox DeviceType = 7 // Set Top Box
)


type ConnectionType int8
const (
    ConnectionEthernet ConnectionType = 1 // 1	Ethernet; Wired Connection
    ConnectionWIFI     ConnectionType = 2 // 2	WIFI
    ConnectionCellular ConnectionType = 3 // 3	Cellular Network - Unknown Generation
    Connection2G       ConnectionType = 4 // 4	Cellular Network - 2G
    Connection3G       ConnectionType = 5 // 5	Cellular Network - 3G
    Connection4G       ConnectionType = 6 // 6	Cellular Network - 4G
    Connection5G       ConnectionType = 7 // 7	Cellular Network - 5G
)

type Device struct {
    Type DeviceType `json:"type,omitempty"`
    UA  string `json:"ua,omitempty"`
    //   ID sanctioned for advertiser use in the clear (i.e., not hashed).
    IFA string `json:"ifa,omitempty"`
    //   Standard “Do Not Track” flag as set in the header by the browser, where 0 = tracking is unrestricted, 1 = do not track.
    DNT int8 `json:"dnt,omitempty"`
    //   “Limit Ad Tracking” signal commercially endorsed (e.g., iOS, Android), where 0 = tracking is unrestricted, 1 = tracking must be limited per commercial guidelines.
    Lmt int8 `json:"lmt,omitempty"`
    //   Device make (e.g., "Apple").
    Make string `json:"make,omitempty"`
    //   Device model (e.g., “iPhone10,1” when the specific device model is known, “iPhone” otherwise).
    Model string `json:"model,omitempty"`
    //   Device operating system.
    OS  OperatingSystem `json:"os,omitempty"`
    //   Device operating system version (e.g., “3.1.2”).
    OSV string `json:"osv,omitempty"`
    //   Hardware version of the device (e.g., “5S” for iPhone 5S).
    HWV string `json:"hwv,omitempty"`
    //   Physical height of the screen in pixels.
    H   int64 `json:"h,omitempty"`
    //   Physical width of the screen in pixels.
    W   int64 `json:"w,omitempty"`
    //   Screen size as pixels per linear inch.
    PPI int64 `json:"ppi,omitempty"`
    //   The ratio of physical pixels to device independent pixels.
    PxRatio float64 `json:"pxratio,omitempty"`
    //   Support for JavaScript, where 0 = no, 1 = yes.
    JS  int8 `json:"js,omitempty"`
    //   Browser language using ISO-639-1-alpha-2.
    Lang string `json:"lang,omitempty"`
    //   IPv4 address closest to device.
    IP  string `json:"ip,omitempty"`
    //   IP address closest to device as IPv6.
    IPv6 string `json:"ipv6,omitempty"`
    //   The value of the “x-forwarded-for” header.
    XFF string `json:"xff,omitempty"`
    //   Indicator of truncation of any of the IP attributes (i.e., ip, ipv6, xff), where 0 = no, 1 = yes (e.g., from 1.2.3.4 to 1.2.3.0).
    IPTr int8 `json:"iptr,omitempty"`
    //   Carrier or ISP (e.g., “VERIZON”) using exchange curated string names
    Carrier string `json:"carrier,omitempty"`
    //   Mobile carrier as the concatenated MCC-MNC code (e.g., “310-005” identifies Verizon Wireless CDMA in the USA).
    MCCMNC string `json:"mccmnc,omitempty"`
    //   MCC and MNC of the SIM card using the same format as mccmnc.
    //   When both values are available, a difference between them reveals that a user is roaming.
    MCCMNCSIM string `json:"mccmncsim,omitempty"`
    //   Network connection type.
    ConType ConnectionType `json:"contype,omitempty"`
    //   Indicates if the geolocation API will be available to JavaScript code running in display ad, where 0 = no, 1 = yes.
    GeoFetch int8 `json:"geofetch,omitempty"`
    //   Location of the device (i.e., typically the user's current location).
    Geo *Geo `json:"geo,omitempty"`
    Ext json.RawMessage `json:"ext,omitempty"`
}
*/

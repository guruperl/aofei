package match

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dh"
	"github.com/genelet/winter/maxmind"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type Attribute struct {
	RPub
	*NativeFormat
	IsApp   bool
	IsVideo bool
	When    time.Time
	IFA     string
	UserID  string
	*demo.Demo
	*maxmind.Geo
	*advice.PzUa
	*dh.DH
	*acl.ACL
}

// NewAttribute creates a new Attribute from a bid request.
func NewAttribute(ctx context.Context, ipSearch *maxmind.IPSearch, bidRequest *openrtb2.BidRequest, rpubMap *RPubMap, when time.Time, pubStr string) (*Attribute, error) {
	device := bidRequest.Device
	if device == nil {
		return nil, nil
	}

	attr := &Attribute{When: when, IsVideo: bidRequest.Imp[0].Video != nil}

	var err error
	attr.IFA, err = getIFA(device)
	if err != nil {
		return nil, err
	}

	if bidRequest.User != nil && (bidRequest.User.BuyerUID != "" || bidRequest.User.ID != "") {
		attr.UserID = bidRequest.User.BuyerUID
		if attr.UserID == "" {
			attr.UserID = bidRequest.User.ID
		}
	}

	user := bidRequest.User
	if user != nil {
		attr.Demo = demo.NewDemo(user.Gender, uint32(user.Yob), getLangs(bidRequest))
	} else {
		attr.Demo = demo.NewDemo("", 0, getLangs(bidRequest))
	}

	attr.Geo, err = getGeo(ctx, ipSearch, device)
	if err != nil {
		return nil, err
	}

	attr.DH = dh.NewDH(when, uint8(attr.Geo.Location.UTCOffset))
	attr.PzUa = getUA(device)
	attr.ACL = getACL(bidRequest, pubStr)
	attr.RPub = rpubMap.GetRPub(attr.ACL)
	sizeID, nativeFormat, err := getSizeIDNative(bidRequest)
	if err != nil {
		return nil, err
	}
	attr.RPub.SizeID = sizeID
	if nativeFormat != nil {
		attr.NativeFormat = nativeFormat
	}

	return attr, nil
}

func MD5Hex(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// getACL returns the acl object from the bid request.
func getACL(bidRequest *openrtb2.BidRequest, pubStr string) *acl.ACL {
	if pubStr == "" && bidRequest.Ext != nil {
		hash := make(map[string]interface{})
		if err := json.Unmarshal(bidRequest.Ext, &hash); err == nil {
			if domain, ok := hash["request_domain"].(string); ok {
				pubStr = domain
			}
		}
	}
	if pubStr == "" {
		pubStr = PUBDefault
	}
	siteStr := SITEDefault
	slotStr := SLOTDefault

	a := &acl.ACL{
		BAdv:  bidRequest.BAdv,
		BApp:  bidRequest.BApp,
		White: bidRequest.ACat,
		Black: bidRequest.BCat,
	}

	var categories []string
	if site := bidRequest.Site; site != nil {
		if pubStr == PUBDefault && (site.Publisher != nil && site.Publisher.ID != "") {
			pubStr = site.Publisher.ID
		}
		if site.Domain != "" {
			siteStr = site.Domain
		} else if site.ID != "" {
			siteStr = site.ID
		}
		if site.Page != "" {
			slotStr = MD5Hex(site.Page)
		}
		if site.Cat != nil {
			categories = append(categories, site.Cat...)
		}
		if site.SectionCat != nil {
			categories = append(categories, site.SectionCat...)
		}
		if site.PageCat != nil {
			categories = append(categories, site.PageCat...)
		}
	} else if app := bidRequest.App; app != nil {
		if pubStr == PUBDefault && (app.Publisher != nil && app.Publisher.ID != "") {
			pubStr = app.Publisher.ID
		}
		if app.Bundle != "" {
			siteStr = app.Bundle
		} else if app.Domain != "" {
			siteStr = app.Domain
		} else if app.ID != "" {
			siteStr = app.ID
		}
		if app.Cat != nil {
			categories = append(categories, app.Cat...)
		}
		if app.SectionCat != nil {
			categories = append(categories, app.SectionCat...)
		}
		if app.PageCat != nil {
			categories = append(categories, app.PageCat...)
		}
	}
	if slotStr == SLOTDefault && bidRequest.Imp[0].TagID != "" {
		slotStr = bidRequest.Imp[0].TagID
	}

	a.PubStr = pubStr
	a.SiteStr = siteStr
	a.SlotStr = slotStr
	a.Categories = categories

	return a
}

// getIFA returns the IFA from the device.
func getIFA(device *openrtb2.Device) (string, error) {
	ifa := device.IFA
	if ifa == "" {
		switch {
		case device.DIDSHA1 != "":
			ifa = device.DIDSHA1
		case device.DIDMD5 != "":
			ifa = device.DIDMD5
		case device.MACSHA1 != "":
			ifa = device.MACSHA1
		case device.MACMD5 != "":
			ifa = device.MACMD5
		case device.UA != "" && device.IP != "":
			h := md5.New()
			if _, err := io.WriteString(h, device.IP+"."+device.UA); err != nil {
				return "", err
			}
			ifa = string(h.Sum(nil))
		default:
		}
	}
	return ifa, nil
}

// getLangs returns the WLangs object from the bid request and device.
func getLangs(bidRequest *openrtb2.BidRequest) []string {
	var langs []string
	if bidRequest.WLang != nil {
		langs = bidRequest.WLang
	} else if bidRequest.Device != nil && bidRequest.Device.Language != "" || bidRequest.Device.LangB != "" {
		if bidRequest.Device.Language != "" {
			langs = append(langs, bidRequest.Device.Language)
		}
		if bidRequest.Device.LangB != "" {
			langs = append(langs, bidRequest.Device.LangB)
		}
	}

	return langs
}

// getGeo returns the Geo object from the device.
func getGeo(_ context.Context, ipSearch *maxmind.IPSearch, device *openrtb2.Device) (*maxmind.Geo, error) {
	pzg := new(maxmind.Geo)
	if device.ConnectionType != nil {
		pzg.ConnectionType = *device.ConnectionType
	}
	if geo := device.Geo; geo != nil {
		if geo.Lat != nil && geo.Lon != nil {
			pzg.Location.Lat = *geo.Lat
			pzg.Location.Lon = *geo.Lon
		}
		pzg.Location.Type = geo.Type
		pzg.Location.Accuracy = geo.Accuracy
		pzg.Location.LastFix = geo.LastFix
		pzg.Location.IPService = geo.IPService
		if geo.UTCOffset != 0 {
			pzg.Location.UTCOffset = geo.UTCOffset
		}
		if geo.Country != "" {
			pzg.CountryID = ipSearch.CountryMap[geo.Country]
			if geo.Region != "" {
				pzg.StateID = ipSearch.StateMap[pzg.CountryID][geo.Region]
			}
		}
		if geo.Metro != "" {
			pzg.DmaID = maxmind.MetroMap[geo.Metro]
		}

		if device.IP != "" && (pzg.CountryID == 0 || pzg.StateID == 0) {
			mm, err := ipSearch.CreatePzGeo(device.IP)
			if err != nil {
				return nil, err
			}
			if pzg.CountryID == 0 {
				pzg.CountryID = mm.Geo.CountryID
			}
			if pzg.StateID == 0 {
				pzg.StateID = mm.Geo.StateID
			}
			if pzg.CityID == 0 {
				pzg.CityID = mm.Geo.CityID
			}
			if pzg.DmaID == 0 {
				pzg.DmaID = mm.Geo.DmaID
			}
			if pzg.Location.UTCOffset == 0 {
				pzg.Location.UTCOffset = mm.Geo.Location.UTCOffset
			}
		}
	}
	return pzg, nil
}

// getUA returns the PzUa object from the device.
func getUA(device *openrtb2.Device) *advice.PzUa {
	pzua := new(advice.PzUa)
	if device.DeviceType != 0 {
		pzua.Device = advice.DeviceTypeType(device.DeviceType)
	}
	if device.Make != "" {
		pzua.Platform = advice.ParseMaker(device.Make)
	}
	// device.Model?
	if device.OS != "" {
		pzua.OS = advice.ParseOS(device.OS)
	}
	if device.OSV != "" {
		pzua.OVersion = advice.ParseOVersion(device.OSV)
	}

	if device.UA != "" && (pzua.OS == 0 || pzua.OVersion == 0 || pzua.Platform == 0 || pzua.Device == 0) {
		mm := advice.GetPzUa(device.UA)
		if pzua.OS == 0 {
			pzua.OS = mm.OS
		}
		if pzua.OVersion == 0 {
			pzua.OVersion = mm.OVersion
		}
		if pzua.Platform == 0 {
			pzua.Platform = mm.Platform
		}
		if pzua.Device == 0 {
			pzua.Device = mm.Device
		}
	}
	return pzua
}

/*
var AttrNames = map[uint32]string{
	1001: "fullday", 1002: "fullhour", 1003: "weekday",
	1010: "language",

	1101: "continent", 1102: "country", 1103: "state", 1104: "city", 1105: "dma",
	1106: "zip", 1107: "isp", 1108: "bandwidth", 1111: "areacode", 1112: "utcoffset", 1110: "lon", 1109: "lat",

	1200: "pzua", 1201: "browser", 1202: "bversion", 1203: "os",
	1204: "oversion", 1205: "platform", 1206: "device",

	1300: "demography", 1301: "gender", 1302: "yob", 1303: "married",
	1304: "income", 1305: "child", 1306: "household", 1307: "ethnicity",
	1308: "education", 1309: "occupation",

	1901: "ifa1", 1902: "ifa2", 1903: "ifa3", 1904: "ifa4", 1905: "ip",
}

var AttrValues = map[string]uint32{
	"fullday": 1001, "fullhour": 1002, "weekday": 1003,
	"language": 1010,

	"continent": 1101, "country": 1102, "state": 1103, "city": 1104, "dma": 1105,
	"zip": 1106, "isp": 1107, "bandwidth": 1108, "areacode": 1111, "utcoffset": 1112, "lon": 1110, "lat": 1109,

	"pzua": 1200, "browser": 1201, "bversion": 1202, "os": 1203,
	"oversion": 1204, "platform": 1205, "device": 1206,

	"demography": 1300, "gender": 1301, "yob": 1302, "married": 1303,
	"income": 1304, "child": 1305, "household": 1306, "ethnicity": 1307,
	"education": 1308, "occupation": 1309,

	"ifa1": 1901, "ifa2": 1902, "ifa3": 1903, "ifa4": 1904, "ip": 1905,
}

type Attrs []uint32

func NewAttrsFromNames(names []string) Attrs {
	var arrs []uint32
	for _, name := range names {
		arrs = append(arrs, AttrValues[name])
	}
	return arrs
}
*/

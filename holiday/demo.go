package holiday

import (
	"context"
	"fmt"
	"math"
	"time"

	uadevice "github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
	ipsearch "github.com/genelet/winter/maxmind"
	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type TagHashArray map[uint32]Attrs

func (self *Controller) getTagsFromBidRequest(ctx context.Context, current time.Time, bidRequest *openrtb2.BidRequest) (TagHashArray, error) {
	ref := make(map[uint32]Attrs)

	if bidRequest.WLang != nil {
		ref[1303] = []uint32{uint32(NewWLangs(bidRequest.WLang))}
	}

	setCats := func(cats []string, start int) {
		if cats != nil {
			cats := NewCATSFromStrings(cats)
			for i, cat := range cats {
				ref[uint32(start+i)] = []uint32{cat}
			}
		}
	}
	if site := bidRequest.Site; site != nil {
		if site.Cat != nil {
			setCats(site.Cat, 1401)
		}
		if site.SectionCat != nil {
			setCats(site.SectionCat, 1501)
		}
		if site.PageCat != nil {
			setCats(site.PageCat, 1601)
		}
	}
	if app := bidRequest.App; app != nil {
		if app.Cat != nil {
			setCats(app.Cat, 1401)
		}
		if app.SectionCat != nil {
			setCats(app.SectionCat, 1501)
		}
		if app.PageCat != nil {
			setCats(app.PageCat, 1601)
		}
	}
	if bidRequest.ACat != nil {
		setCats(bidRequest.ACat, 1701)
	}
	if bidRequest.BCat != nil {
		setCats(bidRequest.BCat, 1801)
	}

	if user := bidRequest.User; user != nil {
		if user.Yob > 0 {
			yob := uint32(user.Yob - 1937)
			ref[1302] = []uint32{yob}
		}
		if user.Gender != "" {
			gender := demo.GenderType_value[user.Gender]
			ref[1301] = []uint32{gender}
		}
	}

	device := bidRequest.Device
	if device != nil {
		var pzgeo *ipsearch.PzGeo
		var err error
		if device.IP != "" {
			if pzgeo, err = self.Ips.CreatePzGeo(device.IP); err != nil {
				return nil, err
			}
			if pzgeo != nil {
				switch {
				case pzgeo.ContinentID > 0:
					ref[1101] = []uint32{uint32(pzgeo.ContinentID)}
				case pzgeo.CountryID > 0:
					ref[1102] = []uint32{uint32(pzgeo.CountryID)}
				case pzgeo.StateCode > 0:
					ref[1103] = []uint32{uint32(pzgeo.StateCode)}
				case pzgeo.CityID > 0:
					ref[1104] = []uint32{uint32(pzgeo.CityID)}
				case pzgeo.DmaID > 0:
					ref[1105] = []uint32{uint32(pzgeo.DmaID)}
				case pzgeo.ZipID > 0:
					ref[1106] = []uint32{uint32(pzgeo.ZipID)}
				case pzgeo.IspID > 0:
					ref[1107] = []uint32{uint32(pzgeo.IspID)}
				case pzgeo.Location.ConnectionType > 0:
					ref[1108] = []uint32{uint32(pzgeo.Location.ConnectionType)}
				case pzgeo.Location.Lat != 0:
					ref[1109] = []uint32{uint32(pzgeo.Location.Lat)}
				case pzgeo.Location.Lon != 0:
					ref[1110] = []uint32{uint32(pzgeo.Location.Lon)}
				case pzgeo.Location.UTCOffset != 0:
					ref[1112] = []uint32{uint32(pzgeo.Location.UTCOffset)}
				default:
				}
			}
		}
		if geo := bidRequest.Device.Geo; geo != nil {
			if geo.Lat != nil && geo.Lon != nil {
				ref[1109] = []uint32{uint32(math.Round(*geo.Lat))}
				ref[1110] = []uint32{uint32(math.Round(*geo.Lon))}
			}
			if geo.Metro != "" {
				if dmaID, ok := ipsearch.MetroMap[geo.Metro]; ok {
					ref[1105] = []uint32{uint32(dmaID)}
				}
			}
			if geo.UTCOffset != 0 {
				ref[1112] = []uint32{uint32(geo.UTCOffset)}
			}
			if pzgeo == nil {
				if geo.Country != "" {
					if countryID, ok := ipsearch.CountryMap[geo.Country]; ok {
						ref[1102] = []uint32{countryID}
					}
				}
				if geo.Region != "" {
					ref[1103] = []uint32{ipsearch.StateCodeToUint32(geo.Region)}
				}
				if geo.City != "" {
					conn := self.Redis
					var cityID uint32
					err = conn.Do(ctx, radix.Cmd(&cityID, "GET", "city:"+geo.Country+":"+geo.Region+":"+geo.City))
					if err != nil {
						return nil, err
					}
					ref[1104] = []uint32{cityID}
				}

			}
		}

		if device.UA != "" {
			pzua := uadevice.GetPzUa(device.UA)
			ref[1203] = []uint32{uint32(pzua.OS)}
			ref[1204] = []uint32{uint32(pzua.OVersion)}
			ref[1205] = []uint32{uint32(pzua.Platform)}
			ref[1206] = []uint32{uint32(pzua.Device)}
		}
	}

	utc := current.UTC()
	ref[1004] = []uint32{uint32(utc.Day())}
	ref[1005] = []uint32{uint32(utc.Hour())}
	ref[1006] = []uint32{uint32(current.Weekday())}
	var z int
	zName := "UTC"
	if i, ok := ref[1112]; ok {
		z = int(i[0])
		zName = fmt.Sprintf("UTC%+d", z)
	}
	loc := time.FixedZone(zName, z*3600)
	localTime := current.In(loc)
	ref[1001] = []uint32{uint32(localTime.Day())}
	ref[1002] = []uint32{uint32(localTime.Hour())}
	ref[1003] = []uint32{uint32(localTime.Weekday())}

	return ref, nil
}

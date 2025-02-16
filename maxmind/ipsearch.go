// Package maxmind provides functionality to search IP addresses and retrieve geographical information.
package maxmind

import (
	"net"

	"github.com/IncSW/geoip2"
	//"github.com/k0kubun/pp/v3"
)

type IPSearch struct {
	filename string
	Reader   *geoip2.CityReader
}

func LoadIPData(fn string) (*IPSearch, error) {
	reader, err := geoip2.NewCityReaderFromFile(fn)
	if err != nil {
		return nil, err
	}
	return &IPSearch{
		filename: fn,
		Reader:   reader,
	}, nil
}

func (self *IPSearch) CreatePzGeo(ip string) (*PzGeo, error) {
	r, err := self.Reader.Lookup(net.ParseIP(ip))
	if err != nil {
		return nil, err
	}

	country, ok := r.Country.Names["zh-CN"]
	if !ok {
		country = r.Country.Names["en"]
	}
	pzg := &PzGeo{
		Continent: r.Continent.Code,
		Country:   country,
	}
	g := Geo{
		CountryID: r.Country.GeoNameID,
		CityID:    r.City.GeoNameID,
		Location: Location{
			Lat:       r.Location.Latitude,
			Lon:       r.Location.Longitude,
			aAccuracy: int64(r.Location.AccuracyRadius),
		},
	}
	if r.Subdivisions != nil {
		pzg.State = r.Subdivisions[0].ISOCode
		g.StateCode = stateCodeToUint32(r.Subdivisions[0].ISOCode)
	}
	if r.City.Names != nil {
		city, ok := r.City.Names["zh-CN"]
		if !ok {
			city = r.City.Names["en"]
		}
		pzg.City = city
	}
	pzg.Geo = g
	return pzg, nil
}

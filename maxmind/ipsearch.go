// Package maxmind provides functionality to search IP addresses and retrieve geographical information.
package maxmind

import (
	"encoding/json"
	"net"
	"os"

	"github.com/IncSW/geoip2"
)

type IPSearch struct {
	CityFile   string                       `json:"city_file"`
	Reader     *geoip2.CityReader           `json:"-"`
	CountryMap map[string]uint32            `json:"country_map,omitempty"`
	StateMap   map[uint32]map[string]uint32 `json:"state_map,omitempty"`
}

func LoadIPData(fn string) (*IPSearch, error) {
	bs, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	ipSearch := &IPSearch{}
	err = json.Unmarshal(bs, ipSearch)
	if err != nil {
		return nil, err
	}

	reader, err := geoip2.NewCityReaderFromFile(ipSearch.CityFile)
	if err != nil {
		return nil, err
	}
	ipSearch.Reader = reader

	return ipSearch, nil
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
			Lat:      r.Location.Latitude,
			Lon:      r.Location.Longitude,
			Accuracy: int64(r.Location.AccuracyRadius),
		},
	}
	if r.Subdivisions != nil {
		pzg.State = r.Subdivisions[0].ISOCode
		g.StateID = self.StateMap[g.CountryID][r.Subdivisions[0].ISOCode]
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

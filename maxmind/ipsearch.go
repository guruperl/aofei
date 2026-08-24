// Package maxmind provides functionality to search IP addresses and retrieve geographical information.
package maxmind

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	cityFile, err := resolveCityFile(fn, ipSearch.CityFile)
	if err != nil {
		return nil, err
	}

	reader, err := geoip2.NewCityReaderFromFile(cityFile)
	if err != nil {
		return nil, err
	}
	ipSearch.CityFile = cityFile
	ipSearch.Reader = reader

	return ipSearch, nil
}

func resolveCityFile(configFile, cityFile string) (string, error) {
	if err := ValidateCityFilePath(cityFile); err != nil {
		return "", err
	}
	clean := filepath.Clean(cityFile)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	return filepath.Join(filepath.Dir(configFile), clean), nil
}

// ValidateCityFilePath rejects ambiguous or unsafe City database paths. A
// relative path is allowed only within the runtime JSON's directory.
func ValidateCityFilePath(cityFile string) error {
	if cityFile == "" || strings.TrimSpace(cityFile) == "" {
		return fmt.Errorf("MaxMind city_file is required")
	}
	if cityFile != strings.TrimSpace(cityFile) {
		return fmt.Errorf("MaxMind city_file must not contain surrounding whitespace")
	}
	if strings.IndexByte(cityFile, 0) >= 0 {
		return fmt.Errorf("MaxMind city_file contains a NUL byte")
	}
	clean := filepath.Clean(cityFile)
	if clean == "." || clean == string(filepath.Separator) {
		return fmt.Errorf("MaxMind city_file %q is not a file path", cityFile)
	}
	if !filepath.IsAbs(clean) && (clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator))) {
		return fmt.Errorf("MaxMind city_file %q escapes its config directory", cityFile)
	}
	return nil
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
			Accuracy:  int64(r.Location.AccuracyRadius),
			UTCOffset: timezoneOffsetMinutes(r.Location.TimeZone),
		},
	}
	if len(r.Subdivisions) > 0 {
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

func timezoneOffsetMinutes(name string) int64 {
	if name == "" {
		return 0
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return 0
	}
	_, offset := time.Now().In(loc).Zone()
	return int64(offset / 60)
}

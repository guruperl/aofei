// Package maxmind provides functionality to search IP addresses and retrieve geographical information.
package maxmind

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/IncSW/geoip2"
	legacyip "github.com/guruperl/aofei/maxmind/ipsearch"
)

type IPSearch struct {
	CityFile   string                       `json:"city_file"`
	Reader     *geoip2.CityReader           `json:"-"`
	Legacy     *legacyip.IPSearch           `json:"-"`
	CountryMap map[string]uint32            `json:"country_map,omitempty"`
	StateMap   map[uint32]map[string]uint32 `json:"state_map,omitempty"`
}

func LoadIPData(fn string) (*IPSearch, error) {
	if strings.EqualFold(filepath.Ext(fn), ".dat") {
		legacy, err := legacyip.LoadIPData(fn)
		if err != nil {
			return nil, err
		}
		return &IPSearch{Legacy: legacy}, nil
	}
	var loaded *IPSearch
	err := withCityPublicationReadLock(fn, func() error {
		var err error
		loaded, err = loadCurrentIPData(fn)
		return err
	})
	return loaded, err
}

func loadCurrentIPData(fn string) (*IPSearch, error) {
	bs, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	ipSearch := &IPSearch{}
	err = json.Unmarshal(bs, ipSearch)
	if err != nil {
		return nil, err
	}
	cityFile, err := ResolveCityFilePath(fn, ipSearch.CityFile)
	if err != nil {
		return nil, err
	}

	reader, err := newCityReaderFromFile(cityFile)
	if err != nil {
		return nil, err
	}
	ipSearch.CityFile = cityFile
	ipSearch.Reader = reader

	return ipSearch, nil
}

// WithCityPublicationLock serializes a complete City generation publication
// against both other publishers and readers loading the selected JSON/asset
// pair.
func WithCityPublicationLock(configPath string, publish func() error) error {
	return withCityPublicationFileLock(configPath, syscall.LOCK_EX, publish)
}

func withCityPublicationReadLock(configPath string, read func() error) error {
	return withCityPublicationFileLock(configPath, syscall.LOCK_SH, read)
}

func withCityPublicationFileLock(configPath string, operation int, work func() error) (err error) {
	if work == nil {
		return errors.New("MaxMind publication operation is nil")
	}
	lockPath := filepath.Join(filepath.Dir(configPath), "."+filepath.Base(configPath)+".lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	locked := false
	defer func() {
		var unlockErr error
		if locked {
			unlockErr = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		}
		err = errors.Join(err, unlockErr, lock.Close())
	}()
	if err := lock.Chmod(0600); err != nil {
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), operation); err != nil {
		return err
	}
	locked = true
	return work()
}

// ResolveCityFilePath resolves a validated City database path relative to its
// runtime JSON. Relative paths cannot escape the JSON directory.
func ResolveCityFilePath(configFile, cityFile string) (string, error) {
	if err := ValidateCityFilePath(cityFile); err != nil {
		return "", err
	}
	clean := filepath.Clean(cityFile)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	return filepath.Join(filepath.Dir(configFile), clean), nil
}

// ValidateCityDatabase parses the database metadata without retaining a file
// descriptor. Panics from malformed third-party decoder input are converted to
// ordinary validation errors.
func ValidateCityDatabase(filename string) error {
	reader, err := newCityReaderFromFile(filename)
	if err != nil {
		return err
	}
	for _, ip := range []string{"0.0.0.0", "127.0.0.1", "255.255.255.255"} {
		if _, err := lookupCity(reader, net.ParseIP(ip)); err != nil && !errors.Is(err, geoip2.ErrNotFound) {
			return fmt.Errorf("validate MaxMind City lookup for %s: %w", ip, err)
		}
	}
	return nil
}

func newCityReaderFromFile(filename string) (reader *geoip2.CityReader, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			reader = nil
			err = fmt.Errorf("malformed MaxMind City database: %v", recovered)
		}
	}()
	reader, err = geoip2.NewCityReaderFromFile(filename)
	if err != nil {
		return nil, fmt.Errorf("load MaxMind City database: %w", err)
	}
	return reader, nil
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
	if self != nil && self.Legacy != nil {
		legacy, err := self.Legacy.CreatePzGeo(ip)
		if err != nil {
			return nil, err
		}
		return &PzGeo{
			Geo: Geo{
				ContinentID: legacy.ContinentID,
				CountryID:   uint32(legacy.CountryID),
				StateID:     uint32(legacy.StateID),
				DmaID:       uint32(legacy.DmaID),
				CityID:      legacy.CityID,
				IspID:       legacy.IspID,
				ZipID:       legacy.ZipID,
				Location:    Location{Lat: legacy.Lat, Lon: legacy.Lon},
			},
			Continent: legacy.Continent,
			Country:   legacy.Country,
			State:     legacy.State,
			Metro:     legacy.Metro,
			City:      legacy.City,
			Zip:       legacy.Zip,
			Isp:       legacy.Isp,
		}, nil
	}
	if self == nil || self.Reader == nil {
		return nil, fmt.Errorf("MaxMind City reader is not loaded")
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address %q", ip)
	}
	r, err := lookupCity(self.Reader, parsedIP)
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

func lookupCity(reader *geoip2.CityReader, ip net.IP) (result *geoip2.CityResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = fmt.Errorf("malformed MaxMind City lookup data: %v", recovered)
		}
	}()
	return reader.Lookup(ip)
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

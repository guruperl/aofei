package maxmind

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestNewOpenRTBGeoFallsBackToIPWithoutDeviceGeo(t *testing.T) {
	p := loadTestIPSearch(t)

	got, err := p.NewOpenRTBGeo(&openrtb2.Device{IP: "128.101.101.101"})
	if err != nil {
		t.Fatal(err)
	}
	if got.CountryID == 0 || got.CityID == 0 || got.Location.UTCOffset == 0 {
		t.Fatalf("NewOpenRTBGeo fallback = %+v, want IP-enriched geo", got)
	}
}

func TestNewOpenRTBGeoPreservesRequestCountryBeforeIPFallback(t *testing.T) {
	p := loadTestIPSearch(t)
	p.CountryMap = map[string]uint32{"ZZ": 12345}

	got, err := p.NewOpenRTBGeo(&openrtb2.Device{
		IP:  "128.101.101.101",
		Geo: &openrtb2.Geo{Country: "ZZ"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.CountryID != 12345 {
		t.Fatalf("CountryID = %d, want request-provided mapped country", got.CountryID)
	}
	if got.CityID == 0 {
		t.Fatalf("CityID = 0, want missing fields filled from IP")
	}
}

func TestNeedsIPGeoIgnoresZeroUTCOffset(t *testing.T) {
	got := needsIPGeo(&Geo{
		CountryID: 1,
		StateID:   2,
		CityID:    3,
		DmaID:     4,
		Location:  Location{UTCOffset: 0},
	})
	if got {
		t.Fatal("needsIPGeo returned true only because UTC offset is zero")
	}
}

func loadTestIPSearch(t *testing.T) *IPSearch {
	t.Helper()
	p, err := LoadIPData(writeTestIPConfig(t, cityMMDBTestPath(t)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func writeTestIPConfig(t *testing.T, cityFile string) string {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "maxmind.json")
	bs, err := json.Marshal(IPSearch{CityFile: cityFile})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, bs, 0600); err != nil {
		t.Fatal(err)
	}
	return configFile
}

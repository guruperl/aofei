package maxmind

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	//"pzutil"
	//"database/sql"
	//_ "github.com/go-sql-driver/mysql"
)

func TestIpsearch(t *testing.T) {
	cityFile := cityMMDBTestPath(t)

	configFile := filepath.Join(t.TempDir(), "maxmind.json")
	config := IPSearch{CityFile: cityFile}
	bs, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, bs, 0600); err != nil {
		t.Fatal(err)
	}

	//p, err := LoadIPData("../etc/qqzeng-ip-utf8.dat");
	p, err := LoadIPData(configFile)
	if err != nil {
		t.Fatal(err)
	}

	ip := "128.101.101.101"

	pzgeo, err := p.CreatePzGeo(ip)
	if err != nil {
		t.Fatal(err)
	}
	if pzgeo.Continent != "NA" || pzgeo.Country == "" || pzgeo.State == "" || pzgeo.City == "" {
		t.Fatalf("CreatePzGeo(%s) = %+v, want populated city-level result", ip, pzgeo)
	}
	if pzgeo.Geo.CountryID == 0 || pzgeo.Geo.CityID == 0 || pzgeo.Geo.Location.Lat == 0 || pzgeo.Geo.Location.Lon == 0 {
		t.Fatalf("CreatePzGeo(%s) geo = %+v, want populated GeoName and location fields", ip, pzgeo.Geo)
	}
}

func TestResolveCityFile(t *testing.T) {
	config := filepath.Join(t.TempDir(), "maxmind.json")
	got, err := resolveCityFile(config, "GeoLite2-City.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(config), "GeoLite2-City.mmdb")
	if got != want {
		t.Fatalf("resolveCityFile() = %q, want %q", got, want)
	}

	absolute := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")
	got, err = resolveCityFile(config, absolute)
	if err != nil {
		t.Fatal(err)
	}
	if got != absolute {
		t.Fatalf("resolveCityFile() = %q, want %q", got, absolute)
	}

	for _, path := range []string{"", ".", "..", "../GeoLite2-City.mmdb", "/", " GeoLite2-City.mmdb"} {
		if _, err := resolveCityFile(config, path); err == nil {
			t.Errorf("resolveCityFile(%q) error = nil", path)
		}
	}
}

func cityMMDBTestPath(t *testing.T) string {
	t.Helper()

	if path := os.Getenv("AOFEI_GEOLITE_CITY_FILE"); path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("AOFEI_GEOLITE_CITY_FILE=%s: %v", path, err)
		}
		return path
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	root := filepath.Dir(filepath.Dir(filename))
	for _, path := range []string{
		filepath.Join(root, "external", "GeoLite2-City.mmdb"),
		filepath.Join(root, "etc", "GeoLite2-City.mmdb"),
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}

	t.Skip("requires GeoLite2 City MMDB at AOFEI_GEOLITE_CITY_FILE, external/GeoLite2-City.mmdb, or etc/GeoLite2-City.mmdb")
	return ""
}

package maxmind

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	legacyip "github.com/guruperl/aofei/maxmind/ipsearch"
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
	got, err := ResolveCityFilePath(config, "GeoLite2-City.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(config), "GeoLite2-City.mmdb")
	if got != want {
		t.Fatalf("resolveCityFile() = %q, want %q", got, want)
	}

	absolute := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")
	got, err = ResolveCityFilePath(config, absolute)
	if err != nil {
		t.Fatal(err)
	}
	if got != absolute {
		t.Fatalf("resolveCityFile() = %q, want %q", got, absolute)
	}

	for _, path := range []string{"", ".", "..", "../GeoLite2-City.mmdb", "/", " GeoLite2-City.mmdb"} {
		if _, err := ResolveCityFilePath(config, path); err == nil {
			t.Errorf("resolveCityFile(%q) error = nil", path)
		}
	}
}

func TestMalformedCityDatabaseReturnsError(t *testing.T) {
	dir := t.TempDir()
	cityFile := filepath.Join(dir, "broken.mmdb")
	if err := os.WriteFile(cityFile, []byte("not an mmdb"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCityDatabase(cityFile); err == nil {
		t.Fatal("ValidateCityDatabase() error = nil")
	}
	configFile := filepath.Join(dir, "maxmind.json")
	data, err := json.Marshal(IPSearch{CityFile: "broken.mmdb"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadIPData(configFile); err == nil {
		t.Fatal("LoadIPData() error = nil")
	}
}

func TestLoadIPDataUsesStrictLegacyDatFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.dat")
	writeLegacyDatFixture(t, path)
	search, err := LoadIPData(path)
	if err != nil {
		t.Fatal(err)
	}
	if search.Reader != nil || search.Legacy == nil {
		t.Fatalf("legacy search = %+v", search)
	}
	geo, err := search.CreatePzGeo("1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if geo.CountryID != 2 || geo.StateID != 3 || geo.CityID != 5 || geo.Country != "country" {
		t.Fatalf("legacy geo = %+v", geo)
	}

	if err := os.WriteFile(path, []byte{1, 2, 3}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadIPData(path); err == nil {
		t.Fatal("malformed legacy LoadIPData() error = nil")
	}
}

func writeLegacyDatFixture(t *testing.T, path string) {
	t.Helper()
	var location bytes.Buffer
	if err := binary.Write(&location, binary.LittleEndian, legacyip.Geo{
		ContinentID: 1,
		CountryID:   2,
		StateID:     3,
		DmaID:       4,
		CityID:      5,
		IspID:       6,
		ZipID:       7,
		Lat:         8.5,
		Lon:         9.5,
	}); err != nil {
		t.Fatal(err)
	}
	location.WriteString("continent|country|state|metro|city|zip|isp")
	firstOffset := uint32(16 + location.Len())
	prefixStart := firstOffset + 12
	data := make([]byte, 16, int(prefixStart)+9)
	binary.LittleEndian.PutUint32(data[0:4], firstOffset)
	binary.LittleEndian.PutUint32(data[4:8], firstOffset)
	binary.LittleEndian.PutUint32(data[8:12], prefixStart)
	binary.LittleEndian.PutUint32(data[12:16], prefixStart)
	data = append(data, location.Bytes()...)
	index := make([]byte, 12)
	binary.LittleEndian.PutUint32(index[0:4], 0x01020300)
	binary.LittleEndian.PutUint32(index[4:8], 0x010203ff)
	index[8], index[9], index[10] = 16, 0, 0
	index[11] = byte(location.Len())
	data = append(data, index...)
	prefix := make([]byte, 9)
	prefix[0] = 1
	data = append(data, prefix...)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
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

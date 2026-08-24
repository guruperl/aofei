package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guruperl/aofei/maxmind"
)

const maxmindTestDriverName = "maxmindtest"

func init() {
	sql.Register(maxmindTestDriverName, maxmindTestDriver{})
}

func TestRunWritesGeneratedConfig(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "maxmind.json")
	configPath := writeTestConfig(t, dir, outputPath, maxmindTestDriverName, "ok")

	if err := run(context.Background(), configPath, "GeoLite2-City.mmdb"); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := readIPSearchConfig(t, outputPath)
	if got.CityFile != "GeoLite2-City.mmdb" {
		t.Fatalf("CityFile = %q", got.CityFile)
	}
	if got.CountryMap["CAN"] != 6251999 {
		t.Fatalf("CountryMap[CAN] = %d", got.CountryMap["CAN"])
	}
	if got.StateMap[6251999]["ON"] != 6093943 {
		t.Fatalf("StateMap[6251999][ON] = %d", got.StateMap[6251999]["ON"])
	}
}

func TestRunReturnsConfigAndDatabaseErrors(t *testing.T) {
	t.Run("missing config path", func(t *testing.T) {
		err := run(context.Background(), "", "GeoLite2-City.mmdb")
		if err == nil || !strings.Contains(err.Error(), "DSP config path is required") {
			t.Fatalf("run() error = %v", err)
		}
	})

	t.Run("missing city path", func(t *testing.T) {
		err := run(context.Background(), "unused", "")
		if err == nil || !strings.Contains(err.Error(), "city mmdb path is required") {
			t.Fatalf("run() error = %v", err)
		}
	})

	t.Run("city path whitespace", func(t *testing.T) {
		err := run(context.Background(), "unused", " GeoLite2-City.mmdb")
		if err == nil || !strings.Contains(err.Error(), "surrounding whitespace") {
			t.Fatalf("run() error = %v", err)
		}
	})

	t.Run("city path escape", func(t *testing.T) {
		err := run(context.Background(), "unused", "../GeoLite2-City.mmdb")
		if err == nil || !strings.Contains(err.Error(), "escapes its config directory") {
			t.Fatalf("run() error = %v", err)
		}
	})

	t.Run("unknown database driver", func(t *testing.T) {
		dir := t.TempDir()
		configPath := writeTestConfig(t, dir, filepath.Join(dir, "maxmind.json"), "missing-driver", "unused")
		err := run(context.Background(), configPath, "GeoLite2-City.mmdb")
		if err == nil || !strings.Contains(err.Error(), "open database") {
			t.Fatalf("run() error = %v", err)
		}
	})
}

func TestBuildIPSearchReturnsQueryErrors(t *testing.T) {
	t.Run("country query", func(t *testing.T) {
		db := openTestDB(t, "country_error")
		defer db.Close()
		_, err := buildIPSearch(context.Background(), db, "GeoLite2-City.mmdb")
		if err == nil || !strings.Contains(err.Error(), "query def_country") {
			t.Fatalf("buildIPSearch() error = %v", err)
		}
	})

	t.Run("state query", func(t *testing.T) {
		db := openTestDB(t, "state_error")
		defer db.Close()
		_, err := buildIPSearch(context.Background(), db, "GeoLite2-City.mmdb")
		if err == nil || !strings.Contains(err.Error(), "query def_state") {
			t.Fatalf("buildIPSearch() error = %v", err)
		}
	})
}

func TestWriteIPSearchAtomic(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "maxmind.json")
	input := &maxmind.IPSearch{
		CityFile:   "GeoLite2-City.mmdb",
		CountryMap: map[string]uint32{"CAN": 6251999},
		StateMap: map[uint32]map[string]uint32{
			6251999: map[string]uint32{"ON": 6093943},
		},
	}

	if err := writeIPSearchAtomic(outputPath, input); err != nil {
		t.Fatalf("writeIPSearchAtomic() error = %v", err)
	}

	got := readIPSearchConfig(t, outputPath)
	if got.CityFile != input.CityFile || got.CountryMap["CAN"] != input.CountryMap["CAN"] {
		t.Fatalf("written config = %+v", got)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".maxmind.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0640 {
		t.Fatalf("output mode = %04o, want 0640", got)
	}
}

func TestWriteIPSearchAtomicReturnsWritePathError(t *testing.T) {
	err := writeIPSearchAtomic(filepath.Join(t.TempDir(), "missing", "maxmind.json"), &maxmind.IPSearch{})
	if err == nil {
		t.Fatal("writeIPSearchAtomic() error = nil")
	}
}

func writeTestConfig(t *testing.T, dir, ipsPath, driverName, dsn string) string {
	t.Helper()
	configPath := filepath.Join(dir, "aofei.json")
	content := map[string]any{
		"ips":           ipsPath,
		"connect_array": []string{driverName, dsn},
	}
	bs, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, bs, 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func readIPSearchConfig(t *testing.T, path string) *maxmind.IPSearch {
	t.Helper()
	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := &maxmind.IPSearch{}
	if err := json.Unmarshal(bs, got); err != nil {
		t.Fatal(err)
	}
	return got
}

func openTestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open(maxmindTestDriverName, dsn)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

type maxmindTestDriver struct{}

func (maxmindTestDriver) Open(name string) (driver.Conn, error) {
	if name == "open_error" {
		return nil, errors.New("open error")
	}
	return maxmindTestConn{dsn: name}, nil
}

type maxmindTestConn struct {
	dsn string
}

func (maxmindTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not implemented")
}

func (maxmindTestConn) Close() error {
	return nil
}

func (maxmindTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not implemented")
}

func (c maxmindTestConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "def_country") {
		if c.dsn == "country_error" {
			return nil, errors.New("country query error")
		}
		return &maxmindTestRows{
			columns: []string{"country_id", "alpha3"},
			values: [][]driver.Value{
				{int64(6251999), "CAN"},
				{int64(1814991), "CHN"},
			},
		}, nil
	}
	if strings.Contains(query, "def_state") {
		if c.dsn == "state_error" {
			return nil, errors.New("state query error")
		}
		return &maxmindTestRows{
			columns: []string{"country_id", "state_code", "state_id"},
			values: [][]driver.Value{
				{int64(6251999), "ON", int64(6093943)},
				{int64(1814991), "HB", int64(1806949)},
			},
		}, nil
	}
	return nil, errors.New("unexpected query")
}

type maxmindTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *maxmindTestRows) Columns() []string {
	return r.columns
}

func (r *maxmindTestRows) Close() error {
	return nil
}

func (r *maxmindTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

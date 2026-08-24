package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	cityPath := filepath.Join(dir, "GeoLite2-City.mmdb")
	if err := os.WriteFile(cityPath, []byte("city fixture"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := runWithCityValidator(context.Background(), configPath, "GeoLite2-City.mmdb", func(staged string) error {
		data, err := os.ReadFile(staged)
		if err != nil {
			return err
		}
		if string(data) != "city fixture" {
			return errors.New("unexpected staged City data")
		}
		return nil
	}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := readIPSearchConfig(t, outputPath)
	if !strings.HasPrefix(filepath.ToSlash(got.CityFile), ".maxmind.json.generations/") {
		t.Fatalf("CityFile = %q, want managed relative generation", got.CityFile)
	}
	managedCity, err := maxmind.ResolveCityFilePath(outputPath, got.CityFile)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(managedCity); err != nil || string(data) != "city fixture" {
		t.Fatalf("managed City data = %q, %v", data, err)
	}
	if got.CountryMap["CAN"] != 6251999 {
		t.Fatalf("CountryMap[CAN] = %d", got.CountryMap["CAN"])
	}
	if got.StateMap[6251999]["ON"] != 6093943 {
		t.Fatalf("StateMap[6251999][ON] = %d", got.StateMap[6251999]["ON"])
	}
}

func TestPublishIPSearchGenerationPreservesCurrentAndPrior(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "maxmind.json")
	validate := func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(data) == "invalid" {
			return errors.New("invalid City database")
		}
		return nil
	}
	publish := func(name, data string) error {
		source := filepath.Join(dir, name)
		if err := os.WriteFile(source, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		return publishIPSearchGeneration(configPath, &maxmind.IPSearch{
			CityFile:   name,
			CountryMap: map[string]uint32{"CAN": uint32(len(data))},
		}, validate)
	}

	if err := publish("first.mmdb", "first"); err != nil {
		t.Fatal(err)
	}
	firstConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	first := readIPSearchConfig(t, configPath)
	firstAsset, err := maxmind.ResolveCityFilePath(configPath, first.CityFile)
	if err != nil {
		t.Fatal(err)
	}

	if err := publish("invalid.mmdb", "invalid"); err == nil || !strings.Contains(err.Error(), "validate staged") {
		t.Fatalf("invalid publish error = %v", err)
	}
	stillCurrent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stillCurrent, firstConfig) {
		t.Fatal("failed validation replaced the selected config")
	}
	if _, err := os.Stat(firstAsset); err != nil {
		t.Fatalf("current asset after failed validation: %v", err)
	}

	if err := publish("second.mmdb", "second"); err != nil {
		t.Fatal(err)
	}
	second := readIPSearchConfig(t, configPath)
	if second.CityFile == first.CityFile {
		t.Fatal("second generation did not change City asset")
	}
	secondAsset, err := maxmind.ResolveCityFilePath(configPath, second.CityFile)
	if err != nil {
		t.Fatal(err)
	}
	assertGenerationCount(t, cityAssetRoot(configPath), 2)
	if _, err := os.Stat(firstAsset); err != nil {
		t.Fatalf("prior asset after second publish: %v", err)
	}

	if err := publish("third.mmdb", "third"); err != nil {
		t.Fatal(err)
	}
	assertGenerationCount(t, cityAssetRoot(configPath), 2)
	if _, err := os.Stat(firstAsset); !os.IsNotExist(err) {
		t.Fatalf("oldest generation still present, stat error = %v", err)
	}

	if err := publish("third.mmdb", "third"); err != nil {
		t.Fatal(err)
	}
	assertGenerationCount(t, cityAssetRoot(configPath), 2)
	if _, err := os.Stat(secondAsset); err != nil {
		t.Fatalf("prior asset after same-content republish: %v", err)
	}
}

func TestCityPublicationLockSerializesPublishers(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "maxmind.json")
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	secondEntered := make(chan struct{})
	var active atomic.Int32
	var overlap atomic.Bool
	done := make(chan error, 2)

	go func() {
		done <- withCityPublicationLock(configPath, func() error {
			if active.Add(1) != 1 {
				overlap.Store(true)
			}
			close(firstEntered)
			<-releaseFirst
			active.Add(-1)
			return nil
		})
	}()
	select {
	case <-firstEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("first publisher did not acquire lock")
	}
	go func() {
		close(secondStarted)
		done <- withCityPublicationLock(configPath, func() error {
			if active.Add(1) != 1 {
				overlap.Store(true)
			}
			active.Add(-1)
			close(secondEntered)
			return nil
		})
	}()
	<-secondStarted
	select {
	case <-secondEntered:
		t.Fatal("second publisher entered before first released the lock")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case <-secondEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("second publisher did not acquire released lock")
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if overlap.Load() {
		t.Fatal("publishers overlapped inside the file lock")
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

func TestCopyHashedFileMismatchPreservesTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mmdb")
	target := filepath.Join(dir, "target.mmdb")
	if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := copyHashedFile(source, target, strings.Repeat("0", 64)); err == nil || !strings.Contains(err.Error(), "changed while it was staged") {
		t.Fatalf("copyHashedFile() error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("target = %q, want old", data)
	}
}

func assertGenerationCount(t *testing.T, root string, want int) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && validGenerationDigest(entry.Name()) {
			count++
		}
	}
	if count != want {
		t.Fatalf("generation count = %d, want %d", count, want)
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

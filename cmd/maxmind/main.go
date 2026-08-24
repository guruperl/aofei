// this runs once, to generate country map and state map
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/atomicfile"
	"github.com/guruperl/aofei/maxmind"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: maxmind -s=dsp_config -city=city.mmdb\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var city string

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.StringVar(&city, "city", os.Getenv("AOFEI_GEOLITE_CITY_FILE"), "MaxMind City MMDB path (or AOFEI_GEOLITE_CITY_FILE)")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	if err := run(ctx, sConf, city); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, configPath, cityPath string) error {
	return runWithCityValidator(ctx, configPath, cityPath, maxmind.ValidateCityDatabase)
}

func runWithCityValidator(ctx context.Context, configPath, cityPath string, validate func(string) error) error {
	if configPath == "" {
		return errors.New("DSP config path is required; set AOFEI or pass -s")
	}
	if cityPath == "" {
		return errors.New("city mmdb path is required; pass -city or set AOFEI_GEOLITE_CITY_FILE")
	}
	if cityPath != strings.TrimSpace(cityPath) {
		return errors.New("city mmdb path must not contain surrounding whitespace")
	}
	if err := maxmind.ValidateCityFilePath(cityPath); err != nil {
		return err
	}
	c, err := dsp.NewConfig(configPath)
	if err != nil {
		return fmt.Errorf("load DSP config %q: %w", configPath, err)
	}
	if c.Ips == "" {
		return errors.New("DSP config ips path is empty")
	}
	if err := c.Validate(dsp.ConfigModeMaxMind); err != nil {
		return err
	}
	db, err := c.OpenDB(ctx)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	ipSearch, err := buildIPSearch(ctx, db, cityPath)
	if err != nil {
		return err
	}

	log.Printf("Writing country and state maps to %s", c.Ips)
	if err := publishIPSearchGeneration(c.Ips, ipSearch, validate); err != nil {
		return fmt.Errorf("publish MaxMind generation %q: %w", c.Ips, err)
	}
	return nil
}

func buildIPSearch(ctx context.Context, db *sql.DB, cityPath string) (*maxmind.IPSearch, error) {
	countryMap, err := loadCountryMap(ctx, db)
	if err != nil {
		return nil, err
	}
	stateMap, err := loadStateMap(ctx, db)
	if err != nil {
		return nil, err
	}
	ipSearch := &maxmind.IPSearch{
		CityFile:   cityPath,
		CountryMap: countryMap,
		StateMap:   stateMap,
	}
	return ipSearch, nil
}

func loadCountryMap(ctx context.Context, db *sql.DB) (map[string]uint32, error) {
	rows, err := db.QueryContext(ctx, `
SELECT country_id, alpha3 FROM def_country`)
	if err != nil {
		return nil, fmt.Errorf("query def_country: %w", err)
	}
	defer rows.Close()

	countryMap := make(map[string]uint32)
	for rows.Next() {
		var countryID uint32
		var alpha3 string
		if err := rows.Scan(&countryID, &alpha3); err != nil {
			return nil, fmt.Errorf("scan def_country: %w", err)
		}
		countryMap[alpha3] = countryID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read def_country rows: %w", err)
	}
	return countryMap, nil
}

func loadStateMap(ctx context.Context, db *sql.DB) (map[uint32]map[string]uint32, error) {
	rows, err := db.QueryContext(ctx, `
SELECT country_id, state_code, state_id 
FROM def_state`)
	if err != nil {
		return nil, fmt.Errorf("query def_state: %w", err)
	}
	defer rows.Close()

	stateMap := make(map[uint32]map[string]uint32)
	for rows.Next() {
		var stateCode string
		var countryID, stateID uint32
		if err := rows.Scan(&countryID, &stateCode, &stateID); err != nil {
			return nil, fmt.Errorf("scan def_state: %w", err)
		}
		if _, ok := stateMap[countryID]; !ok {
			stateMap[countryID] = make(map[string]uint32)
		}
		stateMap[countryID][stateCode] = stateID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read def_state rows: %w", err)
	}
	return stateMap, nil
}

func writeIPSearchAtomic(filename string, ipSearch *maxmind.IPSearch) error {
	return atomicfile.Write(filename, 0640, func(out io.Writer) error {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(ipSearch)
	})
}

func publishIPSearchGeneration(filename string, ipSearch *maxmind.IPSearch, validate func(string) error) error {
	return withCityPublicationLock(filename, func() error {
		return publishIPSearchGenerationLocked(filename, ipSearch, validate)
	})
}

func publishIPSearchGenerationLocked(filename string, ipSearch *maxmind.IPSearch, validate func(string) error) error {
	if ipSearch == nil {
		return errors.New("MaxMind config is nil")
	}
	if validate == nil {
		return errors.New("MaxMind City validator is nil")
	}
	source, err := maxmind.ResolveCityFilePath(filename, ipSearch.CityFile)
	if err != nil {
		return err
	}
	digest, err := hashFile(source)
	if err != nil {
		return fmt.Errorf("hash City database: %w", err)
	}
	assetRoot := cityAssetRoot(filename)
	generationRoot := filepath.Join(assetRoot, digest)
	if err := atomicfile.EnsureDir(generationRoot, 0750); err != nil {
		return fmt.Errorf("create City generation: %w", err)
	}
	assetPath := filepath.Join(generationRoot, "GeoLite2-City.mmdb")
	if err := copyHashedFile(source, assetPath, digest); err != nil {
		return fmt.Errorf("stage City database: %w", err)
	}
	if err := validate(assetPath); err != nil {
		return fmt.Errorf("validate staged City database: %w", err)
	}

	prior := selectedCityGeneration(filename, assetRoot)
	kept := []string{digest, prior}
	if prior == digest {
		rollback, err := newestValidCityGeneration(assetRoot, digest, validate)
		if err != nil {
			return fmt.Errorf("select prior City generation: %w", err)
		}
		kept = append(kept, rollback)
	}
	if err := pruneCityGenerations(assetRoot, kept...); err != nil {
		return fmt.Errorf("prune City generations: %w", err)
	}
	relativeAsset, err := filepath.Rel(filepath.Dir(filename), assetPath)
	if err != nil {
		return err
	}
	if err := maxmind.ValidateCityFilePath(relativeAsset); err != nil {
		return err
	}
	published := *ipSearch
	published.CityFile = relativeAsset
	return writeIPSearchAtomic(filename, &published)
}

func newestValidCityGeneration(assetRoot, exclude string, validate func(string) error) (string, error) {
	type candidate struct {
		name    string
		modTime int64
	}
	entries, err := os.ReadDir(assetRoot)
	if err != nil {
		return "", err
	}
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == exclude || !validGenerationDigest(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		candidates = append(candidates, candidate{name: entry.Name(), modTime: info.ModTime().UnixNano()})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime != candidates[j].modTime {
			return candidates[i].modTime > candidates[j].modTime
		}
		return candidates[i].name > candidates[j].name
	})
	for _, candidate := range candidates {
		asset := filepath.Join(assetRoot, candidate.name, "GeoLite2-City.mmdb")
		if validate(asset) == nil {
			return candidate.name, nil
		}
	}
	return "", nil
}

func withCityPublicationLock(configPath string, publish func() error) (err error) {
	return maxmind.WithCityPublicationLock(configPath, publish)
}

func cityAssetRoot(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "."+filepath.Base(configPath)+".generations")
}

func hashFile(filename string) (digest string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func copyHashedFile(source, target, expectedDigest string) error {
	return atomicfile.Write(target, 0640, func(out io.Writer) error {
		in, err := os.Open(source)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(io.MultiWriter(out, hash), in)
		closeErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if got := fmt.Sprintf("%x", hash.Sum(nil)); got != expectedDigest {
			return errors.New("City database changed while it was staged")
		}
		return nil
	})
}

func selectedCityGeneration(configPath, assetRoot string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var current maxmind.IPSearch
	if err := json.Unmarshal(data, &current); err != nil {
		return ""
	}
	cityPath, err := maxmind.ResolveCityFilePath(configPath, current.CityFile)
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(assetRoot, cityPath)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) != 2 || parts[1] != "GeoLite2-City.mmdb" || !validGenerationDigest(parts[0]) {
		return ""
	}
	return parts[0]
}

func pruneCityGenerations(assetRoot string, keep ...string) error {
	kept := make(map[string]struct{}, len(keep))
	for _, generation := range keep {
		if generation != "" {
			kept[generation] = struct{}{}
		}
	}
	entries, err := os.ReadDir(assetRoot)
	if err != nil {
		return err
	}
	removed := false
	for _, entry := range entries {
		if !entry.IsDir() || !validGenerationDigest(entry.Name()) {
			continue
		}
		if _, ok := kept[entry.Name()]; ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(assetRoot, entry.Name())); err != nil {
			return err
		}
		removed = true
	}
	if removed {
		return atomicfile.SyncDir(assetRoot)
	}
	return nil
}

func validGenerationDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

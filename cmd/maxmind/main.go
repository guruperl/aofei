// this runs once, to generate country map and state map
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/maxmind"
	_ "github.com/go-sql-driver/mysql"
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
	flag.StringVar(&city, "city", "/media/GeoLite2-City.mmdb", "Maxmind city mmdb file")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	if err := run(ctx, sConf, city); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, configPath, cityPath string) error {
	if configPath == "" {
		return errors.New("DSP config path is required; set AOFEI or pass -s")
	}
	if cityPath == "" {
		return errors.New("city mmdb path is required; pass -city")
	}
	c, err := dsp.NewConfig(configPath)
	if err != nil {
		return fmt.Errorf("load DSP config %q: %w", configPath, err)
	}
	if c.Ips == "" {
		return errors.New("DSP config ips path is empty")
	}
	if len(c.ConnectArray) < 2 {
		return errors.New("DSP config connect_array must contain driver and DSN")
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	ipSearch, err := buildIPSearch(ctx, db, cityPath)
	if err != nil {
		return err
	}

	log.Printf("Writing country and state maps to %s", c.Ips)
	if err := writeIPSearchAtomic(c.Ips, ipSearch); err != nil {
		return fmt.Errorf("write MaxMind config %q: %w", c.Ips, err)
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
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	fh, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := fh.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpName)
		}
	}()

	encoder := json.NewEncoder(fh)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ipSearch); err != nil {
		_ = fh.Close()
		return err
	}
	if err := fh.Sync(); err != nil {
		_ = fh.Close()
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return err
	}
	removeTmp = false
	return nil
}

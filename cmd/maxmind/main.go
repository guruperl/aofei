// this runs once, to generate country map and state map
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/maxmind"
	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: maxmind --s=dsp_config --city=city.mmdb\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var city string

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.StringVar(&city, "city", "/media/GeoLite2-City.mmdb", "Maxmind city mmdb file")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	sc, err := dsp.NewController(ctx, sConf)
	if err != nil {
		log.Fatal(err)
	}

	db := sc.DB
	defer sc.Close()

	rows, err := db.QueryContext(ctx, `
SELECT country_id, alpha3 FROM def_country`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var CountryMap = make(map[string]uint32)
	for rows.Next() {
		var countryID uint32
		var alpha3 string
		err = rows.Scan(&countryID, &alpha3)
		if err != nil {
			panic(err)
		}
		CountryMap[alpha3] = countryID
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}

	rows, err = db.QueryContext(ctx, `
SELECT country_id, state_code, state_id 
FROM def_state`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	StateMap := make(map[uint32]map[string]uint32)
	for rows.Next() {
		var stateCode string
		var countryID, stateID uint32
		err = rows.Scan(&countryID, &stateCode, &stateID)
		if err != nil {
			log.Fatal(err)
		}
		if _, ok := StateMap[countryID]; !ok {
			StateMap[countryID] = make(map[string]uint32)
		}
		StateMap[countryID][stateCode] = stateID
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	ipSearch := &maxmind.IPSearch{
		CityFile:   city,
		CountryMap: CountryMap,
		StateMap:   StateMap,
	}

	log.Printf("Writing country and state maps to %s", sc.C.Ips)
	fh, err := os.Create(sc.C.Ips)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()
	encoder := json.NewEncoder(fh)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(ipSearch)
	if err != nil {
		log.Fatal(err)
	}
}

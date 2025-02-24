package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/genelet/winter/genelet"
)

func doCountry(ctx context.Context, db *sql.DB) error {
	dbi := &genelet.DBI{Db: db}
	lists := make([]map[string]interface{}, 0)
	err := dbi.Select_sql(&lists, `
SELECT continent_id, continent_code
FROM def_continent`)
	if err != nil {
		return err
	}
	hash := make(map[string]interface{})
	for _, v := range lists {
		hash[v["continent_code"].(string)] = v["continent_id"]
	}

	f, err := os.Open("GeoLite2-Country-Locations-zh-CN.csv")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip first line
	for scanner.Scan() {
		line := scanner.Text()
		arr := strings.Split(line, `,`)
		if len(arr) != 7 {
			continue
		}
		continentID, ok := hash[arr[2]]
		if !ok {
			fmt.Printf("wrong continent: %v\n", arr)
			continue
		}
		if arr[5] == "" {
			fmt.Printf("wrong country: %v\n", arr)
			continue
		}
		isEuro := "No"
		if arr[6] == "1" {
			isEuro = "Yes"
		}
		_, err := db.ExecContext(ctx, `
INSERT INTO def_country (country_id, country_code, country_name, locale_code, continent_id, is_euro)
VALUES (?, ?, ?, ?, ?, ?)`, arr[0], arr[4], arr[5], arr[1], continentID, isEuro)
		if err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// IN ('US','CN','DE','FR','GB','AU','CA','JP','RU','KR','IN','BR','IT','NL','ES')`)
	_, err = db.ExecContext(ctx, `
UPDATE def_country SET active='Yes' 
WHERE country_code
IN ('US')`)

	return err
}

func doState(ctx context.Context, db *sql.DB) error {
	dbi := &genelet.DBI{Db: db}
	lists := make([]map[string]interface{}, 0)
	hash := make(map[string]interface{})
	err := dbi.Select_sql(&lists, `
SELECT country_id, country_code
FROM def_country`)
	if err != nil {
		return err
	}
	for _, v := range lists {
		hash[v["country_code"].(string)] = v["country_id"]
	}

	r, err := os.Open("GeoLite2-City-Locations-zh-CN.csv")
	if err != nil {
		return err
	}
	defer r.Close()

	names := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Scan() // skip first line
	for scanner.Scan() {
		line := scanner.Text()
		arr := strings.Split(line, `,`)
		if arr[7] != "" {
			names[arr[0]] = arr[7]
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	f, err := os.Open("GeoLite2-City-Locations-en.csv")
	if err != nil {
		return err
	}
	defer f.Close()

	ref := make(map[string]bool)

	re := regexp.MustCompile(`^(\d+),([a-zA-Z]+),([a-zA-Z]+),(("[^"]+")|([^,]+)),([a-zA-Z]+),(("[^"]+")|([^,]+)),([a-zA-Z0-9]*),(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(0|1)$`)
	scanner = bufio.NewScanner(f)
	scanner.Scan() // skip first line
	for scanner.Scan() {
		line := scanner.Text()
		arr := re.FindStringSubmatch(line)
		if len(arr) != 31 {
			continue
		}
		country := arr[7]
		countryID, ok := hash[country]
		if !ok {
			fmt.Printf("wrong country: %v\n", arr)
			continue
		}
		state := arr[11]
		if state == "" {
			continue
		}
		stateName := arr[14]
		if stateName == "" {
			stateName = state
		} else {
			stateName = strings.Trim(stateName, `"`)
		}
		englishName := stateName
		if name, ok := names[arr[1]]; ok {
			stateName = name
		}
		if _, ok := ref[country+"-"+state]; ok {
			continue
		}
		ref[country+"-"+state] = true
		//city := arr[23]
		//dma := arr[26]
		//fmt.Printf("geoname id: %s, country_id: %d, state_code: %s, state_name: %s, english_name: %s\n", arr[1], countryID, state, stateName, englishName)
		//continue
		_, err := db.ExecContext(ctx, `
INSERT INTO def_state (country_id, state_code, state_name, english_name)
VALUES (?,?,?,?)`, countryID, state, stateName, englishName)
		if err != nil {
			return fmt.Errorf("state %d, state_code=%s, %#v => %#v", countryID, state, arr, err)
		}
	}
	return scanner.Err()
}

func doCity(ctx context.Context, db *sql.DB) error {
	f, err := os.Open("metrocodes.csv")
	if err != nil {
		return err
	}
	defer f.Close()

	description := make(map[string]interface{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		arr := strings.Split(line, `,`)
		if len(arr) != 3 {
			continue
		}
		metro := strings.Trim(arr[2], `"`)
		description[metro] = strings.Trim(arr[1], `"`)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	dbi := &genelet.DBI{Db: db}
	lists := make([]map[string]interface{}, 0)
	err = dbi.Select_sql(&lists, `
SELECT s.state_id, s.state_code, c.country_code
FROM def_state s
INNER JOIN def_country c USING (country_id)`)
	if err != nil {
		return err
	}
	hash := make(map[string]map[string]interface{})
	for _, v := range lists {
		if _, ok := hash[v["country_code"].(string)]; !ok {
			hash[v["country_code"].(string)] = make(map[string]interface{})
		}
		hash[v["country_code"].(string)][v["state_code"].(string)] = v["state_id"]
	}

	g, err := os.Open("GeoLite2-City-Locations-en.csv")
	if err != nil {
		return err
	}
	defer g.Close()

	re := regexp.MustCompile(`^(\d+),([a-zA-Z]+),([a-zA-Z]+),(("[^"]+")|([^,]+)),([a-zA-Z]+),(("[^"]+")|([^,]+)),([a-zA-Z0-9]*),(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(("[^"]+")|([^,]+))?,(0|1)$`)
	scanner = bufio.NewScanner(g)
	scanner.Scan() // skip first line
	for scanner.Scan() {
		line := scanner.Text()
		arr := re.FindStringSubmatch(line)
		if len(arr) != 31 {
			continue
		}
		country := arr[7]
		if _, ok := hash[country]; !ok {
			continue
		}
		state := arr[11]
		if state == "" {
			continue
		}
		city := arr[23]
		if city != "" {
			city = strings.Trim(city, `"`)
		}
		stateID, ok := hash[country][state]
		if !ok {
			fmt.Printf("wrong state: %v\n", arr)
			continue
		}
		_, err := db.ExecContext(ctx, `
INSERT INTO def_city (city_id, state_id, city_name)
VALUES (?,?,?)`, arr[1], stateID, city)
		if err != nil {
			return err
		}
		if country == "US" && arr[26] != "" {
			dma := arr[26]
			if desc, ok := description[dma]; ok {
				_, err = db.ExecContext(ctx, `
INSERT INTO def_dma (city_id, metro_code, description)
VALUES (?,?,?)`, arr[1], dma, desc)
				if err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

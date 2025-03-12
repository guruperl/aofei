package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "eightran_goto:12pass34@tcp(vm0)/aofei")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()

	/*
		   	rows, err := db.QueryContext(ctx, `
		   SELECT country_id, alpha3 FROM def_country`)
		   	if err != nil {
		   		panic(err)
		   	}
		   	defer rows.Close()
			fmt.Printf("var CountryMap = map[string]uint32{\n")
		   	for rows.Next() {
		   		var countryID uint32
		   		var alpha3 string
		   		err = rows.Scan(&countryID, &alpha3)
		   		if err != nil {
		   			panic(err)
		   		}
		   		fmt.Printf(`"%s":%d,`+"\n", alpha3, countryID)
		   	}
		   	if err = rows.Err(); err != nil {
		   		panic(err)
		   	}

	*/
	rows, err := db.QueryContext(ctx, `
SELECT country_id, state_code, state_id 
FROM def_state`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	ref := make(map[uint32]map[string]uint32)
	for rows.Next() {
		var stateCode string
		var countryID, stateID uint32
		err = rows.Scan(&countryID, &stateCode, &stateID)
		if err != nil {
			panic(err)
		}
		if _, ok := ref[countryID]; !ok {
			ref[countryID] = make(map[string]uint32)
		}
		ref[countryID][stateCode] = stateID
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
	fmt.Printf("var StateMap = map[uint32]map[string]uint32{\n")
	for k, v := range ref {
		fmt.Printf(`%d: {`+"\n", k)
		for k1, v1 := range v {
			fmt.Printf("\t"+`"%s":%d,`+"\n", k1, v1)
		}
		fmt.Printf("},\n")
	}
	fmt.Printf("}\n")
}

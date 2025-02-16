package main

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "eightran:12pass34@tcp(vm0:3306)/aofei")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = doCountry(context.Background(), db)
	if err != nil {
		log.Println("doCountry:", err)
		panic(err)
	}
	err = doState(context.Background(), db)
	if err != nil {
		log.Println("doState:", err)
		panic(err)
	}
	err = doCity(context.Background(), db)
	if err != nil {
		log.Println("doCity:", err)
		panic(err)
	}

	err = doChannel(context.Background(), db)
	if err != nil {
		log.Println("doChannel:", err)
		panic(err)
	}
}

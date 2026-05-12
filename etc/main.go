package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	config := os.Getenv("AOFEI")
	if config == "" {
		config = "etc/aofei.local.json"
		if _, err := os.Stat(config); err != nil {
			config = "../etc/aofei.local.json"
		}
	}
	cfg, err := dsp.NewConfig(config)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open(cfg.ConnectArray[0], cfg.ConnectArray[1])
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	if err = db.PingContext(ctx); err != nil {
		panic(err)
	}

	switch os.Args[1] {
	case "pub":
		_, err = insertDefaultPub(db)
	default:
	}

	if err != nil {
		panic(err)
	}
}

func insertDefaultPub(db *sql.DB) (*acl.Pub, error) {
	pubMap, err := acl.DBGetPubMap(db)
	if err != nil {
		return nil, err
	}
	if pub, ok := pubMap[acl.PUBDefault]; ok {
		return pub, nil
	}
	return acl.AddPub(db, acl.PUBDefault)
}

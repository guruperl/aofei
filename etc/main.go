package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	control, err := dsp.NewController(context.Background(), "../conf/aofei.json")
	if err != nil {
		panic(err)
	}
	db := control.DB
	defer db.Close()

	ctx := context.Background()

	switch os.Args[1] {
	case "pub":
		_, err = insertDefaultPub(db)
	case "channel":
		err = doChannel(ctx, db)
	case "geography":
		if err = doCountry(ctx, db); err == nil {
			if err = doState(ctx, db); err == nil {
				err = doCity(ctx, db)
			}
		}
	default:
	}

	if err != nil {
		panic(err)
	}
}

func insertDefaultPub(db *sql.DB) (*match.Pub, error) {
	pubMap, err := match.DBGetPubMap(db)
	if err != nil {
		return nil, err
	}
	return pubMap.DBAddNew(db, match.PUBDefault, match.SITEDefault, "", match.SLOTDefault)
}

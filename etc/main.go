package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stdout)
		return
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		usage(os.Stdout)
		return
	}

	if os.Args[1] != "pub" {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}

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

	_, err = insertDefaultPub(db)
	if err != nil {
		panic(err)
	}
}

func usage(out *os.File) {
	fmt.Fprintln(out, "usage: go run ./etc <command>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  pub    Insert the default local publisher if it is missing.")
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

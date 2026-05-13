package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/jobs/midcallback"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mid-callback-retry -s=dsp_config -limit=100 -max-attempts=5 -timeout=2s -dry-run\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	sConf       string
	limit       int
	maxAttempts int
	timeout     time.Duration
	dryRun      bool
	readOnly    bool
)

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.IntVar(&limit, "limit", 100, "maximum due retry rows to process")
	flag.IntVar(&maxAttempts, "max-attempts", 5, "maximum attempts before abandoning a retry row")
	flag.DurationVar(&timeout, "timeout", 2*time.Second, "downstream callback HTTP timeout")
	flag.BoolVar(&dryRun, "dry-run", false, "read due rows without forwarding or updating")
	flag.BoolVar(&readOnly, "read", false, "alias for -dry-run")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	result, err := midcallback.Run(ctx, db, midcallback.Options{
		Limit:       limit,
		MaxAttempts: maxAttempts,
		Timeout:     timeout,
		DryRun:      dryRun || readOnly,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("selected=%d succeeded=%d retrying=%d abandoned=%d\n",
		result.Selected, result.Succeeded, result.Retrying, result.Abandoned)
}

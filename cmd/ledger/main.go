package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/guruperl/aofei/dsp"
	ledgerjob "github.com/guruperl/aofei/internal/jobs/ledger"
	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: ledger -s=dsp_config -interval=divider -daily -stamp=stamp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int
var stamp string
var daily bool

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.IntVar(&interval, "interval", 10, "Divider in minutes")
	flag.StringVar(&stamp, "timestamp", "", "optional. it is MySQL day if daily is set, or 7 digit unix timestamp in minutes")
	flag.BoolVar(&daily, "daily", false, "If set, it will insert daily ledger")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	sc, err := dsp.NewControllerWithOptions(ctx, sConf, dsp.WithoutNATS(), dsp.WithoutMaxMind())
	if err != nil {
		log.Fatal(err)
	}

	db := sc.DB
	defer sc.Close()

	if daily {
		if stamp == "" {
			err = ledgerjob.InsertDaily(db)
			log.Printf("Daily ledger of %s done", time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
		} else {
			err = ledgerjob.InsertDaily(db, stamp)
			log.Printf("Daily ledger of %s done", stamp)
		}
	} else {
		var result ledgerjob.IntervalResult
		var i int
		if stamp == "" {
			result, err = ledgerjob.RunInterval(db, sc.C.LogWinLoss, interval)
		} else if i, err = strconv.Atoi(stamp); err == nil {
			result, err = ledgerjob.RunInterval(db, sc.C.LogWinLoss, interval, i)
		}
		if err == nil && !result.Skipped {
			log.Printf("Ledger %d at %d minutes done", result.Current, interval)
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}

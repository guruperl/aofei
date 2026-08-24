package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
	ledgerjob "github.com/guruperl/aofei/internal/jobs/ledger"
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
var lockTTL time.Duration

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.IntVar(&interval, "interval", 10, "Divider in minutes")
	flag.StringVar(&stamp, "timestamp", "", "optional. it is MySQL day if daily is set, or 7 digit unix timestamp in minutes")
	flag.BoolVar(&daily, "daily", false, "If set, it will insert daily ledger")
	flag.DurationVar(&lockTTL, "lock-ttl", 30*time.Minute, "singleton lock TTL")
}

func main() {
	flag.Parse()
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Validate(dsp.ConfigModeLedger); err != nil {
		log.Fatal(err)
	}
	redis, db, err := c.GetRedisDB(ctx, dsp.ConfigModeLedger)
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()
	defer db.Close()
	err = cmdboot.WithLock(ctx, redis, "aofei:ledger", lockTTL, func(leaseCtx context.Context) error {
		if daily {
			day := stamp
			if day == "" {
				day = time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
			}
			if err := ledgerjob.InsertDailyContext(leaseCtx, db, day); err != nil {
				return err
			}
			log.Printf("Daily ledger of %s done", day)
			return nil
		}

		var result ledgerjob.IntervalResult
		var runErr error
		if stamp == "" {
			result, runErr = ledgerjob.RunIntervalContext(leaseCtx, db, c.LogWinLoss, interval)
		} else {
			i, parseErr := strconv.Atoi(stamp)
			if parseErr != nil {
				return parseErr
			}
			result, runErr = ledgerjob.RunIntervalContext(leaseCtx, db, c.LogWinLoss, interval, i)
		}
		if runErr == nil && !result.Skipped {
			log.Printf("Ledger %d at %d minutes done", result.Current, interval)
		}
		return runErr
	})
	if err != nil {
		log.Fatal(err)
	}
}

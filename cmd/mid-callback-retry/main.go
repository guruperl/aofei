package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
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
	lockTTL     time.Duration
)

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.IntVar(&limit, "limit", 100, "maximum due retry rows to process")
	flag.IntVar(&maxAttempts, "max-attempts", 5, "maximum attempts before abandoning a retry row")
	flag.DurationVar(&timeout, "timeout", 2*time.Second, "downstream callback HTTP timeout")
	flag.BoolVar(&dryRun, "dry-run", false, "read due rows without forwarding or updating")
	flag.BoolVar(&readOnly, "read", false, "alias for -dry-run")
	flag.DurationVar(&lockTTL, "lock-ttl", 30*time.Minute, "singleton lock TTL for mutating runs")
}

func main() {
	flag.Parse()
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Validate(dsp.ConfigModeRetry); err != nil {
		log.Fatal(err)
	}
	redis, db, err := c.GetRedisDB(ctx, dsp.ConfigModeRetry)
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()
	defer db.Close()
	if !(dryRun || readOnly) {
		lock, err := cmdboot.AcquireLock(ctx, redis, "aofei:mid-callback-retry", lockTTL)
		if err != nil {
			log.Fatal(err)
		}
		defer lock.Release(context.Background())
	}

	opts := midcallback.Options{
		Limit:       limit,
		MaxAttempts: maxAttempts,
		Timeout:     timeout,
		DryRun:      dryRun || readOnly,
	}
	backlog, err := midcallback.Backlog(ctx, db, opts)
	if err != nil {
		log.Fatal(err)
	}
	result, err := midcallback.Run(ctx, db, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("due=%d stale_processing=%d selected=%d succeeded=%d retrying=%d abandoned=%d\n",
		backlog.Due, backlog.StaleProcessing, result.Selected, result.Succeeded, result.Retrying, result.Abandoned)
}

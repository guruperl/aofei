package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
	"github.com/guruperl/aofei/internal/jobs/midcallback"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mid-callback-retry -s=dsp_config -limit=100 -max-attempts=5 -timeout=2s [-dry-run|-read] [-json]\n")
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
	jsonOutput  bool
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
	flag.BoolVar(&jsonOutput, "json", false, "write stable JSON summary output")
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
	opts := midcallback.Options{
		Limit:       limit,
		MaxAttempts: maxAttempts,
		Timeout:     timeout,
		DryRun:      dryRun || readOnly,
	}
	run := func(runCtx context.Context) error {
		backlog, err := midcallback.Backlog(runCtx, db, opts)
		if err != nil {
			return err
		}
		result, err := midcallback.Run(runCtx, db, opts)
		if err != nil {
			return err
		}
		return writeRetryReport(os.Stdout, jsonOutput, backlog, result)
	}
	if dryRun || readOnly {
		err = run(ctx)
	} else {
		err = cmdboot.WithLock(ctx, redis, "aofei:mid-callback-retry", lockTTL, run)
	}
	if err != nil {
		log.Fatal(err)
	}
}

type retryReport struct {
	Due             int `json:"due"`
	StaleProcessing int `json:"stale_processing"`
	Selected        int `json:"selected"`
	Succeeded       int `json:"succeeded"`
	Retrying        int `json:"retrying"`
	Abandoned       int `json:"abandoned"`
}

func newRetryReport(backlog midcallback.BacklogStats, result midcallback.Result) retryReport {
	return retryReport{
		Due:             backlog.Due,
		StaleProcessing: backlog.StaleProcessing,
		Selected:        result.Selected,
		Succeeded:       result.Succeeded,
		Retrying:        result.Retrying,
		Abandoned:       result.Abandoned,
	}
}

func writeRetryReport(w io.Writer, asJSON bool, backlog midcallback.BacklogStats, result midcallback.Result) error {
	report := newRetryReport(backlog, result)
	if asJSON {
		enc := json.NewEncoder(w)
		return enc.Encode(report)
	}
	_, err := fmt.Fprintf(w, "due=%d stale_processing=%d selected=%d succeeded=%d retrying=%d abandoned=%d\n",
		report.Due, report.StaleProcessing, report.Selected, report.Succeeded, report.Retrying, report.Abandoned)
	return err
}

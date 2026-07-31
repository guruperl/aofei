// this runs every 15 minutes to update redis caches of pubmap and demand-side
// RAdvs, audiences and creatives. It can also publish the static cache to NATS
// for the spread receiver.
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
	cachejob "github.com/guruperl/aofei/internal/jobs/cache"
	"github.com/nats-io/nats.go"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: redis-cache -s=dsp_config -cache=redis|spread|all|routes -read -update -interval=divider -stamp=stamp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int
var stamp int
var update bool
var read bool
var cacheMode string
var lockTTL time.Duration

const redisCacheMutationLockKey = "aofei:redis-cache"

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.StringVar(&cacheMode, "cache", cachejob.ModeRedis, "cache type")
	flag.BoolVar(&update, "update", false, "update pubmap with new attribute log")
	flag.BoolVar(&read, "read", false, "read caches from redis")
	flag.IntVar(&interval, "interval", 10, "if update, divider in minutes")
	flag.IntVar(&stamp, "timestamp", 0, "if update, fixed timestamp in minutes")
	flag.DurationVar(&lockTTL, "lock-ttl", 30*time.Minute, "singleton lock TTL for mutating runs")
}

func main() {
	flag.Parse()

	if err := cachejob.ValidateMode(cacheMode); err != nil {
		log.Fatal(err)
	}

	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	modes := []dsp.ConfigMode{dsp.ConfigModeCache}
	if cacheMode == cachejob.ModeSpread || cacheMode == cachejob.ModeAll {
		modes = append(modes, dsp.ConfigModeNATS)
	}
	if err := c.Validate(modes...); err != nil {
		log.Fatal(err)
	}
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	redis, db, err := c.GetRedisDB(ctx, dsp.ConfigModeCache)
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()
	defer db.Close()

	if read {
		if err := cachejob.Read(ctx, os.Stdout, c, redis, db, cacheMode); err != nil {
			log.Fatal(err)
		}
		return
	}
	lock, err := cmdboot.AcquireLock(ctx, redis, cacheMutationLockKey(cacheMode), lockTTL)
	if err != nil {
		log.Fatal(err)
	}
	defer lock.Release(context.Background())

	var nc *nats.Conn
	if cacheMode == cachejob.ModeSpread || cacheMode == cachejob.ModeAll {
		nc, err = nats.Connect(c.NatsURL)
		if err != nil {
			log.Fatal(err)
		}
		defer nc.Drain()
	}

	err = cachejob.Run(ctx, c, redis, db, nc, cachejob.Options{
		Mode:           cacheMode,
		UpdatePubMap:   update,
		UpdateInterval: interval,
		UpdateStamp:    stamp,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func cacheMutationLockKey(string) string {
	return redisCacheMutationLockKey
}

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
	fmt.Fprintf(os.Stderr, "usage: redis-cache -s=dsp_config -cache=redis|spread|all|routes -read|-validate-publishers|-validate-middleman [-activation-stage=preflight|fallback|always] -update -interval=divider -stamp=stamp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int
var stamp int
var update bool
var read bool
var validatePublishers bool
var validateMiddleman bool
var activationStage string
var cacheMode string
var lockTTL time.Duration

const redisCacheMutationLockKey = "aofei:redis-cache"

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.StringVar(&cacheMode, "cache", cachejob.ModeRedis, "cache type")
	flag.BoolVar(&update, "update", false, "update pubmap with new attribute log")
	flag.BoolVar(&read, "read", false, "read caches from redis")
	flag.BoolVar(&validatePublishers, "validate-publishers", false, "read-only validation and token manifest for active publisher inventory")
	flag.BoolVar(&validateMiddleman, "validate-middleman", false, "read-only validation of middleman config, routes, publication, and credential references")
	flag.StringVar(&activationStage, "activation-stage", cachejob.MiddlemanStagePreflight, "middleman activation stage: preflight, fallback, or always")
	flag.IntVar(&interval, "interval", 10, "if update, divider in minutes")
	flag.IntVar(&stamp, "timestamp", 0, "if update, fixed timestamp in minutes")
	flag.DurationVar(&lockTTL, "lock-ttl", 30*time.Minute, "singleton lock TTL for mutating runs")
}

func main() {
	flag.Parse()

	if err := cachejob.ValidateMode(cacheMode); err != nil {
		log.Fatal(err)
	}
	if err := validateCommandModes(read, update, validatePublishers, validateMiddleman, activationStage); err != nil {
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
	if validatePublishers {
		modes = []dsp.ConfigMode{dsp.ConfigModeDatabase}
	}
	if err := c.Validate(modes...); err != nil {
		log.Fatal(err)
	}
	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()
	if validatePublishers {
		issuer, err := dsp.NewDirectSSPTokenIssuer(c)
		if err != nil {
			log.Fatal(err)
		}
		db, err := c.OpenDB(ctx)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		if err := cachejob.ValidatePublisherInventory(os.Stdout, db, issuer); err != nil {
			log.Fatal(err)
		}
		return
	}
	redis, db, err := c.GetRedisDB(ctx, dsp.ConfigModeCache)
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()
	defer db.Close()
	if validateMiddleman {
		if err := cachejob.ValidateMiddlemanActivation(ctx, os.Stdout, c, redis, db, activationStage); err != nil {
			log.Fatal(err)
		}
		return
	}

	if read {
		if err := cachejob.Read(ctx, os.Stdout, c, redis, db, cacheMode); err != nil {
			log.Fatal(err)
		}
		return
	}
	var nc *nats.Conn
	if cacheMode == cachejob.ModeSpread || cacheMode == cachejob.ModeAll {
		nc, err = nats.Connect(c.NatsURL)
		if err != nil {
			log.Fatal(err)
		}
		defer nc.Drain()
	}

	err = cmdboot.WithLock(ctx, redis, cacheMutationLockKey(cacheMode), lockTTL, func(leaseCtx context.Context) error {
		return cachejob.Run(leaseCtx, c, redis, db, nc, cachejob.Options{
			Mode:           cacheMode,
			UpdatePubMap:   update,
			UpdateInterval: interval,
			UpdateStamp:    stamp,
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}

func cacheMutationLockKey(string) string {
	return redisCacheMutationLockKey
}

func validateCommandModes(read, update, publishers, middleman bool, stage string) error {
	selected := 0
	for _, enabled := range []bool{read, publishers, middleman} {
		if enabled {
			selected++
		}
	}
	if selected > 1 {
		return fmt.Errorf("-read, -validate-publishers, and -validate-middleman are mutually exclusive")
	}
	if update && (read || publishers || middleman) {
		return fmt.Errorf("-update cannot be combined with a read-only validation mode")
	}
	if !middleman && stage != cachejob.MiddlemanStagePreflight {
		return fmt.Errorf("-activation-stage requires -validate-middleman")
	}
	return nil
}

// this runs every 15 minutes to update redis caches of pubmap and demand-side
// RAdvs, audiences and creatives. It also writes the pubmap to a file.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	"github.com/mediocregopher/radix/v4"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: redis-cache -s=dsp_config -update -interval=divider -stamp=stamp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int
var stamp int
var update bool

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.BoolVar(&update, "update", false, "update pubmap with new attribute log, using interval or timestamp")
	flag.IntVar(&interval, "interval", 10, "divider in minutes")
	flag.IntVar(&stamp, "timestamp", 0, "7-digit fixed timestamp in minutes")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	sc, err := dsp.NewController(ctx, sConf, "stop")
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	pubmap, err := match.DBGetPubMap(sc.DB)
	if err != nil {
		log.Fatal(err)
	}

	if update {
		var stampObject *dsp.Stamp
		if stamp > 0 {
			stampObject = dsp.NewStamp(interval, stamp)
		} else {
			stampObject = dsp.NewStamp(interval)
		}
		fn := sc.C.NewLogfileName(dsp.SUBJECTAttribute, stampObject)
		fh, err := os.Open(fn)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		err = pubmap.DBUpdateIO(sc.DB, fh)
		if err != nil {
			log.Fatal(err)
		}
	}

	name := dsp.PubRedisName(sc.C.RPubMap)
	log.Printf("new PubMap is written to redis %s\n", name)
	arr := []string{name}
	for k, v := range pubmap {
		bs, err := v.Pack()
		if err != nil {
			log.Fatal(err)
		}
		arr = append(arr, k, string(bs))
	}
	err = sc.Redis.Do(ctx, radix.Cmd(nil, "HMSET", arr...))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("new PubMap is written to file %s\n", sc.C.RPubMap)
	jh, err := os.OpenFile(sc.C.RPubMap, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer jh.Close()
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	jh.Write(bs)

	// the following are for demand-side cache, which could be run separately

	log.Printf("Retrieve RAdvs from DB and write to redis")
	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		err = match.DBGetRAdvsToRedis(ctx, sc.Redis, sc.DB, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Retrieve Audiences from DB and write to redis")
	err = match.DBGetAudiencesToRedis(ctx, sc.Redis, sc.DB)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Retrieve Creatives from DB and write to redis")
	err = match.DBGetCreativesToRedis(ctx, sc.Redis, sc.DB)
	if err != nil {
		log.Fatal(err)
	}
}

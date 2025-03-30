// this runs every 15 minutes to update redis caches of pubmap and demand-side
// RAdvs, audiences and creatives. It also writes the pubmap to a file.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: redis-cache -s=dsp_config -read -update -interval=divider -stamp=stamp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int
var stamp int
var update bool
var read bool

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.BoolVar(&update, "update", false, "update pubmap with new attribute log, using interval or timestamp")
	flag.BoolVar(&read, "read", false, "read caches from redis")
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

	//top:= sc.C.Spread
	//err = os.MkdirAll(top, 0755)
	//if err != nil {
	//	log.Fatal(err)
	//}

	if read {
		//if err := spreadRead(ctx, sc, top); err != nil {
		if err := redisRead(ctx, sc); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	pubmap, err := acl.DBGetPubMap(sc.DB)
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
		if err != nil && err == os.ErrNotExist {
			goto Skip
		} else if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		scanner := bufio.NewScanner(fh)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			plus := new(match.AttributePlus)
			err := json.Unmarshal([]byte(line), plus)
			if err != nil {
				log.Fatal(err)
			}
			acl := plus.Attribute.ACL
			siteType := "Web"
			if plus.Attribute.IsApp {
				siteType = "App"
			}
			pub, err := pubmap.DBAddNew(sc.DB, acl.PubStr, acl.SiteStr, siteType, acl.SlotStr)
			if err != nil {
				log.Fatal(err)
			}
			pubmap[acl.PubStr] = pub
		}
		if err = scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
Skip:
	//if err = writeToSpread(ctx, sc, pubmap); err != nil {
	if err = writeToRedis(ctx, sc, pubmap); err != nil {
		log.Fatal(err)
	}
}

func redisRead(ctx context.Context, sc *dsp.Controller) error {
	log.Printf("reading PubMap from redis\n")
	pubmap, err := acl.PubMapFromRedis(ctx, sc.Redis)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", bs)

	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		hash, err := match.RAdvsFromRedisBySizeID(ctx, sc.Redis, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("RAdvs for sizeID %d x %d: %s\n", sizeID2[0], sizeID2[1], bs)
	}

	audiences, err := match.AudienceMapFromRedis(ctx, sc.Redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Audiences: %s\n", bs)

	creatives, err := match.CreativeMapFromRedis(ctx, sc.Redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Creatives: %s\n", bs)
	return nil
}

func writeToRedis(ctx context.Context, sc *dsp.Controller, pubmap acl.PubMap) error {
	log.Printf("new PubMap is written to redis\n")
	err := pubmap.ToRedis(ctx, sc.Redis)
	if err != nil {
		return err
	}

	// the following are for demand-side cache, which could be run separately

	log.Printf("Retrieve RAdvs from DB and write to redis")
	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		err = match.DBGetRAdvsToRedis(ctx, sc.Redis, sc.DB, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
	}

	log.Printf("Retrieve Audiences from DB and write to redis")
	err = match.DBGetAudiencesToRedis(ctx, sc.Redis, sc.DB)
	if err != nil {
		return err
	}

	log.Printf("Retrieve Creatives from DB and write to redis")
	err = match.DBGetCreativesToRedisSpread(ctx, sc.Redis, sc.DB)
	if err != nil {
		return err
	}

	return nil
}

func spreadRead(ctx context.Context, sc *dsp.Controller, top string) error {
	log.Printf("reading PubMap from redis\n")
	pubmap, err := acl.PubMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", bs)

	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		hash, err := match.RAdvsFromIOBySizeID(top, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("RAdvs for sizeID %d x %d: %s\n", sizeID2[0], sizeID2[1], bs)
	}

	audiences, err := match.AudienceMapFromRedis(ctx, sc.Redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Audiences: %s\n", bs)

	creatives, err := match.CreativeMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("Creatives: %s\n", bs)

	return nil
}

func writeToSpread(ctx context.Context, sc *dsp.Controller, pubmap acl.PubMap) error {
	log.Printf("new PubMap is written to redis\n")
	err := pubmap.ToSpread(sc.Nc)
	if err != nil {
		return err
	}

	// the following are for demand-side cache, which could be run separately

	log.Printf("Retrieve RAdvs from DB and write to nats")
	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		err = match.DBGetRAdvsToSpread(ctx, sc.Nc, sc.DB, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
	}

	log.Printf("Retrieve Audiences from DB and write to nats")
	err = match.DBGetAudiencesToSpread(sc.Nc, sc.DB)
	if err != nil {
		return err
	}

	log.Printf("Retrieve Creatives from DB and write to redis")
	err = match.DBGetCreativesToRedisSpread(ctx, sc.Nc, sc.DB)
	if err != nil {
		return err
	}

	return nil
}

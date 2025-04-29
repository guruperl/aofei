// this runs every 15 minutes to update redis caches of pubmap and demand-side
// RAdvs, audiences and creatives. It also writes the pubmap to a file.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"

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
var cache string

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.StringVar(&cache, "cache", "redis", "cache type")
	flag.BoolVar(&update, "update", false, "update pubmap with new attribute log")
	flag.BoolVar(&read, "read", false, "read caches from redis")
	flag.IntVar(&interval, "interval", 10, "if update, divider in minutes")
	flag.IntVar(&stamp, "timestamp", 0, "if update, fixed timestamp in minutes")
	flag.Parse()
}

func main() {
	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	redis, db, err := c.GetRedisDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()
	defer db.Close()

	var top string
	switch cache {
	case "redis":
	default:
		top = c.Spread
		err = os.MkdirAll(top, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	if read {
		switch cache {
		case "spread":
			err = spreadRead(top)
		case "redis":
			err = redisRead(ctx, redis)
		default:
			if err = spreadRead(top); err == nil {
				err = redisRead(ctx, redis)
			}
		}
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	pubmap, err := acl.DBGetPubMap(db)
	if err != nil {
		log.Fatal(err)
	}

	if update {
		err = pubmapUpdate(c, db, pubmap)
		if err != nil {
			log.Fatal(err)
		}
	}

	inactiveDomains, err := acl.InactiveDomains(db)
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(c.NatsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	switch cache {
	case "spread":
		err = writeToSpread(ctx, nc, db, pubmap, inactiveDomains)
	case "redis":
		err = writeToRedis(ctx, redis, db, pubmap, inactiveDomains)
	default:
		if err = writeToSpread(ctx, nc, db, pubmap, inactiveDomains); err == nil {
			err = writeToRedis(ctx, redis, db, pubmap, inactiveDomains)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}

func redisRead(ctx context.Context, redis radix.Client) error {
	log.Printf("reading PubMap from redis\n")
	pubmap, err := acl.PubMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("pubmap:\n%s\n", bs)

	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		hash, err := match.RAdvsFromRedisBySizeID(ctx, redis, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("\nRAdvs for sizeID %d x %d:\n%s\n", sizeID2[0], sizeID2[1], bs)
	}

	audiences, err := match.AudienceMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\nAudiences:\n%s\n", bs)

	creatives, err := match.CreativeMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\nCreatives:\n%s\n", bs)
	return nil
}

func writeToRedis(ctx context.Context, redis radix.Client, db *sql.DB, pubmap acl.PubMap, inactiveDomains []string) error {
	log.Printf("new PubMap is written to redis\n")
	err := pubmap.ToRedis(ctx, redis, inactiveDomains)
	if err != nil {
		return err
	}

	log.Printf("retrieve RAdvs from DB and write to redis")
	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		err = match.DBGetRAdvsToRedisSpreadBySizeID(ctx, redis, db, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
	}

	log.Printf("retrieve Audiences from DB and write to redis")
	err = match.DBGetAudiencesToRedis(ctx, redis, db)
	if err != nil {
		return err
	}

	log.Printf("retrieve Creatives from DB and write to redis")
	err = match.DBGetCreativesToRedisSpread(ctx, redis, db)
	if err != nil {
		return err
	}

	return nil
}

func spreadRead(top string) error {
	log.Printf("reading PubMap from IO\n")
	pubmap, err := acl.PubMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("pubmap:\n%s\n", bs)

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
		fmt.Printf("\nRAdvs for sizeID %d x %d:\n%s\n", sizeID2[0], sizeID2[1], bs)
	}

	audiences, err := match.AudienceMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\nAudiences:\n%s\n", bs)

	creatives, err := match.CreativeMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\nCreatives:\n%s\n", bs)

	return nil
}

func writeToSpread(ctx context.Context, nc *nats.Conn, db *sql.DB, pubmap acl.PubMap, inactiveDomains []string) error {
	log.Printf("new PubMap is written to nats\n")
	err := pubmap.ToSpread(nc, inactiveDomains)
	if err != nil {
		return err
	}

	log.Printf("retrieve RAdvs from DB and write to nats")
	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		err = match.DBGetRAdvsToRedisSpreadBySizeID(ctx, nc, db, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			return err
		}
	}

	log.Printf("retrieve Audiences from DB and write to nats")
	err = match.DBGetAudiencesToSpread(nc, db)
	if err != nil {
		return err
	}

	log.Printf("retrieve Creatives from DB and write to redis")
	err = match.DBGetCreativesToRedisSpread(ctx, nc, db)
	if err != nil {
		return err
	}

	return nil
}

// pubmapUpdate updates the pubmap with new attribute log
func pubmapUpdate(c *dsp.Config, db *sql.DB, pubmap acl.PubMap) error {
	var stampObject *dsp.Stamp
	if stamp > 0 {
		stampObject = dsp.NewStamp(interval, stamp)
	} else {
		stampObject = dsp.NewStamp(interval)
	}
	fn := c.NewLogfileName(dsp.SUBJECTAttribute, stampObject)
	fh, err := os.Open(fn)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
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
		pub, err := pubmap.DBAddNew(db, acl.PubStr, acl.SiteStr, siteType, acl.SlotStr)
		if err != nil {
			log.Fatal(err)
		}
		pubmap[acl.PubStr] = pub
	}
	return scanner.Err()
}

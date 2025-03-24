// this pushs database to Redis cache every 15 minutes or so
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: redis --s=dsp_config\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	sc, err := dsp.NewController(ctx, sConf)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	for _, sizeID2 := range [][2]uint16{
		{64, 64},
		{100, 100},
	} {
		err = match.DBGetRAdvsToRedis(ctx, sc.Redis, sc.DB, match.SizeID2To1(sizeID2[0], sizeID2[1]))
		if err != nil {
			log.Fatal(err)
		}
	}

	err = match.DBGetAudiencesToRedis(ctx, sc.Redis, sc.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = match.DBGetCreativesToRedis(ctx, sc.Redis, sc.DB)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("structured radvs, audiences, and creatives to Redis cache")
}

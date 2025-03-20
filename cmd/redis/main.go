// go run summer.go --log_dir="../../logs/"
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
	fmt.Fprintf(os.Stderr, "usage: redis --s=dsp_config -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n")
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

	err = match.DBGetRAdvsToRedis(ctx, sc.Redis, sc.DB, "App")
	if err == nil {
		err = match.DBGetRAdvsToRedis(ctx, sc.Redis, sc.DB, "Web")
	}
	if err != nil {
		log.Fatal(err)
	}

	err = match.DBGetAudiencesToRedis(ctx, sc.Redis, sc.DB)
	if err != nil {
		log.Fatal(err)
	}
}

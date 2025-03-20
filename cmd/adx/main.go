// go run summer.go --log_dir="../../logs/"
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

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: adx --s=dsp_config -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n")
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

	rpubmap, err := match.DBGetRPubMap(sc.DB)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("new RPubMap is written to %s\n", sc.C.RPubMap)
	fh, err := os.OpenFile(sc.C.RPubMap, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	bs, err := json.MarshalIndent(rpubmap, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fh.Write(bs)
}

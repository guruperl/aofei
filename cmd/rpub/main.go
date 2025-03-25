// this runs daily, after the data of adxes and publishers are inserted or updated.
// the generated RPubMap is put on disk, and the web server has to restart to read it.
package main

import (
	"bufio"
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
	fmt.Fprintf(os.Stderr, "usage: ledger -s=dsp_config -interval=divider -stamp=stamp\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int
var stamp int

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.IntVar(&interval, "interval", 10, "Divider in minutes")
	flag.IntVar(&stamp, "timestamp", 0, "integer, optional. 7 digit unix timestamp in minutes")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	sc, err := dsp.NewController(ctx, sConf)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	rpubmap, err := match.DBGetRPubMap(sc.DB)
	if err != nil {
		log.Fatal(err)
	}

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

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		plus := new(dsp.AttributePlus)
		err := json.Unmarshal([]byte(line), plus)
		if err != nil {
			log.Fatal(err)
		}
		acl := plus.Attribute.ACL
		_, _, _, err = rpubmap.DBAddNew(sc.DB, acl.PubStr, acl.SiteStr, acl.SlotStr)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("new RPubMap is written to %s\n", sc.C.RPubMap)
	bs, err := rpubmap.Pack()
	if err == nil {
		err = sc.Redis.Do(ctx, radix.Cmd(nil, "SET", sc.C.RPubMap, string(bs)))
	}
	if err != nil {
		log.Fatal(err)
	}

	/*
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
	*/
}

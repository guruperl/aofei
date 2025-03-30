// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/genelet/winter/dsp"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: spread -s=dsp_config\n")
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
	sc, err := dsp.NewController(ctx, sConf, "nats")
	if err != nil {
		log.Fatal(err)
	}
	nc := sc.Nc
	defer sc.Close()

	filewriters, err := NewFileWriters(sc.C.Spread)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s]", nc.ConnectedUrl())
	err = filewriters.ReceiveLogs(nc)
	if err != nil {
		log.Fatal(err)
	}
}

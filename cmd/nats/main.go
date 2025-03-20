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
	fmt.Fprintf(os.Stderr, "usage: nats --s=dsp_config -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n")
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

	filewriters, err := dsp.NewFileWriters(sc.C.LogRequest, sc.C.LogResponse, sc.C.LogAttribute, sc.C.LogWinLoss)
	if err != nil {
		log.Fatal(err)
	}

	filewriters.ReceiveLogs(sc.Nc)
}

// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/genelet/winter/dsp"
	"github.com/nats-io/nats.go"

	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: nats-client -s=dsp_config -interval=divider\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var sConf string
var interval int

func init() {
	flag.Usage = usage
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "DSP Config")
	flag.IntVar(&interval, "interval", 10, "Log divider in minutes")
}

func main() {
	flag.Parse()
	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	nc, err := nats.Connect(c.NatsURL, nats.ReconnectWait(20*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Drain()

	filewriters, err := NewFileWriters(c.LogRequest, c.LogResponse, c.LogAttribute, c.LogWinLoss, interval)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s]", nc.ConnectedUrl())
	err = filewriters.ReceiveLogs(nc)
	if err != nil {
		log.Fatal(err)
	}
}

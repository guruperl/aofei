// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	"github.com/nats-io/nats.go"

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
	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	nc, err := nats.Connect(c.NatsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	top := c.Spread
	if err := os.MkdirAll(top, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s]", nc.ConnectedUrl())
	successchan := make(chan bool)
	errchan := make(chan error)

	_, err = nc.Subscribe("*", func(m *nats.Msg) {
		switch m.Subject {
		case dsp.SUBJECTRequest:
			return
		case dsp.SUBJECTResponse:
			return
		case dsp.SUBJECTAttribute:
			return
		case dsp.SUBJECTWinLoss:
			return
		default:
		}

		filename := strings.ReplaceAll(m.Subject, ":", "/")
		dir, base := filepath.Split(filename)
		if strings.HasPrefix(dir, acl.HashNamePubmap) ||
			strings.HasPrefix(dir, match.HashNameAudience) ||
			strings.HasPrefix(dir, match.HashNameSlot) ||
			strings.HasPrefix(dir, match.HashNameCreative) {
			var err error
			if err = os.MkdirAll(fmt.Sprintf("%s/%s", top, dir), os.ModePerm); err == nil {
				var w *os.File
				if w, err = os.OpenFile(fmt.Sprintf("%s/%s/%s", top, dir, base), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666); err == nil {
					defer w.Close()
					_, err = w.Write(m.Data)
				}
			}
			if err != nil {
				errchan <- err
			}
			log.Printf("write %s", m.Subject)
		}
		successchan <- true
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-successchan:
		case errs := <-errchan:
			log.Printf("error: %v", errs)
		}
	}
}

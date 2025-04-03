// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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
		if pure, ok := strings.CutSuffix(base, "cleanup"); ok {
			base = pure
			if err := os.RemoveAll(fmt.Sprintf("%s/%s", top, dir)); err != nil {
				errchan <- err
				return
			}
		}
		if strings.HasPrefix(dir, acl.HashNamePubmap) ||
			strings.HasPrefix(dir, match.HashNameAudience) ||
			strings.HasPrefix(dir, match.HashNameSlot) ||
			strings.HasPrefix(dir, match.HashNameCreative) {
			var err error
			if err = os.MkdirAll(fmt.Sprintf("%s/%s", top, dir), os.ModePerm); err == nil {
				var w *os.File
				if w, err = os.OpenFile(fmt.Sprintf("%s/%s/%s", top, dir, base), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666); err == nil {
					defer w.Close()
					// Acquire exclusive lock.
					// LOCK_NB is not used here, because we want to wait for the lock until it is released by other process.
					if err = syscall.Flock(int(w.Fd()), syscall.LOCK_EX); err == nil {
						defer func() {
							err1 := syscall.Flock(int(w.Fd()), syscall.LOCK_UN)
							if err1 != nil {
								log.Println("Error releasing lock:", err1)
							}
						}()
						_, err = w.Write(m.Data)
					}
				}
			}
			if err != nil {
				errchan <- err
				return
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

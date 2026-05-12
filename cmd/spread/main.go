// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"errors"
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
}

func main() {
	flag.Parse()

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
		handled, err := handleSpreadMessage(top, m)
		if err != nil {
			errchan <- err
			return
		}
		if handled {
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

func handleSpreadMessage(top string, m *nats.Msg) (bool, error) {
	if ignoredLogSubject(m.Subject) {
		return false, nil
	}

	dir, base, ok := spreadSubjectPath(m.Subject)
	if !ok {
		return false, nil
	}

	if pure, ok := strings.CutSuffix(base, "cleanup"); ok {
		base = pure
		if err := os.RemoveAll(filepath.Join(top, dir)); err != nil {
			return true, err
		}
	}

	fullPath := filepath.Join(top, dir, base)
	if string(m.Data) == "DELETE" {
		return true, os.RemoveAll(fullPath)
	}

	if err := os.MkdirAll(filepath.Join(top, dir), os.ModePerm); err != nil {
		return true, err
	}
	return true, writeSnapshot(fullPath, m.Data)
}

func ignoredLogSubject(subject string) bool {
	switch subject {
	case dsp.SUBJECTRequest, dsp.SUBJECTResponse, dsp.SUBJECTAttribute, dsp.SUBJECTWinLoss:
		return true
	default:
		return false
	}
}

func spreadSubjectPath(subject string) (string, string, bool) {
	filename := strings.ReplaceAll(subject, ":", "/")
	if unsafePath(filename) {
		return "", "", false
	}
	dir, base := filepath.Split(filename)
	if dir == "" || base == "" {
		return "", "", false
	}
	if strings.HasPrefix(dir, acl.HashNamePubmap+"/") ||
		strings.HasPrefix(dir, match.HashNameAudience+"/") ||
		strings.HasPrefix(dir, match.HashNameSlot+"/") ||
		strings.HasPrefix(dir, match.HashNameCreative+"/") {
		return dir, base, true
	}
	return "", "", false
}

func unsafePath(filename string) bool {
	if filepath.IsAbs(filename) {
		return true
	}
	for _, part := range strings.Split(filename, "/") {
		if part == "" || part == "." || part == ".." {
			return true
		}
	}
	return false
}

func writeSnapshot(filename string, data []byte) error {
	w, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer w.Close()

	if err := syscall.Flock(int(w.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() {
		if err := syscall.Flock(int(w.Fd()), syscall.LOCK_UN); err != nil && !errors.Is(err, os.ErrClosed) {
			log.Println("Error releasing lock:", err)
		}
	}()

	_, err = w.Write(data)
	return err
}

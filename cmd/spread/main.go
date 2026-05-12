// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
	"github.com/nats-io/nats.go"

	_ "github.com/go-sql-driver/mysql"
)

const (
	spreadSubjectPattern = ">"
	spreadResetBase      = "__reset__"
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
	if err := bootstrapSpreadFromRedis(context.Background(), c, top); err != nil {
		log.Printf("spread bootstrap skipped: %v", err)
	}

	log.Printf("Listening on [%s]", nc.ConnectedUrl())
	successchan := make(chan bool)
	errchan := make(chan error)

	_, err = nc.Subscribe(spreadSubjectPattern, func(m *nats.Msg) {
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

	if base == spreadResetBase {
		return true, resetSpreadDir(top, strings.TrimSuffix(dir, "/"))
	}

	if strings.HasPrefix(dir, match.HashNameSlot+"/") {
		pure, ok := strings.CutSuffix(base, "cleanup")
		if ok {
			base = pure
			if err := os.RemoveAll(filepath.Join(top, dir)); err != nil {
				return true, err
			}
			if base == "" {
				return true, nil
			}
		}
	}

	if base == "" {
		return false, nil
	}

	fullPath := filepath.Join(top, dir, base)
	if string(m.Data) == "DELETE" {
		return true, os.RemoveAll(fullPath)
	}

	return true, writeSnapshot(fullPath, m.Data)
}

func resetSpreadDir(top, family string) error {
	if unsafePath(family) {
		return nil
	}
	return os.RemoveAll(filepath.Join(top, family))
}

func bootstrapSpreadFromRedis(ctx context.Context, c *dsp.Config, top string) error {
	redis, db, err := c.GetRedisDB(ctx)
	if err != nil {
		return err
	}
	defer redis.Close()
	defer db.Close()

	pubmap, err := acl.PubMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	if pubmap != nil {
		if err := resetSpreadDir(top, acl.HashNamePubmap); err != nil {
			return err
		}
		for domain, pub := range pubmap {
			if unsafePath(domain) {
				continue
			}
			data, err := pub.Pack()
			if err != nil {
				return err
			}
			if err := writeSnapshot(filepath.Join(top, acl.HashNamePubmap, domain), data); err != nil {
				return err
			}
		}
	}

	audiences, err := match.AudienceMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	if audiences != nil {
		if err := resetSpreadDir(top, match.HashNameAudience); err != nil {
			return err
		}
		for itemID, audience := range audiences {
			data, err := audience.Pack()
			if err != nil {
				return err
			}
			name := strconv.FormatUint(uint64(itemID), 10)
			if err := writeSnapshot(filepath.Join(top, match.HashNameAudience, name), data); err != nil {
				return err
			}
		}
	}

	creatives, err := match.CreativeMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	if creatives != nil {
		if err := resetSpreadDir(top, match.HashNameCreative); err != nil {
			return err
		}
		for creativeID, creative := range creatives {
			data, err := creative.Pack()
			if err != nil {
				return err
			}
			name := strconv.FormatUint(uint64(creativeID), 10)
			if err := writeSnapshot(filepath.Join(top, match.HashNameCreative, name), data); err != nil {
				return err
			}
		}
	}

	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
		return err
	}
	if err := resetSpreadDir(top, match.HashNameSlot); err != nil {
		return err
	}
	for _, sizeID := range sizeIDs {
		hash, err := match.RAdvsFromRedisBySizeID(ctx, redis, sizeID)
		if err != nil {
			return err
		}
		for slotID, radvs := range hash {
			data, err := radvs.Pack()
			if err != nil {
				return err
			}
			filename := filepath.Join(top, match.HashIONameRAdvs(sizeID), strconv.FormatUint(uint64(slotID), 10))
			if err := writeSnapshot(filename, data); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeSnapshot(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".spread-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := syscall.Flock(int(tmp.Fd()), syscall.LOCK_EX); err != nil {
		tmp.Close()
		return err
	}
	defer func() {
		if err := syscall.Flock(int(tmp.Fd()), syscall.LOCK_UN); err != nil && !errors.Is(err, os.ErrClosed) {
			log.Println("Error releasing lock:", err)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return err
	}
	return syncDir(dir)
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
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

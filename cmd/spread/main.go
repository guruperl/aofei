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
	"sync"
	"syscall"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/cmdboot"
	"github.com/guruperl/aofei/internal/spreadcache"
	"github.com/guruperl/aofei/match"
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

	ctx, stop := cmdboot.SignalContext(context.Background())
	defer stop()

	c, err := dsp.NewConfig(sConf)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Validate(dsp.ConfigModeSpread); err != nil {
		log.Fatal(err)
	}
	nc, err := nats.Connect(c.NatsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	if err := runSpread(ctx, c, nc, log.Default()); err != nil {
		log.Fatal(err)
	}
}

type spreadConn interface {
	ConnectedUrl() string
	Subscribe(string, nats.MsgHandler) (*nats.Subscription, error)
	FlushTimeout(time.Duration) error
	Drain() error
}

func runSpread(ctx context.Context, c *dsp.Config, nc spreadConn, logger *log.Logger) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = log.Default()
	}
	top := c.Spread
	if err := os.MkdirAll(top, os.ModePerm); err != nil {
		return err
	}
	if err := bootstrapSpreadFromRedis(ctx, c, top); err != nil {
		logger.Printf("spread bootstrap skipped: %v", err)
	}
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		return err
	}

	_, err = nc.Subscribe(spreadSubjectPattern, func(m *nats.Msg) {
		handled, err := receiver.handle(m)
		if err != nil {
			logger.Printf("error: %v", err)
			return
		}
		if handled {
			logger.Printf("write %s", m.Subject)
		}
	})
	if err != nil {
		return err
	}
	if err := nc.FlushTimeout(5 * time.Second); err != nil {
		return err
	}
	logger.Printf("Listening on [%s]", nc.ConnectedUrl())

	<-ctx.Done()
	if err := nc.Drain(); err != nil {
		return err
	}
	return nil
}

type spreadGenerationReceiver struct {
	mu        sync.Mutex
	top       string
	active    uint64
	committed uint64
	expected  spreadcache.Manifest
	seen      map[string]string
}

func newSpreadGenerationReceiver(top string) (*spreadGenerationReceiver, error) {
	committed, ok, err := spreadcache.CurrentSequence(top)
	if err != nil {
		return nil, err
	}
	if !ok {
		committed = 0
	}
	return &spreadGenerationReceiver{top: top, committed: committed}, nil
}

func (r *spreadGenerationReceiver) handle(m *nats.Msg) (bool, error) {
	if m == nil {
		return false, errors.New("spread message is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch m.Subject {
	case spreadcache.BeginSubject:
		manifest, err := spreadcache.ParseManifest(m.Data)
		if err != nil {
			return true, err
		}
		return true, r.begin(manifest)
	case spreadcache.DataSubject:
		sequence, err := spreadcache.ParseSequence(m.Header.Get(spreadcache.GenerationHeader))
		if err != nil {
			return true, err
		}
		return true, r.put(sequence, m.Header.Get(spreadcache.OriginalSubjectHeader), m.Data)
	case spreadcache.CommitSubject:
		sequence, err := spreadcache.ParseSequence(string(m.Data))
		if err != nil {
			return true, err
		}
		return true, r.commit(sequence)
	default:
		if r.committed != 0 {
			if _, _, ok := spreadSubjectPath(m.Subject); ok {
				return false, nil
			}
		}
		return handleSpreadMessage(r.top, m)
	}
}

func (r *spreadGenerationReceiver) begin(manifest spreadcache.Manifest) error {
	if manifest.Sequence <= r.committed || manifest.Sequence < r.active {
		return nil
	}
	if manifest.Sequence == r.active {
		if r.expected != manifest {
			return fmt.Errorf("spread generation %d manifest changed", manifest.Sequence)
		}
		return nil
	}
	if r.active != 0 {
		if err := os.RemoveAll(spreadcache.GenerationRoot(r.top, r.active)); err != nil {
			return err
		}
	}
	root := spreadcache.GenerationRoot(r.top, manifest.Sequence)
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	for _, family := range []string{acl.HashNamePubmap, match.HashNameAudience, match.HashNameCreative, match.HashNameSlot} {
		if err := os.MkdirAll(filepath.Join(root, family), 0750); err != nil {
			return err
		}
	}
	r.active = manifest.Sequence
	r.expected = manifest
	r.seen = make(map[string]string)
	return nil
}

func (r *spreadGenerationReceiver) put(sequence uint64, subject string, data []byte) error {
	if sequence != r.active || sequence <= r.committed {
		return nil
	}
	relative, ok := spreadcache.RelativePath(subject)
	if !ok {
		return fmt.Errorf("unsupported spread generation subject %q", subject)
	}
	digest := spreadcache.Digest(data)
	if prior, ok := r.seen[subject]; ok && prior != digest {
		return fmt.Errorf("spread generation %d subject %q changed", sequence, subject)
	}
	if err := writeSnapshot(filepath.Join(spreadcache.GenerationRoot(r.top, sequence), relative), data); err != nil {
		return err
	}
	r.seen[subject] = digest
	return nil
}

func (r *spreadGenerationReceiver) commit(sequence uint64) error {
	if sequence <= r.committed {
		return nil
	}
	if sequence != r.active {
		return nil
	}
	if len(r.seen) != r.expected.EntryCount {
		return fmt.Errorf("spread generation %d is incomplete: received %d of %d entries", sequence, len(r.seen), r.expected.EntryCount)
	}
	if digest := spreadcache.ManifestDigest(r.seen); digest != r.expected.SHA256 {
		return fmt.Errorf("spread generation %d manifest digest mismatch", sequence)
	}
	if err := cleanupSpreadGenerations(r.top, r.committed, sequence); err != nil {
		return err
	}
	if err := spreadcache.Commit(r.top, sequence); err != nil {
		return err
	}
	r.committed = sequence
	r.active = 0
	r.expected = spreadcache.Manifest{}
	r.seen = nil
	return nil
}

func cleanupSpreadGenerations(top string, keep ...uint64) error {
	kept := make(map[uint64]struct{}, len(keep))
	for _, sequence := range keep {
		if sequence != 0 {
			kept[sequence] = struct{}{}
		}
	}
	root := filepath.Dir(spreadcache.GenerationRoot(top, 1))
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sequence, err := strconv.ParseUint(entry.Name(), 10, 64)
		if err != nil || sequence == 0 {
			continue
		}
		if _, ok := kept[sequence]; ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
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

	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		return err
	}
	messages := make([]spreadcache.Message, 0)
	pubmap, err := acl.PubMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	if pubmap != nil {
		for domain, pub := range pubmap {
			if unsafePath(domain) {
				continue
			}
			data, err := pub.Pack()
			if err != nil {
				return err
			}
			messages = append(messages, spreadcache.Message{Subject: acl.HashNamePubmap + ":" + domain, Data: data})
		}
	}

	audiences, err := match.AudienceMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	if audiences != nil {
		for itemID, audience := range audiences {
			data, err := audience.Pack()
			if err != nil {
				return err
			}
			name := strconv.FormatUint(uint64(itemID), 10)
			messages = append(messages, spreadcache.Message{Subject: match.HashNameAudience + ":" + name, Data: data})
		}
	}

	creatives, err := match.CreativeMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	if creatives != nil {
		for creativeID, creative := range creatives {
			data, err := creative.Pack()
			if err != nil {
				return err
			}
			name := strconv.FormatUint(uint64(creativeID), 10)
			messages = append(messages, spreadcache.Message{Subject: match.HashNameCreative + ":" + name, Data: data})
		}
	}

	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
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
			subject := match.HashNameRAdvs(sizeID) + ":" + strconv.FormatUint(uint64(slotID), 10)
			messages = append(messages, spreadcache.Message{Subject: subject, Data: data})
		}
	}
	sequence, err := spreadcache.NextSequence(ctx, redis, receiver.committed)
	if err != nil {
		return err
	}
	manifest, err := spreadcache.NewManifest(sequence, messages)
	if err != nil {
		return err
	}
	if err := receiver.begin(manifest); err != nil {
		return err
	}
	for _, message := range messages {
		if err := receiver.put(sequence, message.Subject, message.Data); err != nil {
			return err
		}
	}
	return receiver.commit(sequence)
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

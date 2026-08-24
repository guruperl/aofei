// this receives 4 logs from web server. It should run as a service, after the nats server is up.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/atomicfile"
	"github.com/guruperl/aofei/internal/cmdboot"
	cachejob "github.com/guruperl/aofei/internal/jobs/cache"
	"github.com/guruperl/aofei/internal/opsmetrics"
	"github.com/guruperl/aofei/internal/spreadcache"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
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
	if err := atomicfile.EnsureDir(top, 0750); err != nil {
		return err
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

	// Install and flush the subscription before best-effort bootstrap so a
	// publication cannot disappear in the startup gap. Existing disk state or a
	// generation received during the flush is already a complete snapshot and
	// must not be overwritten by a sequential Redis reconstruction.
	if !receiver.hasCommittedGeneration() {
		if err := bootstrapSpreadFromRedis(ctx, c, receiver); err != nil {
			opsmetrics.RecordSpread("bootstrap_failed")
			logger.Printf("spread bootstrap skipped: %v", err)
		} else {
			opsmetrics.RecordSpread("bootstrap_succeeded")
		}
	}

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
	selected  bool
	expected  spreadcache.Manifest
	seen      map[string]string
	prepare   func(string) error
}

func newSpreadGenerationReceiver(top string) (*spreadGenerationReceiver, error) {
	committed, ok, err := spreadcache.CurrentSequence(top)
	if err != nil {
		return nil, err
	}
	if !ok {
		committed = 0
	}
	selected := false
	if committed != 0 {
		info, statErr := os.Stat(spreadcache.GenerationRoot(top, committed))
		switch {
		case statErr == nil && info.IsDir():
			selected = true
		case statErr == nil:
			return nil, fmt.Errorf("spread generation %d is not a directory", committed)
		case errors.Is(statErr, os.ErrNotExist):
			// Preserve the numeric floor but allow bootstrap to repair the
			// missing selected directory.
		case statErr != nil:
			return nil, statErr
		}
	}
	return &spreadGenerationReceiver{top: top, committed: committed, selected: selected}, nil
}

func (r *spreadGenerationReceiver) hasCommittedGeneration() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.selected
}

func (r *spreadGenerationReceiver) committedSequence() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.committed
}

func (r *spreadGenerationReceiver) install(sequence uint64, messages []spreadcache.Message) error {
	manifest, err := spreadcache.NewManifest(sequence, messages)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.begin(manifest); err != nil {
		return err
	}
	for _, message := range messages {
		if err := r.put(sequence, message.Subject, message.Data); err != nil {
			return err
		}
	}
	return r.commit(sequence)
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
			opsmetrics.RecordSpread("generation_rejected")
			return true, err
		}
		return true, r.begin(manifest)
	case spreadcache.DataSubject:
		sequence, err := spreadcache.ParseSequence(m.Header.Get(spreadcache.GenerationHeader))
		if err != nil {
			opsmetrics.RecordSpread("generation_rejected")
			return true, err
		}
		err = r.put(sequence, m.Header.Get(spreadcache.OriginalSubjectHeader), m.Data)
		if err != nil {
			opsmetrics.RecordSpread("generation_rejected")
		}
		return true, err
	case spreadcache.CommitSubject:
		sequence, err := spreadcache.ParseSequence(string(m.Data))
		if err != nil {
			opsmetrics.RecordSpread("generation_rejected")
			return true, err
		}
		return true, r.commit(sequence)
	default:
		if r.committed != 0 {
			if _, _, ok := spreadSubjectPath(m.Subject); ok {
				return false, nil
			}
		}
		handled, err := handleSpreadMessage(r.top, m)
		if handled {
			outcome := "legacy_write_succeeded"
			if err != nil {
				outcome = "legacy_write_failed"
			}
			opsmetrics.RecordSpread(outcome)
		}
		return handled, err
	}
}

func (r *spreadGenerationReceiver) begin(manifest spreadcache.Manifest) error {
	if manifest.Sequence <= r.committed || manifest.Sequence < r.active {
		opsmetrics.RecordSpread("generation_stale")
		return nil
	}
	if manifest.Sequence == r.active {
		if r.expected != manifest {
			opsmetrics.RecordSpread("generation_rejected")
			return fmt.Errorf("spread generation %d manifest changed", manifest.Sequence)
		}
		opsmetrics.RecordSpread("generation_stale")
		return nil
	}
	root := spreadcache.GenerationRoot(r.top, manifest.Sequence)
	prepare := r.prepare
	if prepare == nil {
		prepare = prepareSpreadGenerationRoot
	}
	if err := prepare(root); err != nil {
		opsmetrics.RecordSpread("generation_rejected")
		return err
	}
	if r.active != 0 {
		if err := os.RemoveAll(spreadcache.GenerationRoot(r.top, r.active)); err != nil {
			_ = os.RemoveAll(root)
			opsmetrics.RecordSpread("generation_rejected")
			return err
		}
	}
	r.active = manifest.Sequence
	r.expected = manifest
	r.seen = make(map[string]string)
	opsmetrics.RecordSpread("generation_started")
	return nil
}

func prepareSpreadGenerationRoot(root string) error {
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	for _, family := range []string{acl.HashNamePubmap, match.HashNameAudience, match.HashNameCreative, match.HashNameSlot} {
		if err := atomicfile.EnsureDir(filepath.Join(root, family), 0750); err != nil {
			return err
		}
	}
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
		opsmetrics.RecordSpread("generation_stale")
		return nil
	}
	if sequence != r.active {
		opsmetrics.RecordSpread("generation_stale")
		return nil
	}
	if len(r.seen) != r.expected.EntryCount {
		opsmetrics.RecordSpread("generation_incomplete")
		return fmt.Errorf("spread generation %d is incomplete: received %d of %d entries", sequence, len(r.seen), r.expected.EntryCount)
	}
	if digest := spreadcache.ManifestDigest(r.seen); digest != r.expected.SHA256 {
		opsmetrics.RecordSpread("generation_rejected")
		return fmt.Errorf("spread generation %d manifest digest mismatch", sequence)
	}
	if err := cleanupSpreadGenerations(r.top, r.committed, sequence); err != nil {
		opsmetrics.RecordSpread("generation_rejected")
		return err
	}
	if err := spreadcache.Commit(r.top, sequence); err != nil {
		opsmetrics.RecordSpread("generation_rejected")
		return err
	}
	r.committed = sequence
	r.selected = true
	r.active = 0
	r.expected = spreadcache.Manifest{}
	r.seen = nil
	opsmetrics.RecordSpread("generation_committed")
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

func bootstrapSpreadFromRedis(ctx context.Context, c *dsp.Config, receiver *spreadGenerationReceiver) error {
	if receiver == nil {
		return errors.New("spread generation receiver is nil")
	}
	redis, db, err := c.GetRedisDB(ctx)
	if err != nil {
		return err
	}
	defer redis.Close()
	defer db.Close()
	return cmdboot.WithLock(ctx, redis, cachejob.MutationLockKey, 30*time.Minute, func(leaseCtx context.Context) error {
		if receiver.hasCommittedGeneration() {
			return nil
		}
		return bootstrapSpreadSnapshot(leaseCtx, redis, db, receiver)
	})
}

func bootstrapSpreadSnapshot(ctx context.Context, redis radix.Client, db *sql.DB, receiver *spreadGenerationReceiver) error {
	if receiver.hasCommittedGeneration() {
		return nil
	}
	pubmap, err := acl.DBGetPubMapContext(ctx, db)
	if err != nil {
		return err
	}
	if err := acl.ValidateCommercialPubMap(pubmap); err != nil {
		return fmt.Errorf("publisher inventory readiness: %w", err)
	}

	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
		return err
	}
	messages, err := cachejob.BuildSpreadGeneration(ctx, db, pubmap, sizeIDs)
	if err != nil {
		return err
	}
	sequence, err := spreadcache.NextSequence(ctx, redis, receiver.committedSequence())
	if err != nil {
		return err
	}
	return receiver.install(sequence, messages)
}

func writeSnapshot(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := atomicfile.EnsureDir(dir, 0750); err != nil {
		return err
	}
	return atomicfile.Write(filename, 0640, func(out io.Writer) error {
		_, err := out.Write(data)
		return err
	})
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

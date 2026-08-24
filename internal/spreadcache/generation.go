package spreadcache

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/guruperl/aofei/internal/atomicfile"
	"github.com/mediocregopher/radix/v4"
)

const (
	BeginSubject          = "_aofei.cache.generation.begin"
	DataSubject           = "_aofei.cache.generation.data"
	CommitSubject         = "_aofei.cache.generation.commit"
	GenerationHeader      = "Aofei-Cache-Generation"
	OriginalSubjectHeader = "Aofei-Cache-Subject"
	CounterKey            = "aofei:spread:generation"

	currentFile    = ".aofei-current"
	generationsDir = ".aofei-generations"
	selectionLock  = ".aofei-current.lock"
)

type Message struct {
	Subject string
	Data    []byte
}

type Manifest struct {
	Sequence   uint64 `json:"sequence"`
	EntryCount int    `json:"entry_count"`
	SHA256     string `json:"sha256"`
}

func NextSequence(ctx context.Context, redis radix.Client, floors ...uint64) (uint64, error) {
	if redis == nil {
		return 0, errors.New("spread generation redis client is nil")
	}
	var floor uint64
	for _, candidate := range floors {
		if candidate > floor {
			floor = candidate
		}
	}
	var sequence int64
	const advanceSequence = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local floor = tonumber(ARGV[1])
if current < floor then
  redis.call('SET', KEYS[1], ARGV[1])
end
return redis.call('INCR', KEYS[1])`
	if err := redis.Do(ctx, radix.Cmd(&sequence, "EVAL", advanceSequence, "1", CounterKey, SequenceString(floor))); err != nil {
		return 0, err
	}
	if sequence <= 0 {
		return 0, fmt.Errorf("spread generation sequence is invalid: %d", sequence)
	}
	return uint64(sequence), nil
}

func NewManifest(sequence uint64, messages []Message) (Manifest, error) {
	if sequence == 0 {
		return Manifest{}, errors.New("spread generation sequence is zero")
	}
	entries := make(map[string]string, len(messages))
	for _, message := range messages {
		if _, ok := RelativePath(message.Subject); !ok {
			return Manifest{}, fmt.Errorf("unsupported spread generation subject %q", message.Subject)
		}
		if _, ok := entries[message.Subject]; ok {
			return Manifest{}, fmt.Errorf("duplicate spread generation subject %q", message.Subject)
		}
		entries[message.Subject] = Digest(message.Data)
	}
	return Manifest{Sequence: sequence, EntryCount: len(entries), SHA256: ManifestDigest(entries)}, nil
}

func (m Manifest) Marshal() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (m Manifest) Validate() error {
	if m.Sequence == 0 {
		return errors.New("spread generation sequence is zero")
	}
	if m.EntryCount < 0 {
		return errors.New("spread generation entry count is negative")
	}
	digest, err := hex.DecodeString(m.SHA256)
	if err != nil || len(digest) != sha256.Size {
		return errors.New("spread generation manifest digest is invalid")
	}
	if m.EntryCount == 0 && m.SHA256 != ManifestDigest(nil) {
		return errors.New("empty spread generation manifest digest is invalid")
	}
	return nil
}

func Digest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func ManifestDigest(entries map[string]string) string {
	subjects := make([]string, 0, len(entries))
	for subject := range entries {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	hash := sha256.New()
	var size [8]byte
	for _, subject := range subjects {
		digest := entries[subject]
		binary.BigEndian.PutUint64(size[:], uint64(len(subject)))
		hash.Write(size[:])
		hash.Write([]byte(subject))
		binary.BigEndian.PutUint64(size[:], uint64(len(digest)))
		hash.Write(size[:])
		hash.Write([]byte(digest))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func SequenceString(sequence uint64) string {
	return strconv.FormatUint(sequence, 10)
}

func ParseSequence(value string) (uint64, error) {
	sequence, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || sequence == 0 {
		return 0, fmt.Errorf("invalid spread generation sequence %q", value)
	}
	return sequence, nil
}

func RelativePath(subject string) (string, bool) {
	parts := strings.Split(subject, ":")
	switch {
	case len(parts) == 2 && parts[0] == "pubmap" && safePart(parts[1]):
		return filepath.Join(parts...), true
	case len(parts) == 2 && (parts[0] == "audience" || parts[0] == "creative") && canonicalUint32(parts[1]):
		return filepath.Join(parts...), true
	case len(parts) == 3 && parts[0] == "slot" && canonicalUint32(parts[1]) && canonicalUint32(parts[2]):
		return filepath.Join(parts...), true
	default:
		return "", false
	}
}

func GenerationRoot(top string, sequence uint64) string {
	return filepath.Join(top, generationsDir, SequenceString(sequence))
}

// NewStagingRoot creates a private generation tree that cannot collide with a
// second receiver staging the same published sequence.
func NewStagingRoot(top string, sequence uint64) (string, error) {
	if sequence == 0 {
		return "", errors.New("spread generation sequence is zero")
	}
	root := filepath.Join(top, generationsDir)
	if err := atomicfile.EnsureDir(root, 0750); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(root, "."+SequenceString(sequence)+"-staging-")
	if err != nil {
		return "", err
	}
	if err := os.Chmod(staging, 0750); err != nil {
		_ = os.RemoveAll(staging)
		return "", err
	}
	return staging, nil
}

func CurrentSequence(top string) (uint64, bool, error) {
	data, err := os.ReadFile(filepath.Join(top, currentFile))
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	sequence, err := ParseSequence(string(data))
	if err != nil {
		return 0, false, err
	}
	return sequence, true, nil
}

func Resolve(top string) (string, error) {
	sequence, ok, err := CurrentSequence(top)
	if err != nil || !ok {
		return top, err
	}
	root := GenerationRoot(top, sequence)
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("resolve spread generation %d: %w", sequence, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("spread generation %d is not a directory", sequence)
	}
	return root, nil
}

// WithResolved holds the shared side of the generation-selection lock for one
// complete read. Cleanup and pointer replacement cannot prune or supersede the
// resolved root until read returns.
func WithResolved(top string, read func(string) error) error {
	return WithResolvedContext(context.Background(), top, read)
}

// WithResolvedContext is WithResolved with cancellation while waiting for the
// shared selection lock.
func WithResolvedContext(ctx context.Context, top string, read func(string) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if read == nil {
		return errors.New("spread generation reader is nil")
	}
	return withSelectionLock(ctx, top, false, func() error {
		root, err := Resolve(top)
		if err != nil {
			return err
		}
		return read(root)
	})
}

func Commit(top string, sequence uint64) error {
	return CommitContext(context.Background(), top, sequence)
}

// CommitContext switches the selected generation only while the owning
// operation remains valid. Selection is serialized across processes and never
// moves the pointer backward.
func CommitContext(ctx context.Context, top string, sequence uint64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if sequence == 0 {
		return errors.New("spread generation sequence is zero")
	}
	if err := atomicfile.EnsureDir(top, 0750); err != nil {
		return err
	}
	return withSelectionLock(ctx, top, true, func() error {
		current, ok, err := CurrentSequence(top)
		if err != nil {
			return err
		}
		if ok && current >= sequence {
			return nil
		}
		return writeCurrentContext(ctx, top, sequence)
	})
}

// InstallContext atomically installs one private staging tree and selects it.
// It returns the sequence selected at completion, which can be higher than the
// candidate when another process won the race first.
func InstallContext(ctx context.Context, top string, sequence uint64, staging string) (selected uint64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if sequence == 0 {
		return 0, errors.New("spread generation sequence is zero")
	}
	if err := validateStagingRoot(top, sequence, staging); err != nil {
		return 0, err
	}
	if err := atomicfile.EnsureDir(top, 0750); err != nil {
		return 0, err
	}
	err = withSelectionLock(ctx, top, true, func() error {
		current, ok, err := CurrentSequence(top)
		if err != nil {
			return err
		}
		if ok && current >= sequence {
			selected = current
			return os.RemoveAll(staging)
		}
		info, err := os.Stat(staging)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("spread generation staging path %q is not a directory", staging)
		}
		root := GenerationRoot(top, sequence)
		if err := os.RemoveAll(root); err != nil {
			return err
		}
		if err := os.Rename(staging, root); err != nil {
			return err
		}
		if err := atomicfile.SyncDir(filepath.Dir(root)); err != nil {
			return err
		}
		if err := cleanupOlderGenerations(ctx, top, current, sequence); err != nil {
			return err
		}
		if err := writeCurrentContext(ctx, top, sequence); err != nil {
			return err
		}
		selected = sequence
		return nil
	})
	return selected, err
}

func writeCurrentContext(ctx context.Context, top string, sequence uint64) error {
	return atomicfile.WriteContext(ctx, filepath.Join(top, currentFile), 0640, func(out io.Writer) error {
		_, err := fmt.Fprintf(out, "%d\n", sequence)
		return err
	})
}

func validateStagingRoot(top string, sequence uint64, staging string) error {
	root := filepath.Clean(filepath.Join(top, generationsDir))
	clean := filepath.Clean(staging)
	if filepath.Dir(clean) != root {
		return fmt.Errorf("spread generation staging path %q is outside %q", staging, root)
	}
	prefix := "." + SequenceString(sequence) + "-staging-"
	if base := filepath.Base(clean); !strings.HasPrefix(base, prefix) || len(base) == len(prefix) {
		return fmt.Errorf("spread generation staging path %q does not match sequence %d", staging, sequence)
	}
	return nil
}

func withSelectionLock(ctx context.Context, top string, exclusive bool, selectGeneration func() error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(top, selectionLock), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	locked := false
	defer func() {
		var unlockErr error
		if locked {
			unlockErr = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		}
		err = errors.Join(err, unlockErr, lock.Close())
	}()
	if err := lock.Chmod(0600); err != nil {
		return err
	}

	const retryDelay = 10 * time.Millisecond
	operation := syscall.LOCK_SH
	if exclusive {
		operation = syscall.LOCK_EX
	}
	for {
		err := syscall.Flock(int(lock.Fd()), operation|syscall.LOCK_NB)
		if err == nil {
			locked = true
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return selectGeneration()
}

func cleanupOlderGenerations(ctx context.Context, top string, keep ...uint64) error {
	kept := make(map[uint64]struct{}, len(keep))
	var ceiling uint64
	for _, sequence := range keep {
		if sequence != 0 {
			kept[sequence] = struct{}{}
			if sequence > ceiling {
				ceiling = sequence
			}
		}
	}
	root := filepath.Join(top, generationsDir)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() {
			continue
		}
		sequence, staging, ok := generationEntrySequence(entry.Name())
		if !ok || sequence > ceiling {
			continue
		}
		if _, ok := kept[sequence]; ok && !staging {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func generationEntrySequence(name string) (sequence uint64, staging bool, ok bool) {
	if parsed, err := strconv.ParseUint(name, 10, 64); err == nil && parsed != 0 {
		return parsed, false, true
	}
	if !strings.HasPrefix(name, ".") {
		return 0, false, false
	}
	sequencePart, suffix, found := strings.Cut(strings.TrimPrefix(name, "."), "-staging-")
	if !found || suffix == "" {
		return 0, false, false
	}
	parsed, err := strconv.ParseUint(sequencePart, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false, false
	}
	return parsed, true, true
}

func canonicalUint32(value string) bool {
	parsed, err := strconv.ParseUint(value, 10, 32)
	return err == nil && strconv.FormatUint(parsed, 10) == value
}

func safePart(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, `/\\`)
}

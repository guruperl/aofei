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

func Commit(top string, sequence uint64) error {
	if sequence == 0 {
		return errors.New("spread generation sequence is zero")
	}
	if err := atomicfile.EnsureDir(top, 0750); err != nil {
		return err
	}
	return atomicfile.Write(filepath.Join(top, currentFile), 0640, func(out io.Writer) error {
		_, err := fmt.Fprintf(out, "%d\n", sequence)
		return err
	})
}

func canonicalUint32(value string) bool {
	parsed, err := strconv.ParseUint(value, 10, 32)
	return err == nil && strconv.FormatUint(parsed, 10) == value
}

func safePart(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, `/\\`)
}

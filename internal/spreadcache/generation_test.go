package spreadcache

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func TestNextSequenceAdvancesPastCommittedFloor(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	sequence, err := NextSequence(ctx, client, 41)
	if err != nil {
		t.Fatal(err)
	}
	if sequence != 42 {
		t.Fatalf("sequence = %d, want 42", sequence)
	}
	sequence, err = NextSequence(ctx, client, 2)
	if err != nil {
		t.Fatal(err)
	}
	if sequence != 43 {
		t.Fatalf("sequence = %d, want 43", sequence)
	}
}

func TestManifestRejectsDuplicateAndUnsafeSubjects(t *testing.T) {
	for _, messages := range [][]Message{
		{{Subject: "creative:7", Data: []byte("one")}, {Subject: "creative:7", Data: []byte("two")}},
		{{Subject: "creative:../escape", Data: []byte("bad")}},
		{{Subject: "slot:1:2cleanup", Data: []byte("legacy")}},
	} {
		if _, err := NewManifest(1, messages); err == nil {
			t.Fatalf("NewManifest(%#v) error = nil", messages)
		}
	}
}

func TestManifestEncodingStaysBoundedAsInventoryGrows(t *testing.T) {
	messages := make([]Message, 1000)
	for i := range messages {
		messages[i] = Message{Subject: "creative:" + SequenceString(uint64(i+1)), Data: []byte("payload")}
	}
	manifest, err := NewManifest(1, messages)
	if err != nil {
		t.Fatal(err)
	}
	data, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 256 {
		t.Fatalf("manifest encoding grew to %d bytes", len(data))
	}
}

func TestCommitAndResolveGeneration(t *testing.T) {
	top := t.TempDir()
	root := GenerationRoot(top, 7)
	if err := os.MkdirAll(filepath.Join(root, "creative"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := Commit(top, 7); err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != root {
		t.Fatalf("resolved root = %q, want %q", resolved, root)
	}
}

func TestResolveRejectsMissingCommittedGeneration(t *testing.T) {
	top := t.TempDir()
	if err := Commit(top, 9); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(top); err == nil {
		t.Fatal("Resolve error = nil, want missing generation rejection")
	}
}

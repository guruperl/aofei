package spreadcache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func privateTempDir(t testing.TB) string {
	t.Helper()
	path := t.TempDir()
	if err := os.Chmod(path, 0750); err != nil {
		t.Fatal(err)
	}
	return path
}

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
	top := privateTempDir(t)
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
	top := privateTempDir(t)
	if err := Commit(top, 9); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(top); err == nil {
		t.Fatal("Resolve error = nil, want missing generation rejection")
	}
}

func TestCommitNeverMovesSelectionBackward(t *testing.T) {
	top := privateTempDir(t)
	if err := Commit(top, 12); err != nil {
		t.Fatal(err)
	}
	if err := Commit(top, 11); err != nil {
		t.Fatal(err)
	}
	sequence, ok, err := CurrentSequence(top)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || sequence != 12 {
		t.Fatalf("current sequence = %d, %t; want 12, true", sequence, ok)
	}
}

func TestSelectRemovesOnlySupersededGenerations(t *testing.T) {
	top := privateTempDir(t)
	for _, sequence := range []uint64{7, 8, 10} {
		if err := os.MkdirAll(GenerationRoot(top, sequence), 0750); err != nil {
			t.Fatal(err)
		}
	}
	if err := Commit(top, 8); err != nil {
		t.Fatal(err)
	}
	staging, err := NewStagingRoot(top, 9)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := InstallContext(context.Background(), top, 9, staging)
	if err != nil {
		t.Fatal(err)
	}
	if selected != 9 {
		t.Fatalf("selected sequence = %d, want 9", selected)
	}
	if _, err := os.Stat(GenerationRoot(top, 7)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("superseded generation still exists: %v", err)
	}
	for _, sequence := range []uint64{8, 9, 10} {
		if _, err := os.Stat(GenerationRoot(top, sequence)); err != nil {
			t.Fatalf("retained generation %d: %v", sequence, err)
		}
	}
}

func TestWithResolvedRetainsRootUntilReadCompletes(t *testing.T) {
	top := privateTempDir(t)
	root := GenerationRoot(top, 1)
	if err := os.MkdirAll(root, 0750); err != nil {
		t.Fatal(err)
	}
	if err := Commit(top, 1); err != nil {
		t.Fatal(err)
	}
	reading := make(chan struct{})
	release := make(chan struct{})
	readDone := make(chan error, 1)
	go func() {
		readDone <- WithResolved(top, func(resolved string) error {
			if resolved != root {
				return errors.New("resolved unexpected generation")
			}
			close(reading)
			<-release
			return nil
		})
	}()
	<-reading

	staging, err := NewStagingRoot(top, 2)
	if err != nil {
		t.Fatal(err)
	}
	installDone := make(chan error, 1)
	go func() {
		_, err := InstallContext(context.Background(), top, 2, staging)
		installDone <- err
	}()
	select {
	case err := <-installDone:
		t.Fatalf("generation selection completed during retained read: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("resolved root was pruned during read: %v", err)
	}
	close(release)
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	if err := <-installDone; err != nil {
		t.Fatal(err)
	}
}

func TestCommitContextCancelsWhileWaitingForSelectionLock(t *testing.T) {
	top := privateTempDir(t)
	if err := Commit(top, 12); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(top, selectionLock), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	err = CommitContext(ctx, top, 13)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CommitContext error = %v, want deadline exceeded", err)
	}
	sequence, ok, err := CurrentSequence(top)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || sequence != 12 {
		t.Fatalf("current sequence = %d, %t; want 12, true", sequence, ok)
	}
}

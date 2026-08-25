package atomicfile

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWriteContextCancellationPreservesPriorFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	err := WriteContext(ctx, path, 0640, func(out io.Writer) error {
		if _, err := out.Write([]byte("new")); err != nil {
			return err
		}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteContext error = %v, want context canceled", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("snapshot = %q, want prior content", data)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".snapshot.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestWriteReplacesFileWithPrivateDurableTemporary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot")
	if err := os.WriteFile(path, []byte("old"), 0666); err != nil {
		t.Fatal(err)
	}

	if err := Write(path, 0640, func(out io.Writer) error {
		_, err := out.Write([]byte("new"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("snapshot = %q, want new", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0640 {
		t.Fatalf("snapshot mode = %04o, want 0640", got)
	}
}

func TestWriteFailurePreservesPriorFileAndRemovesTemporary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("encode failed")
	err := Write(path, 0640, func(out io.Writer) error {
		if _, err := out.Write([]byte("partial")); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Write() error = %v, want %v", err, wantErr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("snapshot = %q, want old", data)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".snapshot.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestEnsureDirValidatesWithoutMutatingExistingDirectory(t *testing.T) {
	parent := t.TempDir()
	for name, mode := range map[string]os.FileMode{
		"wide":          0777,
		"shared-setgid": 0775 | os.ModeSetgid,
		"sticky":        0777 | os.ModeSticky,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(parent, name)
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			before := fullMode(t, path)
			if err := EnsureDir(path, 0750); err == nil {
				t.Fatal("over-permissive existing directory was accepted")
			}
			if after := fullMode(t, path); after != before {
				t.Fatalf("existing mode changed from %v to %v", before, after)
			}
		})
	}

	for name, mode := range map[string]os.FileMode{
		"private":        0700,
		"safe-setgid":    0750 | os.ModeSetgid,
		"private-sticky": 0700 | os.ModeSticky,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(parent, name)
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			before := fullMode(t, path)
			if err := EnsureDir(path, 0750); err != nil {
				t.Fatal(err)
			}
			if after := fullMode(t, path); after != before {
				t.Fatalf("existing mode changed from %v to %v", before, after)
			}
		})
	}

	nested := filepath.Join(parent, "one", "two")
	if err := EnsureDir(nested, 0750); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Join(parent, "one"), 0750)
	assertMode(t, nested, 0750)
}

func TestEnsureDirConcurrentCreationDoesNotChmodRaceWinner(t *testing.T) {
	nested := filepath.Join(t.TempDir(), "shared", "nested")
	start := make(chan struct{})
	errors := make(chan error, 32)
	var wait sync.WaitGroup
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			errors <- EnsureDir(nested, 0750)
		}()
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertMode(t, filepath.Dir(nested), 0750)
	assertMode(t, nested, 0750)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}

func fullMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode() & (os.ModePerm | os.ModeSetgid | os.ModeSticky)
}

package atomicfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

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

func TestEnsureDirTightensWithoutBroadening(t *testing.T) {
	parent := t.TempDir()
	wide := filepath.Join(parent, "wide")
	if err := os.Mkdir(wide, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(wide, 0777); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDir(wide, 0750); err != nil {
		t.Fatal(err)
	}
	assertMode(t, wide, 0750)

	private := filepath.Join(parent, "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDir(private, 0750); err != nil {
		t.Fatal(err)
	}
	assertMode(t, private, 0700)

	nested := filepath.Join(parent, "one", "two")
	if err := EnsureDir(nested, 0750); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Join(parent, "one"), 0750)
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

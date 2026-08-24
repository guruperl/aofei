// Package atomicfile provides durable local-file replacement helpers.
package atomicfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureDir creates path and any missing parents, syncing each new directory
// entry before returning. An existing final directory is tightened to at most
// perm, but a more restrictive mode is preserved.
func EnsureDir(path string, perm fs.FileMode) error {
	if path == "" {
		return errors.New("directory path is empty")
	}
	perm &= fs.ModePerm
	if perm == 0 {
		return errors.New("directory permission is empty")
	}

	clean := filepath.Clean(path)
	missing := make([]string, 0, 4)
	for candidate := clean; ; candidate = filepath.Dir(candidate) {
		info, err := os.Stat(candidate)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("directory path %q is not a directory", candidate)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		missing = append(missing, candidate)
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return fmt.Errorf("no existing parent for directory %q", clean)
		}
	}

	for i := len(missing) - 1; i >= 0; i-- {
		dir := missing[i]
		if err := os.Mkdir(dir, perm); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		if err := os.Chmod(dir, perm); err != nil {
			return err
		}
		if err := SyncDir(filepath.Dir(dir)); err != nil {
			return err
		}
	}

	info, err := os.Stat(clean)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("directory path %q is not a directory", clean)
	}
	current := info.Mode().Perm()
	tightened := current & perm
	if tightened != current {
		if err := os.Chmod(clean, tightened); err != nil {
			return err
		}
		return SyncDir(clean)
	}
	return nil
}

// Write replaces filename only after write succeeds and the temporary file is
// flushed and closed. It then syncs the containing directory so the rename is
// durable. The caller must create the containing directory first.
func Write(filename string, perm fs.FileMode, write func(io.Writer) error) error {
	if filename == "" {
		return errors.New("file path is empty")
	}
	if write == nil {
		return errors.New("atomic file writer is nil")
	}
	perm &= fs.ModePerm
	if perm == 0 {
		return errors.New("file permission is empty")
	}

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if err := write(tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(tmpName, filename); err != nil {
		return err
	}
	return SyncDir(dir)
}

// SyncDir flushes directory metadata and closes the directory handle.
func SyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return err
	}
	return dir.Close()
}

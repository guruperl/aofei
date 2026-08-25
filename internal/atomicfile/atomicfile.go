// Package atomicfile provides durable local-file replacement helpers.
package atomicfile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/guruperl/aofei/internal/opsmetrics"
)

// EnsureDir creates path and any missing parents, syncing each new directory
// entry before returning. An existing final directory is never chmod'd: it may
// be more restrictive than perm, but permissions broader than perm fail.
func EnsureDir(path string, perm fs.FileMode) (err error) {
	defer func() {
		outcome := "directory_succeeded"
		if err != nil {
			outcome = "directory_failed"
		}
		opsmetrics.RecordFilesystem(outcome)
	}()
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
		created := false
		if err := os.Mkdir(dir, perm); err != nil {
			if !errors.Is(err, fs.ErrExist) {
				return err
			}
			info, statErr := os.Stat(dir)
			if statErr != nil {
				return statErr
			}
			if !info.IsDir() {
				return fmt.Errorf("directory path %q is not a directory", dir)
			}
		} else {
			created = true
		}
		if created {
			// Override umask only for a directory this call created. A path won by
			// another process is existing operator-owned state and is validated
			// below instead of mutated.
			if err := os.Chmod(dir, perm); err != nil {
				return err
			}
			if err := SyncDir(filepath.Dir(dir)); err != nil {
				return err
			}
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
	if extra := current &^ perm; extra != 0 {
		return fmt.Errorf("existing directory %q permissions %04o exceed allowed %04o", clean, current, perm)
	}
	return nil
}

// Write replaces filename only after write succeeds and the temporary file is
// flushed and closed. It then syncs the containing directory so the rename is
// durable. The caller must create the containing directory first.
func Write(filename string, perm fs.FileMode, write func(io.Writer) error) (err error) {
	return WriteContext(context.Background(), filename, perm, write)
}

// WriteContext is Write with a cancellation gate before replacement. If the
// context ends while the temporary file is being prepared, the selected file
// is left unchanged. Once rename starts, directory sync still completes so a
// successful replacement is never reported without its durability step.
func WriteContext(ctx context.Context, filename string, perm fs.FileMode, write func(io.Writer) error) (err error) {
	defer func() {
		outcome := "write_succeeded"
		if err != nil {
			outcome = "write_failed"
		}
		opsmetrics.RecordFilesystem(outcome)
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	closed = true
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return err
	}
	return SyncDir(dir)
}

// SyncDir flushes directory metadata and closes the directory handle.
func SyncDir(path string) (err error) {
	defer func() {
		if err != nil {
			opsmetrics.RecordFilesystem("sync_failed")
		}
	}()
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

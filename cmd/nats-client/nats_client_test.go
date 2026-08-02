package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/guruperl/aofei/dsp"
	"github.com/nats-io/nats.go"
)

func TestFileWritersWriteKnownSubjects(t *testing.T) {
	fw := newTestFileWriters(t)
	defer fw.Close()

	subjects := map[string]string{
		dsp.SUBJECTRequest:   "request payload",
		dsp.SUBJECTResponse:  "response payload",
		dsp.SUBJECTAttribute: "attribute payload",
		dsp.SUBJECTWinLoss:   "winloss payload",
	}
	for subject, payload := range subjects {
		if err := fw.WriteLog(subject, []byte(payload)); err != nil {
			t.Fatalf("WriteLog(%s): %v", subject, err)
		}
	}
	fw.Close()

	for subject, payload := range subjects {
		dir := subject
		if subject == dsp.SUBJECTWinLoss {
			dir = "winloss"
		}
		got, err := os.ReadFile(filepath.Join(fw.requestRoot(), dir, subject+".100"))
		if err != nil {
			t.Fatalf("read %s log: %v", subject, err)
		}
		if string(got) != payload+"\n" {
			t.Fatalf("%s log = %q, want %q", subject, string(got), payload+"\n")
		}
	}
}

func TestFileWritersIgnoredSubjectIsObservable(t *testing.T) {
	fw := newTestFileWriters(t)
	defer fw.Close()

	err := fw.WriteLog("unknown", []byte("payload"))
	if !errors.Is(err, ErrIgnoredSubject) {
		t.Fatalf("WriteLog unknown error = %v, want ErrIgnoredSubject", err)
	}
}

func TestFileWritersRotationReopensFiles(t *testing.T) {
	fw := newTestFileWriters(t)
	defer fw.Close()

	current := 100
	fw.current = func(int) int { return current }
	if err := fw.WriteLog(dsp.SUBJECTRequest, []byte("first")); err != nil {
		t.Fatal(err)
	}
	current = 101
	if err := fw.WriteLog(dsp.SUBJECTRequest, []byte("second")); err != nil {
		t.Fatal(err)
	}
	fw.Close()

	assertFile(t, filepath.Join(fw.request, "request.100"), "first\n")
	assertFile(t, filepath.Join(fw.request, "request.101"), "second\n")
}

func TestFileWritersUsePrivatePermissions(t *testing.T) {
	fw := newTestFileWriters(t)
	if err := fw.WriteLog(dsp.SUBJECTWinLoss, []byte("ledger input")); err != nil {
		t.Fatal(err)
	}
	fw.Close()

	for _, dir := range []string{fw.request, fw.response, fw.attribute, fw.winloss} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		assertPrivateMode(t, dir, info.Mode().Perm(), true)
	}
	info, err := os.Stat(filepath.Join(fw.winloss, "winloss.100"))
	if err != nil {
		t.Fatal(err)
	}
	assertPrivateMode(t, info.Name(), info.Mode().Perm(), false)
}

func TestFileWritersReturnDiskPathFailure(t *testing.T) {
	fw := newTestFileWriters(t)
	defer fw.Close()
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("block child creation"), 0600); err != nil {
		t.Fatal(err)
	}
	fw.request = filepath.Join(blocker, "request")
	if err := fw.WriteLog(dsp.SUBJECTRequest, []byte("must fail")); err == nil {
		t.Fatal("write to invalid disk path succeeded")
	}
}

func TestEnqueueLogMessageCopiesDataAndReportsFullQueue(t *testing.T) {
	msgs := make(chan logMessage, 1)
	errs := make(chan error, 1)
	data := []byte("first")
	enqueueLogMessage(msgs, errs, dsp.SUBJECTRequest, data)
	data[0] = 'X'

	msg := <-msgs
	if string(msg.data) != "first" {
		t.Fatalf("queued data = %q, want copied payload", string(msg.data))
	}

	msgs <- logMessage{subject: dsp.SUBJECTRequest, data: []byte("occupied")}
	enqueueLogMessage(msgs, errs, dsp.SUBJECTRequest, []byte("dropped"))
	select {
	case err := <-errs:
		if err == nil || !strings.Contains(err.Error(), "queue full") {
			t.Fatalf("queue full error = %v", err)
		}
	default:
		t.Fatal("expected queue full error")
	}
}

func TestRunNATSClientDrainsOnContextCancelAndFlushesQueuedLogs(t *testing.T) {
	root := t.TempDir()
	cfg := &dsp.Config{
		LogRequest:   filepath.Join(root, "request"),
		LogResponse:  filepath.Join(root, "response"),
		LogAttribute: filepath.Join(root, "attribute"),
		LogWinLoss:   filepath.Join(root, "winloss"),
	}
	nc := newFakeNATSLogConn()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- runNATSClient(ctx, cfg, nc, 10, log.New(io.Discard, "", 0))
	}()
	nc.waitSubscribed(t)
	nc.publish(dsp.SUBJECTWinLoss, []byte("queued winloss"))
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("runNATSClient did not exit after context cancellation")
	}
	if !nc.drained() {
		t.Fatal("NATS connection was not drained")
	}
	matches, err := filepath.Glob(filepath.Join(cfg.LogWinLoss, "winloss.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("winloss files = %v, want one file", matches)
	}
	assertFile(t, matches[0], "queued winloss\n")
}

func TestFileWritersPruneOnlyExpiredKnownLogFiles(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	dirs := map[string]string{
		dsp.SUBJECTRequest:   filepath.Join(root, "request"),
		dsp.SUBJECTResponse:  filepath.Join(root, "response"),
		dsp.SUBJECTAttribute: filepath.Join(root, "attribute"),
		dsp.SUBJECTWinLoss:   filepath.Join(root, "winloss"),
	}
	fw, err := newFileWritersWithRetention(dirs[dsp.SUBJECTRequest], dirs[dsp.SUBJECTResponse], dirs[dsp.SUBJECTAttribute], dirs[dsp.SUBJECTWinLoss], 10, 24*time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	fw.Close()
	old := now.Add(-25 * time.Hour)
	for subject, dir := range dirs {
		name := filepath.Join(dir, subject+".old")
		if err := os.WriteFile(name, []byte("expired"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(name, old, old); err != nil {
			t.Fatal(err)
		}
		fresh := filepath.Join(dir, subject+".fresh")
		if err := os.WriteFile(fresh, []byte("fresh"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	unrelated := filepath.Join(dirs[dsp.SUBJECTRequest], "operator-note.old")
	if err := os.WriteFile(unrelated, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(unrelated, old, old); err != nil {
		t.Fatal(err)
	}

	fw, err = newFileWritersWithRetention(dirs[dsp.SUBJECTRequest], dirs[dsp.SUBJECTResponse], dirs[dsp.SUBJECTAttribute], dirs[dsp.SUBJECTWinLoss], 10, 24*time.Hour, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()
	for subject, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, subject+".old")); !os.IsNotExist(err) {
			t.Fatalf("expired %s log was not removed: %v", subject, err)
		}
		if _, err := os.Stat(filepath.Join(dir, subject+".fresh")); err != nil {
			t.Fatalf("fresh %s log was removed: %v", subject, err)
		}
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func newTestFileWriters(t *testing.T) *FileWriters {
	t.Helper()
	root := t.TempDir()
	fw, err := NewFileWriters(
		filepath.Join(root, "request"),
		filepath.Join(root, "response"),
		filepath.Join(root, "attribute"),
		filepath.Join(root, "winloss"),
		10,
	)
	if err != nil {
		t.Fatal(err)
	}
	fw.existing = 100
	fw.current = func(int) int { return 100 }
	return fw
}

func (self *FileWriters) requestRoot() string {
	return filepath.Dir(self.request)
}

func assertFile(t *testing.T, name, want string) {
	t.Helper()
	got, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", name, string(got), want)
	}
}

func assertPrivateMode(t *testing.T, name string, mode os.FileMode, isDir bool) {
	t.Helper()
	if mode&0007 != 0 {
		t.Fatalf("%s mode %04o has world permissions", name, mode)
	}
	if mode&0022 != 0 {
		t.Fatalf("%s mode %04o has group/other write permissions", name, mode)
	}
	if isDir && mode&0700 != 0700 {
		t.Fatalf("%s mode %04o is not owner-accessible", name, mode)
	}
	if !isDir && mode&0600 != 0600 {
		t.Fatalf("%s mode %04o is not owner-readable/writable", name, mode)
	}
}

type fakeNATSLogConn struct {
	mu         sync.Mutex
	handler    nats.MsgHandler
	wasDrained bool
	subscribed chan struct{}
}

func newFakeNATSLogConn() *fakeNATSLogConn {
	return &fakeNATSLogConn{subscribed: make(chan struct{})}
}

func (f *fakeNATSLogConn) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handler = handler
	close(f.subscribed)
	return nil, nil
}

func (f *fakeNATSLogConn) Drain() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wasDrained = true
	return nil
}

func (f *fakeNATSLogConn) ConnectedUrl() string {
	return "nats://test"
}

func (f *fakeNATSLogConn) publish(subject string, data []byte) {
	f.mu.Lock()
	handler := f.handler
	f.mu.Unlock()
	if handler != nil {
		handler(&nats.Msg{Subject: subject, Data: data})
	}
}

func (f *fakeNATSLogConn) waitSubscribed(t *testing.T) {
	t.Helper()
	select {
	case <-f.subscribed:
	case <-time.After(time.Second):
		t.Fatal("fake NATS subscription was not registered")
	}
}

func (f *fakeNATSLogConn) drained() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.wasDrained
}

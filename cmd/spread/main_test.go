package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/guruperl/aofei/dsp"
)

func TestHandleSpreadMessageWritesSnapshot(t *testing.T) {
	top := t.TempDir()
	msg := &nats.Msg{Subject: "creative:7", Data: []byte("first")}
	handled, err := handleSpreadMessage(top, msg)
	if err != nil {
		t.Fatalf("handle first message: %v", err)
	}
	if !handled {
		t.Fatalf("expected creative subject to be handled")
	}

	msg.Data = []byte("second")
	handled, err = handleSpreadMessage(top, msg)
	if err != nil {
		t.Fatalf("handle second message: %v", err)
	}
	if !handled {
		t.Fatalf("expected creative subject to be handled")
	}

	data, err := os.ReadFile(filepath.Join(top, "creative", "7"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("expected snapshot write, got %q", data)
	}
}

func TestSpreadSubscriptionReceivesNestedSubjects(t *testing.T) {
	if spreadSubjectPattern != ">" {
		t.Fatalf("spread subject pattern = %q, want tail wildcard", spreadSubjectPattern)
	}
}

func TestHandleSpreadMessageWritesDottedPublisherSubject(t *testing.T) {
	top := t.TempDir()
	handled, err := handleSpreadMessage(top, &nats.Msg{
		Subject: "pubmap:example.com",
		Data:    []byte("publisher"),
	})
	if err != nil {
		t.Fatalf("handle dotted publisher: %v", err)
	}
	if !handled {
		t.Fatalf("expected dotted publisher subject to be handled")
	}
	assertSpreadFile(t, filepath.Join(top, "pubmap", "example.com"), "publisher")
}

func TestHandleSpreadMessageCleanupSubject(t *testing.T) {
	top := t.TempDir()
	path := filepath.Join(top, "slot", "4194368", "10")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	handled, err := handleSpreadMessage(top, &nats.Msg{
		Subject: "slot:4194368:10cleanup",
		Data:    []byte("new"),
	})
	if err != nil {
		t.Fatalf("handle cleanup: %v", err)
	}
	if !handled {
		t.Fatalf("expected slot cleanup subject to be handled")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read new file: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("expected new snapshot after cleanup, got %q", data)
	}
}

func TestHandleSpreadMessageSlotCleanupOnlySubject(t *testing.T) {
	top := t.TempDir()
	path := filepath.Join(top, "slot", "4194368", "10")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	handled, err := handleSpreadMessage(top, &nats.Msg{
		Subject: "slot:4194368:cleanup",
	})
	if err != nil {
		t.Fatalf("handle cleanup only: %v", err)
	}
	if !handled {
		t.Fatalf("expected slot cleanup subject to be handled")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected slot directory cleanup, stat err=%v", err)
	}
}

func TestHandleSpreadMessageCleanupSuffixOnlyAppliesToSlots(t *testing.T) {
	top := t.TempDir()
	handled, err := handleSpreadMessage(top, &nats.Msg{
		Subject: "pubmap:examplecleanup",
		Data:    []byte("publisher"),
	})
	if err != nil {
		t.Fatalf("handle cleanup-suffixed publisher: %v", err)
	}
	if !handled {
		t.Fatalf("expected cleanup-suffixed publisher subject to be handled")
	}
	assertSpreadFile(t, filepath.Join(top, "pubmap", "examplecleanup"), "publisher")
}

func TestHandleSpreadMessageResetSubject(t *testing.T) {
	top := t.TempDir()
	path := filepath.Join(top, "creative", "7")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	handled, err := handleSpreadMessage(top, &nats.Msg{
		Subject: "creative:__reset__",
	})
	if err != nil {
		t.Fatalf("handle reset: %v", err)
	}
	if !handled {
		t.Fatalf("expected reset subject to be handled")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected creative reset, stat err=%v", err)
	}
}

func TestHandleSpreadMessageMultipleSlotRefresh(t *testing.T) {
	top := t.TempDir()
	old := filepath.Join(top, "slot", "4194368", "10")
	if err := os.MkdirAll(filepath.Dir(old), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(old, []byte("old"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	for _, msg := range []*nats.Msg{
		{Subject: "slot:4194368:10cleanup", Data: []byte("slot-10")},
		{Subject: "slot:4194368:20", Data: []byte("slot-20")},
	} {
		handled, err := handleSpreadMessage(top, msg)
		if err != nil {
			t.Fatalf("handle %s: %v", msg.Subject, err)
		}
		if !handled {
			t.Fatalf("expected %s to be handled", msg.Subject)
		}
	}

	assertSpreadFile(t, filepath.Join(top, "slot", "4194368", "10"), "slot-10")
	assertSpreadFile(t, filepath.Join(top, "slot", "4194368", "20"), "slot-20")
}

func TestHandleSpreadMessageDelete(t *testing.T) {
	top := t.TempDir()
	path := filepath.Join(top, "audience", "5")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	handled, err := handleSpreadMessage(top, &nats.Msg{
		Subject: "audience:5",
		Data:    []byte("DELETE"),
	})
	if err != nil {
		t.Fatalf("handle delete: %v", err)
	}
	if !handled {
		t.Fatalf("expected audience subject to be handled")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file deletion, stat err=%v", err)
	}
}

func TestHandleSpreadMessageIgnoresSubjects(t *testing.T) {
	for _, subject := range []string{"request", "response", "attribute", "winloss", "unknown:1", "creative", "creative:..:escape"} {
		handled, err := handleSpreadMessage(t.TempDir(), &nats.Msg{
			Subject: subject,
			Data:    []byte("data"),
		})
		if err != nil {
			t.Fatalf("handle %s: %v", subject, err)
		}
		if handled {
			t.Fatalf("expected %s to be ignored", subject)
		}
	}
}

type fakeSpreadConn struct {
	handler nats.MsgHandler
	drained bool
}

func (f *fakeSpreadConn) ConnectedUrl() string { return "fake://spread" }

func (f *fakeSpreadConn) Subscribe(_ string, handler nats.MsgHandler) (*nats.Subscription, error) {
	f.handler = handler
	return nil, nil
}

func (f *fakeSpreadConn) Drain() error {
	f.drained = true
	return nil
}

func TestRunSpreadExitsOnContextCancelAndDrains(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	conn := &fakeSpreadConn{}
	var logs bytes.Buffer
	cfg := &dsp.Config{
		Spread:                      t.TempDir(),
		ConnectArray:                []string{"mysql", "missing"},
		Redis:                       &dsp.Red{Network: "tcp", Addr: "127.0.0.1:1"},
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		MiddlemanCallbackTTLSeconds: 86700,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
		MiddlemanCallbackBaseURL:    "http://localhost",
	}

	if err := runSpread(ctx, cfg, conn, log.New(&logs, "", 0)); err != nil {
		t.Fatal(err)
	}
	if conn.handler == nil {
		t.Fatal("expected spread subscription handler to be registered")
	}
	if !conn.drained {
		t.Fatal("expected NATS connection to be drained on shutdown")
	}
	if !strings.Contains(logs.String(), "spread bootstrap skipped") {
		t.Fatalf("logs = %q, want bootstrap skip note", logs.String())
	}
}

type failingDrainSpreadConn struct {
	fakeSpreadConn
}

func (f *failingDrainSpreadConn) Drain() error {
	return errors.New("drain failed")
}

func TestRunSpreadReturnsDrainError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := &dsp.Config{
		Spread:                      t.TempDir(),
		ConnectArray:                []string{"mysql", "missing"},
		Redis:                       &dsp.Red{Network: "tcp", Addr: "127.0.0.1:1"},
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		MiddlemanCallbackTTLSeconds: 86700,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
		MiddlemanCallbackBaseURL:    "http://localhost",
	}

	err := runSpread(ctx, cfg, &failingDrainSpreadConn{}, log.New(&bytes.Buffer{}, "", 0))
	if err == nil || !strings.Contains(err.Error(), "drain failed") {
		t.Fatalf("runSpread error = %v, want drain failed", err)
	}
}

func assertSpreadFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, string(data), want)
	}
}

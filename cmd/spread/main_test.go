package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nats-io/nats.go"
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

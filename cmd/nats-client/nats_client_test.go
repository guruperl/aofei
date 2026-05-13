package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guruperl/aofei/dsp"
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

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
	"time"

	"github.com/nats-io/nats.go"

	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/internal/spreadcache"
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

func TestSpreadGenerationPublishesOnlyCompleteOrderedSnapshot(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	messages := []spreadcache.Message{
		{Subject: "pubmap:example.com", Data: []byte("publisher")},
		{Subject: "creative:7", Data: []byte("creative")},
		{Subject: "slot:4194368:10", Data: []byte("slot")},
	}
	sendSpreadGeneration(t, receiver, 1, messages, nil)
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "pubmap", "example.com"), "publisher")
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "creative")
	assertSpreadFile(t, filepath.Join(root, "slot", "4194368", "10"), "slot")
}

func TestSpreadGenerationReconnectGapPreservesCommittedSnapshot(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	sendSpreadGeneration(t, receiver, 1, []spreadcache.Message{
		{Subject: "creative:7", Data: []byte("old")},
	}, nil)

	next := []spreadcache.Message{
		{Subject: "creative:7", Data: []byte("new")},
		{Subject: "audience:8", Data: []byte("audience")},
	}
	manifest, err := spreadcache.NewManifest(2, next)
	if err != nil {
		t.Fatal(err)
	}
	begin, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(&nats.Msg{Subject: spreadcache.BeginSubject, Data: begin}); err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(spreadDataMessage(2, next[0])); err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(&nats.Msg{Subject: spreadcache.CommitSubject, Data: []byte("2")}); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("incomplete commit error = %v", err)
	}
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "old")

	if _, err := receiver.handle(spreadDataMessage(2, next[1])); err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(&nats.Msg{Subject: spreadcache.CommitSubject, Data: []byte("2")}); err != nil {
		t.Fatal(err)
	}
	root, err = spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "new")
	assertSpreadFile(t, filepath.Join(root, "audience", "8"), "audience")
}

func TestSpreadGenerationDuplicateMessagesAreIdempotent(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	messages := []spreadcache.Message{{Subject: "creative:7", Data: []byte("stable")}}
	sendSpreadGeneration(t, receiver, 1, messages, func(begin, data, commit *nats.Msg) {
		for _, message := range []*nats.Msg{begin, begin, data, data, commit, commit} {
			if _, err := receiver.handle(message); err != nil {
				t.Fatal(err)
			}
		}
	})
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "stable")
}

func TestSpreadGenerationDigestMismatchPreservesCommittedSnapshot(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	sendSpreadGeneration(t, receiver, 1, []spreadcache.Message{{Subject: "creative:7", Data: []byte("old")}}, nil)
	messages := []spreadcache.Message{{Subject: "creative:7", Data: []byte("new")}}
	manifest, err := spreadcache.NewManifest(2, messages)
	if err != nil {
		t.Fatal(err)
	}
	manifestData, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(&nats.Msg{Subject: spreadcache.BeginSubject, Data: manifestData}); err != nil {
		t.Fatal(err)
	}
	corrupt := spreadcache.Message{Subject: messages[0].Subject, Data: []byte("corrupt")}
	if _, err := receiver.handle(spreadDataMessage(2, corrupt)); err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(&nats.Msg{Subject: spreadcache.CommitSubject, Data: []byte("2")}); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("corrupt commit error = %v", err)
	}
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "old")
}

func TestSpreadGenerationOverlappingProducerCannotDeleteNewerSnapshot(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	oldMessages := []spreadcache.Message{{Subject: "creative:7", Data: []byte("old-producer")}}
	oldManifest, err := spreadcache.NewManifest(1, oldMessages)
	if err != nil {
		t.Fatal(err)
	}
	oldBegin, err := oldManifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := receiver.handle(&nats.Msg{Subject: spreadcache.BeginSubject, Data: oldBegin}); err != nil {
		t.Fatal(err)
	}

	newMessages := []spreadcache.Message{{Subject: "creative:7", Data: []byte("new-producer")}}
	sendSpreadGeneration(t, receiver, 2, newMessages, nil)
	for _, delayed := range []*nats.Msg{
		spreadDataMessage(1, oldMessages[0]),
		{Subject: spreadcache.CommitSubject, Data: []byte("1")},
		{Subject: spreadcache.BeginSubject, Data: oldBegin},
	} {
		if _, err := receiver.handle(delayed); err != nil {
			t.Fatal(err)
		}
	}
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "new-producer")
}

func TestSpreadGenerationIgnoresLegacyMutationAfterActivation(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	sendSpreadGeneration(t, receiver, 1, []spreadcache.Message{{Subject: "creative:7", Data: []byte("current")}}, nil)
	handled, err := receiver.handle(&nats.Msg{Subject: "creative:7", Data: []byte("legacy")})
	if err != nil {
		t.Fatal(err)
	}
	if handled {
		t.Fatal("legacy mutation reported handled after generation activation")
	}
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "current")
}

func TestSpreadGenerationRetainsCurrentAndPreviousSnapshots(t *testing.T) {
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	for sequence := uint64(1); sequence <= 3; sequence++ {
		sendSpreadGeneration(t, receiver, sequence, []spreadcache.Message{{
			Subject: "creative:7",
			Data:    []byte(spreadcache.SequenceString(sequence)),
		}}, nil)
	}
	if _, err := os.Stat(spreadcache.GenerationRoot(top, 1)); !os.IsNotExist(err) {
		t.Fatalf("old generation still exists: %v", err)
	}
	for _, sequence := range []uint64{2, 3} {
		if _, err := os.Stat(spreadcache.GenerationRoot(top, sequence)); err != nil {
			t.Fatalf("retained generation %d: %v", sequence, err)
		}
	}
}

func TestSpreadGenerationNATSReconnectIntegration(t *testing.T) {
	url := os.Getenv("AOFEI_TEST_NATS_URL")
	if url == "" {
		t.Skip("set AOFEI_TEST_NATS_URL to run NATS reconnect integration")
	}
	top := t.TempDir()
	receiver, err := newSpreadGenerationReceiver(top)
	if err != nil {
		t.Fatal(err)
	}
	callbackErr := make(chan error, 1)
	subscriberReconnected := make(chan struct{}, 1)
	subscriber, err := nats.Connect(url, nats.ReconnectWait(10*time.Millisecond), nats.MaxReconnects(100), nats.ReconnectHandler(func(*nats.Conn) {
		subscriberReconnected <- struct{}{}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer subscriber.Close()
	if _, err := subscriber.Subscribe(">", func(message *nats.Msg) {
		if _, err := receiver.handle(message); err != nil {
			select {
			case callbackErr <- err:
			default:
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := subscriber.FlushTimeout(2 * time.Second); err != nil {
		t.Fatal(err)
	}

	publisherReconnected := make(chan struct{}, 1)
	publisher, err := nats.Connect(url, nats.ReconnectWait(10*time.Millisecond), nats.MaxReconnects(100), nats.ReconnectHandler(func(*nats.Conn) {
		publisherReconnected <- struct{}{}
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	if err := publisher.ForceReconnect(); err != nil {
		t.Fatal(err)
	}
	if err := subscriber.ForceReconnect(); err != nil {
		t.Fatal(err)
	}
	for name, reconnected := range map[string]<-chan struct{}{
		"publisher":  publisherReconnected,
		"subscriber": subscriberReconnected,
	} {
		select {
		case <-reconnected:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s did not reconnect", name)
		}
	}
	messages := []spreadcache.Message{
		{Subject: "creative:7", Data: []byte("after-reconnect")},
		{Subject: "slot:1:2", Data: []byte("ordered")},
	}
	publishNATSGeneration(t, publisher, 1001, messages)

	deadline := time.Now().Add(5 * time.Second)
	for {
		sequence, ok, err := spreadcache.CurrentSequence(top)
		if err != nil {
			t.Fatal(err)
		}
		if ok && sequence == 1001 {
			break
		}
		select {
		case err := <-callbackErr:
			t.Fatal(err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("reconnected NATS generation was not committed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	root, err := spreadcache.Resolve(top)
	if err != nil {
		t.Fatal(err)
	}
	assertSpreadFile(t, filepath.Join(root, "creative", "7"), "after-reconnect")
	assertSpreadFile(t, filepath.Join(root, "slot", "1", "2"), "ordered")
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
	flushed bool
}

func (f *fakeSpreadConn) ConnectedUrl() string { return "fake://spread" }

func (f *fakeSpreadConn) Subscribe(_ string, handler nats.MsgHandler) (*nats.Subscription, error) {
	f.handler = handler
	return nil, nil
}

func (f *fakeSpreadConn) FlushTimeout(time.Duration) error {
	f.flushed = true
	return nil
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
	if !conn.flushed {
		t.Fatal("expected spread subscription to be flushed before readiness")
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

func sendSpreadGeneration(t *testing.T, receiver *spreadGenerationReceiver, sequence uint64, messages []spreadcache.Message, custom func(begin, data, commit *nats.Msg)) {
	t.Helper()
	manifest, err := spreadcache.NewManifest(sequence, messages)
	if err != nil {
		t.Fatal(err)
	}
	manifestData, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	begin := &nats.Msg{Subject: spreadcache.BeginSubject, Data: manifestData}
	commit := &nats.Msg{Subject: spreadcache.CommitSubject, Data: []byte(spreadcache.SequenceString(sequence))}
	if custom != nil {
		var data *nats.Msg
		if len(messages) > 0 {
			data = spreadDataMessage(sequence, messages[0])
		}
		custom(begin, data, commit)
		return
	}
	if _, err := receiver.handle(begin); err != nil {
		t.Fatal(err)
	}
	for _, message := range messages {
		if _, err := receiver.handle(spreadDataMessage(sequence, message)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := receiver.handle(commit); err != nil {
		t.Fatal(err)
	}
}

func spreadDataMessage(sequence uint64, message spreadcache.Message) *nats.Msg {
	out := nats.NewMsg(spreadcache.DataSubject)
	out.Header.Set(spreadcache.GenerationHeader, spreadcache.SequenceString(sequence))
	out.Header.Set(spreadcache.OriginalSubjectHeader, message.Subject)
	out.Data = message.Data
	return out
}

func publishNATSGeneration(t *testing.T, publisher *nats.Conn, sequence uint64, messages []spreadcache.Message) {
	t.Helper()
	manifest, err := spreadcache.NewManifest(sequence, messages)
	if err != nil {
		t.Fatal(err)
	}
	data, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.PublishMsg(&nats.Msg{Subject: spreadcache.BeginSubject, Data: data}); err != nil {
		t.Fatal(err)
	}
	for _, message := range messages {
		if err := publisher.PublishMsg(spreadDataMessage(sequence, message)); err != nil {
			t.Fatal(err)
		}
	}
	if err := publisher.PublishMsg(&nats.Msg{Subject: spreadcache.CommitSubject, Data: []byte(spreadcache.SequenceString(sequence))}); err != nil {
		t.Fatal(err)
	}
	if err := publisher.FlushTimeout(5 * time.Second); err != nil {
		t.Fatal(err)
	}
}

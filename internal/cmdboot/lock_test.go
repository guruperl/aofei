package cmdboot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func lockTestClient(t *testing.T, address string) radix.Client {
	t.Helper()
	client, err := (radix.PoolConfig{Size: 1}).New(context.Background(), "tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestAcquireLockRenewsLeaseAndExcludesSecondOwner(t *testing.T) {
	server := miniredis.RunT(t)
	client := lockTestClient(t, server.Addr())
	lock, err := AcquireLock(context.Background(), client, "job", 90*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release(context.Background())
	time.Sleep(160 * time.Millisecond)
	second, err := AcquireLock(context.Background(), client, "job", time.Second)
	if second != nil || !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second lock=%v err=%v, want held", second, err)
	}
	if err := lock.Err(); err != nil {
		t.Fatalf("lease renewal: %v", err)
	}
}

func TestLockReportsLeaseLossAndCannotDeleteSuccessor(t *testing.T) {
	server := miniredis.RunT(t)
	client := lockTestClient(t, server.Addr())
	lock, err := AcquireLock(context.Background(), client, "job", 60*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	lock.cancel()
	<-lock.done
	if err := client.Do(context.Background(), radix.Cmd(nil, "SET", "job", "successor", "PX", "1000")); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	var value string
	if err := client.Do(context.Background(), radix.Cmd(&value, "GET", "job")); err != nil {
		t.Fatal(err)
	}
	if value != "successor" {
		t.Fatalf("release removed successor value %q", value)
	}

	lost, err := AcquireLock(context.Background(), client, "lost", 60*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	deadline := time.Now().Add(time.Second)
	for lost.Err() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if lost.Err() == nil {
		t.Fatal("lease loss was not reported")
	}
}

package cmdboot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

type transientLockClient struct {
	radix.Client
	mu     sync.Mutex
	errors []error
	calls  int
}

type delayedLockClient struct {
	radix.Client
	delay time.Duration
	once  sync.Once
}

func (c *delayedLockClient) Addr() net.Addr { return c.Client.Addr() }

func (c *delayedLockClient) Do(ctx context.Context, action radix.Action) error {
	err := c.Client.Do(ctx, action)
	c.once.Do(func() { time.Sleep(c.delay) })
	return err
}

func (c *transientLockClient) Addr() net.Addr { return c.Client.Addr() }

func (c *transientLockClient) Do(ctx context.Context, action radix.Action) error {
	c.mu.Lock()
	c.calls++
	var err error
	if len(c.errors) != 0 {
		err = c.errors[0]
		c.errors = c.errors[1:]
	}
	c.mu.Unlock()
	if err != nil {
		return err
	}
	return c.Client.Do(ctx, action)
}

func scriptedLockTime(start time.Time, waits int) (func() time.Time, func(context.Context, time.Duration) bool) {
	current := start
	return func() time.Time { return current }, func(ctx context.Context, delay time.Duration) bool {
		if waits == 0 || ctx.Err() != nil {
			return false
		}
		waits--
		current = current.Add(delay)
		return true
	}
}

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

func TestAcquireLockRejectsAcknowledgementAfterConfirmedWindow(t *testing.T) {
	server := miniredis.RunT(t)
	base := lockTestClient(t, server.Addr())
	client := &delayedLockClient{Client: base, delay: 50 * time.Millisecond}

	lock, err := AcquireLock(context.Background(), client, "delayed", 20*time.Millisecond)
	if lock != nil || !errors.Is(err, ErrLeaseUncertain) {
		t.Fatalf("lock=%v err=%v, want uncertain acquisition rejection", lock, err)
	}
	if server.Exists("delayed") {
		t.Fatal("stale acquisition acknowledgement left an owned lease behind")
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

func TestLockRetriesTransientRenewalInsideConfirmedWindow(t *testing.T) {
	server := miniredis.RunT(t)
	base := lockTestClient(t, server.Addr())
	if err := base.Do(context.Background(), radix.Cmd(nil, "SET", "job", "owner", "PX", "60000")); err != nil {
		t.Fatal(err)
	}
	client := &transientLockClient{Client: base, errors: []error{errors.New("temporary redis failure")}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now, wait := scriptedLockTime(time.Now(), 2)
	lock := &Lock{
		client: client, key: "job", token: "owner", ttl: 90 * time.Millisecond,
		ctx: ctx, cancel: cancel, done: make(chan struct{}), now: now, wait: wait,
		confirmedAt: now(),
	}

	lock.maintain(ctx)

	if err := lock.Err(); err != nil {
		t.Fatalf("transient renewal became lease loss: %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("renewal calls = %d, want transient failure plus successful retry", client.calls)
	}
	if ctx.Err() != nil {
		t.Fatal("successful retry canceled lease-owned work")
	}
}

func TestLockInitialRenewalDelayStaysInsideShortLease(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var delay time.Duration
	lock := &Lock{
		ttl: 3 * time.Millisecond, ctx: ctx, cancel: cancel, done: make(chan struct{}),
		now: time.Now,
	}
	lock.wait = func(_ context.Context, got time.Duration) bool {
		delay = got
		return false
	}

	lock.maintain(ctx)

	if delay <= 0 || delay >= lock.ttl {
		t.Fatalf("initial renewal delay = %s, want inside %s lease", delay, lock.ttl)
	}
}

func TestLockCancelsImmediatelyOnConfirmedTokenMismatch(t *testing.T) {
	server := miniredis.RunT(t)
	client := lockTestClient(t, server.Addr())
	if err := client.Do(context.Background(), radix.Cmd(nil, "SET", "job", "successor", "PX", "60000")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	now, wait := scriptedLockTime(time.Now(), 1)
	lock := &Lock{
		client: client, key: "job", token: "owner", ttl: 90 * time.Millisecond,
		ctx: ctx, cancel: cancel, done: make(chan struct{}), now: now, wait: wait,
		confirmedAt: now(),
	}

	lock.maintain(ctx)

	if !errors.Is(lock.Err(), ErrLockHeld) {
		t.Fatalf("lease error = %v, want confirmed ownership loss", lock.Err())
	}
	if ctx.Err() == nil {
		t.Fatal("confirmed ownership loss did not cancel work")
	}
}

func TestLockStopsAtLastConfirmedDeadlineAfterUncertainty(t *testing.T) {
	server := miniredis.RunT(t)
	base := lockTestClient(t, server.Addr())
	client := &transientLockClient{Client: base, errors: make([]error, 20)}
	for i := range client.errors {
		client.errors[i] = errors.New("redis unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	now, wait := scriptedLockTime(time.Now(), 20)
	lock := &Lock{
		client: client, key: "job", token: "owner", ttl: 90 * time.Millisecond,
		ctx: ctx, cancel: cancel, done: make(chan struct{}), now: now, wait: wait,
		confirmedAt: now(),
	}

	lock.maintain(ctx)

	if !errors.Is(lock.Err(), ErrLeaseUncertain) {
		t.Fatalf("lease error = %v, want uncertain ownership deadline", lock.Err())
	}
	if ctx.Err() == nil {
		t.Fatal("expired confirmed lease window did not cancel work")
	}
	if now().Before(lock.confirmedAt.Add(lock.ttl)) {
		t.Fatalf("work stopped at %s before confirmed deadline %s", now(), lock.confirmedAt.Add(lock.ttl))
	}
}

func TestWithLockReleasesBeforeReturningWorkFailure(t *testing.T) {
	server := miniredis.RunT(t)
	client := lockTestClient(t, server.Addr())
	want := errors.New("work failed")
	err := WithLock(context.Background(), client, "job", time.Second, func(ctx context.Context) error {
		if ctx.Err() != nil {
			t.Fatal("lease work started with canceled context")
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("WithLock error = %v, want work failure", err)
	}
	if server.Exists("job") {
		t.Fatal("WithLock returned before releasing owned lease")
	}
}

func TestRedisLockOwnershipIntegration(t *testing.T) {
	address := os.Getenv("AOFEI_LOCK_REDIS_ADDR")
	if address == "" {
		t.Skip("AOFEI_LOCK_REDIS_ADDR is not set")
	}
	client := lockTestClient(t, address)
	key := fmt.Sprintf("aofei:test:lock:%d", time.Now().UnixNano())
	lock, err := AcquireLock(context.Background(), client, key, 150*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lock.Release(context.Background()) })
	time.Sleep(220 * time.Millisecond)
	if err := lock.Err(); err != nil {
		t.Fatalf("real Redis renewal failed: %v", err)
	}
	second, err := AcquireLock(context.Background(), client, key, time.Second)
	if second != nil || !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second lock=%v err=%v, want held", second, err)
	}
	if err := lock.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	replacement, err := AcquireLock(context.Background(), client, key, time.Second)
	if err != nil {
		t.Fatalf("replacement acquisition: %v", err)
	}
	if err := replacement.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
}

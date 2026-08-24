package cmdboot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mediocregopher/radix/v4"
)

var ErrLockHeld = errors.New("singleton lock is already held")
var ErrLeaseUncertain = errors.New("singleton lease ownership is uncertain")

const lockReleaseTimeout = 5 * time.Second

type Lock struct {
	client      radix.Client
	key         string
	token       string
	ttl         time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	now         func() time.Time
	wait        func(context.Context, time.Duration) bool
	confirmedAt time.Time
	errMu       sync.Mutex
	err         error
}

func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}

func AcquireLock(ctx context.Context, client radix.Client, key string, ttl time.Duration) (*Lock, error) {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	confirmedAt := time.Now()
	var reply string
	if err := client.Do(ctx, radix.Cmd(&reply, "SET", key, token, "NX", "PX", fmt.Sprintf("%d", lockTTLMillis(ttl)))); err != nil {
		return nil, err
	}
	if reply != "OK" {
		return nil, ErrLockHeld
	}
	leaseCtx, cancel := context.WithCancel(ctx)
	lock := &Lock{
		client: client, key: key, token: token, ttl: ttl,
		ctx: leaseCtx, cancel: cancel, done: make(chan struct{}),
		now: time.Now, wait: waitForLockDelay, confirmedAt: confirmedAt,
	}
	go lock.maintain(leaseCtx)
	return lock, nil
}

// WithLock acquires a renewable lease, runs work with the lease-owned context,
// and explicitly releases before returning any work or lease failure. Work
// must honor the supplied context so it stops no later than the last confirmed
// lease deadline.
func WithLock(ctx context.Context, client radix.Client, key string, ttl time.Duration, work func(context.Context) error) error {
	if work == nil {
		return fmt.Errorf("singleton work is required")
	}
	lock, err := AcquireLock(ctx, client, key, ttl)
	if err != nil {
		return err
	}
	workErr := work(lock.Context())
	releaseCtx, cancel := context.WithTimeout(context.Background(), lockReleaseTimeout)
	releaseErr := lock.Release(releaseCtx)
	cancel()
	return errors.Join(workErr, lock.Err(), releaseErr)
}

// Context is canceled on confirmed ownership loss, when renewal uncertainty
// reaches the last confirmed expiry, or when the acquisition parent ends.
func (l *Lock) Context() context.Context {
	if l == nil || l.ctx == nil {
		return context.Background()
	}
	return l.ctx
}

func (l *Lock) Release(ctx context.Context) error {
	if l == nil || l.client == nil {
		return nil
	}
	if l.cancel != nil {
		l.cancel()
	}
	if l.done != nil {
		<-l.done
	}
	script := radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)
	var deleted int
	return l.client.Do(ctx, script.Cmd(&deleted, []string{l.key}, l.token))
}

// Err reports a lease-renewal failure. Mutating commands check it before
// reporting success; their durable writes must still be idempotent because a
// dependency partition can always make lease ownership uncertain.
func (l *Lock) Err() error {
	if l == nil {
		return nil
	}
	l.errMu.Lock()
	defer l.errMu.Unlock()
	return l.err
}

func (l *Lock) maintain(ctx context.Context) {
	defer close(l.done)
	interval := l.ttl / 3
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	now := l.now
	if now == nil {
		now = time.Now
	}
	wait := l.wait
	if wait == nil {
		wait = waitForLockDelay
	}
	confirmedAt := l.confirmedAt
	if confirmedAt.IsZero() {
		confirmedAt = now()
	}
	delay := interval
	transientFailures := 0
	script := radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0`)
	for {
		if !wait(ctx, delay) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		deadline := confirmedAt.Add(l.ttl)
		started := now()
		if !started.Before(deadline) {
			l.recordLeaseFailure(fmt.Errorf("%w after renewal deadline", ErrLeaseUncertain))
			return
		}
		attemptCtx, cancel := context.WithDeadline(ctx, deadline)
		var renewed int
		err := l.client.Do(attemptCtx, script.Cmd(&renewed, []string{l.key}, l.token, fmt.Sprintf("%d", lockTTLMillis(l.ttl))))
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err == nil && renewed == 1 {
			confirmedAt = started
			transientFailures = 0
			delay = interval
			continue
		}
		if err == nil {
			l.recordLeaseFailure(ErrLockHeld)
			return
		}
		transientFailures++
		remaining := deadline.Sub(now())
		if remaining <= 0 {
			l.recordLeaseFailure(fmt.Errorf("%w after renewal failure: %v", ErrLeaseUncertain, err))
			return
		}
		delay = lockRenewalRetryDelay(l.ttl, remaining, transientFailures)
	}
}

func (l *Lock) recordLeaseFailure(cause error) {
	l.errMu.Lock()
	if l.err == nil {
		l.err = fmt.Errorf("singleton lease %s lost: %w", l.key, cause)
	}
	l.errMu.Unlock()
	if l.cancel != nil {
		l.cancel()
	}
}

func waitForLockDelay(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func lockRenewalRetryDelay(ttl, remaining time.Duration, failures int) time.Duration {
	delay := ttl / 30
	if delay < 10*time.Millisecond {
		delay = 10 * time.Millisecond
	}
	if delay > time.Second {
		delay = time.Second
	}
	for i := 1; i < failures && delay < ttl/6; i++ {
		delay *= 2
	}
	if maximum := ttl / 6; maximum >= 10*time.Millisecond && delay > maximum {
		delay = maximum
	}
	if delay > remaining {
		return remaining
	}
	return delay
}

func lockTTLMillis(ttl time.Duration) int64 {
	millis := ttl.Milliseconds()
	if millis < 1 {
		return 1
	}
	return millis
}

func randomToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

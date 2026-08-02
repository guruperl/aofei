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

type Lock struct {
	client radix.Client
	key    string
	token  string
	ttl    time.Duration
	cancel context.CancelFunc
	done   chan struct{}
	errMu  sync.Mutex
	err    error
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
	var reply string
	if err := client.Do(ctx, radix.Cmd(&reply, "SET", key, token, "NX", "PX", fmt.Sprintf("%d", lockTTLMillis(ttl)))); err != nil {
		return nil, err
	}
	if reply != "OK" {
		return nil, ErrLockHeld
	}
	leaseCtx, cancel := context.WithCancel(ctx)
	lock := &Lock{client: client, key: key, token: token, ttl: ttl, cancel: cancel, done: make(chan struct{})}
	go lock.maintain(leaseCtx)
	return lock, nil
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
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	script := radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0`)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var renewed int
			err := l.client.Do(ctx, script.Cmd(&renewed, []string{l.key}, l.token, fmt.Sprintf("%d", lockTTLMillis(l.ttl))))
			if err == nil && renewed == 1 {
				continue
			}
			if err == nil {
				err = ErrLockHeld
			}
			l.errMu.Lock()
			l.err = fmt.Errorf("singleton lease %s lost: %w", l.key, err)
			l.errMu.Unlock()
			return
		}
	}
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

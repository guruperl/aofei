package cmdboot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/mediocregopher/radix/v4"
)

var ErrLockHeld = errors.New("singleton lock is already held")

type Lock struct {
	client radix.Client
	key    string
	token  string
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
	if err := client.Do(ctx, radix.Cmd(&reply, "SET", key, token, "NX", "EX", fmt.Sprintf("%.0f", ttl.Seconds()))); err != nil {
		return nil, err
	}
	if reply != "OK" {
		return nil, ErrLockHeld
	}
	return &Lock{client: client, key: key, token: token}, nil
}

func (l *Lock) Release(ctx context.Context) error {
	if l == nil || l.client == nil {
		return nil
	}
	script := radix.NewEvalScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0`)
	var deleted int
	return l.client.Do(ctx, script.Cmd(&deleted, []string{l.key}, l.token))
}

func randomToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

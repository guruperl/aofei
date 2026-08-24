package match

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type CacheSink interface {
	ResetRAdvs(ctx context.Context, sizeID uint32) error
	PutRAdvs(ctx context.Context, sizeID, slotID uint32, data []byte, cleanup bool) error
	CleanupRAdvs(ctx context.Context, sizeID uint32) error
	PutAudience(ctx context.Context, itemID uint32, data []byte) error
	PutCreative(ctx context.Context, creativeID uint32, data []byte) error
}

// ErrUnsafeRedisLiveReset reports an attempted destructive rebuild of a live
// Redis family outside the atomic generation publisher.
var ErrUnsafeRedisLiveReset = errors.New("refusing non-atomic reset of live redis cache family")

// RedisCacheSink writes cache entries to Redis. Internal item-level live sinks
// deliberately refuse family resets: deleting a live hash before repopulating
// it would expose an empty or partial generation. Reusable family writers must
// use NewRedisCacheGenerationSink while constructing a complete namespaced
// generation, then publish that generation atomically.
type RedisCacheSink struct {
	client    radix.Client
	keySuffix string
}

// newRedisCacheSink returns a sink for compatible item-level live writes. It
// cannot reset a live family. Full-family callers must use the exported
// generation constructor instead.
func newRedisCacheSink(client radix.Client) RedisCacheSink {
	return RedisCacheSink{client: client}
}

// NewRedisCacheGenerationSink returns a sink whose writes are isolated under
// keySuffix until the caller atomically publishes the complete generation.
func NewRedisCacheGenerationSink(client radix.Client, keySuffix string) (RedisCacheSink, error) {
	if client == nil {
		return RedisCacheSink{}, errors.New("redis cache generation client is nil")
	}
	if keySuffix == "" {
		return RedisCacheSink{}, errors.New("redis cache generation suffix is empty")
	}
	return RedisCacheSink{client: client, keySuffix: keySuffix}, nil
}

func (s RedisCacheSink) key(name string) string {
	return name + s.keySuffix
}

func (s RedisCacheSink) ResetRAdvs(ctx context.Context, sizeID uint32) error {
	if s.keySuffix == "" {
		return ErrUnsafeRedisLiveReset
	}
	return s.client.Do(ctx, radix.Cmd(nil, "DEL", s.key(HashNameRAdvs(sizeID))))
}

func (s RedisCacheSink) PutRAdvs(ctx context.Context, sizeID, slotID uint32, data []byte, _ bool) error {
	return s.client.Do(ctx, radix.Cmd(nil, "HSET", s.key(HashNameRAdvs(sizeID)), strconv.FormatUint(uint64(slotID), 10), string(data)))
}

func (s RedisCacheSink) CleanupRAdvs(context.Context, uint32) error {
	return nil
}

func (s RedisCacheSink) PutAudience(ctx context.Context, itemID uint32, data []byte) error {
	return s.client.Do(ctx, radix.Cmd(nil, "HSET", s.key(HashNameAudience), strconv.FormatUint(uint64(itemID), 10), string(data)))
}

func (s RedisCacheSink) PutCreative(ctx context.Context, creativeID uint32, data []byte) error {
	return s.client.Do(ctx, radix.Cmd(nil, "HSET", s.key(HashNameCreative), strconv.FormatUint(uint64(creativeID), 10), string(data)))
}

type SpreadCacheSink struct {
	Conn *nats.Conn
}

func (s SpreadCacheSink) ResetRAdvs(context.Context, uint32) error {
	return nil
}

func (s SpreadCacheSink) PutRAdvs(_ context.Context, sizeID, slotID uint32, data []byte, cleanup bool) error {
	subject := fmt.Sprintf("%s:%d", HashNameRAdvs(sizeID), slotID)
	if cleanup {
		subject += "cleanup"
	}
	return s.Conn.Publish(subject, data)
}

func (s SpreadCacheSink) CleanupRAdvs(_ context.Context, sizeID uint32) error {
	return publishRAdvsSpreadCleanup(s.Conn, sizeID)
}

func (s SpreadCacheSink) PutAudience(_ context.Context, itemID uint32, data []byte) error {
	return s.Conn.Publish(fmt.Sprintf("%s:%d", HashNameAudience, itemID), data)
}

func (s SpreadCacheSink) PutCreative(_ context.Context, creativeID uint32, data []byte) error {
	return s.Conn.Publish(fmt.Sprintf("%s:%d", HashNameCreative, creativeID), data)
}

func CacheSinkFor(conn any) (CacheSink, error) {
	switch t := conn.(type) {
	case CacheSink:
		return t, nil
	case radix.Client:
		return newRedisCacheSink(t), nil
	case *nats.Conn:
		return SpreadCacheSink{Conn: t}, nil
	default:
		return nil, fmt.Errorf("unsupported cache sink type: %T", conn)
	}
}

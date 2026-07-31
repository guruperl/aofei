package match

import (
	"context"
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

type RedisCacheSink struct {
	Client    radix.Client
	KeySuffix string
}

func (s RedisCacheSink) key(name string) string {
	return name + s.KeySuffix
}

func (s RedisCacheSink) ResetRAdvs(ctx context.Context, sizeID uint32) error {
	return s.Client.Do(ctx, radix.Cmd(nil, "DEL", s.key(HashNameRAdvs(sizeID))))
}

func (s RedisCacheSink) PutRAdvs(ctx context.Context, sizeID, slotID uint32, data []byte, _ bool) error {
	return s.Client.Do(ctx, radix.Cmd(nil, "HSET", s.key(HashNameRAdvs(sizeID)), strconv.FormatUint(uint64(slotID), 10), string(data)))
}

func (s RedisCacheSink) CleanupRAdvs(context.Context, uint32) error {
	return nil
}

func (s RedisCacheSink) PutAudience(ctx context.Context, itemID uint32, data []byte) error {
	return s.Client.Do(ctx, radix.Cmd(nil, "HSET", s.key(HashNameAudience), strconv.FormatUint(uint64(itemID), 10), string(data)))
}

func (s RedisCacheSink) PutCreative(ctx context.Context, creativeID uint32, data []byte) error {
	return s.Client.Do(ctx, radix.Cmd(nil, "HSET", s.key(HashNameCreative), strconv.FormatUint(uint64(creativeID), 10), string(data)))
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
		return RedisCacheSink{Client: t}, nil
	case *nats.Conn:
		return SpreadCacheSink{Conn: t}, nil
	default:
		return nil, fmt.Errorf("unsupported cache sink type: %T", conn)
	}
}

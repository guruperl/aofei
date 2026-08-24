package match

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/internal/cachegeneration"
	"github.com/mediocregopher/radix/v4"
)

type cancelAfterResetSink struct {
	cancel context.CancelFunc
	puts   int
}

func (s *cancelAfterResetSink) ResetRAdvs(context.Context, uint32) error {
	s.cancel()
	return nil
}
func (s *cancelAfterResetSink) PutRAdvs(context.Context, uint32, uint32, []byte, bool) error {
	s.puts++
	return nil
}
func (*cancelAfterResetSink) CleanupRAdvs(context.Context, uint32) error { return nil }
func (*cancelAfterResetSink) PutAudience(context.Context, uint32, []byte) error {
	return nil
}
func (*cancelAfterResetSink) PutCreative(context.Context, uint32, []byte) error {
	return nil
}

func TestRAdvGenerationPackingStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sink := &cancelAfterResetSink{cancel: cancel}
	err := radvHashToCacheSinkBySizeID(ctx, sink, map[uint32]RAdvs{
		7: {{Demand: Demand{CreativeID: 11}}},
	}, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("radvHashToCacheSinkBySizeID error = %v, want context canceled", err)
	}
	if sink.puts != 0 {
		t.Fatalf("RAdv writes after cancellation = %d, want 0", sink.puts)
	}
}

func TestRedisCacheSinkRejectsLiveFamilyReset(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	key := HashNameRAdvs(1)
	if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "7", "old")); err != nil {
		t.Fatal(err)
	}
	sink, err := CacheSinkFor(client)
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.ResetRAdvs(ctx, 1); !errors.Is(err, ErrUnsafeRedisLiveReset) {
		t.Fatalf("ResetRAdvs error = %v, want %v", err, ErrUnsafeRedisLiveReset)
	}
	var got string
	if err := client.Do(ctx, radix.Cmd(&got, "HGET", key, "7")); err != nil {
		t.Fatal(err)
	}
	if got != "old" {
		t.Fatalf("live cache value = %q, want old", got)
	}
}

func TestRedisCacheGenerationSinkIsolatesFamilyWrites(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	const suffix = ":generation-test"
	key := HashNameRAdvs(1)
	if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "7", "old")); err != nil {
		t.Fatal(err)
	}
	sink, err := NewRedisCacheGenerationSink(client, suffix)
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.ResetRAdvs(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := sink.PutRAdvs(ctx, 1, 8, []byte("new"), false); err != nil {
		t.Fatal(err)
	}

	var live string
	if err := client.Do(ctx, radix.Cmd(&live, "HGET", key, "7")); err != nil {
		t.Fatal(err)
	}
	if live != "old" {
		t.Fatalf("live cache value = %q, want old", live)
	}
	var staged string
	if err := client.Do(ctx, radix.Cmd(&staged, "HGET", key+suffix, "8")); err != nil {
		t.Fatal(err)
	}
	if staged != "new" {
		t.Fatalf("staged cache value = %q, want new", staged)
	}
	var marker string
	if err := client.Do(ctx, radix.Cmd(&marker, "HGET", key+suffix, cachegeneration.MarkerField)); err != nil {
		t.Fatal(err)
	}
	if marker != cachegeneration.MarkerValue {
		t.Fatalf("staged cache marker = %q, want %q", marker, cachegeneration.MarkerValue)
	}
}

func TestRedisCacheGenerationSinkRequiresNamespace(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := NewRedisCacheGenerationSink(client, ""); err == nil {
		t.Fatal("NewRedisCacheGenerationSink error = nil, want empty namespace rejection")
	}
}

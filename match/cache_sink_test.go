package match

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

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

package cache

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
)

func TestValidateMode(t *testing.T) {
	for _, mode := range []string{ModeRedis, ModeSpread, ModeAll, ModeRoutes} {
		if err := ValidateMode(mode); err != nil {
			t.Fatalf("ValidateMode(%q) = %v", mode, err)
		}
	}
	if err := ValidateMode("bad"); err == nil {
		t.Fatal("ValidateMode(bad) = nil, want error")
	}
}

func TestRunRejectsInvalidUpdateInterval(t *testing.T) {
	err := Run(context.TODO(), nil, nil, nil, nil, Options{Mode: ModeRedis})
	if err == nil {
		t.Fatal("Run with zero update interval = nil, want error")
	}
}

func TestSwapRedisStaticCachesReplacesOneCompleteGeneration(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	liveHash := func(key string) {
		t.Helper()
		if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "value", "old")); err != nil {
			t.Fatal(err)
		}
	}
	shadowHash := func(key string) {
		t.Helper()
		if err := client.Do(ctx, radix.Cmd(nil, "HSET", key+redisShadowSuffix, "value", "new")); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience, match.HashNameCreative} {
		liveHash(key)
	}
	for _, key := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience} {
		shadowHash(key)
	}
	for _, key := range []string{match.HashNameMiddlemanRoutes, match.HashNameMiddlemanRoutesV2} {
		if err := client.Do(ctx, radix.Cmd(nil, "SET", key, "old")); err != nil {
			t.Fatal(err)
		}
		if err := client.Do(ctx, radix.Cmd(nil, "SET", key+redisShadowSuffix, "new")); err != nil {
			t.Fatal(err)
		}
	}
	newSlot := match.HashNameRAdvs(1)
	oldSlot := match.HashNameRAdvs(2)
	liveHash(newSlot)
	liveHash(oldSlot)
	shadowHash(newSlot)

	if err := swapRedisStaticCaches(ctx, client, []uint32{1}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience, newSlot} {
		var got string
		if err := client.Do(ctx, radix.Cmd(&got, "HGET", key, "value")); err != nil {
			t.Fatal(err)
		}
		if got != "new" {
			t.Fatalf("%s value = %q, want new", key, got)
		}
	}
	for _, key := range []string{match.HashNameMiddlemanRoutes, match.HashNameMiddlemanRoutesV2} {
		var got string
		if err := client.Do(ctx, radix.Cmd(&got, "GET", key)); err != nil {
			t.Fatal(err)
		}
		if got != "new" {
			t.Fatalf("%s value = %q, want new", key, got)
		}
	}
	for _, key := range []string{match.HashNameCreative, oldSlot} {
		var exists int
		if err := client.Do(ctx, radix.Cmd(&exists, "EXISTS", key)); err != nil {
			t.Fatal(err)
		}
		if exists != 0 {
			t.Fatalf("%s still exists after empty/obsolete swap", key)
		}
	}
}

func TestWriteToRedisBuildFailureLeavesLiveGeneration(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Do(ctx, radix.Cmd(nil, "HSET", acl.HashNamePubmap, "value", "old")); err != nil {
		t.Fatal(err)
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT t.slot_id").WillReturnError(errors.New("build failed"))
	if err := WriteToRedis(ctx, client, db, nil, []uint32{1}); err == nil {
		t.Fatal("WriteToRedis error = nil, want build failure")
	}
	var got string
	if err := client.Do(ctx, radix.Cmd(&got, "HGET", acl.HashNamePubmap, "value")); err != nil {
		t.Fatal(err)
	}
	if got != "old" {
		t.Fatalf("live pubmap value = %q, want old generation", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAttributeLogScannerAcceptsLargeLines(t *testing.T) {
	line := bytes.Repeat([]byte("x"), maxAttributeLogLineBytes)
	scanner := newAttributeLogScanner(bytes.NewReader(append(line, '\n')))
	if !scanner.Scan() {
		t.Fatalf("Scan = false: %v", scanner.Err())
	}
	if got := len(scanner.Bytes()); got != len(line) {
		t.Fatalf("line length = %d, want %d", got, len(line))
	}
}

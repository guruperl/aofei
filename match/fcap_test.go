package match

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

// TestFcap tests the Fcap struct.
func TestFcap(t *testing.T) {
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	fcap := NewFcap(now)
	if fcap.Total != 0 {
		t.Errorf("Total should be 0, but got %d", fcap.Total)
	}
	if fcap.StartYM != 1 {
		t.Errorf("StartYM should be 1, but got %d", fcap.StartYM)
	}
	if fcap.StartDHM != 2048 {
		t.Errorf("StartDHM should be 2048, but got %d", fcap.StartDHM)
	}
	if fcap.Last != 0 {
		t.Errorf("Last should be 0, but got %d", fcap.Last)
	}
}

func TestBothCapRefreshClickPeriod(t *testing.T) {
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	when := start.Add(11 * time.Minute)
	bothcap := NewBothCap(start)
	bothcap.Cli.Refresh(start)

	bothcap.Refresh(when, RAdv{Cap: Cap{CapPeriod: 60, ClickPeriod: 10}}, false, true)

	if bothcap.Cli.Total != 1 {
		t.Fatalf("Cli.Total = %d, want reset count 1", bothcap.Cli.Total)
	}
	if !bothcap.Cli.GetStart().Equal(when) {
		t.Fatalf("Cli.GetStart() = %s, want %s", bothcap.Cli.GetStart(), when)
	}
}

func TestBothCapRefreshExpiredImpAndClick(t *testing.T) {
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	when := start.Add(2 * time.Hour)
	bothcap := NewBothCap(start)
	bothcap.Imp.Refresh(start)
	bothcap.Cli.Refresh(start)

	bothcap.Refresh(when, RAdv{Cap: Cap{CapPeriod: 30, ClickPeriod: 10}}, true, true)

	if bothcap.Imp.Total != 1 {
		t.Fatalf("Imp.Total = %d, want reset count 1", bothcap.Imp.Total)
	}
	if bothcap.Cli.Total != 1 {
		t.Fatalf("Cli.Total = %d, want reset count 1", bothcap.Cli.Total)
	}
	if !bothcap.Imp.GetStart().Equal(when) {
		t.Fatalf("Imp.GetStart() = %s, want %s", bothcap.Imp.GetStart(), when)
	}
	if !bothcap.Cli.GetStart().Equal(when) {
		t.Fatalf("Cli.GetStart() = %s, want %s", bothcap.Cli.GetStart(), when)
	}
}

func TestFcapRefreshSaturatesTotal(t *testing.T) {
	when := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	fcap := NewFcap(when)
	fcap.Total = 254
	fcap.Refresh(when)
	fcap.Refresh(when)
	if fcap.Total != 255 {
		t.Fatalf("Total = %d, want saturated 255", fcap.Total)
	}
}

func TestFcapUTCFormatCoversNinetyDaysWithoutMinuteWrap(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, time.March, 7, 1, 30, 0, 0, location)
	last := start.Add(90 * 24 * time.Hour)
	state := NewBothCap(start)
	state.Imp.Refresh(last)
	packed, err := state.Pack()
	if err != nil {
		t.Fatal(err)
	}
	legacySize := binary.Size(legacyBothCapWire{})
	if len(packed) < legacySize+3 || string(packed[legacySize:legacySize+2]) != bothCapFormatMagic || packed[legacySize+2] != bothCapFormatUTC {
		t.Fatalf("wire prefix = %x, want versioned UTC format", packed)
	}
	var legacyReader legacyBothCapWire
	if err := binary.Read(bytes.NewReader(packed), binary.LittleEndian, &legacyReader); err != nil {
		t.Fatal(err)
	}
	if legacyReader.Imp.Total != state.Imp.Total || legacyReader.Imp.Last != ^uint16(0) {
		t.Fatalf("legacy-prefix view = %+v, want safe saturated compatibility", legacyReader.Imp)
	}
	roundTrip, err := UnpackBothCap(packed)
	if err != nil {
		t.Fatal(err)
	}
	if !roundTrip.Imp.GetStart().Equal(start.UTC().Truncate(time.Minute)) {
		t.Fatalf("start = %s, want %s", roundTrip.Imp.GetStart(), start.UTC())
	}
	if !roundTrip.Imp.GetLast().Equal(last.UTC().Truncate(time.Minute)) {
		t.Fatalf("last = %s, want %s", roundTrip.Imp.GetLast(), last.UTC())
	}
	if got := roundTrip.Imp.SinceStart(last); got != ^uint16(0) {
		t.Fatalf("90-day elapsed view = %d, want saturated %d", got, ^uint16(0))
	}
	if roundTrip.Imp.Last != ^uint16(0) {
		t.Fatalf("legacy last view = %d, want saturated diagnostic value", roundTrip.Imp.Last)
	}
}

func TestFcapFutureAndOutOfOrderTimesClampSafely(t *testing.T) {
	now := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)
	future := NewFcap(now.Add(time.Hour))
	future.Total = 2
	if got := future.SinceStart(now); got != 0 {
		t.Fatalf("future start elapsed = %d, want zero", got)
	}
	if (Cap{CapNumber: 2, CapPeriod: 60}).CanServe(now, BothCap{Imp: future}) {
		t.Fatal("future cap state bypassed an already reached limit")
	}
	last := future.GetLast()
	future.Refresh(now)
	if !future.GetLast().Equal(last) {
		t.Fatalf("out-of-order refresh moved last backward: %s -> %s", last, future.GetLast())
	}
}

func TestLegacyBothCapReadsAreUTCAndUpgradeOnWrite(t *testing.T) {
	type legacyFcap struct {
		Total    uint8
		StartYM  uint8
		StartDHM uint16
		Last     uint16
	}
	type legacyBothCap struct {
		Imp legacyFcap
		Cli legacyFcap
	}
	start := time.Date(2026, time.March, 8, 1, 30, 0, 0, time.UTC)
	legacy := legacyFcap{
		Total: 3, StartYM: uint8((start.Year()-FCAPStartYear)<<4 + int(start.Month())),
		StartDHM: uint16(start.Day()<<11 + start.Hour()<<6 + start.Minute()), Last: 45,
	}
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, legacyBothCap{Imp: legacy, Cli: legacy}); err != nil {
		t.Fatal(err)
	}
	originalLocal := time.Local
	defer func() { time.Local = originalLocal }()
	for _, name := range []string{"America/Los_Angeles", "Asia/Shanghai"} {
		location, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		time.Local = location
		state, err := UnpackBothCap(buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if !state.Imp.GetStart().Equal(start) || !state.Imp.GetLast().Equal(start.Add(45*time.Minute)) {
			t.Fatalf("legacy state under %s = %s/%s", name, state.Imp.GetStart(), state.Imp.GetLast())
		}
		upgraded, err := state.Pack()
		if err != nil {
			t.Fatal(err)
		}
		legacySize := binary.Size(legacyBothCapWire{})
		if len(upgraded) == len(buf.Bytes()) || string(upgraded[legacySize:legacySize+2]) != bothCapFormatMagic || upgraded[legacySize+2] != bothCapFormatUTC {
			t.Fatalf("upgraded wire under %s = %x", name, upgraded)
		}
	}
}

func TestFcapElapsedBoundariesDoNotWrap(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	fcap := NewFcap(start)
	if got := fcap.SinceStart(start.Add(45 * 24 * time.Hour)); got != 64800 {
		t.Fatalf("45-day elapsed = %d, want 64800", got)
	}
	if got := fcap.SinceStart(start.Add(90 * 24 * time.Hour)); got != ^uint16(0) {
		t.Fatalf("90-day elapsed = %d, want saturated %d", got, ^uint16(0))
	}
}

func TestMustRefreshBothCapConcurrent(t *testing.T) {
	server := miniredis.RunT(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := (radix.PoolConfig{Size: 4}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	pid := "fcap-concurrent-test"
	itemID := uint32(123)
	key := HashNameBothCap(pid)
	if err := client.Do(ctx, radix.Cmd(nil, "DEL", key)); err != nil {
		t.Fatal(err)
	}

	when := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			errs <- MustRefreshBothCap(ctx, client, when, pid, itemID, Cap{CapPeriod: 60}, true, false)
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	var data []byte
	if err := client.Do(ctx, radix.Cmd(&data, "HGET", key, "123")); err != nil {
		t.Fatal(err)
	}
	bothcap, err := UnpackBothCap(data)
	if err != nil {
		t.Fatal(err)
	}
	if bothcap.Imp.Total != 2 {
		t.Fatalf("Imp.Total = %d, want 2", bothcap.Imp.Total)
	}
	var ttl int64
	if err := client.Do(ctx, radix.Cmd(&ttl, "TTL", key)); err != nil {
		t.Fatal(err)
	}
	if ttl <= 0 {
		t.Fatalf("TTL = %d, want positive", ttl)
	}
}

func TestMustRefreshBothCapDoesNotShortenLongerTTL(t *testing.T) {
	server := miniredis.RunT(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	key := HashNameBothCap("fcap-ttl-test")
	defer client.Do(ctx, radix.Cmd(nil, "DEL", key))
	if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "1", "old")); err != nil {
		t.Fatal(err)
	}
	if err := client.Do(ctx, radix.Cmd(nil, "EXPIRE", key, "7200")); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := MustRefreshBothCapWithTTL(ctx, client, when, "fcap-ttl-test", 2, Cap{CapPeriod: 60}, time.Hour, true, false); err != nil {
		t.Fatal(err)
	}
	var ttl int64
	if err := client.Do(ctx, radix.Cmd(&ttl, "TTL", key)); err != nil {
		t.Fatal(err)
	}
	if ttl < 7100 {
		t.Fatalf("TTL = %d, want existing longer TTL preserved", ttl)
	}
}

func TestBothCapsToRedisWithTTLPreservesLongerExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	key := HashNameBothCap("bulk-longer-ttl")
	if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "1", "old")); err != nil {
		t.Fatal(err)
	}
	if err := client.Do(ctx, radix.Cmd(nil, "EXPIRE", key, "7200")); err != nil {
		t.Fatal(err)
	}
	state := NewBothCap(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC))
	state.Imp.Refresh(state.Imp.GetStart())
	if err := BothCapsToRedisWithTTL(ctx, client, "bulk-longer-ttl", map[uint32]BothCap{2: state}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if ttl := server.TTL(key); ttl < 119*time.Minute {
		t.Fatalf("TTL = %s, want existing longer TTL preserved", ttl)
	}
	if server.HGet(key, "2") == "" {
		t.Fatal("bulk cap data was not committed")
	}
}

func TestBothCapsToRedisWithTTLAtomicallyAddsExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	state := NewBothCap(time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC))
	for _, pid := range []string{"bulk-new", "bulk-persistent"} {
		key := HashNameBothCap(pid)
		if pid == "bulk-persistent" {
			if err := client.Do(ctx, radix.Cmd(nil, "HSET", key, "1", "old")); err != nil {
				t.Fatal(err)
			}
			if ttl := server.TTL(key); ttl != 0 {
				t.Fatalf("persistent setup TTL = %s, want 0", ttl)
			}
		}
		if err := BothCapsToRedisWithTTL(ctx, client, pid, map[uint32]BothCap{2: state}, time.Hour); err != nil {
			t.Fatal(err)
		}
		if server.HGet(key, "2") == "" {
			t.Fatalf("%s data was not committed", pid)
		}
		if ttl := server.TTL(key); ttl <= 0 {
			t.Fatalf("%s TTL = %s, want data and positive expiry together", pid, ttl)
		}
	}
}

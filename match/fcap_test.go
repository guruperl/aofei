package match

import (
	"bytes"
	"context"
	"encoding/binary"
	"expvar"
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

func TestBothCapPackKeepsLegacyPrefixInLocalWallTime(t *testing.T) {
	local := time.FixedZone("UTC-8", -8*60*60)
	originalLocal := time.Local
	time.Local = local
	defer func() { time.Local = originalLocal }()

	start := time.Date(2026, time.August, 1, 12, 15, 0, 0, time.UTC)
	last := start.Add(45 * time.Minute)
	state := NewBothCap(start)
	state.Imp.Refresh(last)
	packed, err := state.Pack()
	if err != nil {
		t.Fatal(err)
	}

	var legacy legacyBothCapWire
	if err := binary.Read(bytes.NewReader(packed), binary.LittleEndian, &legacy); err != nil {
		t.Fatal(err)
	}
	legacyImp := Fcap{
		Total:    legacy.Imp.Total,
		StartYM:  legacy.Imp.StartYM,
		StartDHM: legacy.Imp.StartDHM,
		Last:     legacy.Imp.Last,
	}
	if got := legacyImp.GetStart(); !got.Equal(start) {
		t.Fatalf("legacy-prefix start = %s, want local wall-clock instant %s", got, start)
	}
	if got := legacyImp.GetLast(); !got.Equal(last) {
		t.Fatalf("legacy-prefix last = %s, want %s", got, last)
	}

	current, err := UnpackBothCap(packed)
	if err != nil {
		t.Fatal(err)
	}
	if got := current.Imp.GetStart(); !got.Equal(start) {
		t.Fatalf("v2 start = %s, want UTC instant %s", got, start)
	}
	if got := current.Imp.GetLast(); !got.Equal(last) {
		t.Fatalf("v2 last = %s, want UTC instant %s", got, last)
	}
}

func TestBothCapFormatMetricsUseFixedKeys(t *testing.T) {
	legacyBefore := expvarMapValue(metricBothCapFormats, "legacy")
	v2Before := expvarMapValue(metricBothCapFormats, "utc_v2")
	malformedBefore := expvarMapValue(metricBothCapFormats, "malformed")

	legacy := new(bytes.Buffer)
	if err := binary.Write(legacy, binary.LittleEndian, legacyBothCapWire{}); err != nil {
		t.Fatal(err)
	}
	if _, err := UnpackBothCap(legacy.Bytes()); err != nil {
		t.Fatal(err)
	}
	packed, err := NewBothCap(time.Now()).Pack()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UnpackBothCap(packed); err != nil {
		t.Fatal(err)
	}
	if _, err := UnpackBothCap([]byte("dynamic-format-id")); err == nil {
		t.Fatal("malformed cap payload succeeded")
	}
	if got := expvarMapValue(metricBothCapFormats, "legacy") - legacyBefore; got != 1 {
		t.Fatalf("legacy format metric delta = %d, want 1", got)
	}
	if got := expvarMapValue(metricBothCapFormats, "utc_v2") - v2Before; got != 1 {
		t.Fatalf("v2 format metric delta = %d, want 1", got)
	}
	if got := expvarMapValue(metricBothCapFormats, "malformed") - malformedBefore; got != 1 {
		t.Fatalf("malformed format metric delta = %d, want 1", got)
	}
	if metricBothCapFormats.Get("dynamic-format-id") != nil {
		t.Fatal("dynamic cap-format metric key was published")
	}
}

func expvarMapValue(metric *expvar.Map, key string) int64 {
	value, _ := metric.Get(key).(*expvar.Int)
	if value == nil {
		return 0
	}
	return value.Value()
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

func TestLegacyBothCapReadsUseLocalWallClockAndUpgradeToUTC(t *testing.T) {
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
	// Legacy fields carry wall-clock components only; the pre-D04 reader
	// interpreted them in time.Local. Use one fixed wall-clock and verify each
	// deployment zone observes the same local instant, then that the write
	// upgrade preserves the absolute instant.
	wall := time.Date(2026, time.March, 8, 1, 30, 0, 0, time.UTC)
	legacy := legacyFcap{
		Total:    3,
		StartYM:  uint8((wall.Year()-FCAPStartYear)<<4 + int(wall.Month())),
		StartDHM: uint16(wall.Day()<<11 + wall.Hour()<<6 + wall.Minute()),
		Last:     45,
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
		wantStart := time.Date(2026, time.March, 8, 1, 30, 0, 0, location)
		wantLast := wantStart.Add(45 * time.Minute)
		if !state.Imp.GetStart().Equal(wantStart) || !state.Imp.GetLast().Equal(wantLast) {
			t.Fatalf("legacy state under %s = %s/%s, want %s/%s", name, state.Imp.GetStart(), state.Imp.GetLast(), wantStart, wantLast)
		}
		upgraded, err := state.Pack()
		if err != nil {
			t.Fatal(err)
		}
		legacySize := binary.Size(legacyBothCapWire{})
		if len(upgraded) == len(buf.Bytes()) || string(upgraded[legacySize:legacySize+2]) != bothCapFormatMagic || upgraded[legacySize+2] != bothCapFormatUTC {
			t.Fatalf("upgraded wire under %s = %x", name, upgraded)
		}
		roundTrip, err := UnpackBothCap(upgraded)
		if err != nil {
			t.Fatal(err)
		}
		if !roundTrip.Imp.GetStart().Equal(wantStart.UTC().Truncate(time.Minute)) {
			t.Fatalf("upgraded start under %s = %s, want %s", name, roundTrip.Imp.GetStart(), wantStart.UTC())
		}
	}
}

func TestLegacyStartInIsLocationAwareWithoutGlobalTimezone(t *testing.T) {
	// Exercises the location-parameterized reader directly so the conversion
	// is verified without mutating time.Local.
	wall := time.Date(2026, time.March, 8, 1, 30, 0, 0, time.UTC)
	fcap := Fcap{
		StartYM:  uint8((wall.Year()-FCAPStartYear)<<4 + int(wall.Month())),
		StartDHM: uint16(wall.Day()<<11 + wall.Hour()<<6 + wall.Minute()),
	}
	la := time.FixedZone("UTC-8", -8*3600)
	shanghai := time.FixedZone("UTC+8", 8*3600)
	wantLA := time.Date(2026, time.March, 8, 1, 30, 0, 0, la)
	wantShanghai := time.Date(2026, time.March, 8, 1, 30, 0, 0, shanghai)
	if got := fcap.legacyStartIn(la); !got.Equal(wantLA) {
		t.Fatalf("legacy start in UTC-8 = %s, want %s", got, wantLA)
	}
	if got := fcap.legacyStartIn(shanghai); !got.Equal(wantShanghai) {
		t.Fatalf("legacy start in UTC+8 = %s, want %s", got, wantShanghai)
	}
	if got := wantLA.UTC(); !got.Equal(time.Date(2026, time.March, 8, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("UTC-8 instant in UTC = %s, want 2026-03-08T09:30Z", got)
	}
	if got := wantShanghai.UTC(); !got.Equal(time.Date(2026, time.March, 7, 17, 30, 0, 0, time.UTC)) {
		t.Fatalf("UTC+8 instant in UTC = %s, want 2026-03-07T17:30Z", got)
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

func TestMustRefreshBothCapEventMarkerUsesAbsoluteDeadline(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	eventTTL := 5 * time.Second
	applied, err := MustRefreshBothCapOnceWithTTL(ctx, client, time.Now(), "absolute-event-expiry", 1, Cap{CapThrottle: 1}, time.Hour, "event:absolute", eventTTL, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("cap update was not applied")
	}
	if ttl := server.TTL("event:absolute"); ttl <= 0 || ttl > eventTTL {
		t.Fatalf("event marker TTL = %s, want positive and no later than %s", ttl, eventTTL)
	}
}

func TestMustRefreshBothCapEventMarkerValueIsNonAuthoritative(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	const eventKey = "event:opaque-label"
	if err := client.Do(ctx, radix.Cmd(nil, "SET", eventKey, "not-a-password", "EX", "60")); err != nil {
		t.Fatal(err)
	}
	applied, err := MustRefreshBothCapOnceWithTTL(ctx, client, time.Now(), "marker-value", 1, Cap{CapThrottle: 1}, time.Hour, eventKey, time.Minute, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("an existing opaque marker permitted a second cap update")
	}
	if server.Exists(HashNameBothCap("marker-value")) {
		t.Fatal("an arbitrary marker value caused cap state mutation")
	}
	if value, err := server.Get(eventKey); err != nil || value != "not-a-password" {
		t.Fatalf("marker value was interpreted or rewritten: value=%q error=%v", value, err)
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

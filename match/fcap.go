package match

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"expvar"
	"fmt"
	"strconv"
	"time"

	"github.com/mediocregopher/radix/v4"
)

const (
	FCAPStartYear         = 2025
	bothCapRefreshRetries = 8
	defaultBothCapTTL     = 90 * 24 * time.Hour
	bothCapCleanupTimeout = 2 * time.Second
	bothCapFormatMagic    = "BC"
	bothCapFormatUTC      = byte(2)
	fcapFormatUTC         = uint8(2)
	maxUnixMinute         = int64((1<<63 - 1) / 60)
	minUnixMinute         = int64((-1 << 63) / 60)
)

var ErrBothCapRefreshConflict = errors.New("bothcap refresh conflict")

var (
	metricBothCapRefreshes        = expvar.NewInt("aofei_bothcap_refresh_total")
	metricBothCapRefreshRetries   = expvar.NewInt("aofei_bothcap_refresh_retries_total")
	metricBothCapRefreshConflicts = expvar.NewInt("aofei_bothcap_refresh_conflicts_total")
	metricBothCapRefreshLastMS    = expvar.NewInt("aofei_bothcap_refresh_last_ms")
)

// Fcap is frequency-cap state. The exported calendar fields retain the legacy
// diagnostic view. New state uses unexported UTC epoch-minute fields so the
// public Go shape remains source compatible while the Redis wire format can
// cover the configured retention without local-time or uint16 wrap.
type Fcap struct {
	Total          uint8  `json:"total"`
	StartYM        uint8  `json:"start_ym"`
	StartDHM       uint16 `json:"start_dhm"`
	Last           uint16 `json:"last"`
	format         uint8
	startUTCMinute int64
	lastUTCMinute  int64
}

type legacyFcapWire struct {
	Total    uint8
	StartYM  uint8
	StartDHM uint16
	Last     uint16
}

type legacyBothCapWire struct {
	Imp legacyFcapWire
	Cli legacyFcapWire
}

// NewFcap creates a new Fcap instance from time
func NewFcap(when time.Time) Fcap {
	when = when.UTC().Truncate(time.Minute)
	fcap := Fcap{
		format:         fcapFormatUTC,
		startUTCMinute: when.Unix() / 60,
		lastUTCMinute:  when.Unix() / 60,
	}
	fcap.setLegacyView(when, when)
	return fcap
}

// Refresh adds one more count and update the last access time
func (self *Fcap) Refresh(when time.Time) {
	if self == nil {
		return
	}
	self.ensureUTC()
	if self.Total < ^uint8(0) {
		self.Total++
	}
	whenMinute := when.UTC().Unix() / 60
	if whenMinute > self.lastUTCMinute {
		self.lastUTCMinute = whenMinute
	}
	self.Last = saturatedMinutes(self.lastUTCMinute - self.startUTCMinute)
}

// GetStart gets starting time in time
func (self Fcap) GetStart() time.Time {
	if self.format == fcapFormatUTC {
		return time.Unix(self.startUTCMinute*60, 0).UTC()
	}
	return self.legacyStartUTC()
}

// GetLast gets last access time in time
func (self Fcap) GetLast() time.Time {
	if self.format == fcapFormatUTC {
		return time.Unix(self.lastUTCMinute*60, 0).UTC()
	}
	return self.legacyStartUTC().Add(time.Duration(self.Last) * time.Minute)
}

// SinceStart reports minutes passed since the start
func (self Fcap) SinceStart(when time.Time) uint16 {
	return saturatedMinutes(elapsedMinutes(self.GetStart(), when))
}

// SinceLast reports minutes passed since the last access
func (self Fcap) SinceLast(when time.Time) uint16 {
	return saturatedMinutes(elapsedMinutes(self.GetLast(), when))
}

func (self Fcap) legacyStartUTC() time.Time {
	return time.Date(FCAPStartYear+int(self.StartYM>>4), time.Month(15&self.StartYM), int(self.StartDHM>>11), int(31&(self.StartDHM>>6)), int(63&self.StartDHM), 0, 0, time.UTC)
}

func (self *Fcap) ensureUTC() {
	if self.format == fcapFormatUTC {
		return
	}
	start := self.legacyStartUTC()
	last := start.Add(time.Duration(self.Last) * time.Minute)
	self.format = fcapFormatUTC
	self.startUTCMinute = start.Unix() / 60
	self.lastUTCMinute = last.Unix() / 60
}

func (self *Fcap) setLegacyView(start, last time.Time) {
	start = start.UTC()
	self.StartYM = uint8((start.Year()-FCAPStartYear)<<4 + int(start.Month()))
	self.StartDHM = uint16(start.Day()<<11 + start.Hour()<<6 + start.Minute())
	self.Last = saturatedMinutes(elapsedMinutes(start, last.UTC()))
}

func elapsedMinutes(start, when time.Time) int64 {
	minutes := int64(when.UTC().Sub(start.UTC()) / time.Minute)
	if minutes < 0 {
		return 0
	}
	return minutes
}

func saturatedMinutes(minutes int64) uint16 {
	if minutes <= 0 {
		return 0
	}
	if minutes > int64(^uint16(0)) {
		return ^uint16(0)
	}
	return uint16(minutes)
}

type BothCap struct {
	Imp Fcap
	Cli Fcap
}

// NewBothCap creates a new BothCap instance
func NewBothCap(when time.Time) BothCap {
	return BothCap{
		Imp: NewFcap(when),
		Cli: NewFcap(when),
	}
}

// Pack packs the BothCap into bytes
func (self BothCap) Pack() ([]byte, error) {
	self.Imp.ensureUTC()
	self.Cli.ensureUTC()
	self.Imp.setLegacyView(self.Imp.GetStart(), self.Imp.GetLast())
	self.Cli.setLegacyView(self.Cli.GetStart(), self.Cli.GetLast())
	buf := new(bytes.Buffer)
	legacy := legacyBothCapWire{
		Imp: legacyFcapWire{Total: self.Imp.Total, StartYM: self.Imp.StartYM, StartDHM: self.Imp.StartDHM, Last: self.Imp.Last},
		Cli: legacyFcapWire{Total: self.Cli.Total, StartYM: self.Cli.StartYM, StartDHM: self.Cli.StartDHM, Last: self.Cli.Last},
	}
	if err := binary.Write(buf, binary.LittleEndian, legacy); err != nil {
		return nil, err
	}
	_, _ = buf.WriteString(bothCapFormatMagic)
	_ = buf.WriteByte(bothCapFormatUTC)
	for _, current := range []Fcap{self.Imp, self.Cli} {
		if err := binary.Write(buf, binary.LittleEndian, current.startUTCMinute); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, current.lastUTCMinute); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// PackString packs the BothCap into RawURL string
func (self BothCap) PackString() (string, error) {
	data, err := self.Pack()
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// UnpackBothCap unpacks the BothCap from bytes
func UnpackBothCap(data []byte) (BothCap, error) {
	legacySize := binary.Size(legacyBothCapWire{})
	if len(data) < legacySize {
		return BothCap{}, fmt.Errorf("unsupported bothcap payload format")
	}
	var legacy legacyBothCapWire
	if err := binary.Read(bytes.NewReader(data[:legacySize]), binary.LittleEndian, &legacy); err != nil {
		return BothCap{}, err
	}
	legacyState := BothCap{
		Imp: Fcap{Total: legacy.Imp.Total, StartYM: legacy.Imp.StartYM, StartDHM: legacy.Imp.StartDHM, Last: legacy.Imp.Last},
		Cli: Fcap{Total: legacy.Cli.Total, StartYM: legacy.Cli.StartYM, StartDHM: legacy.Cli.StartDHM, Last: legacy.Cli.Last},
	}
	if len(data) == legacySize {
		return legacyState, nil
	}
	if len(data) != legacySize+3+2*(8+8) || string(data[legacySize:legacySize+2]) != bothCapFormatMagic || data[legacySize+2] != bothCapFormatUTC {
		return BothCap{}, fmt.Errorf("unsupported bothcap payload format")
	}
	buf := bytes.NewReader(data[legacySize+3:])
	decode := func(total uint8) (Fcap, error) {
		var startMinute, lastMinute int64
		if err := binary.Read(buf, binary.LittleEndian, &startMinute); err != nil {
			return Fcap{}, err
		}
		if err := binary.Read(buf, binary.LittleEndian, &lastMinute); err != nil {
			return Fcap{}, err
		}
		if startMinute < minUnixMinute || startMinute > maxUnixMinute || lastMinute < minUnixMinute || lastMinute > maxUnixMinute {
			return Fcap{}, fmt.Errorf("bothcap UTC minute is outside the supported range")
		}
		if lastMinute < startMinute {
			return Fcap{}, fmt.Errorf("bothcap last minute precedes start minute")
		}
		start := time.Unix(startMinute*60, 0).UTC()
		last := time.Unix(lastMinute*60, 0).UTC()
		fcap := Fcap{Total: total, format: fcapFormatUTC, startUTCMinute: startMinute, lastUTCMinute: lastMinute}
		fcap.setLegacyView(start, last)
		return fcap, nil
	}
	imp, err := decode(legacy.Imp.Total)
	if err != nil {
		return BothCap{}, err
	}
	cli, err := decode(legacy.Cli.Total)
	if err != nil {
		return BothCap{}, err
	}
	return BothCap{Imp: imp, Cli: cli}, nil
}

// UnpackBothCapString unpacks the BothCap from RawURL string
func UnpackBothCapString(text string) (BothCap, error) {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return BothCap{}, err
	}
	return UnpackBothCap(data)
}

func (self *BothCap) Refresh(when time.Time, block RAdv, isImp bool, isCli bool) {
	imp := self.Imp
	cli := self.Cli
	if isImp {
		if !block.Cap.ValidPeriodImp(when, imp) {
			imp = NewFcap(when)
		}
		imp.Refresh(when)
	}
	if isCli {
		if !block.Cap.ValidPeriodCli(when, cli) {
			cli = NewFcap(when)
		}
		cli.Refresh(when)
	}
	(*self).Imp = imp
	(*self).Cli = cli
}

func HashNameBothCap(pid string) string {
	return fmt.Sprintf("bothcap:%s", pid)
}

// MustRefreshBothCap reads bothcap from Redis and refreshes it. And write it back to Redis.
func MustRefreshBothCap(ctx context.Context, conn radix.Client, when time.Time, pid string, itemID uint32, cap Cap, isImp bool, isCli bool) error {
	return MustRefreshBothCapWithTTL(ctx, conn, when, pid, itemID, cap, defaultBothCapTTL, isImp, isCli)
}

// MustRefreshBothCapWithTTL refreshes cap state and bounds idle Redis retention.
func MustRefreshBothCapWithTTL(ctx context.Context, conn radix.Client, when time.Time, pid string, itemID uint32, cap Cap, ttl time.Duration, isImp bool, isCli bool) error {
	_, err := mustRefreshBothCapWithTTL(ctx, conn, when, pid, itemID, cap, ttl, "", 0, isImp, isCli)
	return err
}

// MustRefreshBothCapOnceWithTTL refreshes cap state at most once for eventKey.
// The cap value and event marker are committed in the same Redis transaction.
func MustRefreshBothCapOnceWithTTL(ctx context.Context, conn radix.Client, when time.Time, pid string, itemID uint32, cap Cap, ttl time.Duration, eventKey string, eventTTL time.Duration, isImp bool, isCli bool) (bool, error) {
	if eventKey == "" {
		return false, fmt.Errorf("bothcap event key is empty")
	}
	if eventTTL <= 0 {
		return false, fmt.Errorf("bothcap event TTL must be positive")
	}
	return mustRefreshBothCapWithTTL(ctx, conn, when, pid, itemID, cap, ttl, eventKey, eventTTL, isImp, isCli)
}

func mustRefreshBothCapWithTTL(ctx context.Context, conn radix.Client, when time.Time, pid string, itemID uint32, cap Cap, ttl time.Duration, eventKey string, eventTTL time.Duration, isImp bool, isCli bool) (bool, error) {
	if !isImp && !isCli {
		return false, nil
	}
	if ttl <= 0 {
		return false, fmt.Errorf("bothcap TTL must be positive")
	}
	started := time.Now()
	defer func() {
		metricBothCapRefreshLastMS.Set(time.Since(started).Milliseconds())
	}()
	metricBothCapRefreshes.Add(1)
	if conn == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	key := HashNameBothCap(pid)
	itemIDStr := fmt.Sprintf("%d", itemID)
	ttlSeconds := int64((ttl + time.Second - 1) / time.Second)
	eventTTLSeconds := int64((eventTTL + time.Second - 1) / time.Second)
	for attempt := 0; attempt < bothCapRefreshRetries; attempt++ {
		var retry bool
		var applied bool
		err := conn.Do(ctx, radix.WithConn(key, func(ctx context.Context, redisConn radix.Conn) (err error) {
			state := bothCapConnectionClean
			watchKeys := []string{key}
			if eventKey != "" {
				watchKeys = append(watchKeys, eventKey)
			}
			if err = redisConn.Do(ctx, radix.Cmd(nil, "WATCH", watchKeys...)); err != nil {
				return err
			}
			state = bothCapConnectionWatching
			defer func() {
				if err == nil || state == bothCapConnectionClean {
					return
				}
				if cleanupErr := cleanupBothCapConnection(redisConn, state); cleanupErr != nil {
					err = cleanupErr
				}
			}()

			if eventKey != "" {
				var exists int
				if err = redisConn.Do(ctx, radix.Cmd(&exists, "EXISTS", eventKey)); err != nil {
					return err
				}
				if exists != 0 {
					err = redisConn.Do(ctx, radix.Cmd(nil, "UNWATCH"))
					if err == nil {
						state = bothCapConnectionClean
					}
					return err
				}
			}

			var data []byte
			if err = redisConn.Do(ctx, radix.Cmd(&data, "HGET", key, itemIDStr)); err != nil {
				return err
			}
			var currentTTL int64
			if err = redisConn.Do(ctx, radix.Cmd(&currentTTL, "TTL", key)); err != nil {
				return err
			}

			var bothcap BothCap
			if len(data) > 0 {
				bothcap, err = UnpackBothCap(data)
				if err != nil {
					return err
				}
			} else {
				bothcap = NewBothCap(when)
			}
			bothcap.Refresh(when, RAdv{Cap: cap}, isImp, isCli)

			packed, err := bothcap.Pack()
			if err != nil {
				return err
			}
			if err = redisConn.Do(ctx, radix.Cmd(nil, "MULTI")); err != nil {
				return err
			}
			state = bothCapConnectionMulti
			if err = redisConn.Do(ctx, radix.Cmd(nil, "HSET", key, itemIDStr, string(packed))); err != nil {
				return err
			}
			if currentTTL < ttlSeconds {
				if err = redisConn.Do(ctx, radix.Cmd(nil, "EXPIRE", key, strconv.FormatInt(ttlSeconds, 10))); err != nil {
					return err
				}
			}
			if eventKey != "" {
				if err = redisConn.Do(ctx, radix.Cmd(nil, "SETNX", eventKey, "1")); err != nil {
					return err
				}
				if err = redisConn.Do(ctx, radix.Cmd(nil, "EXPIRE", eventKey, strconv.FormatInt(eventTTLSeconds, 10))); err != nil {
					return err
				}
			}
			var result []int
			execResult := radix.Maybe{Rcv: &result}
			if err = redisConn.Do(ctx, radix.Cmd(&execResult, "EXEC")); err != nil {
				return err
			}
			state = bothCapConnectionClean
			if execResult.Null {
				retry = true
			} else {
				applied = true
			}
			return nil
		}))
		if err != nil {
			return false, err
		}
		if !retry {
			return applied, nil
		}
		metricBothCapRefreshRetries.Add(1)
	}
	metricBothCapRefreshConflicts.Add(1)
	return false, ErrBothCapRefreshConflict
}

type bothCapConnectionState uint8

const (
	bothCapConnectionClean bothCapConnectionState = iota
	bothCapConnectionWatching
	bothCapConnectionMulti
)

func cleanupBothCapConnection(conn radix.Conn, state bothCapConnectionState) error {
	command := "UNWATCH"
	if state == bothCapConnectionMulti {
		command = "DISCARD"
	}
	ctx, cancel := context.WithTimeout(context.Background(), bothCapCleanupTimeout)
	defer cancel()
	if err := conn.Do(ctx, radix.Cmd(nil, command)); err != nil {
		// Do not wrap the Redis error: WithConn pools must see this as a
		// connection-state failure and discard the connection instead of
		// returning potentially watched/transactional state to the pool.
		return fmt.Errorf("bothcap Redis connection cleanup with %s failed: %v", command, err)
	}
	return nil
}

func BothCapsToRedis(ctx context.Context, conn radix.Client, pid string, bothcaps map[uint32]BothCap) error {
	return BothCapsToRedisWithTTL(ctx, conn, pid, bothcaps, defaultBothCapTTL)
}

var bothCapsToRedisWithTTLScript = radix.NewEvalScript(`
redis.call("HSET", KEYS[1], unpack(ARGV, 2))
local current_ttl = redis.call("TTL", KEYS[1])
local requested_ttl = tonumber(ARGV[1])
if current_ttl < requested_ttl then
  redis.call("EXPIRE", KEYS[1], requested_ttl)
end
return 1`)

// BothCapsToRedisWithTTL writes cap state with bounded idle Redis retention.
func BothCapsToRedisWithTTL(ctx context.Context, conn radix.Client, pid string, bothcaps map[uint32]BothCap, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("bothcap TTL must be positive")
	}
	args := make([]string, 1, 1+2*len(bothcaps))
	for itemID, bothcap := range bothcaps {
		data, err := bothcap.Pack()
		if err != nil {
			return err
		}
		args = append(args, fmt.Sprintf("%d", itemID), string(data))
	}
	if len(args) == 1 {
		return nil
	}
	ttlSeconds := int64((ttl + time.Second - 1) / time.Second)
	key := HashNameBothCap(pid)
	args[0] = strconv.FormatInt(ttlSeconds, 10)
	return conn.Do(ctx, bothCapsToRedisWithTTLScript.Cmd(nil, []string{key}, args...))
}

// BothCapsFromRedis retrieves bothcaps from Redis.
func BothCapsFromRedis(ctx context.Context, conn radix.Client, pid string, itemIDs []string) (map[uint32]BothCap, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}

	data := make([]string, len(itemIDs))
	arr := append([]string{HashNameBothCap(pid)}, itemIDs...)
	err := conn.Do(ctx, radix.Cmd(&data, "HMGET", arr...))
	if err != nil {
		return nil, err
	}
	bothcaps := make(map[uint32]BothCap)
	for i, sdata := range data {
		if len(sdata) == 0 {
			continue
		}
		itemID, err := strconv.Atoi(itemIDs[i])
		if err != nil {
			return nil, err
		}
		bothcap, err := UnpackBothCap([]byte(sdata))
		if err != nil {
			return nil, err
		}
		bothcaps[uint32(itemID)] = bothcap
	}
	if len(bothcaps) == 0 {
		return nil, nil
	}
	return bothcaps, nil
}

// BothCapsCleanupExpired removes expired bothcaps from Redis.
func BothCapsCleanupExpired(ctx context.Context, conn radix.Client, pid string, itemIDs []string) error {
	arr := append([]string{HashNameBothCap(pid)}, itemIDs...)
	return conn.Do(ctx, radix.Cmd(nil, "HDEL", arr...))
}

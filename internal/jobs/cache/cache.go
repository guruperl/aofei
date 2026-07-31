package cache

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

const (
	redisShadowSuffix        = ":next"
	maxAttributeLogLineBytes = 8 * 1024 * 1024
)

const (
	ModeRedis  = "redis"
	ModeSpread = "spread"
	ModeAll    = "all"
	ModeRoutes = "routes"
)

type Options struct {
	Mode           string
	UpdatePubMap   bool
	UpdateInterval int
	UpdateStamp    int
}

func ValidateMode(mode string) error {
	switch mode {
	case ModeRedis, ModeSpread, ModeAll, ModeRoutes:
		return nil
	default:
		return fmt.Errorf("unknown cache mode %q; expected redis, spread, all, or routes", mode)
	}
}

func Run(ctx context.Context, c *dsp.Config, redis radix.Client, db *sql.DB, nc *nats.Conn, opts Options) error {
	if err := ValidateMode(opts.Mode); err != nil {
		return err
	}
	if opts.UpdateInterval <= 0 {
		return fmt.Errorf("cache update interval must be positive")
	}
	if opts.Mode == ModeRoutes {
		return match.DBGetMiddlemanRoutesToRedis(ctx, redis, db)
	}
	if opts.Mode == ModeSpread || opts.Mode == ModeAll {
		if nc == nil {
			return fmt.Errorf("cache mode %q requires NATS", opts.Mode)
		}
		if c.Spread == "" {
			return fmt.Errorf("cache mode %q requires spread directory", opts.Mode)
		}
		if err := os.MkdirAll(c.Spread, 0755); err != nil {
			return err
		}
	}

	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
		return err
	}

	pubmap, err := acl.DBGetPubMap(db)
	if err != nil {
		return err
	}

	if opts.UpdatePubMap {
		if err := UpdatePubMap(c, db, pubmap, opts.UpdateInterval, opts.UpdateStamp); err != nil {
			return err
		}
	}

	switch opts.Mode {
	case ModeSpread:
		return WriteToSpread(ctx, nc, db, pubmap, sizeIDs)
	case ModeRedis:
		return WriteToRedis(ctx, redis, db, pubmap, sizeIDs)
	case ModeAll:
		if err := WriteToSpread(ctx, nc, db, pubmap, sizeIDs); err != nil {
			return err
		}
		return WriteToRedis(ctx, redis, db, pubmap, sizeIDs)
	default:
		return ValidateMode(opts.Mode)
	}
}

func Read(ctx context.Context, out io.Writer, c *dsp.Config, redis radix.Client, db *sql.DB, mode string) error {
	if err := ValidateMode(mode); err != nil {
		return err
	}
	if mode == ModeRoutes {
		return RedisRoutesRead(ctx, out, redis)
	}
	sizeIDs, err := match.DBGetActiveCreativeSizeIDs(ctx, db)
	if err != nil {
		return err
	}
	switch mode {
	case ModeSpread:
		return SpreadRead(out, c.Spread, sizeIDs)
	case ModeRedis:
		return RedisRead(ctx, out, redis, sizeIDs)
	case ModeAll:
		if err := SpreadRead(out, c.Spread, sizeIDs); err != nil {
			return err
		}
		return RedisRead(ctx, out, redis, sizeIDs)
	case ModeRoutes:
		return RedisRoutesRead(ctx, out, redis)
	default:
		return ValidateMode(mode)
	}
}

func RedisRoutesRead(ctx context.Context, out io.Writer, redis radix.Client) error {
	middlemanRoutes, err := match.MiddlemanRouteCacheFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(middlemanRoutes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Middleman routes:\n%s\n", bs)
	return nil
}

func RedisRead(ctx context.Context, out io.Writer, redis radix.Client, sizeIDs []uint32) error {
	pubmap, err := acl.PubMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "pubmap:\n%s\n", bs)

	for _, sizeID := range sizeIDs {
		width, height := match.SizeID1To2(sizeID)
		hash, err := match.RAdvsFromRedisBySizeID(ctx, redis, sizeID)
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "\nRAdvs for sizeID %d x %d:\n%s\n", width, height, bs)
	}

	audiences, err := match.AudienceMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nAudiences:\n%s\n", bs)

	creatives, err := match.CreativeMapFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nCreatives:\n%s\n", bs)

	middlemanRoutes, err := match.MiddlemanRouteCacheFromRedis(ctx, redis)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(middlemanRoutes, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nMiddleman routes:\n%s\n", bs)
	return nil
}

func WriteToRedis(ctx context.Context, redis radix.Client, db *sql.DB, pubmap acl.PubMap, sizeIDs []uint32) error {
	if err := cleanupRedisShadowCaches(ctx, redis); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = cleanupRedisShadowCaches(context.Background(), redis)
		}
	}()

	if err := pubmap.ToRedisKeys(ctx, redis, acl.HashNamePubmap+redisShadowSuffix, acl.HashNamePubByID+redisShadowSuffix); err != nil {
		return err
	}

	sink := match.RedisCacheSink{Client: redis, KeySuffix: redisShadowSuffix}
	for _, sizeID := range sizeIDs {
		if err := match.DBGetRAdvsToRedisSpreadBySizeID(ctx, sink, db, sizeID); err != nil {
			return err
		}
	}

	if err := match.DBGetAudiencesToCache(ctx, sink, db); err != nil {
		return err
	}

	if err := match.DBGetCreativesToRedisSpread(ctx, sink, db); err != nil {
		return err
	}

	if err := match.DBGetMiddlemanRoutesToRedisKeys(ctx, redis, db, match.HashNameMiddlemanRoutes+redisShadowSuffix, match.HashNameMiddlemanRoutesV2+redisShadowSuffix); err != nil {
		return err
	}
	if err := swapRedisStaticCaches(ctx, redis, sizeIDs); err != nil {
		return err
	}
	committed = true
	return cleanupRedisShadowCaches(ctx, redis)
}

func cleanupRedisShadowCaches(ctx context.Context, redis radix.Client) error {
	keys := []string{
		acl.HashNamePubmap + redisShadowSuffix,
		acl.HashNamePubByID + redisShadowSuffix,
		match.HashNameAudience + redisShadowSuffix,
		match.HashNameCreative + redisShadowSuffix,
		match.HashNameMiddlemanRoutes + redisShadowSuffix,
		match.HashNameMiddlemanRoutesV2 + redisShadowSuffix,
	}
	if err := redis.Do(ctx, radix.Cmd(nil, "DEL", keys...)); err != nil {
		return err
	}
	return DeleteRedisKeysByPattern(ctx, redis, match.HashNameSlot+":*"+redisShadowSuffix)
}

func swapRedisStaticCaches(ctx context.Context, redis radix.Client, sizeIDs []uint32) error {
	type keyPair struct {
		live   string
		shadow string
		exists bool
	}
	pairs := []keyPair{
		{live: acl.HashNamePubmap, shadow: acl.HashNamePubmap + redisShadowSuffix},
		{live: acl.HashNamePubByID, shadow: acl.HashNamePubByID + redisShadowSuffix},
		{live: match.HashNameAudience, shadow: match.HashNameAudience + redisShadowSuffix},
		{live: match.HashNameCreative, shadow: match.HashNameCreative + redisShadowSuffix},
		{live: match.HashNameMiddlemanRoutes, shadow: match.HashNameMiddlemanRoutes + redisShadowSuffix},
		{live: match.HashNameMiddlemanRoutesV2, shadow: match.HashNameMiddlemanRoutesV2 + redisShadowSuffix},
	}
	newSlots := make(map[string]struct{})
	for _, sizeID := range sizeIDs {
		live := match.HashNameRAdvs(sizeID)
		if _, ok := newSlots[live]; ok {
			continue
		}
		newSlots[live] = struct{}{}
		pairs = append(pairs, keyPair{live: live, shadow: live + redisShadowSuffix})
	}
	for i := range pairs {
		var exists int
		if err := redis.Do(ctx, radix.Cmd(&exists, "EXISTS", pairs[i].shadow)); err != nil {
			return err
		}
		pairs[i].exists = exists != 0
	}
	liveSlots, err := redisLiveSlotKeys(ctx, redis)
	if err != nil {
		return err
	}
	return redis.Do(ctx, radix.WithConn("", func(ctx context.Context, conn radix.Conn) (err error) {
		if err = conn.Do(ctx, radix.Cmd(nil, "MULTI")); err != nil {
			return err
		}
		inMulti := true
		defer func() {
			if err != nil && inMulti {
				_ = conn.Do(ctx, radix.Cmd(nil, "DISCARD"))
			}
		}()
		for _, pair := range pairs {
			if pair.exists {
				err = conn.Do(ctx, radix.Cmd(nil, "RENAME", pair.shadow, pair.live))
			} else {
				err = conn.Do(ctx, radix.Cmd(nil, "DEL", pair.live))
			}
			if err != nil {
				return err
			}
		}
		for _, key := range liveSlots {
			if _, ok := newSlots[key]; ok {
				continue
			}
			if err = conn.Do(ctx, radix.Cmd(nil, "DEL", key)); err != nil {
				return err
			}
		}
		err = conn.Do(ctx, radix.Cmd(nil, "EXEC"))
		inMulti = false
		return err
	}))
}

func redisLiveSlotKeys(ctx context.Context, redis radix.Client) ([]string, error) {
	var keys []string
	scanner := (radix.ScannerConfig{Command: "SCAN", Pattern: match.HashNameSlot + ":*", Count: 256}).New(redis)
	for {
		var key string
		if !scanner.Next(ctx, &key) {
			break
		}
		suffix := strings.TrimPrefix(key, match.HashNameSlot+":")
		if _, err := strconv.ParseUint(suffix, 10, 32); err == nil {
			keys = append(keys, key)
		}
	}
	if err := scanner.Close(); err != nil {
		return nil, err
	}
	sort.Strings(keys)
	return keys, nil
}

func SpreadRead(out io.Writer, top string, sizeIDs []uint32) error {
	pubmap, err := acl.PubMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err := json.MarshalIndent(pubmap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "pubmap:\n%s\n", bs)

	for _, sizeID := range sizeIDs {
		width, height := match.SizeID1To2(sizeID)
		hash, err := match.RAdvsFromIOBySizeID(top, sizeID)
		if err != nil {
			return err
		}
		bs, err = json.MarshalIndent(hash, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "\nRAdvs for sizeID %d x %d:\n%s\n", width, height, bs)
	}

	audiences, err := match.AudienceMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(audiences, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nAudiences:\n%s\n", bs)

	creatives, err := match.CreativeMapFromIO(top)
	if err != nil {
		return err
	}
	bs, err = json.MarshalIndent(creatives, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nCreatives:\n%s\n", bs)
	return nil
}

func WriteToSpread(ctx context.Context, nc *nats.Conn, db *sql.DB, pubmap acl.PubMap, sizeIDs []uint32) error {
	if nc == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	for _, family := range []string{acl.HashNamePubmap, match.HashNameAudience, match.HashNameCreative, match.HashNameSlot} {
		if err := PublishSpreadReset(nc, family); err != nil {
			return err
		}
	}

	if err := pubmap.ToSpread(nc); err != nil {
		return err
	}

	for _, sizeID := range sizeIDs {
		if err := match.DBGetRAdvsToRedisSpreadBySizeID(ctx, nc, db, sizeID); err != nil {
			return err
		}
	}

	if err := match.DBGetAudiencesToSpread(nc, db); err != nil {
		return err
	}

	return match.DBGetCreativesToRedisSpread(ctx, nc, db)
}

func ResetRedisStaticCaches(ctx context.Context, redis radix.Client) error {
	for _, name := range []string{acl.HashNamePubmap, acl.HashNamePubByID, match.HashNameAudience, match.HashNameCreative, match.HashNameMiddlemanRoutes, match.HashNameMiddlemanRoutesV2} {
		if err := redis.Do(ctx, radix.Cmd(nil, "DEL", name)); err != nil {
			return err
		}
	}
	return DeleteRedisKeysByPattern(ctx, redis, match.HashNameSlot+":*")
}

func DeleteRedisKeysByPattern(ctx context.Context, redis radix.Client, pattern string) error {
	const chunk = 256
	var keys []string
	scanner := (radix.ScannerConfig{Command: "SCAN", Pattern: pattern, Count: chunk}).New(redis)
	for {
		var key string
		if !scanner.Next(ctx, &key) {
			break
		}
		keys = append(keys, key)
		if len(keys) == chunk {
			if err := redis.Do(ctx, radix.Cmd(nil, "DEL", keys...)); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	if err := scanner.Close(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return redis.Do(ctx, radix.Cmd(nil, "DEL", keys...))
	}
	return nil
}

func PublishSpreadReset(nc *nats.Conn, family string) error {
	return nc.Publish(family+":__reset__", nil)
}

func UpdatePubMap(c *dsp.Config, db *sql.DB, pubmap acl.PubMap, interval, stamp int) error {
	if interval <= 0 {
		return fmt.Errorf("cache update interval must be positive")
	}
	var stampObject *dsp.Stamp
	if stamp > 0 {
		stampObject = dsp.NewStamp(interval, stamp)
	} else {
		stampObject = dsp.NewStamp(interval)
	}
	fn := c.NewLogfileName(dsp.SUBJECTAttribute, stampObject)
	fh, err := os.Open(fn)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer fh.Close()

	scanner := newAttributeLogScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		plus := new(match.AttributePlus)
		if err := json.Unmarshal([]byte(line), plus); err != nil {
			return err
		}
		acl := plus.Attribute.ACL
		siteType := "Web"
		if plus.Attribute.IsApp {
			siteType = "App"
		}
		pub, err := pubmap.DBAddNew(db, acl.PubStr, acl.SiteStr, siteType, acl.SlotStr)
		if err != nil {
			return err
		}
		pubmap[acl.PubStr] = pub
	}
	return scanner.Err()
}

func newAttributeLogScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxAttributeLogLineBytes+1)
	return scanner
}

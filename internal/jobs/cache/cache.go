package cache

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
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
	if err := ResetRedisStaticCaches(ctx, redis); err != nil {
		return err
	}

	if err := pubmap.ToRedis(ctx, redis); err != nil {
		return err
	}

	for _, sizeID := range sizeIDs {
		if err := match.DBGetRAdvsToRedisSpreadBySizeID(ctx, redis, db, sizeID); err != nil {
			return err
		}
	}

	if err := match.DBGetAudiencesToRedis(ctx, redis, db); err != nil {
		return err
	}

	if err := match.DBGetCreativesToRedisSpread(ctx, redis, db); err != nil {
		return err
	}

	return match.DBGetMiddlemanRoutesToRedis(ctx, redis, db)
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

	scanner := bufio.NewScanner(fh)
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

// Package adminapi exposes the narrow Aofei surface used by the sibling
// pzdesign Summer admin module.
package adminapi

import (
	"context"
	"database/sql"
	"net/url"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/advice"
	"github.com/guruperl/aofei/demo"
	"github.com/guruperl/aofei/dh"
	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/uploaded"
	"github.com/mediocregopher/radix/v4"
)

type RPub = match.RPub
type Audience = match.Audience

var (
	String2CAT                = acl.String2CAT
	CAT2String                = acl.CAT2String
	HashNameMiddlemanRoutesV2 = match.HashNameMiddlemanRoutesV2
)

func AddPub(db *sql.DB, domain string) (*acl.Pub, error) {
	return acl.AddPub(db, domain)
}

func DBGetPubByID(db *sql.DB, pubID string) (*acl.Pub, string, error) {
	return acl.DBGetPubByID(db, pubID)
}

func SizeID2To1(w, h uint16) uint32 {
	return match.SizeID2To1(w, h)
}

func SizeID1To2(sizeID uint32) (uint16, uint16) {
	return match.SizeID1To2(sizeID)
}

func UAResetArgs(args url.Values) {
	advice.UAResetArgs(args)
}

func DemoResetArgs(args url.Values) {
	demo.DemoResetArgs(args)
}

func DHResetArgs(args url.Values) {
	dh.DHResetArgs(args)
}

func UploadedResetArgs(args url.Values) {
	uploaded.UploadedResetArgs(args)
}

func NewDHAudience() *dh.DHAudience {
	return new(dh.DHAudience)
}

func NewDemoAudience() *demo.DemoAudience {
	return new(demo.DemoAudience)
}

func NewUaAudience() *advice.UaAudience {
	return new(advice.UaAudience)
}

func NewUploadAudience() *uploaded.UploadAudience {
	return new(uploaded.UploadAudience)
}

func DBGetCreativesToRedisSpread(ctx context.Context, conn any, db *sql.DB, extra ...string) error {
	return match.DBGetCreativesToRedisSpread(ctx, conn, db, extra...)
}

func DBGetAudience(db *sql.DB, itemID uint32) (*Audience, error) {
	return match.DBGetAudience(db, itemID)
}

func DBGetRAdvsToRedisSpreadByItemID(ctx context.Context, conn any, db *sql.DB, itemID uint32, top ...string) error {
	return match.DBGetRAdvsToRedisSpreadByItemID(ctx, conn, db, itemID, top...)
}

func DBDeleteRAdvsToRedisSpreadByItemID(ctx context.Context, conn any, db *sql.DB, itemID uint32, top ...string) error {
	return match.DBDeleteRAdvsToRedisSpreadByItemID(ctx, conn, db, itemID, top...)
}

func UploadSingle(ctx context.Context, redis radix.Client, advID uint32, marker, line string) error {
	return uploaded.UploadSingle(ctx, redis, advID, marker, line)
}

type MiddlemanRouteCacheMetadata = match.MiddlemanRouteCacheMetadata
type MiddlemanRouteCache = match.MiddlemanRouteCache

func MiddlemanRouteCacheFromRedis(ctx context.Context, redis radix.Client) (*MiddlemanRouteCache, error) {
	return match.MiddlemanRouteCacheFromRedis(ctx, redis)
}

func DBGetMiddlemanRouteHighWater(ctx context.Context, db *sql.DB) (string, error) {
	return match.DBGetMiddlemanRouteHighWater(ctx, db)
}

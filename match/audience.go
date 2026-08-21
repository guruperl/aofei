// Package match provides functionality for handling audience matching and database operations.
package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/advice"
	"github.com/guruperl/aofei/demo"
	"github.com/guruperl/aofei/dh"
	"github.com/guruperl/aofei/maxmind"
	"github.com/guruperl/aofei/uploaded"
	"github.com/nats-io/nats.go"

	"github.com/mediocregopher/radix/v4"
)

type Audience struct {
	*dh.DHAudience
	*acl.ACLAudience
	*maxmind.GeoAudience
	*demo.DemoAudience
	*advice.UaAudience
	*uploaded.UploadAudience
}

func (self *Audience) Has(attr *Attribute) bool {
	if self == nil || attr == nil {
		return false
	}

	d := attr.Demo
	g := attr.Geo
	u := attr.PzUa
	dh := attr.DH
	a := attr.ACL

	if self.GeoAudience != nil && !self.GeoAudience.Has(g) {
		log.Printf("geo audience not match")
		return false
	}
	if self.DemoAudience != nil && !self.DemoAudience.Has(d) {
		log.Printf("demo audience not match")
		return false
	}
	if self.UaAudience != nil && !self.UaAudience.Has(u) {
		log.Printf("ua audience not match")
		return false
	}
	if self.DHAudience != nil && !self.DHAudience.Has(dh) {
		log.Printf("dh audience not match")
		return false
	}
	if self.ACLAudience != nil && !self.ACLAudience.Has(a) {
		log.Printf("acl audience not match")
		return false
	}

	return true
}

// Pack serializes the audience into a byte slice.
func (self *Audience) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := writeCachePayloadHeader(buf, cachePayloadKindAudience, cachePayloadVersionAudience); err != nil {
		return nil, err
	}
	err := gob.NewEncoder(buf).Encode(self)
	return buf.Bytes(), err
}

// PackIO packs the audience into an IO writer.
func (self *Audience) PackIO(w *bytes.Buffer) error {
	if err := writeCachePayloadHeader(w, cachePayloadKindAudience, cachePayloadVersionAudience); err != nil {
		return err
	}
	return gob.NewEncoder(w).Encode(self)
}

// UnpackAudience deserializes the audience from a byte slice.
func UnpackAudience(data []byte) (*Audience, error) {
	var err error
	data, err = unpackCachePayload(data, cachePayloadKindAudience, cachePayloadVersionAudience)
	if err != nil {
		return nil, err
	}
	audience := new(Audience)
	buf := bytes.NewReader(data)
	err = gob.NewDecoder(buf).Decode(audience)
	return audience, err
}

// UnpackAudienceIO deserializes the audience from an IO reader.
func UnpackAudienceIO(r io.Reader) (*Audience, error) {
	data, err := readAllCachePayload(r, cachePayloadKindAudience, cachePayloadVersionAudience)
	if err != nil {
		return nil, err
	}
	audience := new(Audience)
	err = gob.NewDecoder(bytes.NewReader(data)).Decode(audience)
	return audience, err
}

// getItemIDs retrieves item IDs from the database for active audiences.
func getItemIDs(db *sql.DB) ([]uint32, error) {
	rows, err := db.Query(`
	SELECT item_id
	FROM adv_item
	INNER JOIN adv_campaign USING (campaign_id)
	INNER JOIN adv USING (adv_id)
	WHERE adv_item.active="Yes"
	AND adv_campaign.active="Yes" 
	AND adv.active="Yes"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itemIDs []uint32
	for rows.Next() {
		var itemID uint32
		err = rows.Scan(&itemID)
		if err != nil {
			return nil, err
		}
		itemIDs = append(itemIDs, itemID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return itemIDs, nil
}

// DBGetAudiencesToRedis retrieves audiences from the database and inserts them into Redis.
func DBGetAudiencesToRedis(ctx context.Context, conn radix.Client, db *sql.DB) error {
	return DBGetAudiencesToCache(ctx, RedisCacheSink{Client: conn}, db)
}

// DBGetAudiencesToCache writes database audiences through the supplied cache sink.
func DBGetAudiencesToCache(ctx context.Context, sink CacheSink, db *sql.DB) error {
	itemIDs, err := getItemIDs(db)
	if err != nil {
		return fmt.Errorf("failed to get item IDs: %w", err)
	}

	for _, itemID := range itemIDs {
		aud, err := DBGetAudience(db, itemID)
		if err == nil {
			if aud == nil {
				continue
			}
			var data []byte
			data, err = aud.Pack()
			if err == nil {
				err = sink.PutAudience(ctx, itemID, data)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// DBGetAudiencesToSpread retrieves audiences from the database and publishes them to nats.
func DBGetAudiencesToSpread(conn *nats.Conn, db *sql.DB) error {
	itemIDs, err := getItemIDs(db)
	if err != nil {
		return fmt.Errorf("failed to get item IDs: %w", err)
	}

	for _, itemID := range itemIDs {
		aud, err := DBGetAudience(db, itemID)
		if err == nil {
			err = aud.ToSpread(conn, itemID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func DBGetAudience(db *sql.DB, itemID uint32) (*Audience, error) {
	a, err := acl.DBGetACLAudience(db, itemID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT tv.value_id, an.attrname_id, an.attrname, av.attrvalue_id
FROM adv_targetname tn
INNER JOIN adv_targetvalue tv USING (targetname_id)
INNER JOIN adv_attrname an USING (attrname_id)
LEFT JOIN adv_attrvalue av ON (an.attrname_id=av.attrname_id AND tv.value_id=av.attrvalue_id)
WHERE tn.item_id=?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idh, igeo, idemo, iua, iupload int
	dhA := new(dh.DHAudience)
	geoA := new(maxmind.GeoAudience)
	demoA := new(demo.DemoAudience)
	uaA := new(advice.UaAudience)
	uploadA := new(uploaded.UploadAudience)
	for rows.Next() {
		var valueID, attrnameID uint32
		var attrvalueID sql.NullInt64
		var attrname string
		err = rows.Scan(&valueID, &attrnameID, &attrname, &attrvalueID)
		if err != nil {
			return nil, err
		}

		idh += dhA.DBFillDhAudience(attrname, valueID)
		igeo += geoA.DBFillGeoAudience(attrname, valueID)
		idemo += demoA.DBFillDemoAudience(attrname, valueID)
		iua += uaA.DBFillUaAudience(attrname, valueID)
		iupload += uploadA.DBFillUploadAudience(attrname, valueID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if a == nil && idh == 0 && igeo == 0 && idemo == 0 && iua == 0 && iupload == 0 {
		return nil, nil
	}

	return &Audience{
		DHAudience:     dhA,
		GeoAudience:    geoA,
		DemoAudience:   demoA,
		UaAudience:     uaA,
		ACLAudience:    a,
		UploadAudience: uploadA,
	}, nil
}

const (
	HashNameAudience = "audience"
)

// ToRedis inserts audience into Redis.
func (self *Audience) ToRedis(ctx context.Context, conn radix.Client, itemID uint32) error {
	bs, err := self.Pack()
	if err == nil {
		err = RedisCacheSink{Client: conn}.PutAudience(ctx, itemID, bs)
	}
	return err
}

// ToSpread publishes the audience to NATS.
func (self *Audience) ToSpread(conn *nats.Conn, itemID uint32) error {
	bs, err := self.Pack()
	if err != nil {
		return err
	}
	return SpreadCacheSink{Conn: conn}.PutAudience(context.Background(), itemID, bs)
}

// AudienceFromRedis retrieves audience data from Redis.
func AudienceFromRedis(ctx context.Context, conn radix.Client, itemID uint32) (*Audience, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNameAudience, strconv.FormatUint(uint64(itemID), 10)))
	if err != nil {
		return nil, err
	}
	return UnpackAudience(bs)
}

// AudienceFromIO retrieves audience data from IO.
func AudienceFromIO(top string, itemID uint32) (*Audience, error) {
	fh, err := os.Open(fmt.Sprintf("%s/%s/%d", top, HashNameAudience, itemID))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return UnpackAudienceIO(fh)
}

// AudienceMapFromRedis retrieves all multiple audience data by itemID from Redis
func AudienceMapFromRedis(ctx context.Context, conn radix.Client) (map[uint32]*Audience, error) {
	var arr []string
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", HashNameAudience))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil // No audiences found in Redis
	}
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows // Invalid format
	}

	// Decode each key-value pair into the Audiences map
	audiences := make(map[uint32]*Audience)
	for i := 0; i < len(arr); i += 2 {
		itemID, err := strconv.ParseUint(arr[i], 10, 32)
		if err != nil {
			return nil, err
		}
		data := []byte(arr[i+1])
		audience, err := UnpackAudience(data)
		if err != nil {
			return nil, err
		}
		audiences[uint32(itemID)] = audience
	}
	return audiences, nil
}

// AudienceMapFromIO retrieves all multiple audience data by itemID from IO
func AudienceMapFromIO(top string) (map[uint32]*Audience, error) {
	files, err := os.ReadDir(top + "/" + HashNameAudience)
	if err != nil {
		return nil, err
	}
	hash := make(map[uint32]*Audience)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		itemID, err := strconv.ParseUint(file.Name(), 10, 32)
		if err != nil {
			return nil, err
		}
		audience, err := AudienceFromIO(top, uint32(itemID))
		if err != nil {
			return nil, err
		}
		hash[uint32(itemID)] = audience
	}
	return hash, nil
}

type Audiences []*Audience

// AudiencesFromRedis retrieves multiple audience data from Redis for specified RAdvs
func (self RAdvs) AudiencesFromRedis(ctx context.Context, conn radix.Client) (Audiences, error) {
	arr := []string{HashNameAudience}
	for _, radv := range self {
		arr = append(arr, fmt.Sprintf("%d", radv.ItemID))
	}
	data := make([]string, len(self))
	err := conn.Do(ctx, radix.Cmd(&data, "HMGET", arr...))
	if err != nil {
		return nil, err
	}

	audiences := make([]*Audience, len(self))
	for i, d := range data {
		if len(d) == 0 {
			continue
		}
		audience, err := UnpackAudience([]byte(d))
		if err != nil {
			return nil, err
		}
		audiences[i] = audience
	}
	return audiences, nil
}

// AudiencesFromIO retrieves multiple audience data from Redis for specified RAdvs
func (self RAdvs) AudiencesFromIO(top string) (Audiences, error) {
	audiences := make([]*Audience, len(self))
	for i, radv := range self {
		audience, err := AudienceFromIO(top, radv.ItemID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		audiences[i] = audience
	}
	return audiences, nil
}

// Match checks if the audiences match the given attribute and returns a slice of booleans.
func (self Audiences) Match(attr *Attribute) []bool {
	matches := make([]bool, len(self))
	for i, a := range self {
		if a != nil {
			matches[i] = a.Has(attr)
		} else {
			matches[i] = true
		}
	}
	return matches
}

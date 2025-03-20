// Package match provides functionality for handling audience matching and database operations.
package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"strconv"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dh"
	"github.com/genelet/winter/maxmind"

	"github.com/mediocregopher/radix/v4"
)

type Audience struct {
	*dh.DHAudience
	*acl.ACLAudience
	*maxmind.GeoAudience
	*demo.DemoAudience
	*advice.UaAudience
}

func (self *Audience) Has(attr *Attribute) bool {
	d := attr.Demo
	g := attr.Geo
	u := attr.PzUa
	dh := attr.DH
	a := attr.ACL

	if self.GeoAudience != nil && !self.GeoAudience.Has(g) {
		return false
	}
	if self.DemoAudience != nil && !self.DemoAudience.Has(d) {
		return false
	}
	if self.UaAudience != nil && !self.UaAudience.Has(u) {
		return false
	}
	if self.DHAudience != nil && !self.DHAudience.Has(dh) {
		return false
	}
	if self.ACLAudience != nil && !self.ACLAudience.Has(a) {
		return false
	}

	return true
}

// Pack serializes the audience into a byte slice.
func (self *Audience) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(self)
	return buf.Bytes(), err
}

// UnpackAudience deserializes the audience from a byte slice.
func UnpackAudience(data []byte) (*Audience, error) {
	audience := new(Audience)
	buf := bytes.NewReader(data)
	err := gob.NewDecoder(buf).Decode(audience)
	return audience, err
}

// DBGetAudiencesToRedis retrieves audiences from the database and inserts them into Redis.
func DBGetAudiencesToRedis(ctx context.Context, conn radix.Client, db *sql.DB) error {
	rows, err := db.Query(`
SELECT item_id
FROM adv_item
INNER JOIN adv_campaign USING (campaign_id)
INNER JOIN adv USING (adv_id)
WHERE adv_item.active="Yes"
AND adv_campaign.active="Yes" 
AND adv.active="Yes"`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var itemIDs []uint32
	for rows.Next() {
		var itemID uint32
		err = rows.Scan(&itemID)
		if err != nil {
			return err
		}
		itemIDs = append(itemIDs, itemID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, itemID := range itemIDs {
		aud, err := DBGetAudience(db, itemID)
		if err == nil {
			err = aud.ToRedis(ctx, conn, itemID)
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

	var idh, igeo, idemo, iua int
	dhA := new(dh.DHAudience)
	geoA := new(maxmind.GeoAudience)
	demoA := new(demo.DemoAudience)
	uaA := new(advice.UaAudience)
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
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if a == nil && idh == 0 && igeo == 0 && idemo == 0 && iua == 0 {
		return nil, nil
	}

	return &Audience{DHAudience: dhA, GeoAudience: geoA, DemoAudience: demoA, UaAudience: uaA, ACLAudience: a}, nil
}

const (
	HashNameAudience = "audience"
)

// ToRedis inserts audience into Redis.
func (self *Audience) ToRedis(ctx context.Context, conn radix.Client, itemID uint32) error {
	bs, err := self.Pack()
	if err == nil {
		err = conn.Do(ctx, radix.Cmd(nil, "HSET", HashNameAudience, strconv.FormatUint(uint64(itemID), 10), string(bs)))
	}
	return err
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

type Audiences []*Audience

// AudiencesFromRedis retrieves multiple audience data from Redis.
func AudiencesFromRedis(ctx context.Context, conn radix.Client, itemIDs []string) (Audiences, error) {
	data := make([]string, len(itemIDs))
	arr := append([]string{HashNameAudience}, itemIDs...)
	err := conn.Do(ctx, radix.Cmd(&data, "HMGET", arr...))
	if err != nil {
		return nil, err
	}

	audiences := make([]*Audience, len(itemIDs))
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

func (self Audiences) Match(attr *Attribute) []bool {
	matches := make([]bool, len(self))
	for i := range self {
		//for i, a := range self {
		//		if a != nil {
		//			matches[i] = a.Has(attr)
		//		} else {
		matches[i] = true
		//		}
	}
	return matches
}

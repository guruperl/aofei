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

// AudieceFromRedis retrieves audience data from Redis.
func AudienceFromRedis(ctx context.Context, conn radix.Client, itemID uint32) (*Audience, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNameAudience, strconv.FormatUint(uint64(itemID), 10)))
	if err != nil {
		return nil, err
	}
	return UnpackAudience(bs)
}

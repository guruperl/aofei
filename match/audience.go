// Package match provides functionality for handling audience matching and database operations.
package match

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dh"
	"github.com/genelet/winter/maxmind"
	"github.com/genelet/winter/pzutil"

	"github.com/mediocregopher/radix/v4"
)

type Audience struct {
	*dh.DHAudience
	*CategoryAudience
	*maxmind.GeoAudience
	*demo.DemoAudience
	*advice.UaAudience
}

func (self *Audience) Pack() ([]byte, error) {
	return pzutil.PackObject(self)
}

func UnpackAudience(data []byte) (*Audience, error) {
	audience := new(Audience)
	err := pzutil.UnpackObject(data, audience)
	return audience, err
}

func DBGetAudience(db *sql.DB, itemID uint32) (*Audience, error) {
	var output *Audience
	cat, err := DBGetCategoryAudience(db, itemID)
	if err != nil {
		return nil, err
	}
	if cat != nil {
		output = &Audience{
			CategoryAudience: cat,
		}
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

	rows, err = db.Query(`
SELECT c.channel_name
FROM ch_belong b
INNER JOIN def_channel c USING channel_id
INNER JOIN adv_item i ON (b.entitytype_id=41 AND b.entity_id=i.campaign_id)
WHERE i.item_id=?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var owns []string
	for rows.Next() {
		var channelName string
		err = rows.Scan(&channelName)
		if err != nil {
			return nil, err
		}
		owns = append(owns, channelName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	var channelOrder string
	err = db.QueryRow(`
SELECT channel_order
FROM adv_item
WHERE item_id=?`, itemID).Scan(&channelOrder)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	rows, err = db.Query(`
SELECT c.channel_name
FROM ch_ac a
INNER JOIN def_channel c USING channel_id
INNER JOIN adv_item i ON (a.entitytype_id=42 AND a.entity_id=i.item_id)
WHERE i.item_id=?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var channelName string
		err = rows.Scan(&channelName)
		if err != nil {
			return nil, err
		}
		cats = append(cats, channelName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if output == nil {
		if idh == 0 && igeo == 0 && idemo == 0 && iua == 0 {
			return nil, nil
		} else {
			output = new(Audience)
		}
	}

	if idh > 0 {
		output.DHAudience = dhA
	}
	if igeo > 0 {
		output.GeoAudience = geoA
	}
	if idemo > 0 {
		output.DemoAudience = demoA
	}
	if iua > 0 {
		output.UaAudience = uaA
	}

	return output, nil
}

// ToRedis inserts audience data into Redis.
func (self *Audience) ToRedis(ctx context.Context, conn radix.Client, itemID uint32, data []byte) error {
	bs, err := self.Pack()
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "SET", "audience:"+strconv.FormatUint(uint64(itemID), 10), string(bs)))
}

// FromRedis retrieves audience data from Redis.
func FromRedis(ctx context.Context, conn radix.Client, itemID uint32) (*Audience, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "GET", "audience:"+strconv.FormatUint(uint64(itemID), 10)))
	if err != nil {
		return nil, err
	}
	return UnpackAudience(bs)
}

func (self *Audience) Has(attr *Attribute) bool {
	d := attr.Demo
	g := attr.Geo
	u := attr.PzUa
	dh := attr.DH
	white := attr.White
	black := attr.Black
	pub := attr.PubLevel
	site := attr.SiteLevel
	page := attr.PageLevel

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
	if self.CategoryAudience != nil && !self.CategoryAudience.Has(white, black, pub, site, page) {
		return false
	}

	return true
}

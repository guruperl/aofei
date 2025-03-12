// Package chac is for channel black and white lists
package chac

import (
	"fmt"
	"net/url"

	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Topics(extra ...url.Values) error {
	entitytypeID := extra[0].Get("entitytype_id")
	entityID := extra[0].Get("entity_id")
	level := extra[0].Get("level")
	ARGS := self.ARGS
	if entitytypeID == "" {
		entitytypeID = ARGS.Get("entitytype_id")
	}
	if entityID == "" {
		entityID = ARGS.Get("entity_id")
	}

	var err error
	switch entitytypeID {
	case "32":
		err = self.GetArgs(ARGS, `
SELECT channel_order FROM pub_slot WHERE slot_id=?`, entityID)
	case "42":
		err = self.GetArgs(ARGS, `
SELECT channel_order FROM adv_item WHERE item_id=?`, entityID)
	default:
		//return fmt.Errorf("wrong id %s of type %s", entityID, entitytypeID)
	}
	if err != nil {
		return err
	}

	// just to get a shorter list
	if level == "" {
		level = "1"
	}
	if level == "" {
		return self.SelectSQL(self.LISTS, `
SELECT c.channel_id, c.channel_name, b.chbelong_id, a.chac_id
FROM def_channel c
LEFT JOIN ch_belong b ON (c.channel_id=b.channel_id AND b.entitytype_id=? AND b.entity_id=?)
LEFT JOIN ch_ac     a ON (c.channel_id=a.channel_id AND a.entitytype_id=? AND a.entity_id=?)
`, entitytypeID, entityID, entitytypeID, entityID)
	}
	return self.SelectSQL(self.LISTS, `
SELECT c.channel_id, c.channel_name, b.chbelong_id, a.chac_id
FROM def_channel c
LEFT JOIN ch_belong b ON (c.channel_id=b.channel_id AND b.entitytype_id=? AND b.entity_id=?)
LEFT JOIN ch_ac     a ON (c.channel_id=a.channel_id AND a.entitytype_id=? AND a.entity_id=?)
WHERE c.level=?`, entitytypeID, entityID, entitytypeID, entityID, level)
}

func (self *Model) InsertBelong(extra ...url.Values) error {
	ARGS := self.ARGS
	if ARGS.Get("belong_ids") == "" {
		return nil
	}

	sql := `INSERT INTO ch_belong (entitytype_id, entity_id, channel_id) VALUES `
	n := 0
	for _, id := range ARGS["belong_ids"] {
		if pzutil.IsDigit(id) {
			n++
			sql += "(" + ARGS.Get("entitytype_id") + "," + ARGS.Get("entity_id") + "," + id + "),"
		}
	}
	if n == 0 {
		return nil
	}
	return self.DoSQL(sql[:len(sql)-1])
}

func (self *Model) InsertAc(extra ...url.Values) error {
	ARGS := self.ARGS
	if ARGS.Get("ac_ids") == "" {
		return nil
	}

	sql := `INSERT INTO ch_ac (entitytype_id, entity_id, channel_id) VALUES `
	n := 0
	for _, id := range ARGS["ac_ids"] {
		if pzutil.IsDigit(id) {
			n++
			sql += "(" + ARGS.Get("entitytype_id") + "," + ARGS.Get("entity_id") + "," + id + "),"
		}
	}
	if n == 0 {
		return nil
	}
	return self.DoSQL(sql[:len(sql)-1])
}

func (self *Model) Update(extra ...url.Values) error {
	ARGS := self.ARGS
	entitytypeID := ARGS.Get("entitytype_id")
	entityID := ARGS.Get("entity_id")
	channelOrder := ARGS.Get("channel_order")
	if (entitytypeID == "32" || entitytypeID == "42") && channelOrder == "" {
		return fmt.Errorf("channel_order is empty")
	}

	var err error
	if err = self.DoSQL(`
DELETE FROM ch_belong WHERE entitytype_id=? AND entity_id=?`, entitytypeID, entityID); err == nil {
		if err = self.DoSQL(`
DELETE FROM ch_ac WHERE entitytype_id=? AND entity_id=?`, entitytypeID, entityID); err == nil {
			if entitytypeID == "32" || entitytypeID == "42" {
				err = self.DoSQL(`
UPDATE `+ARGS.Get("table")+`
SET channel_order=? WHERE `+ARGS.Get("idname")+`=?`, channelOrder, entityID)
			}
			if err == nil {
				if err = self.InsertAc(extra...); err == nil {
					err = self.InsertBelong(extra...)
				}
			}
		}
	}
	return err
}

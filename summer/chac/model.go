// Package chac is for channel black and white lists
package chac

import (
	"net/url"

	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Topics(extra ...url.Values) error {
	entitytype_id := extra[0].Get("entitytype_id")
	entity_id := extra[0].Get("entity_id")
	level := extra[0].Get("level")
	ARGS := self.ARGS
	if entitytype_id == "" {
		entitytype_id = ARGS.Get("entitytype_id")
	}
	if entity_id == "" {
		entity_id = ARGS.Get("entity_id")
	}
	if level == "" {
		level = ARGS.Get("level")
		if level == "" {
			level = "1"
		}
	}

	var err error
	if entitytype_id == "32" {
		err = self.GetArgs(ARGS,
			`SELECT mychannel, channel_order FROM pub_slot WHERE slot_id=?`, entity_id)
	} else if entitytype_id == "31" {
		err = self.GetArgs(ARGS,
			`SELECT channel_order FROM pub_site WHERE site_id=?`, entity_id)
	} else {
		err = self.GetArgs(ARGS,
			`SELECT channel_order FROM adv_campaign WHERE campaign_id=?`, entity_id)
	}
	if err != nil {
		return err
	}

	return self.SelectSQL(self.LISTS,
		`SELECT c.channel_id, c.channel_name, b.chbelong_id, a.chac_id
FROM def_channel c
LEFT JOIN ch_belong b ON (c.channel_id=b.channel_id AND b.entitytype_id=? AND b.entity_id=?)
LEFT JOIN ch_ac     a ON (c.channel_id=a.channel_id AND a.entitytype_id=? AND a.entity_id=?)
WHERE c.level=?`, entitytype_id, entity_id, entitytype_id, entity_id, level)
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
	entitytype_id := ARGS.Get("entitytype_id")
	entity_id := ARGS.Get("entity_id")
	channel_order := ARGS.Get("channel_order")

	err := self.DoSQL(
		`DELETE FROM ch_belong WHERE entitytype_id=? AND entity_id=?`,
		entitytype_id, entity_id)
	if err == nil {
		err = self.DoSQL(
			`DELETE FROM ch_ac WHERE entitytype_id=? AND entity_id=?`,
			entitytype_id, entity_id)
	}
	if err != nil {
		return err
	}

	if entitytype_id == "32" {
		if channel_order == "" {
			channel_order = "Inherit"
		}
		mychannel := ARGS.Get("mychannel")
		if mychannel == "" {
			mychannel = "Inherit"
		}

		err = self.DoSQL(
			`UPDATE `+ARGS.Get("table")+` SET mychannel=?, channel_order=? WHERE `+ARGS.Get("idname")+`=?`, mychannel, channel_order, entity_id)
		if err != nil {
			return err
		}

		if mychannel == "Inherit" && channel_order == "Inherit" {
			return nil
		} else if mychannel == "Inherit" {
			return self.InsertAc(extra...)
		} else if channel_order == "Inherit" {
			return self.InsertBelong(extra...)
		} else {
			if err = self.InsertAc(extra...); err == nil {
				return self.InsertBelong(extra...)
			}
			return err
		}
	}

	if channel_order == "" {
		channel_order = "Black"
	}
	err = self.DoSQL(
		`UPDATE `+ARGS.Get("table")+` SET channel_order=? WHERE `+ARGS.Get("idname")+`=?`, channel_order, entity_id)
	if err != nil {
		return err
	}

	if err = self.InsertAc(extra...); err == nil {
		return self.InsertBelong(extra...)
	}
	return err
}

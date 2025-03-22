// Package item handles the item model.
package item

import (
	"errors"
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Review(extra ...url.Values) error {
	return self.DoSQL(
		`UPDATE adv_item SET active="New" WHERE active="Prepare" AND item_id=?`, self.ARGS.Get("item_id"))
}

func (self *Model) normalizeCreatives() error {
	ARGS := self.ARGS
	lists := make([]map[string]interface{}, 0)
	err := self.SelectSQL(&lists, `
SELECT creative_id, weight, size_id FROM adv_item 
WHERE active='Yes'
AND item_id=?`, ARGS.Get("item_id"))
	if err != nil {
		return err
	}
	ref := make(map[uint32]map[uint32]float32)
	for _, item := range lists {
		creative_id := uint32(item["creative_id"].(int64))
		size_id := uint32(item["size_id"].(int64))
		if _, ok := ref[size_id]; !ok {
			ref[size_id] = make(map[uint32]float32)
		}
		ref[size_id][creative_id] = float32(item["weight"].(int64))
	}
	for _, v := range ref {
		var total float32
		for _, w := range v {
			total += w
		}
		for id, w := range v {
			v[id] = w / total
			err = self.DoSQL(`
UPDATE adv_item SET weight=? WHERE creative_id=?`, v[id], id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *Model) Authen(extra ...url.Values) error {
	ARGS := self.ARGS
	err := self.normalizeCreatives()
	if err != nil {
		return err
	}
	if ARGS.Get("agent_level") == "1" {
		return self.DoSQL(
			`UPDATE adv_item SET active=? WHERE active="New" AND item_id=?`, ARGS.Get("active"), ARGS.Get("item_id"))
	}
	return self.DoSQL(
		`UPDATE adv_item SET active=? WHERE active IN ("Pass2", "New") AND item_id=?`, ARGS.Get("active"), ARGS.Get("item_id"))
}

func (self *Model) Insert(extra ...url.Values) error {
	ARGS := self.ARGS
	err := self.GetArgs(ARGS,
		`SELECT active AS item_active FROM adv_item WHERE item_id=?`, ARGS.Get("item_id"))
	if err != nil {
		return err
	}
	if summer.Grep([]string{"Pass2", "New", "Yes"}, ARGS.Get("item_active")) {
		return errors.New("in reviewing")
	}
	return self.Model.Insert(extra...)
}

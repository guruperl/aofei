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

func (self *Model) Authen(extra ...url.Values) error {
	ARGS := self.ARGS
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

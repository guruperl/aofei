// Package site provides a filter for the summer package.
package site

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action

	if ARGS.Get("_gadmin") != "1" && (action == "insert" || action == "update") {
		if ARGS.Get("active") != "" {
			ARGS.Del("active")
		}
	}

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	who := self.RoleValue

	if who == "pub" && action == "topics" {
		extra["active"] = []string{"New", "Yes"}
	} else if who == "admin" && action == "topics" {
		if pubID := ARGS.Get("pub_id"); pubID != "" {
			extra.Set("pub_id", pubID)
		}
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	//ARGS := self.R.Form
	action := self.Action
	//role  := self.RoleValue
	lists := *model.LISTS

	if action == "edit" {
		item := lists[0]
		summer.TranslateOne(item, "access_order", "access_order_g")
	}

	return nil
}

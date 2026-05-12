package dsp

import (
	"crypto/sha1"
	"encoding/hex"
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
	who := self.RoleValue
	if ARGS.Get("_gadmin") == "1" {
		who = "admin"
	}

	if who == "admin" && action == "insert" {
		h := sha1.New()
		h.Write([]byte(ARGS.Get("passwd") + ARGS.Get("email")))
		ARGS.Set("passwd", hex.EncodeToString(h.Sum(nil)))
	} else if who != "admin" && action == "update" {
		ARGS.Del("active")
	}

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	ARGS := self.R.Form
	if self.RoleValue == "admin" && self.Action == "insert" {
		return model.Randomid("mid_dsp", "dsp_id", 0, 16777216, 10)
	}
	if self.RoleValue == "dsp" {
		extra.Set("dsp_id", ARGS.Get("dsp_id"))
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	return self.Filter.After(&model.Model)
}

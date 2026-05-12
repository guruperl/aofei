package bidder

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

	if self.RoleValue != "admin" {
		ARGS := self.R.Form
		ARGS.Del("credential_ref")
		ARGS.Del("credential_status")
		ARGS.Del("active")
	}
	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	if self.RoleValue == "dsp" {
		dspID := self.R.Form.Get("dsp_id")
		extra.Set("dsp_id", dspID)
		if self.Action == "insert" {
			self.R.Form.Set("dsp_id", dspID)
			self.R.Form.Set("credential_status", "Missing")
			self.R.Form.Set("active", "No")
		}
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	return self.Filter.After(&model.Model)
}

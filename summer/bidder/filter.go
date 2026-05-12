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
		ARGS.Del("synthetic_campaign_id")
		ARGS.Del("synthetic_item_id")
		ARGS.Del("synthetic_creative_id")
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

	if self.RoleValue == "adv" {
		advID := self.R.Form.Get("adv_id")
		extra.Set("adv_id", advID)
		if self.Action == "insert" {
			self.R.Form.Set("adv_id", advID)
			self.R.Form.Set("credential_status", "Missing")
			self.R.Form.Set("active", "No")
		}
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}
	if self.RoleValue != "admin" {
		for _, item := range *model.LISTS {
			for _, field := range operatorFields {
				delete(item, field)
			}
		}
	}
	return nil
}

var operatorFields = []string{
	"synthetic_campaign_id",
	"synthetic_item_id",
	"synthetic_creative_id",
	"credential_ref",
	"credential_status",
	"active",
}

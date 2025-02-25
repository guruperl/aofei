package payment

import (
	"net/url"

	"github.com/genelet/winter/summer"
	// "github.com/golang/glog"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	// ARGS := self.R.Form
	// action := self.Action
	// who := self.RoleValue

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	//who := self.RoleValue
	if action == "topics" || action == "edit" {
		ARGS.Set("_gtable_saved", model.CurrentTable)
		model.CurrentTable = "view_payment"
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	who := self.RoleValue
	if action == "topics" || action == "edit" {
		model.CurrentTable = ARGS.Get("_gtable_saved")
		if who == "adv" {
			err := model.CallOnce(map[string]interface{}{"model": "adv", "action": "edit"})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

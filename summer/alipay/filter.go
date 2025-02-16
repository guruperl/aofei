package alipay

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

	//ARGS := self.R.Form
	//action := self.Action
	//who := self.Role_value

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	who := self.Role_value

	if who == "adv" && action == "topics" {
		extra.Set("adv_id", ARGS.Get("adv_id"))
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	//action := self.Action
	//who := self.Role_value
	//lists := *model.LISTS
	//other := *model.OTHER

	return nil
}

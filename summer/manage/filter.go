package manage

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

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	action := self.Action
	//who := self.RoleValue
	ARGS := self.R.Form
	//lists := *model.LISTS
	//other := *model.OTHER

	if action == "login_as" {
		role := ARGS.Get("role")
		uri := "/"
		if role == "adv" {
			uri = "/goto/adv/g/campaign?action=topics"
			return self.SetLoginAs(role, ARGS.Get("email"), uri, model.DB)
		} else if role == "pub" {
			uri = "/goto/pub/g/site?action=topics"
			return self.SetLoginAs(role, ARGS.Get("email"), uri, model.DB)
		} else if role == "agent" {
			uri = "/goto/agent/g/adv?action=topics"
			return self.SetLoginAs(role, ARGS.Get("login"), uri, model.DB)
		}
	}

	return nil
}

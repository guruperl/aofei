package manage

import (
	"net/url"
	"github.com/genelet/winter/summer"
)

type Filter struct {
    summer.Filter
}

func (self *Filter)Preset() error {
	if err := self.Filter.Preset(); err != nil { return err }

	return nil
}

func (self *Filter)Before(model *Model, extra url.Values, nextextra url.Values)  error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil { return err }

	return nil
}

func (self *Filter)After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil { return err }

	action := self.Action
	//who := self.Role_value
	ARGS := self.R.Form
	//lists := *model.LISTS
	//other := *model.OTHER

	if action=="login_as" {
		role := ARGS.Get("role")
		uri := "/"
		if role=="adv" {
			uri = "/goto/adv/e/campaign?action=topics"
			return self.Set_login_as(role, ARGS.Get("email"), uri, model.Db)
		} else if role=="pub" {
			uri = "/goto/pub/e/site?action=topics"
			return self.Set_login_as(role, ARGS.Get("email"), uri, model.Db)
		} else if role=="agent" {
			uri = "/goto/agent/e/adv?action=topics"
			return self.Set_login_as(role, ARGS.Get("login"), uri, model.Db)
		}
	}

	return nil
}

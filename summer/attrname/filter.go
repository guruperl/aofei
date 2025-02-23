package attrname

import (
	"net/url"
	"strings"

	"github.com/genelet/winter/summer"
	"github.com/golang/glog"
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
	//who := self.Role_value

	if action == "insert" {
		for _, v := range strings.Split(ARGS.Get("value"), ",") {
			ARGS.Add("values", strings.TrimSpace(v))
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
	who := self.Role_value

	if action == "insert" {
		err := model.Randomid("adv_attrname", "attrname_id", 10000, 16777216, 10)
		if err != nil {
			return err
		}
		glog.Infof("whole args 11111111: %#v", ARGS)
	} else if who == "adv" && action == "delete" {
		extra.Set("adv_id", ARGS.Get("adv_id"))
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	return self.Filter.After(&model.Model)
}

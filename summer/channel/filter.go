package channel

import (
	"net/url"
	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter)Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil { return err }

	ARGS := self.R.Form
	action := self.Action
//	who := self.Role_value

	if action=="topics" {
		if level := ARGS.Get("level"); level != "" {
			extra.Set("level", level)
		}
	}

	return nil
}

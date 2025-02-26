package adv

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Dashboard(extra ...url.Values) error {
	return self.Edit(extra...)
}

func (self *Model) Takedown(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.DoSQL("UPDATE adv SET active=? WHERE adv_id=?", ARGS.Get("active"), ARGS.Get("adv_id"))
}

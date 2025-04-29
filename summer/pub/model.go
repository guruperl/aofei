package pub

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
	return self.DoSQL(`
UPDATE pub SET active=? WHERE pub_id=?`, ARGS.Get("active"), ARGS.Get("pub_id"))
}

func (self *Model) Insert(extra ...url.Values) error {
	ARGS := self.ARGS
	if ARGS.Get("_gadmin") == "1" {
		return self.DoSQL(`
UPDATE pub SET total_balance_id=? WHERE pub_id=?`, ARGS.Get("total_balance_id"), ARGS.Get("pub_id"))
	}
	return self.Model.Insert(extra...)
}

package campaign

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Authen(extra ...url.Values) error {
	ARGS := self.ARGS
	if ARGS.Get("agent_level") == "1" {
		return self.DoSQL(
			`UPDATE adv_campaign SET active=? WHERE active="New" AND campaign_id=?`, ARGS.Get("active"), ARGS.Get("campaign_id"))
	}
	return self.DoSQL(
		`UPDATE adv_campaign SET active=? WHERE active IN ("Pass2", "New") AND campaign_id=?`, ARGS.Get("active"), ARGS.Get("campaign_id"))
}

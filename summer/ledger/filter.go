package ledger

import (
	"net/url"
	"time"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) Get_all() (map[string][]string, []string) {
	who := self.Role_value
	if who == "pub" {
		self.Fks = map[string][]string{"pub": {"pub_id", ""}}
	} else if who == "adv" {
		self.Fks = map[string][]string{"adv": {"adv_id", ""}}
	} else {
		self.Fks = map[string][]string{"pub": {"campaign_id", "campaign_md5"}}
	}

	return self.Filter.Get_all()
}

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	//who := self.Role_value

	if summer.Grep([]string{"topicsAdv24Hours", "topicsAdvTopItems", "topicsAdvTopSlots", "topicsPub24Hours", "topicsPubTopSlots", "topicsPubTopCampaigns"}, action) {
		if ARGS.Get("day") == "" {
			day_time := time.Now().AddDate(0, 0, -1).String()
			ARGS.Set("day", day_time[0:10])
		}
		if ARGS.Get("idays") == "" {
			ARGS.Set("idays", "0")
		}
		if ARGS.Get("top") == "" {
			ARGS.Set("top", "200")
		}
	}

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	return self.Filter.Before(&model.Model, extra, nextextra)
}

func (self *Filter) After(model *Model) error {
	return self.Filter.After(&model.Model)
}

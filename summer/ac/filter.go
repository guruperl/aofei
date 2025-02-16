package ac

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) Get_all() (map[string][]string, []string) {
	action := self.Action
	ARGS := self.R.Form
	entitytype_id := ARGS.Get("entitytype_id")
	if entitytype_id == "3" {
		if action == "startnew" {
			self.Fks = map[string][]string{"pub": {"pub_id", "", "campaign_id", "campaign_md5"}}
		} else {
			self.Fks = map[string][]string{"pub": {"pub_id", ""}}
		}
	} else if entitytype_id == "31" {
		self.Fks = map[string][]string{"pub": {"site_id", "site_md5"}}
	} else if entitytype_id == "4" {
		self.Fks = map[string][]string{"adv": {"adv_id", ""}}
	} else if entitytype_id == "41" {
		self.Fks = map[string][]string{"adv": {"campaign_id", "campaign_md5"}}
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
	//action := self.Action

	eid := ARGS.Get("entitytype_id")
	ARGS.Set("table", summer.TABLES[eid][0])
	ARGS.Set("idname", summer.TABLES[eid][1])
	ARGS.Set("entity_id", ARGS.Get(ARGS.Get("idname")))

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	return self.Filter.Before(&model.Model, extra, nextextra)
}

func (self *Filter) After(model *Model) error {
	return self.Filter.After(&model.Model)
}

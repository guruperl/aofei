package balance

import (
	"net/url"
	"strconv"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) GetAll() (map[string][]string, []string) {
	ARGS := self.R.Form
	entitytype_id := ARGS.Get("entitytype_id")
	if entitytype_id == "41" {
		self.Fks = map[string][]string{"adv": {"campaign_id", "campaign_md5"}}
	} else if entitytype_id == "42" {
		self.Fks = map[string][]string{"adv": {"item_id", "item_md5"}}
	} else {
		self.Fks = map[string][]string{"pub": {"item_id", "item_md5"}}
	}
	return self.Filter.GetAll()
}

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	ARGS := self.R.Form
	//action := self.Action
	//who := self.RoleValue

	entitytype_id := ARGS.Get("entitytype_id")
	idname := summer.TABLES[entitytype_id][1]

	ARGS.Set("table", summer.TABLES[entitytype_id][0])
	ARGS.Set("idname", idname)
	ARGS.Set("entity_id", ARGS.Get(idname))

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	action := self.Action
	if action == "update" {
		if extra.Get("campaign_id") != "" {
			extra.Del("campaign_id")
		}
		if extra.Get("item_id") != "" {
			extra.Del("item_id")
		}
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	lists := *model.LISTS
	ARGS := self.R.Form
	action := self.Action
	//who := self.RoleValue

	if action == "topics" {
		for _, item := range lists {
			balance_id := strconv.FormatInt(item["balance_id"].(int64), 10)
			ARGS.Set(item["which"].(string), balance_id)
		}
	}

	return nil
}

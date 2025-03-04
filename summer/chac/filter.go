package chac

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) GetAll() (map[string][]string, []string) {
	switch self.R.Form.Get("entitytype_id") {
	case "31":
		self.Fks = map[string][]string{"pub": {"site_id", "site_md5"}}
	case "32":
		self.Fks = map[string][]string{"pub": {"slot_id", "slot_md5"}}
	case "41":
		self.Fks = map[string][]string{"adv": {"campaign_id", "campaign_md5"}}
	case "42":
		self.Fks = map[string][]string{"adv": {"item_id", "item_md5"}}
	default:
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

	entitytypeID := ARGS.Get("entitytype_id")
	idname := summer.TABLES[entitytypeID][1]

	ARGS.Set("table", summer.TABLES[entitytypeID][0])
	ARGS.Set("idname", idname)
	ARGS.Set("entity_id", ARGS.Get(idname))

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	return self.Filter.Before(&model.Model, extra, nextextra)
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	//	ARGS := self.R.Form
	action := self.Action
	//	who := self.RoleValue
	lists := *model.LISTS
	//other := *model.OTHER

	if action == "topics" {
		summer.TranslateOne(lists, "channel_name", "channel_name_g")
	}

	return nil
}

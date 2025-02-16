package item

import (
	"net/url"
	"strings"

	"github.com/genelet/winter/summer"
	// "database/sql"
	// hitem "github.com/genelet/winter/holiday/item"
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
	who := self.Role_value

	if who == "adv" && (action == "insert" || action == "update") {
		for _, name := range []string{"fl_language", "fl_device", "fl_position", "fl_content"} {
			if ARGS.Get(name) != "" {
				ARGS.Set(name, strings.Join(ARGS[name], ","))
			}
		}
		summer.SetSizeID(ARGS)
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

	if who == "admin" && action == "topics" {
		if ARGS.Get("campaign_id") != "" {
			extra.Set("campaign_id", ARGS.Get("campaign_id"))
		}
	} else if who == "agent" && action == "topics" {
		if ARGS.Get("agent_level") == "1" {
			extra["active"] = []string{"Yes", "New"}
		} else {
			extra["active"] = []string{"Yes", "New", "Pass2"}
		}
	} else if who == "pub" && action == "topics" {
		extra["active"] = []string{"Yes", "Pass2", "New"}
	} else if action == "topics" {
		extra["active"] = []string{"Yes", "Pass2", "New", "Pause", "Prepare"}
	} else if who == "adv" && action == "insert" {
		model.Current_table = "adv_balance"
		if err := self.BalanceBefore(&model.Model); err != nil {
			return err
		}
		model.Current_table = "adv_item"
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	//ARGS := self.R.Form
	action := self.Action
	//who := self.Role_value
	lists := *model.LISTS
	other := *model.OTHER

	if action == "topics" {
		for _, item := range lists {
			if item["startx"] != nil {
				startx := item["startx"].(string)
				item["startx"] = startx[5 : len(startx)-9]
			}
			if item["endx"] != nil {
				endx := item["endx"].(string)
				item["endx"] = endx[5 : len(endx)-9]
			}
			summer.SetWH(item)
		}
		summer.TranslateOne(lists, "qa_mime", "qa_chinese")
	} else if action == "startnew" {
		for _, name := range []string{"language", "device", "position", "content"} {
			other["fl_"+name] = summer.LARGES[name]
			summer.TranslateOne(other["fl_"+name], "which", "label_chinese")
		}
		for _, name := range []string{"mime", "creative"} {
			other["qa_"+name] = summer.LARGES[name]
			summer.TranslateOne(other["qa_"+name], "which", "label_chinese")
		}
	} else if action == "edit" {
		item := lists[0]
		summer.SetWH(item)
		for _, name := range []string{"language", "device", "position", "content"} {
			str := ""
			if item["fl_"+name] != nil {
				str = item["fl_"+name].(string)
			}
			other["fl_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["fl_"+name], "which", "label_chinese")
		}
		for _, name := range []string{"mime", "creative"} {
			str := ""
			if item["qa_"+name] != nil {
				str = item["qa_"+name].(string)
			}
			other["qa_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["qa_"+name], "which", "label_chinese")
		}
	}

	/*
	   	if action=="update" || (action=="authen" && ARGS.Get("active")=="Yes") {
	   		taodb, err := sql.Open(self.C.Custom["taoType"], self.C.Custom["taoAccount"])
	   		if err != nil { return err }
	   		hmodel := new(hitem.Model)
	   		err = hmodel.Load(self.C.ProjectRoot+"/src/holiday/item/component.json")
	   		if err != nil { return err }
	   		hmodel.Db = taodb
	   		err = hmodel.ExecSQL("drop table if exists item_"+ARGS.Get("item_id"))
	   		if err != nil { return err }
	   		lists := make([]map[string]interface{},0)
	   		err = model.Select_sql(&lists,
	   `SELECT i.item_id, creative_id, weight, item_click, cpc_fc, cpc_length, content
	   FROM adv_item i
	   INNER JOIN adv_campaign c USING (campaign_id)
	   INNER JOIN adv_creative r USING (item_id)
	   WHERE i.item_id=? AND c.active="Yes" AND i.active="Yes"`, ARGS.Get("item_id"))
	   		if err != nil { return err }
	   		for _, one := range lists {
	   			err = hmodel.Insert(one)
	   			if err != nil { return err }
	   		}
	   		taodb.Close()
	   	}
	*/

	return nil
}

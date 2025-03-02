// Package campaign has insert:
// 1) into to adv_balance needs to do before action, to get balance_id
// 2) ac, channel - InserAc, channel - InsertBelong after, to get entity_id
package campaign

import (
	"net/url"

	"github.com/genelet/winter/summer"
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
	//who := self.RoleValue

	if ARGS.Get("_gadmin") != "1" && (action == "insert" || action == "update") {
		if ARGS.Get("active") != "" {
			ARGS.Del("active")
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
	who := self.RoleValue

	if who == "admin" && action == "topics" {
		if ARGS.Get("adv_id") != "" {
			extra.Set("adv_id", ARGS.Get("adv_id"))
		}
	} else if who == "agent" && action == "topics" {
		if ARGS.Get("agent_level") == "1" {
			extra["c.active"] = []string{"Yes", "New"}
		} else {
			extra["c.active"] = []string{"Yes", "New", "Pass2"}
		}
	} else if action == "topics" {
		extra["c.active"] = []string{"Yes", "New", "Pass2", "Pause"}
	} else if who == "adv" && action == "insert" {
		model.CurrentTable = "adv_balance"
		if err := self.BalanceBefore(&model.Model); err != nil {
			return err
		}
		model.CurrentTable = "adv_campaign"
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	//ARGS := self.R.Form
	action := self.Action
	//who := self.RoleValue
	lists := *model.LISTS
	other := *model.OTHER

	if action == "startnew" {
		summer.TranslateOne(other["channel_topics"], "channel_name", "channel_name_g")
	} else if action == "edit" {
		item := lists[0]
		summer.TranslateOne(item, "access_order", "access_order_g")
	}

	/*
	   	if action=="update" || (action=="authen" && ARGS.Get("active")=="Yes") {
	           taodb, err := sql.Open(self.C.Custom["taoType"], self.C.Custom["taoAccount"])
	           if err != nil { return err }
	           hmodel := new(hitem.Model)
	           err = hmodel.Load(self.C.ProjectRoot+"/src/holiday/item/component.json")
	           if err != nil { return err }
	           hmodel.Db = taodb

	   		items := make([]map[string]interface{},0)
	   		err = model.SelectSQL(&items,
	   `SELECT item_id
	   FROM adv_item i
	   INNER JOIN adv_campaign c USING (campaign_id)
	   WHERE i.campaign_id=? AND c.active="Yes" AND i.active="Yes"`, ARGS.Get("campaign_id"))
	           if err != nil { return err }
	   		for _, one := range items {
	   			sql := fmt.Sprintf("drop table if exists item_%d", one["item_id"])
	   			err = hmodel.ExecSQL(sql)
	   			if err != nil { return err }
	   		}
	           lists := make([]map[string]interface{},0)
	           err = model.SelectSQL(&lists,
	   `SELECT i.item_id, creative_id, weight, item_click, cpc_fc, cpc_length, content
	   FROM adv_item i
	   INNER JOIN adv_campaign c USING (campaign_id)
	   INNER JOIN adv_creative r USING (item_id)
	   WHERE i.campaign_id=? AND c.active="Yes" AND i.active="Yes"`, ARGS.Get("campaign_id"))
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

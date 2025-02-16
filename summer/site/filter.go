package site

import (
	"net/url"
	"strconv"

	//	"strings"
	"github.com/genelet/winter/summer"
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

	if (who == "pub" || who == "admin") && (action == "insert" || action == "update") {
		qa_site := summer.GetSiteScoreArgs(ARGS)
		ARGS.Set("qa_site", strconv.FormatUint(uint64(qa_site), 10))
		fl_camp := summer.GetCampaignScoreArgs(ARGS)
		ARGS.Set("fl_campaign", strconv.FormatUint(uint64(fl_camp), 10))
	}
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
	who := self.Role_value

	if who == "pub" && action == "topics" {
		extra["active"] = []string{"New", "Yes"}
	} else if who == "admin" && action == "topics" {
		if pub_id := ARGS.Get("pub_id"); pub_id != "" {
			extra.Set("pub_id", pub_id)
		}
	}
	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	//role  := self.Role_value
	lists := *model.LISTS
	other := *model.OTHER

	if action == "edit" {
		item := lists[0]
		site := summer.UnpackSite((uint32(item["qa_site"].(int64))))
		for k, v := range site.InHash() {
			item[k] = v
		}
		camp := summer.UnpackCampaign((uint32(item["fl_campaign"].(int64))))
		for k, v := range camp.InHash() {
			item[k] = v
		}
		summer.TranslateOne(item, "access_order", "access_order_g")
		summer.TranslateOne(item["chac_topics"], "channel_name", "channel_name_g")
	} else if action == "startnew" {
		summer.TranslateOne(other["channel_topics"], "channel_name", "channel_name_g")
	} else if action == "insert" {
		item := lists[0]
		ARGS.Set("entitytype_id", "31")
		// in genelet model, auto id is returned as string
		ARGS.Set("entity_id", item["site_id"].(string))

		if ARGS.Get("belong_ids") != "" {
			err := model.Call_once(map[string]interface{}{"model": "chac", "action": "insertBelong"})
			if err != nil {
				return err
			}
		}
		if ARGS.Get("channel_order") != "" && ARGS.Get("ac_ids") != "" {
			err := model.Call_once(map[string]interface{}{"model": "chac", "action": "insertAc"})
			if err != nil {
				return err
			}
		}
	} else if action == "update" {
		ARGS.Set("table", "pub_site")
		ARGS.Set("idname", "site_id")
		ARGS.Set("entitytype_id", "31")
		ARGS.Set("entity_id", ARGS.Get("site_id"))
		err := model.Call_once(map[string]interface{}{"model": "chac", "action": "update"})
		if err != nil {
			return err
		}
	}

	return nil
}

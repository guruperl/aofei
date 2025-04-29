package item

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

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
	who := self.RoleValue

	if who == "adv" {
		ARGS.Set("entitytype_id", "42")
	}

	if who == "adv" && (action == "insert" || action == "update") {
		slot := summer.GetSlotScoreArgs(ARGS)
		ARGS.Set("fl_slot", strconv.FormatUint(uint64(slot), 10))
		item := summer.GetItemScoreArgs(ARGS)
		ARGS.Set("qa_item", strconv.FormatUint(uint64(item), 10))
		if ARGS.Get("page_cap") != "" {
			i, err := strconv.Atoi(ARGS.Get("page_cap"))
			if err != nil {
				return err
			}
			if i > 255 {
				return fmt.Errorf("page_cap should be less than 256")
			}
		}
		for _, v := range []string{"cpc_fc", "cpm_fc"} {
			if ARGS.Get(v) != "" {
				i, err := strconv.Atoi(ARGS.Get(v))
				if err != nil {
					return err
				}
				if i > 65535 {
					return fmt.Errorf("%s should be less than 65536", v)
				}
			}
		}
		for _, name := range []string{"fl_language", "fl_device", "fl_position"} {
			if ARGS.Get(name) != "" {
				ARGS.Set(name, strings.Join(ARGS[name], ","))
			}
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
	if ARGS.Get("_gadmin") == "1" {
		who = "admin"
	}

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
		model.CurrentTable = "adv_balance"
		if err := self.BalanceBefore(&model.Model); err != nil {
			return err
		}
		model.CurrentTable = "adv_item"
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	who := self.RoleValue
	if ARGS.Get("_gadmin") == "1" {
		who = "admin"
	}
	lists := *model.LISTS
	other := *model.OTHER

	if action == "startnew" {
		for _, name := range []string{"language", "device", "position"} {
			other["fl_"+name] = summer.LARGES[name]
			summer.TranslateOne(other["fl_"+name], "label", "label_chinese")
		}
		for _, name := range []string{"mime", "creative", "expnd"} {
			other["qa_"+name] = summer.LARGES[name]
			summer.TranslateOne(other["qa_"+name], "label", "label_chinese")
		}
	} else if action == "edit" {
		item := lists[0]
		campItem := summer.UnpackItem((uint32(item["qa_item"].(int64))))
		for k, v := range campItem.InHash() {
			item[k] = v
		}
		slot := summer.UnpackSlot((uint32(item["fl_slot"].(int64))))
		for k, v := range slot.InHash() {
			item[k] = v
		}
		for _, name := range []string{"language", "device", "position"} {
			str := ""
			if item["fl_"+name] != nil {
				str = item["fl_"+name].(string)
			}
			other["fl_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["fl_"+name], "label", "label_chinese")
		}
		for _, name := range []string{"mime", "creative", "expnd"} {
			str := ""
			if item["qa_"+name] != nil {
				str = item["qa_"+name].(string)
			}
			other["qa_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["qa_"+name], "label", "label_chinese")
		}
	} else if action == "topics" {
		for _, item := range lists {
			if item["startx"] != nil {
				startx := item["startx"].(string)
				item["startx"] = startx[5 : len(startx)-9]
			}
			if item["endx"] != nil {
				endx := item["endx"].(string)
				item["endx"] = endx[5 : len(endx)-9]
			}
		}
		summer.TranslateOne(lists, "qa_mime", "qa_chinese")
	} else if (who == "admin" || who == "agent") && action == "update" {
		// this is to recalculate the weight of active creatives, after an update
		// to ensure the weights are normalized to 1.0
		err := model.DoSQL(`
UPDATE adv_creative r
INNER JOIN (
	SELECT SUM(weight) AS weight
	FROM adv_creative
	WHERE item_id=? AND active="Yes"
) AS t ON r.item_id = ?
SET r.weight= r.weight/t.weight`, ARGS.Get("item_id"), ARGS.Get("item_id"))
		if err != nil {
			return fmt.Errorf("failed to recalculate weight: %s", err.Error())
		}
	}

	return nil
}

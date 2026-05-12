// Package slot describes slot.
package slot

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/match"
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
	//	who := self.RoleValue

	if action == "insert" || action == "update" {
		qaSlot := summer.GetSlotScoreArgs(ARGS)
		ARGS.Set("qa_slot", strconv.FormatUint(uint64(qaSlot), 10))
		flItem := summer.GetItemScoreArgs(ARGS)
		ARGS.Set("fl_item", strconv.FormatUint(uint64(flItem), 10))
		err := summer.SetSizeID(ARGS)
		if err != nil {
			return err
		}
		for _, name := range []string{"fl_mime", "fl_creative", "fl_expnd"} {
			if ARGS.Get(name) != "" {
				ARGS.Set(name, strings.Join(ARGS[name], ","))
			}
		}
		err = summer.SetSizeID(ARGS)
		if err != nil {
			return err
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
	//who := self.RoleValue

	if action == "topics" {
		if siteID := ARGS.Get("site_id"); siteID != "" {
			extra.Set("site_id", siteID)
		}
		extra["active"] = []string{"Yes", "New"}
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	lists := *model.LISTS
	other := *model.OTHER

	if action == "startnew" {
		for _, name := range []string{"language", "device", "position"} {
			other["qa_"+name] = summer.LargeOptions(name)
			summer.TranslateOne(other["qa_"+name], "label", "label_chinese")
		}
		for _, name := range []string{"mime", "creative", "expnd"} {
			other["fl_"+name] = summer.LargeOptions(name)
			summer.TranslateOne(other["fl_"+name], "label", "label_chinese")
		}
		summer.TranslateOne(other["channel_topics"], "channel_name", "channel_name_g")
	} else if action == "edit" {
		item := lists[0]
		item["size_id"] = int64(4194368) // 64x64
		summer.SetWH(item)
		slot := summer.UnpackSlot((uint32(item["qa_slot"].(int64))))
		for k, v := range slot.InHash() {
			item[k] = v
		}
		campItem := summer.UnpackItem((uint32(item["fl_item"].(int64))))
		for k, v := range campItem.InHash() {
			item[k] = v
		}
		for _, name := range []string{"language", "device", "position"} {
			str := ""
			if item["qa_"+name] != nil {
				str = item["qa_"+name].(string)
			}
			other["qa_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["qa_"+name], "label", "label_chinese")
		}
		for _, name := range []string{"mime", "creative", "expnd"} {
			str := ""
			if item["fl_"+name] != nil {
				str = item["fl_"+name].(string)
			}
			other["fl_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["fl_"+name], "label", "label_chinese")
		}
		summer.TranslateOne(item["chac_topics"], "channel_name", "channel_name_g")
	} else if action == "topics" {
		summer.TranslateOne(lists, "qa_device", "qa_device_g")
		ARGS.Set("serverUrl", "/")
		ARGS.Set("serverScript", "/bid")
		pubID, err := strconv.ParseUint(ARGS.Get("pub_id"), 10, 32)
		if err != nil {
			return err
		}
		siteID, err := strconv.ParseUint(ARGS.Get("site_id"), 10, 32)
		if err != nil {
			return err
		}
		ARGS.Set("site_str", summer.PackTwo(uint32(pubID), uint32(siteID)))
		for _, item := range lists {
			slotID := uint32(item["slot_id"].(int64))
			item["w"] = uint16(64)
			item["h"] = uint16(64)
			sizeID := uint32(4194368) // 64x64
			item["size_id"] = int64(sizeID)
			summer.SetWH(item)
			item["slot_str"] = summer.PackTwo(slotID, sizeID)
			var err error
			item["code"], err = match.RPub{PubID: uint32(pubID), SiteID: uint32(siteID), SlotID: slotID, SizeID: sizeID}.PackString()
			if err != nil {
				return err
			}
			item["mediaTypes"] = mimeFormat(item)
			if created := item["created"]; created != nil {
				c := created.(string)
				item["created"] = c[:len(c)-9]
			}
		}
	} else if action == "insert" {
		item := lists[0]
		ARGS.Set("entitytype_id", "32")
		ARGS.Set("entity_id", item["slot_id"].(string))
		ARGS.Set("othertype_id", "4")

		if ARGS.Get("belong_ids") != "" {
			err := model.CallOnce(map[string]interface{}{"model": "chac", "action": "insertBelong"})
			if err != nil {
				return err
			}
		}
		if ARGS.Get("ac_ids") != "" {
			err := model.CallOnce(map[string]interface{}{"model": "chac", "action": "insertAc"})
			if err != nil {
				return err
			}
		}
	} else if action == "update" {
		ARGS.Set("table", "pub_slot")
		ARGS.Set("idname", "slot_id")
		ARGS.Set("entitytype_id", "32")
		ARGS.Set("entity_id", ARGS.Get("slot_id"))
		ARGS.Set("othertype_id", "4")
		err := model.CallOnce(map[string]interface{}{"model": "chac", "action": "update"})
		if err != nil {
			return err
		}
	}

	return nil
}

func mimeFormat(item map[string]interface{}) string {
	w := item["w"].(uint16)
	h := item["h"].(uint16)
	hash := make(map[string]string)
	sizeStr := `[` + strconv.Itoa(int(w)) + `,` + strconv.Itoa(int(h)) + `]`

	flMime := item["fl_mime"].(string)
	switch flMime {
	case "Iframe":
		hash["iframe"] = `{wrong:` + sizeStr + `}`
	default:
		hash["native"] = `{image:` + sizeStr + `}`
	}
	str := ""
	for k, v := range hash {
		str += "\t\t\t" + k + ":" + v + ",\n"
	}
	return str[:len(str)-2]
}

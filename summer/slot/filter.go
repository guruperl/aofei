// Description: Filter for slot.
package slot

import (
	"net/url"
	"strconv"
	"strings"

	//"github.com/golang/glog"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/pzutil"
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
	//	who := self.Role_value

	if action == "insert" || action == "update" {
		for _, name := range []string{"fl_mime"} {
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
	//who := self.Role_value

	if action == "topics" {
		if site_id := ARGS.Get("site_id"); site_id != "" {
			extra.Set("site_id", site_id)
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
	//role  := self.Role_value
	lists := *model.LISTS
	other := *model.OTHER

	if action == "startnew" {
		for _, name := range []string{"language", "device", "position", "content"} {
			other["qa_"+name] = summer.LARGES[name]
			summer.TranslateOne(other["qa_"+name], "which", "label_chinese")
		}
		for _, name := range []string{"mime", "creative"} {
			other["fl_"+name] = summer.LARGES[name]
			summer.TranslateOne(other["fl_"+name], "which", "label_chinese")
		}
		summer.TranslateOne(other["channel_topics"], "channel_name", "channel_name_g")
	} else if action == "edit" {
		item := lists[0]
		summer.SetWH(item)
		for _, name := range []string{"language", "device", "position", "content"} {
			str := ""
			if item["qa_"+name] != nil {
				str = item["qa_"+name].(string)
			}
			other["qa_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["qa_"+name], "which", "label_chinese")
		}
		for _, name := range []string{"mime", "creative"} {
			str := ""
			if item["fl_"+name] != nil {
				str = item["fl_"+name].(string)
			}
			other["fl_"+name] = self.AfterItemSet(name, str)
			summer.TranslateOne(other["fl_"+name], "which", "label_chinese")
		}
		summer.TranslateOne(item["chac_topics"], "channel_name", "channel_name_g")
	} else if action == "topics" {
		summer.TranslateOne(lists, "qa_device", "qa_device_g")
		c := model.Storage["Ssp"].(*pzutil.Config)
		ARGS.Set("serverUrl", c.ServerURL)
		ARGS.Set("serverScript", c.ServerURL+c.Handle["ssp"])
		pub_id, _ := strconv.ParseUint(ARGS.Get("pub_id"), 10, 32)
		site_id, _ := strconv.ParseUint(ARGS.Get("site_id"), 10, 32)
		ARGS.Set("site_str", pzutil.PackTwo(uint32(pub_id), uint32(site_id)))
		for _, item := range lists {
			slot_id := uint32(item["slot_id"].(int64))
			size_id := uint32(item["size_id"].(int64))
			summer.SetWH(item)
			item["slot_str"] = pzutil.PackTwo(slot_id, size_id)
			var err error
			item["code"], err = match.RPub{PubID: uint32(pub_id), SiteID: uint32(site_id), SlotID: slot_id, SizeID: size_id}.Pack1()
			if err != nil {
				return err
			}
			item["mediaTypes"] = mime_format(item)
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

		if ARGS.Get("mychannel") != "Inherit" && ARGS.Get("belong_ids") != "" {
			err := model.Call_once(map[string]interface{}{"model": "chac", "action": "insertBelong"})
			if err != nil {
				return err
			}
		}
		if ARGS.Get("channel_order") != "Inherit" && ARGS.Get("ac_ids") != "" {
			err := model.Call_once(map[string]interface{}{"model": "chac", "action": "insertAc"})
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
		err := model.Call_once(map[string]interface{}{"model": "chac", "action": "update"})
		if err != nil {
			return err
		}
	}

	return nil
}

func mime_format(item map[string]interface{}) string {
	w := item["w"].(uint16)
	h := item["h"].(uint16)
	hash := make(map[string]string)
	size_str := `[` + strconv.Itoa(int(w)) + `,` + strconv.Itoa(int(h)) + `]`

	fl_mime := item["fl_mime"].(string)
	switch fl_mime {
	case "Iframe":
		hash["iframe"] = `{wrong:` + size_str + `}`
	default:
		hash["native"] = `{image:` + size_str + `}`
	}
	str := ""
	for k, v := range hash {
		str += "\t\t\t" + k + ":" + v + ",\n"
	}
	return str[:len(str)-2]
}

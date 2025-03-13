// Package targetname is for campaign targeting management
package targetname

import (
	"database/sql"
	"net/url"
	"strconv"

	uadevice "github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
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
	//who := self.RoleValue

	if ARGS.Get("entitytype_id") == "41" {
		ARGS.Set("entity_id", ARGS.Get("campaign_id"))
	} else if ARGS.Get("entitytype_id") == "42" {
		ARGS.Set("entity_id", ARGS.Get("item_id"))
	}

	if action == "insert" || action == "update" {
		uadevice.ResetArgs(ARGS)
	}

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action

	if action == "topics" {
		extra.Set("tn.item_id", ARGS.Get("item_id"))
	} else if action == "delete" {
		extra.Set("item_id", ARGS.Get("item_id"))
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
	lists := *model.LISTS
	other := *model.OTHER

	if action == "topics" {
		var channelOrder string
		err := model.DB.QueryRow(`
SELECT channel_order FROM adv_item WHERE item_id = ?`, ARGS.Get("item_id")).Scan(&channelOrder)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		other["channel_order"] = channelOrder
		summer.TranslateOne(other["chac_topics"], "channel_name", "channel_name_g")

		isps := make(map[uint32][]interface{})
		states := make(map[uint32][]interface{})
		cities := make(map[string]map[uint32][]interface{})
		customs := make(map[string]map[uint32][]interface{})
		dmas := make(map[string]map[uint32][]interface{})
		if other["targetname_topicsIsps"] != nil {
			for _, item := range other["targetname_topicsIsps"].([]map[string]interface{}) {
				ispName := item["isp_name"].(string)
				ispID := uint32(item["isp_id"].(int64))
				isps[ispID] = []interface{}{ispName, item["value_id"]}
			}
			other["isp"] = isps
			delete(other, "targetname_topicsIsps")
		}

		if other["targetname_topicsStates"] != nil {
			for _, item := range other["targetname_topicsStates"].([]map[string]interface{}) {
				stateName := item["state_name"].(string)
				stateID := uint32(item["state_id"].(int64))
				states[stateID] = []interface{}{stateName, item["value_id"], item["state_code"], item["country_name"]}
				cities[stateName] = make(map[uint32][]interface{})
			}
			other["state"] = states
			delete(other, "targetname_topicsStates")
		}

		if other["targetname_topicsCities"] != nil {
			for _, item := range other["targetname_topicsCities"].([]map[string]interface{}) {
				stateName := item["state_name"].(string)
				cityName := item["city_name"].(string)
				cityID := uint32(item["city_id"].(int64))
				cities[stateName][cityID] = []interface{}{cityName, item["value_id"]}
			}
			other["city"] = cities
			delete(other, "targetname_topicsCities")
		}

		if other["targetname_topicsDmas"] != nil {
			for _, item := range other["targetname_topicsDmas"].([]map[string]interface{}) {
				cityName := item["city_name"].(string)
				if _, ok := dmas[cityName]; !ok {
					dmas[cityName] = make(map[uint32][]interface{})
				}
				metroCode := item["metro_code"].(string)
				dmaID := uint32(item["dma_id"].(int64))
				dmas[cityName][dmaID] = []interface{}{metroCode, item["value_id"]}
			}
			other["dma"] = dmas
			delete(other, "targetname_topicsDmas")
		}

		if other["targetname_topicsCustom"] != nil {
			for _, item := range other["targetname_topicsCustom"].([]map[string]interface{}) {
				attrname := item["attrname"].(string) + "_" + strconv.FormatInt(item["attrname_id"].(int64), 10)
				attrvalueID := uint32(item["attrvalue_id"].(int64))
				if _, ok := customs[attrname]; !ok {
					customs[attrname] = make(map[uint32][]interface{})
				}
				customs[attrname][attrvalueID] = []interface{}{item["value"], item["value_id"]}
			}
			other["custom"] = customs
			delete(other, "targetname_topicsCustom")
		}

		/* the standard Topics lists, one state selected and two selected for hour and day
		SELECT tn.targetname_id, tv.targetvalue_id, tv.value_id, an.attrname_id, an.attrname, av.attrvalue_id
		FROM adv_targetname tn
		INNER JOIN adv_targetvalue tv USING (targetname_id)
		INNER JOIN adv_attrname an USING (attrname_id)
		LEFT JOIN adv_attrvalue av ON (an.attrname_id=av.attrname_id AND tv.value_id=av.attrvalue_id)
		WHERE (tn.item_id =5)
		ORDER BY targetname_id DESC LIMIT 100 OFFSET 0;
				+---------------+----------------+----------+-------------+----------+--------------+
				| targetname_id | targetvalue_id | value_id | attrname_id | attrname | attrvalue_id |
				+---------------+----------------+----------+-------------+----------+--------------+
				|             4 |              5 |        1 |        1002 | fullhour |         NULL |
				|             4 |              6 |        2 |        1002 | fullhour |         NULL |
				|             3 |              3 |        1 |        1003 | weekday  |         NULL |
				|             3 |              4 |        6 |        1003 | weekday  |         NULL |
				|             2 |              2 |     3263 |        1103 | state    |         NULL |
				+---------------+----------------+----------+-------------+----------+--------------+
		*/
		ref := make(map[string]map[int]bool)
		var utcoffsetFound bool
		var utcoffset int
		hours := make(map[int]bool)
		days := make(map[int]bool)
		uaAud := new(uadevice.UaAudience)
		for _, item := range lists {
			attrname := item["attrname"].(string)
			if _, ok := ref[attrname]; !ok {
				ref[attrname] = make(map[int]bool)
			}
			valueID := int(item["value_id"].(int64))
			switch attrname {
			case "utcoffset":
				utcoffsetFound = true
				utcoffset = int(int32(uint32(item["value_id"].(int64))))
			case "fullhour":
				hours[valueID] = true
			case "weekday":
				days[valueID] = true
			case "isp", "state", "city", "dma":
			case "os":
				uaAud.UaOSs = uint32(valueID)
			case "oversion":
				uaAud.UaOVersions = uint32(valueID)
			case "platform":
				uaAud.UaPlatforms = uint32(valueID)
			case "browser":
				uaAud.UaBrowsers = uint32(valueID)
			case "device":
				uaAud.UaDevices = uint32(valueID)
			default:
				ref[attrname][valueID] = true
			}
		}

		utcoffsets := make(map[int][]interface{})
		for i := -11; i < 13; i++ {
			str := strconv.Itoa(i)
			selected := false
			if utcoffsetFound && i == utcoffset {
				selected = true
			}
			utcoffsets[i] = []interface{}{str, selected}
		}
		other["utcoffset"] = utcoffsets
		if utcoffsetFound {
			other["utcoffsetFound"] = "true"
		} else {
			other["utcoffsetFound"] = "false"
		}

		fullhours := make(map[int][]interface{})
		for i := 0; i < 24; i++ {
			str := strconv.Itoa(i)
			selected := hours[i]
			fullhours[i] = []interface{}{str, selected}
		}
		other["fullhour"] = fullhours

		weekdays := make(map[int][]interface{})
		WEEK := map[int]string{0: "Sun", 1: "Mon", 2: "Tue", 3: "Wed", 4: "Thu", 5: "Fri", 6: "Sat"}
		for i, name := range WEEK {
			selected := days[i]
			weekdays[i] = []interface{}{name, selected}
		}
		other["weekday"] = weekdays
		other["weekdayChinese"] = summer.Translate(other["weekday"])

		other["pzua"] = uaAud.Tmpls()
		other["pzuaChinese"] = summer.Translate(other["pzua"])
		other["pzAttrs"] = uadevice.UaAttrs()
		other["pzAttrsChinese"] = summer.Translate(other["pzAttrs"])

		demos := make(map[string]map[int][]interface{})
		for attrname, val := range demo.DemoNames() {
			item := make(map[int][]interface{})
			for valueID, name := range val {
				item[int(valueID)] = []interface{}{name, ref[attrname][int(valueID)]}
			}
			demos[attrname] = item
		}
		other["demo"] = demos
		other["demoChinese"] = summer.Translate(other["demo"])
		other["dAttrs"] = demo.DemoAttrs()
		other["dAttrsChinese"] = summer.Translate(other["dAttrs"])
	} else if who == "adv" && action == "insert" {
		ARGS.Set("table", "adv_item")
		ARGS.Set("idname", "item_id")
		ARGS.Set("entity_id", ARGS.Get("item_id"))
		ARGS.Set("entitytype_id", "42")
		err := model.CallOnce(map[string]interface{}{"model": "chac", "action": "update"})
		if err != nil {
			return err
		}
	}

	return nil
}

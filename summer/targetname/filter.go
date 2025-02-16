package targetname

import (
	"net/url"
	"strconv"

	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/summer"
	"github.com/genelet/winter/uadevice"
	// "database/sql"
	// hitem "github.com/genelet/winter/holiday/target"
)

type Filter struct {
	summer.Filter
}

// filter is initialized and re-used. if reset fks, please reset all, no default
func (self *Filter) Get_all() (map[string][]string, []string) {
	ARGS := self.R.Form

	entitytype_id := ARGS.Get("entitytype_id")
	if entitytype_id == "41" {
		self.Fks = map[string][]string{"adv": {"campaign_id", "campaign_md5", "targetname_id", "targetname_md5"}}
	} else if entitytype_id == "42" {
		self.Fks = map[string][]string{"adv": {"item_id", "item_md5", "targetname_id", "targetname_md5"}}
	} else {
		self.Fks = map[string][]string{"pub": {"item_id", "item_md5", "targetname_id", "targetname_md5"}}
	}

	return self.Filter.Get_all()
}

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	ARGS := self.R.Form
	//who := self.Role_value

	if ARGS.Get("entitytype_id") == "41" {
		ARGS.Set("entity_id", ARGS.Get("campaign_id"))
	} else if ARGS.Get("entitytype_id") == "42" {
		ARGS.Set("entity_id", ARGS.Get("item_id"))
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
		extra.Set("tn.campaign_id", ARGS.Get("campaign_id"))
	} else if action == "delete" {
		extra.Set("campaign_id", ARGS.Get("campaign_id"))
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
		isps := make(map[uint32][]interface{})
		states := make(map[uint32][]interface{})
		cities := make(map[string]map[uint32][]interface{})
		customs := make(map[string]map[uint32][]interface{})
		dmas := make(map[string]map[uint32][]interface{})
		if other["targetname_topicsIsps"] != nil {
			for _, item := range other["targetname_topicsIsps"].([]map[string]interface{}) {
				isp_name := item["isp_name"].(string)
				isp_id := uint32(item["isp_id"].(int64))
				isps[isp_id] = []interface{}{isp_name, item["value_id"]}
			}
			other["isp"] = isps
			delete(other, "targetname_topicsIsps")
		}

		if other["targetname_topicsStates"] != nil {
			for _, item := range other["targetname_topicsStates"].([]map[string]interface{}) {
				state_name := item["state_name"].(string)
				state_id := uint32(item["state_id"].(int64))
				states[state_id] = []interface{}{state_name, item["value_id"]}
				cities[state_name] = make(map[uint32][]interface{})
				dmas[state_name] = make(map[uint32][]interface{})
			}
			other["state"] = states
			delete(other, "targetname_topicsStates")
		}

		if other["targetname_topicsCities"] != nil {
			for _, item := range other["targetname_topicsCities"].([]map[string]interface{}) {
				state_name := item["state_name"].(string)
				city_name := item["city_name"].(string)
				city_id := uint32(item["city_id"].(int64))
				cities[state_name][city_id] = []interface{}{city_name, item["value_id"]}
			}
			other["city"] = cities
			delete(other, "targetname_topicsCities")
		}

		if other["targetname_topicsDmas"] != nil {
			for _, item := range other["targetname_topicsDmas"].([]map[string]interface{}) {
				state_name := item["state_name"].(string)
				metro_code := item["metro_code"].(string)
				dma_id := uint32(item["dma_id"].(int64))
				dmas[state_name][dma_id] = []interface{}{metro_code, item["value_id"]}
			}
			other["dma"] = dmas
			delete(other, "targetname_topicsDmas")
		}

		if other["targetname_topicsCustom"] != nil {
			for _, item := range other["targetname_topicsCustom"].([]map[string]interface{}) {
				attrname := item["attrname"].(string) + "_" + strconv.FormatInt(item["attrname_id"].(int64), 10)
				attrvalue_id := uint32(item["attrvalue_id"].(int64))
				if _, ok := customs[attrname]; !ok {
					customs[attrname] = make(map[uint32][]interface{})
				}
				customs[attrname][attrvalue_id] = []interface{}{item["value"], item["value_id"]}
			}
			other["custom"] = customs
			delete(other, "targetname_topicsCustom")
		}

		ref := make(map[string]map[uint32]bool)
		hours := uint32(0)
		days := uint32(0)
		for _, item := range lists {
			attrname := item["attrname"].(string)
			if _, ok := ref[attrname]; !ok {
				ref[attrname] = make(map[uint32]bool)
			}
			value_id := uint32(item["value_id"].(int64))
			switch attrname {
			case "weekhour":
				hours = value_id
			case "weekday":
				days = value_id
			case "isp", "state", "city", "dma":
			default:
				ref[attrname][value_id] = true
			}
		}

		pzuas := make(map[string]map[uint32][]interface{})
		for attrname, val := range uadevice.UaNames() {
			item := make(map[uint32][]interface{})
			for value_id, name := range val {
				item[value_id] = []interface{}{name, ref[attrname][value_id]}
			}
			pzuas[attrname] = item
		}
		other["pzua"] = pzuas
		other["pzuaChinese"] = summer.Translate(other["pzua"])
		other["pzAttrs"] = uadevice.UaAttrs()
		other["pzAttrsChinese"] = summer.Translate(other["pzAttrs"])

		demos := make(map[string]map[uint32][]interface{})
		for attrname, val := range demo.DemoNames() {
			item := make(map[uint32][]interface{})
			for value_id, name := range val {
				item[value_id] = []interface{}{name, ref[attrname][value_id]}
			}
			demos[attrname] = item
		}
		other["demo"] = demos
		other["demoChinese"] = summer.Translate(other["demo"])
		other["dAttrs"] = demo.DemoAttrs()
		other["dAttrsChinese"] = summer.Translate(other["dAttrs"])

		weekhours := make(map[uint32][]interface{})
		for i := uint32(0); i < 24; i++ {
			str := strconv.FormatUint(uint64(i), 10)
			weekhours[i] = []interface{}{str, ((1 << i) & hours) > 0}
		}
		other["weekhour"] = weekhours

		weekdays := make(map[uint32][]interface{})
		WEEK := map[uint32]string{0: "Sun", 1: "Mon", 2: "Tue", 3: "Wed", 4: "Thu", 5: "Fri", 6: "Sat"}
		for i, name := range WEEK {
			weekdays[i] = []interface{}{name, ((1 << i) & days) > 0}
		}
		other["weekday"] = weekdays
		other["weekdayChinese"] = summer.Translate(other["weekday"])
	}

	/*
	   if action=="insert" {
	       taodb, err := sql.Open(self.C.Custom["taoType"], self.C.Custom["taoAccount"])
	       if err != nil { return err }
	       tmodel := new(hitem.Model)
	       err = tmodel.Load(self.C.ProjectRoot+"/src/holiday/target/component.json")
	       if err != nil { return err }
	       tmodel.Db = taodb
	       err = tmodel.ExecSQL("drop table if exists target_"+ARGS.Get("campaign_id"))
	       if err != nil { return err }
	       for _, one := range lists {
	           err = tmodel.Insert(one)
	           if err != nil { return err }
	       }
	       taodb.Close()
	   }
	*/

	return nil
}

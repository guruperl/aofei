// Package targetname is for campaign targeting management
package targetname

import (
	"database/sql"
	"net/url"
	"strconv"

	"github.com/genelet/winter/advice"
	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dh"
	"github.com/genelet/winter/summer"
	"github.com/genelet/winter/uploaded"
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
		advice.UAResetArgs(ARGS)
		demo.DemoResetArgs(ARGS)
		dh.DHResetArgs(ARGS)
		uploaded.UploadedResetArgs(ARGS)
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
		countries := make(map[uint32][]interface{})
		states := make(map[uint32][]interface{})
		cities := make(map[string]map[uint32][]interface{})
		customs := make(map[string]map[uint32][]interface{})
		dmas := make(map[string]map[uint32][]interface{})

		if other["targetname_editACL"] != nil {
			item := other["targetname_editACL"].([]map[string]interface{})[0]
			other["orderSiteTypes"] = item["access_order"]
			other["aclSiteTypes"] = item["fl_sitetypes"]
			delete(other, "targetname_editACL")
			aclChinese := make(map[uint32][]any)
			for _, item := range other["targetname_topicsACL"].([]map[string]interface{}) {
				siteID := uint32(item["site_id"].(int64))
				aclChinese[siteID] = []any{item["other_id"], item["site_type"], item["domain"], item["foreign_id"]}
			}
			other["aclChinese"] = aclChinese
			delete(other, "targetname_topicsACL")
		}

		if other["targetname_topicsIsps"] != nil {
			for _, item := range other["targetname_topicsIsps"].([]map[string]interface{}) {
				ispName := item["isp_name"].(string)
				ispID := uint32(item["isp_id"].(int64))
				isps[ispID] = []interface{}{ispName, item["value_id"]}
			}
			//other["isp"] = isps
			delete(other, "targetname_topicsIsps")
		}

		if other["targetname_topicsCountries"] != nil {
			for _, item := range other["targetname_topicsCountries"].([]map[string]interface{}) {
				countryName := item["country_name"].(string)
				countryID := uint32(item["country_id"].(int64))
				countries[countryID] = []interface{}{countryName, item["value_id"]}
			}
			other["country"] = countries
			delete(other, "targetname_topicsCountries")
		}

		if other["targetname_topicsStates"] != nil {
			for _, item := range other["targetname_topicsStates"].([]map[string]interface{}) {
				stateName := item["state_name"].(string)
				stateID := uint32(item["state_id"].(int64))
				states[stateID] = []interface{}{stateName, item["value_id"], item["state_code"], item["country_name"]}
				cities[stateName] = make(map[uint32][]interface{})
			}
			//other["state"] = states
			delete(other, "targetname_topicsStates")
		}

		if other["targetname_topicsCities"] != nil {
			for _, item := range other["targetname_topicsCities"].([]map[string]interface{}) {
				stateName := item["state_name"].(string)
				cityName := item["city_name"].(string)
				cityID := uint32(item["city_id"].(int64))
				cities[stateName][cityID] = []interface{}{cityName, item["value_id"]}
			}
			//other["city"] = cities
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
			//other["dma"] = dmas
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
		dhAud := new(dh.DHAudience)
		demAud := new(demo.DemoAudience)
		uaAud := new(advice.UaAudience)
		uploadedAud := new(uploaded.UploadAudience)
		for _, item := range lists {
			attrname := item["attrname"].(string)
			valueID := uint32(int(item["value_id"].(int64)))
			dhAud.DBFillDhAudience(attrname, valueID)
			demAud.DBFillDemoAudience(attrname, valueID)
			uaAud.DBFillUaAudience(attrname, valueID)
			uploadedAud.DBFillUploadAudience(attrname, valueID)
		}

		other["dtChinese"] = summer.Translate(dhAud.Tmpls())
		other["pzuaChinese"] = summer.Translate(uaAud.Tmpls())
		other["demoChinese"] = summer.Translate(demAud.Tmpls())
		other["uploadChinese"] = summer.Translate(uploadedAud.Tmpls())
		/*

					<div class="tab-pane fade" id="t4">
						<h4>自定义Upload</h4>
			            <div class="form-group row">
			                <div class="col-sm-12">
			                    <div class="form-check form-check-inline">{{range $k, $v:= .Other.uploadChinese}}
			                        <input class="form-check-input" type=checkbox name="uploads" {{if index $v 1}}checked{{end}} value="{{$k}}" />
			                        <label class="form-check-label">{{index $v 0}}</label>{{end}}
			                    </div>
			                </div>
			            </div>
			            <h3>自定义标签定向</h3>
			            {{range $key, $val := .Other.custom}}<h4>{{$key}}</h4>
			            <div class="form-group row">
			                <div class="col-sm-12">
			                    <div class="form-check form-check-inline">{{range $k, $v:= $val}}
			                        <input class="form-check-input" type=checkbox name="{{$key}}" valu
			e="{{$k}}" {{if index $v 1}}checked{{end}}>
			                        <label class="form-check-label">{{index $v 0}}</label>{{end}}
			                    </div>
			                </div>
			            </div>{{end}}
			        </div>

		*/
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

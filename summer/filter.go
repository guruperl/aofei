package summer

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/genelet/winter/genelet"
	"github.com/genelet/winter/match"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type Filter struct {
	genelet.Filter
}

var TABLES = map[string][]string{
	"3":  {"pub", "pub_id"},
	"31": {"pub_site", "site_id"},
	"32": {"pub_slot", "slot_id"},
	"4":  {"adv", "adv_id"},
	"41": {"adv_campaign", "campaign_id"},
	"42": {"adv_item", "item_id"},
}

var LARGES = map[string][]map[string]interface{}{
	"language": {
		{"which": "EN", "label": "English", "default": true},
		{"which": "ZH", "label": "Chinese", "default": false},
		{"which": "JA", "label": "Japanese", "default": false},
		{"which": "KO", "label": "Korean", "default": false},
		{"which": "RU", "label": "Russian", "default": false},
		{"which": "FR", "label": "French", "default": false},
		{"which": "DE", "label": "German", "default": false},
		{"which": "ES", "label": "Spanish", "default": false},
		{"which": "IT", "label": "Italian", "default": false},
		{"which": "PT", "label": "Portuguese", "default": false},
		{"which": "NL", "label": "Dutch", "default": false},
		{"which": "TR", "label": "Turkish", "default": false},
		{"which": "AR", "label": "Arabic", "default": false},
		{"which": "HE", "label": "Hebrew", "default": false},
		{"which": "VI", "label": "Vietnamese", "default": false},
		{"which": "TH", "label": "Thai", "default": false},
		{"which": "UK", "label": "Ukrainian", "default": false},
		{"which": "ID", "label": "Indonesian", "default": false},
		{"which": "FA", "label": "Persian", "default": false},
		{"which": "EL", "label": "Greek", "default": false},
		{"which": "PL", "label": "Polish", "default": false},
		{"which": "CS", "label": "Czech", "default": false},
		{"which": "SK", "label": "Slovak", "default": false},
		{"which": "HU", "label": "Hungarian", "default": false},
		{"which": "RO", "label": "Romanian", "default": false},
		{"which": "BG", "label": "Bulgarian", "default": false},
		{"which": "SR", "label": "Serbian", "default": false},
		{"which": "FI", "label": "Finnish", "default": false},
		{"which": "SV", "label": "Swedish", "default": false},
		{"which": "DA", "label": "Danish", "default": false},
		{"which": "NO", "label": "Norwegian", "default": false},
		{"which": "Other", "label": "Other", "default": false},
	},
	"mime": {
		{"which": "0", "label": "Unknown", "default": true},
		{"which": "1", "label": "XHTML Text", "default": true},
		{"which": "2", "label": "XHTML Banner", "default": true},
		{"which": "3", "label": "Javascript", "default": false},
		{"which": "4", "label": "Iframe", "default": true},
	},
	"device": {
		{"which": "0", "label": "Unknown Device", "default": true},
		{"which": "1", "label": "Mobile/Tablet - General", "default": true},
		{"which": "2", "label": "Personal Computer", "default": true},
		{"which": "3", "label": "Connected TV", "default": false},
		{"which": "4", "label": "Phone", "default": true},
		{"which": "5", "label": "Tablet", "default": true},
		{"which": "6", "label": "Connected Device", "default": false},
		{"which": "7", "label": "Set Top Box", "default": false},
	},
	"position": {
		{"which": "0", "label": "Unknown Position", "default": true},
		{"which": "1", "label": "Above The Fold", "default": true},
		{"which": "2", "label": "Locked (i.e., fixed position)", "default": false},
		{"which": "3", "label": "Below The Fold", "default": true},
		{"which": "4", "label": "Header", "default": true},
		{"which": "5", "label": "Footer", "default": true},
		{"which": "6", "label": "Sidebar", "default": true},
		{"which": "7", "label": "Fullscreen", "default": false},
	},
	"expnd": {
		{"which": "0", "label": "Stable", "default": true},
		{"which": "1", "label": "Left", "default": false},
		{"which": "2", "label": "Right", "default": false},
		{"which": "3", "label": "Up", "default": false},
		{"which": "4", "label": "Down", "default": false},
		{"which": "5", "label": "Full Screen", "default": false},
	},
	"creative": {
		{"which": "0", "label": "Normal Creative", "default": true},
		{"which": "1", "label": "Audio Ad (Autoplay)", "default": false},
		{"which": "2", "label": "Audio Ad (User Initiated)", "default": false},
		{"which": "3", "label": "Expandable (Automatic)", "default": false},
		{"which": "4", "label": "Expandable (User Initiated - Click)", "default": false},
		{"which": "5", "label": "Expandable (User Initiated - Rollover)", "default": false},
		{"which": "6", "label": "In-Banner Video Ad (Autoplay)", "default": false},
		{"which": "7", "label": "In-Banner Video Ad (User Initiated)", "default": false},
		{"which": "8", "label": "Pop (e.g., Over, Under, or Upon Exit)", "default": false},
		{"which": "9", "label": "Provocative or Suggestive Imagery", "default": false},
		{"which": "10", "label": "Shaky, Flashing, Flickering, Extreme Animation, Smileys", "default": false},
		{"which": "11", "label": "Surveys", "default": false},
		{"which": "12", "label": "Text Only", "default": false},
		{"which": "13", "label": "User Interactive (e.g., Embedded Games)", "default": false},
		{"which": "14", "label": "Windows Dialog or Alert Style", "default": false},
		{"which": "15", "label": "Has Audio On/Off Button", "default": false},
		{"which": "16", "label": "Ad Provides Skip Button (e.g. VPAID-rendered skip button on pre-roll video)", "default": false},
		{"which": "17", "label": "Adobe Flash", "default": false},
		{"which": "18", "label": "Responsive; Sizeless; Fluid (i.e., creatives that dynamically resize to environment)", "default": false},
	},
}

func SetSizeID(args url.Values) error {
	if args.Get("w") == "" || args.Get("h") == "" {
		return fmt.Errorf("w or h is empty")
	}
	w, err := strconv.Atoi(args.Get("w"))
	if err != nil {
		return err
	}
	h, err := strconv.Atoi(args.Get("h"))
	if err != nil {
		return err
	}
	if w > 65535 || h > 65535 {
		return fmt.Errorf("w or h are over 65535")
	}
	args.Set("size_id", fmt.Sprintf("%d", match.SizeID2To1(uint16(w), uint16(h))))
	return nil
}

func SetWH(item map[string]interface{}) {
	sizeid := item["size_id"].(int64)
	item["w"], item["h"] = match.SizeID1To2(uint32(sizeid))
}

func (self *Filter) Preset() error {
	r := self.R
	ARGS := r.Form

	action := self.Action
	if action == "insert" || action == "insupd" {
		ARGS.Set("ip", self.Base.GetIP())
		if i64, err := strconv.ParseInt(ARGS.Get("_gtime"), 10, 64); err == nil {
			y, m, d := time.Unix(i64, 0).Date()
			ARGS.Set("created", fmt.Sprintf("%d-%d-%d", y, m, d))
		}
		for k, v := range ARGS {
			if k[len(k)-1:] == "_id" {
				isDigit := true
				for _, val := range v {
					if !IsDigit(val) {
						isDigit = false
					}
				}
				if !isDigit {
					ARGS.Del(k)
				}
			}
		}
	} else if action == "topics" {
		if ARGS.Get("pageno") == "" {
			ARGS.Set("pageno", "1")
		}
		if ARGS.Get("rowcount") == "" {
			ARGS.Set("rowcount", "100")
		}
		if ARGS.Get("sortreverse") == "" {
			ARGS.Set("sortreverse", "1")
		}
	}

	return nil
}

func (self *Filter) BalanceBefore(model *Model) error {
	ARGS := self.R.Form

	total := url.Values{}
	for _, name := range []string{"limit_spend", "limit_imp", "limit_cli"} {
		value := ARGS.Get(name)
		if value != "" && value != "0" && value != "0.0" {
			total.Set(name, value)
		}
	}
	if len(total) > 0 {
		if err := model.InsertHash(total); err != nil {
			return err
		}
		ARGS.Set("total_balance_id", strconv.FormatInt(model.LastID, 10))
	}
	daily := url.Values{}
	for name, v := range map[string]string{"daily_spend": "limit_spend", "daily_imp": "limit_imp", "daily_cli": "limit_cli"} {
		value := ARGS.Get(name)
		if value != "" && value != "0" && value != "0.0" {
			daily.Set(v, value)
		}
	}
	if len(daily) > 0 {
		if err := model.InsertHash(daily); err != nil {
			return err
		}
		ARGS.Set("daily_balance_id", strconv.FormatInt(model.LastID, 10))
	}

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	return nil
}

func (self *Filter) AfterItemSet(name, str string) []map[string]interface{} {
	values := make([]string, 0)
	if len(str) > 0 {
		values = strings.Split(str, ",")
	}

	lists := make([]map[string]interface{}, 0)
	for _, item := range LARGES[name] {
		if Grep(values, item["which"].(string)) {
			item["selected"] = true
		} else {
			item["selected"] = false
		}
		lists = append(lists, item)
	}

	return lists
}

func (self *Filter) After(model *Model) error {
	ARGS := self.R.Form
	other := *model.OTHER

	who := self.RoleValue
	action := self.Action
	obj := ARGS.Get("_gobj")

	if genelet.Grep([]string{"item", "slot"}, obj) && genelet.Grep([]string{"startnew", "edit"}, action) {
		other["slotsChinese"] = Translate(other["slots"])
		other["slotsDefault"] = CreateSlot("", "", "", "", "", "", "", "", "", "", "").InHash()
		other["slotAttrs"] = GetSlotAttrs()
		other["slotAttrsChinese"] = Translate(other["slotAttrs"])
		other["items"] = GetItemNames()
		other["itemsChinese"] = Translate(other["items"])
		other["itemsDefault"] = CreateItem("", "", "", "", "", "").InHash()
		other["itemAttrs"] = GetItemAttrs()
		other["itemAttrsChinese"] = Translate(other["itemAttrs"])
	}

	// creative caches belong to a specific campaign, item or creative are published
	if genelet.Grep([]string{"item", "campaign", "creative"}, obj) && genelet.Grep([]string{"insert", "update"}, action) {
		var par []string
		switch obj {
		case "item":
			par = []string{"item_id", ARGS.Get("item_id")}
		case "campaign":
			par = []string{"campaign_id", ARGS.Get("campaign_id")}
		case "creative":
			par = []string{"creative_id", ARGS.Get("creative_id")}
		default:
		}
		if nc, ok := model.Storage["Nc"]; ok && nc != nil {
			if err := match.DBGetCreativesToRedisSpread(context.Background(), nc.(*nats.Conn), model.DB, par...); err != nil {
				return err
			}
		}
		if redis, ok := model.Storage["Redis"]; ok && redis != nil {
			if err := match.DBGetCreativesToRedisSpread(context.Background(), redis.(radix.Client), model.DB, par...); err != nil {
				return err
			}
		}
	}

	var itemID uint32
	// audience of item is published
	if (obj == "targetname" && action == "insert") ||
		(who == "admin" && obj == "item" && action == "update") {
		if str := ARGS.Get("item_id"); str != "" {
			id, err := strconv.Atoi(str)
			if err != nil {
				return fmt.Errorf("item_id should be a number")
			}
			itemID = uint32(id)
		} else {
			return fmt.Errorf("item_id is empty")
		}
	}

	if obj == "targetname" && action == "insert" {
		aud, err := match.DBGetAudience(model.DB, itemID)
		if err == nil && aud != nil {
			if nc, ok := model.Storage["Nc"]; ok && nc != nil {
				err = aud.ToSpread(nc.(*nats.Conn), itemID)
			}
			if redis, ok := model.Storage["Redis"]; ok && redis != nil {
				err = aud.ToRedis(context.Background(), redis.(radix.Client), itemID)
			}
		}
		if err != nil {
			return fmt.Errorf("failed to update audience: %s", err.Error())
		}
	}

	if who == "admin" && obj == "item" && action == "update" {
		var err error
		if nc, ok := model.Storage["Nc"]; ok && nc != nil {
			top := model.Storage["Spread"].(string)
			switch ARGS.Get("how") {
			case "Get":
				err = match.DBGetRAdvsToRedisSpreadByItemID(context.Background(), nc.(*nats.Conn), model.DB, itemID, top)
			case "Delete":
				err = match.DBDeleteRAdvsToRedisSpreadByItemID(context.Background(), nc.(*nats.Conn), model.DB, itemID, top)
			}
		}
		if redis, ok := model.Storage["Redis"]; ok && redis != nil {
			switch ARGS.Get("how") {
			case "Get":
				err = match.DBGetRAdvsToRedisSpreadByItemID(context.Background(), redis.(radix.Client), model.DB, itemID)
			case "Delete":
				err = match.DBDeleteRAdvsToRedisSpreadByItemID(context.Background(), redis.(radix.Client), model.DB, itemID)
			}
		}
		if err != nil {
			return fmt.Errorf("failed to update advs: %s", err.Error())
		}

	}

	return nil
}

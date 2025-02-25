package summer

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/genelet/winter/genelet"
	"github.com/genelet/winter/pzutil"
	"github.com/mediocregopher/radix/v4"
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
	"mime": {
		{"which": "MimeUnknown", "label": "Unknown", "default": true},
		{"which": "JSMime", "label": "Javascript", "default": false},
		{"which": "Iframe", "label": "Iframe", "default": true},
		{"which": "XHTMLText", "label": "MobileText", "default": true},
		{"which": "XHTMLBanner", "label": "MobileHtml", "default": true},
	},
	"device": {
		{"which": "DeviceUnknown", "label": "Unknown Device", "default": true},
		{"which": "DeviceMobile", "label": "Mobile/Tablet - General", "default": true},
		{"which": "DevicePC", "label": "Personal Computer", "default": true},
		{"which": "DeviceTV", "label": "Connected TV", "default": false},
		{"which": "DevicePhone", "label": "Phone", "default": true},
		{"which": "DeviceTablet", "label": "Tablet", "default": true},
		{"which": "DeviceConnected", "label": "Connected Device", "default": false},
		{"which": "DeviceSetTopBox", "label": "Set Top Box", "default": false},
	},
	"position": {
		{"which": "PositionUnknown", "label": "Unknown Position", "default": true},
		{"which": "PositionAboveFold", "label": "Above The Fold", "default": true},
		{"which": "PositionLocked", "label": "Locked (i.e., fixed position)", "default": false},
		{"which": "PositionBelowFold", "label": "Below The Fold", "default": true},
		{"which": "PositionHeader", "label": "Header", "default": true},
		{"which": "PositionFooter", "label": "Footer", "default": true},
		{"which": "PositionSideBar", "label": "Sidebar", "default": true},
		{"which": "PositionFullScreen", "label": "Fullscreen", "default": false},
	},
	"content": {
		{"which": "ContentUnknown", "label": "Unknown Content", "default": true},
		{"which": "ContentVideo", "label": "Video (i.e., video file or stream such as Internet TV broadcasts)", "default": true},
		{"which": "ContentGame", "label": "Game (i.e., an interactive software game)", "default": false},
		{"which": "ContentMusic", "label": "Music (i.e., audio file or stream such as Internet radio broadcasts)", "default": false},
		{"which": "ContentApp", "label": "Application (i.e., an interactive software application)", "default": true},
		{"which": "ContentText", "label": "Text (i.e., primarily textual document such as a web page, eBook or a news article)", "default": true},
		{"which": "ContentOther", "label": "Other (i.e., none of the other categories applies)", "default": true},
	},
	"creative": {
		{"which": "AttrUnknown", "label": "Normal Creative", "default": true},
		{"which": "AttrAudioAuto", "label": "Audio Ad (Autoplay)", "default": false},
		{"which": "AttrAudioUser", "label": "Audio Ad (User Initiated)", "default": false},
		{"which": "AttrExpandableAuto", "label": "Expandable (Automatic)", "default": false},
		{"which": "AttrExpandableUserClick", "label": "Expandable (User Initiated - Click)", "default": false},
		{"which": "AttrExpandableUserRollover", "label": "Expandable (User Initiated - Rollover)", "default": false},
		{"which": "AttrVideoAuto", "label": "In-Banner Video Ad (Autoplay)", "default": false},
		{"which": "AttrVideoUser", "label": "In-Banner Video Ad (User Initiated)", "default": false},
		{"which": "AttrPop", "label": "Pop (e.g., Over, Under, or Upon Exit)", "default": false},
		{"which": "AttrProvocative", "label": "Provocative or Suggestive Imagery", "default": false},
		{"which": "AttrExtremeAnimation", "label": "Shaky, Flashing, Flickering, Extreme Animation, Smileys", "default": false},
		{"which": "AttrSurvey", "label": "Surveys", "default": false},
		{"which": "AttrTextOnly", "label": "Text Only", "default": false},
		{"which": "AttrInteractive", "label": "User Interactive (e.g., Embedded Games)", "default": false},
		{"which": "AttrWindowsDialog", "label": "Windows Dialog or Alert Style", "default": false},
		{"which": "AttrHasAudioToggleButton", "label": "Has Audio On/Off Button", "default": false},
		{"which": "AttrHasSkipButton", "label": "Ad Provides Skip Button (e.g. VPAID-rendered skip button on pre-roll video)", "default": false},
		{"which": "AttrFlash", "label": "Adobe Flash", "default": false},
		{"which": "AttrResponsive", "label": "Responsive; Sizeless; Fluid (i.e., creatives that dynamically resize to environment)", "default": false},
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
	args.Set("size_id", fmt.Sprintf("%d", pzutil.GetSizeID(uint16(w), uint16(h))))
	return nil
}

func SetWH(item map[string]interface{}) {
	sizeid := item["size_id"].(int64)
	item["w"], item["h"] = pzutil.GetSizes(uint32(sizeid))
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
					if !pzutil.IsDigit(val) {
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

	action := self.Action
	obj := ARGS.Get("_gobj")

	if genelet.Grep([]string{"campaign", "site"}, obj) && genelet.Grep([]string{"startnew", "edit"}, action) {
		other["sites"] = GetSiteNames()
		other["sitesChinese"] = Translate(other["sites"])
		other["sitesDefault"] = CreateSite("", "", "", "", "", "", "", "", "", "", "").InHash()
		other["siteAttrs"] = GetSiteAttrs()
		other["siteAttrsChinese"] = Translate(other["siteAttrs"])
		other["campaigns"] = GetCampaignNames()
		other["campaignsChinese"] = Translate(other["campaigns"])
		other["campaignsDefault"] = CreateCampaign("", "", "", "", "", "").InHash()
		other["campaignAttrs"] = GetCampaignAttrs()
		other["campaignAttrsChinese"] = Translate(other["campaignAttrs"])
	}

	var err error
	if (obj == "site" && (action == "delete" || action == "update")) ||
		(obj == "chac" && ARGS.Get("entitytype_id") == "31" && action == "update") {
		siteid := ARGS.Get("site_id")
		if siteid == "" && ARGS.Get("entitytype_id") == "31" {
			siteid = ARGS.Get("entity_id")
		}
		conn := (model.Storage)["Redis"].(radix.Client)
		c := (model.Storage)["Ssp"].(*pzutil.Config)
		err = conn.Do(context.Background(), radix.Cmd(nil, "DEL", c.SITE+":"+siteid))
	} else if obj == "slot" && (action == "delete" || action == "update") {
		slotid := ARGS.Get("slot_id")
		if err = model.DoSQL("DELETE FROM pub_weight WHERE slot_id=?", slotid); err == nil {
			conn := (model.Storage)["Redis"].(radix.Client)
			c := (model.Storage)["Ssp"].(*pzutil.Config)
			err = conn.Do(context.Background(), radix.Cmd(nil, "DEL", c.SLOT+":"+slotid))
		}
	} else if (obj == "targetname" && (action == "delete" || action == "insert")) ||
		(obj == "campaign" && (action == "delete" || action == "update")) {
		campaignid := ARGS.Get("campaign_id")
		conn := (model.Storage)["Redis"].(radix.Client)
		c := (model.Storage)["Ssp"].(*pzutil.Config)
		err = conn.Do(context.Background(), radix.Cmd(nil, "HDEL", c.AUDIENCE, campaignid))
	} else if (obj == "item" || obj == "creative") && (action == "delete" || action == "update") {
		itemid := ARGS.Get("item_id")
		p := (model.Storage)["Redis"].(radix.Client)
		c := (model.Storage)["Ssp"].(*pzutil.Config)
		err = p.Do(context.Background(), radix.Cmd(nil, "HDEL", c.ITEM, itemid))
	}

	return err
}

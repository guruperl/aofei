package pub

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strconv"

	"github.com/genelet/winter/acl"
	"github.com/genelet/winter/genelet"
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
	if ARGS.Get("_gadmin") == "1" {
		who = "admin"
	}

	if (who == "pub" && action == "updatepass") ||
		(who == "web" && (action == "insert" || action == "resetpass")) {
		if ARGS.Get("firstname") == "" {
			ARGS.Set("firstname", ARGS.Get("lastname"))
		}
		if ARGS.Get("passwd") == ARGS.Get("confirm") {
			h := sha1.New()
			h.Write([]byte(ARGS.Get("passwd") + ARGS.Get("email")))
			ARGS.Set("passwd", hex.EncodeToString(h.Sum(nil)))
			ARGS.Del("confirm")
		} else {
			return genelet.Err(3102)
		}
	}

	if who == "web" && (action == "activate" || action == "startreset" || action == "resetpass") {
		if ARGS.Get("md5") != genelet.Digest(self.C.Secret, ARGS.Get("pub_id"), ARGS.Get("email"), ARGS.Get("stamp"), ARGS.Get("firstname"), ARGS.Get("lastname")) {
			return genelet.Err(3102)
		}
	} else if who == "admin" && action == "insert" {
		// needed for validation but not actually passed to the db
		for _, str := range []string{"email", "passwd", "firstname", "lastname", "address_id", "active", "access_order"} {
			ARGS.Set(str, "1")
		}
	} else if who != "admin" && action == "update" {
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

	action := self.Action
	who := self.RoleValue
	ARGS := self.R.Form
	if ARGS.Get("_gadmin") == "1" {
		who = "admin"
	}

	if who == "admin" && action == "insert" {
		p, err := acl.AddPub(model.DB, ARGS.Get("domain"))
		if err != nil {
			return err
		}
		ARGS.Set("pub_id", strconv.FormatInt(int64(p.PubID), 10))
		dbi := genelet.DBI{DB: model.DB}
		err = dbi.DoSQL(`
INSERT INTO adv_balance (limit_imp, created) VALUES (?, NOW())`, ARGS.Get("limit_imp"))
		if err != nil {
			return err
		}
		ARGS.Set("total_balance_id", strconv.FormatInt(dbi.LastID, 10))
	} else if who == "web" && action == "insert" {
		if err := model.Randomid("pub", "pub_id", 0, 16777216, 10); err != nil {
			return err
		}
	} else if who != "admin" && action == "update" {
		model.CurrentTable = "adv_balance"
		if err := self.BalanceBefore(&model.Model); err != nil {
			return err
		}
		model.CurrentTable = "pub"
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
		for _, item := range lists {
			created := item["created"].(string)
			item["created"] = created[:len(created)-9]
		}
	} else if who == "web" && action == "insert" {
		email := ARGS.Get("email")
		ARGS.Set("stamp", ARGS.Get("_gtime"))
		ARGS.Set("md5", genelet.Digest(self.C.Secret, ARGS.Get("pub_id"), email, ARGS.Get("stamp"), ARGS.Get("firstname"), ARGS.Get("lastname")))
		ARGS.Set("serverUrl", self.C.ServerURL)
		other["_gmail"] = map[string]interface{}{
			"To":      email,
			"Subject": "Welcome to Aofei Publisher Service",
			"file":    self.C.Template + "/" + who + "/pub/insert.mail." + self.ChartagValue}
	} else if who == "web" && action == "retrieve" && len(lists) > 0 {
		item := lists[0]
		email := item["email"].(string)
		pubID := strconv.FormatInt(item["pub_id"].(int64), 10)
		ARGS.Set("stamp", ARGS.Get("_gtime"))
		ARGS.Set("md5", genelet.Digest(self.C.Secret, pubID, email, ARGS.Get("stamp"), item["firstname"].(string), item["lastname"].(string)))
		ARGS.Set("serverUrl", self.C.ServerURL)
		other["_gmail"] = map[string]interface{}{
			"To":      email,
			"Subject": "Reset Aofei Publisher Password",
			"file":    self.C.Template + "/" + who + "/pub/retrieve.mail." + self.ChartagValue}
	}

	return nil
}

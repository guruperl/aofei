package summer

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestModel(t *testing.T) {
	c, err := genelet.NewConfig("../conf/gotest.json")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}

	model := new(Model)
	model.DB = db
	model.CurrentTable = "testing"
	model.SORTBY = "sortby"
	model.SORTREVERSE = "sortreverse"
	model.PAGENO = "pageno"
	model.ROWCOUNT = "rowcount"
	model.TOTALNO = "totalno"
	model.MAXPAGENO = "max_pageno"
	model.FIELD = "field"
	model.EMPTIES = "empties"

	ret := model.ExecSQL(`drop table if exists testing_f`)
	if ret != nil {
		t.Errorf("create table testing_f failed %s", ret.Error())
	}
	ret = model.ExecSQL(`drop table if exists testing`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}
	ret = model.ExecSQL(`CREATE TABLE testing (id int(10) unsigned NOT NULL, email varchar(255) not null, address_id int(10) unsigned DEFAULT NULL, active enum('Yes','No','New') default 'New', primary key (id))`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}

	add := new(Model)
	comp := genelet.NewComponent("address/component.json")
	genelet.Invoke0(add, "Initialize", comp)
	storage := map[string]interface{}{"address": add}

	args := make(url.Values)
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	extra := []url.Values{{}}
	model.SetDefaults(args, &lists, &other, storage)

	model.CurrentKey = "id"
	model.InsertPars = []string{"id", "email", "address_id"}
	model.EditPars = []string{"id", "email", "address_id", "active"}
	model.UpdatePars = []string{"id", "email", "address_id"}

	args["email"] = []string{"a_email"}
	args["contact"] = []string{"b_contact"}
	args["contact_email"] = args["email"]
	args["company"] = []string{"b_company"}

	err = model.Randomid("testing", "id", 100, 200, 10)
	if err != nil {
		t.Fatal(err)
	}
	err = model.Insert(extra...)
	if err != nil {
		t.Fatal(err)
	}
	result := other["address_insert"].([]map[string]interface{})
	if result[0]["company"].(string) != "b_company" {
		t.Errorf("%v", other)
	}
	addressID := result[0]["address_id"].(string)
	address := lists[0]["address_id"].(string)
	if addressID != address {
		t.Errorf("%s", addressID)
		t.Errorf("%s", address)
		t.Errorf("%v", lists)
	}

	lists = make([]map[string]interface{}, 0)
	extra = []url.Values{{}}
	err = model.Edit(extra...)
	if err != nil {
		t.Fatal(err)
	}
	if lists[0]["contact"].(string) != "b_contact" {
		t.Errorf("%v", lists)
	}

	err = model.Activate(extra...)
	if err == nil {
		lists = make([]map[string]interface{}, 0)
		err = model.Edit(extra...)
	}
	if err != nil {
		t.Fatal(err)
	}
	if lists[0]["active"].(string) != "Yes" {
		t.Errorf("%v", lists)
	}

	lists = make([]map[string]interface{}, 0)
	extra = []url.Values{{}}
	args["email"] = []string{"c_email"}
	args["contact"] = []string{"c_contact"}
	err = model.Update(extra...)
	if err == nil {
		err = model.Edit(extra...)
	}
	if err != nil {
		t.Fatal(err)
	}
	if lists[0]["contact"].(string) != "c_contact" || lists[0]["email"].(string) != "c_email" {
		t.Errorf("%v", lists)
	}

	if _, err = db.Exec("DROP TABLE testing"); err == nil {
		if _, err = db.Exec("DELETE FROM add_address WHERE address_id>=21"); err == nil {
			_, err = db.Exec("ALTER TABLE add_address AUTO_INCREMENT=21")
		}
	}
	if err != nil {
		t.Errorf("%v", err)
	}

	model.CurrentTable = "pub_slot"

	storage = make(map[string]interface{})

	args = make(url.Values)
	lists = make([]map[string]interface{}, 0)
	other = make(map[string]interface{})
	extra = []url.Values{{}}
	model.SetDefaults(args, &lists, &other, storage)

	model.CurrentKey = "slot_id"
	model.EditPars = []string{"slot_id", "site_id", "slot_name", "size_id", "qa_device", "qa_position", "qa_content", "mychannel", "channel_order", "created", "active"}

	args["slot_id"] = []string{"125"}
	err = model.Edit(extra...)
	if err != nil {
		t.Fatal(err)
	}
	one := lists[0]
	if one["size_id"].(int64) != 5 ||
		one["active"].(string) != "Yes" ||
		one["slot_name"].(string) != "slot 125" ||
		one["channel_order"].(string) != "Inherit" ||
		one["mychannel"].(string) != "Inherit" ||
		one["slot_id"].(int64) != 125 ||
		one["site_id"].(int64) != 25 {
		t.Errorf("%v", lists)
	}
}

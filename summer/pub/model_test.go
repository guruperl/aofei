package pub

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestModel(t *testing.T) {
	c := genelet.NewConfig("../../../conf/config.json")
	db, err := sql.Open(c.Db[0], c.Db[1])
	if err != nil {
		panic(err)
	}

	comp := genelet.NewComponent("component.json")

	model := new(Model)
	model.Db = db
	add := new(Model)
	add.Db = db
	model.Initialize(comp)
	add.Initialize(comp)

	storage := map[string]interface{}{"pub": add}

	ret := model.Exec_sql(`drop table if exists testing`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}
	ret = model.Exec_sql(`CREATE TABLE testing (id int(10) unsigned NOT NULL, email varchar(255) not null, address_id int(10) unsigned DEFAULT NULL, active enum('Yes','No','New') default 'New', primary key (id), FOREIGN KEY (address_id) REFERENCES add_address (address_id) ON UPDATE CASCADE)`)
	if ret != nil {
		t.Errorf("create table testing failed %s", ret.Error())
	}

	args := make(url.Values)
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	extra := []url.Values{{}}
	model.Set_defaults(args, &lists, &other, storage)

	model.Current_key = "id"
	model.Insert_pars = []string{"id", "email", "address_id"}
	model.Edit_pars = []string{"id", "email", "address_id", "active"}
	model.Update_pars = []string{"id", "email", "address_id"}

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
	result := other["summer_Model_Insert"].([]map[string]interface{})
	if result[0]["company"].(string) != "b_company" {
		t.Errorf("%v", other)
	}
	address_id := result[0]["address_id"].(string)
	address := lists[0]["address_id"].(string)
	if address_id != address {
		t.Errorf("%s", address_id)
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
	args["id"] = []string{"160"}
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
}

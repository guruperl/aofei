package slot

import (
    "testing"
    "database/sql"
)

func TestModel(t *testing.T) {
    db, err := sql.Open("taosSql", "root:taosdata@/tcp(127.0.0.1:0)/holiday?parseTime=false")
    if err != nil { panic(err) }
	model := new(Model)
	err = model.Load("component.json")
    if err != nil { panic(err) }

	model.Db = db

	release := 1
	err = model.ExecSQL(`drop table if exists slot_1`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists slot`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`CREATE TABLE slot (ts timestamp, slot_id int, adv_id int, campaign_id int, item_id int, cost_type tinyint, price float, endx int, cpm_fc smallint, cpm_length int, cpm_throttle int, cpc_fc smallint, cpc_length int) TAGS (release int)`)
	if err != nil { panic(err) }

	lists := []map[string]interface{}{
{"slot_id":111,"adv_id":2,"campaign_id":22,"item_id":222,"cost_type":1,"price":1.11},
{"slot_id":111,"adv_id":3,"campaign_id":33,"item_id":333,"cost_type":2,"price":2.22,"endx":1600000000},
{"slot_id":111,"adv_id":4,"campaign_id":44,"item_id":444,"cost_type":3,"price":3.33,"endx":1600000000},
{"slot_id":555,"adv_id":2,"campaign_id":22,"item_id":222,"cost_type":1,"price":1.11},
{"slot_id":555,"adv_id":3,"campaign_id":33,"item_id":333,"cost_type":2,"price":2.22,"endx":1600000000},
{"slot_id":555,"adv_id":4,"campaign_id":44,"item_id":666,"cost_type":3,"price":3.33,"endx":1600000000},
}
	ARGS := map[string]interface{}{"release":release}
	model.ARGS = ARGS
	err = model.CreateTable()
	if err != nil { panic(err) }
	err = model.Inserts(lists)
	if err != nil { panic(err) }

	model.ARGS = ARGS
	model.LISTS = make([]map[string]interface{},0)
	err = model.ReleaseTopics(map[string]interface{}{"slot_id":111})
	if err != nil { panic(err) }

	results := model.LISTS
	if len(results) != 3 {
		t.Errorf("%v", len(results))
	}
	if results[0]["adv_id"].(int) != 2 || results[0]["price"].(float32) != 1.11 {
		t.Errorf("%T,%v", results[0]["adv_id"], results[0]["adv_id"])
		t.Errorf("%T,%v", results[0]["price"], results[0]["price"])
	}
	if results[1]["adv_id"].(int) != 3 || results[1]["price"].(float32) != 2.22 {
		t.Errorf("%T,%v", results[1]["adv_id"], results[1]["adv_id"])
		t.Errorf("%T,%v", results[1]["price"], results[1]["price"])
	}
	if results[2]["adv_id"].(int) != 4 || results[2]["price"].(float32) != 3.33 {
		t.Errorf("%T,%v", results[2]["adv_id"], results[2]["adv_id"])
		t.Errorf("%T,%v", results[2]["price"], results[2]["price"])
	}

	db.Close()
}

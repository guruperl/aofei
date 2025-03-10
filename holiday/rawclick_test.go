package holiday

import (
	"time"
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

	err = model.ExecSQL(`drop table if exists rawclick_111_4`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_222_4`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_333_4`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_111_5`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_222_5`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_333_5`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_111_6`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_222_6`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick_333_6`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawclick`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`CREATE TABLE rawclick (click_id timestamp, imp_id timestamp, raw_id timestamp, user_id timestamp, slot_id int, creative_id int, item_id int, cost_type tinyint, price float, campaign_id int, adv_id int) TAGS (pub_id int, site_id int)`)
	if err != nil { panic(err) }

	h1 := map[string]interface{}{"raw_id":7,"user_id":8,"slot_id":4,"creative_id":44444,"item_id":4444,"campaign_id":444,"adv_id":44,"cost_type":1,"price":4.44}
	h2 := map[string]interface{}{"raw_id":7,"user_id":8,"slot_id":5,"creative_id":55555,"item_id":5555,"campaign_id":555,"adv_id":55,"cost_type":2,"price":5.55}
	h3 := map[string]interface{}{"raw_id":7,"user_id":8,"slot_id":6,"creative_id":66666,"item_id":6666,"campaign_id":666,"adv_id":66,"cost_type":1,"price":6.66}

	for _, pid := range []int{111,222,333} {
		for _, iid := range []int{777,888,999} {
			ARGS := map[string]interface{}{"imp_id":iid, "pub_id":pid}
			ARGS["site_id"] = 40
			model.ARGS = ARGS
			err = model.Insert(h1)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
			ARGS["site_id"] = 50
			model.ARGS = ARGS
			err = model.Insert(h2)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
			ARGS["site_id"] = 60
			model.ARGS = ARGS
			err = model.Insert(h3)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
		}
	}

    model.ARGS = make(map[string]interface{})
    err = model.Topics(map[string]interface{}{"pub_id":111})
    if err != nil { panic(err) }

    results := model.LISTS
    if len(results) != 9 {
        t.Errorf("%v", results)
	}
	if results[0]["imp_id"].(int64) != 777 ||
		results[0]["pub_id"].(int)  != 111 ||
		results[0]["slot_id"].(int) != 4   ||
		results[0]["site_id"].(int) != 40  ||
		results[0]["item_id"].(int) != 4444||
		results[0]["creative_id"].(int) != 44444 {
        t.Errorf("%v", results[0])
    }

	db.Close()
}

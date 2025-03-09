package target

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

	err = model.ExecSQL(`drop table if exists target_111`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists target_222`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists target_333`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists target`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`CREATE TABLE target (ts timestamp, attrname_id int, value_id int) TAGS (campaign_id int)`)
	if err != nil { panic(err) }

	h1 := map[string]interface{}{"value_id":1}
	h2 := map[string]interface{}{"value_id":2}
	h3 := map[string]interface{}{"value_id":3}

	for _, cid := range []int{111,222,333} {
		for _, aid := range []int{81,82,83} {
			ARGS := map[string]interface{}{"attrname_id":aid, "campaign_id":cid}
			model.ARGS = ARGS
			err = model.Insert(h1)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
			err = model.Insert(h2)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
			err = model.Insert(h3)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
		}
	}

    model.ARGS = make(map[string]interface{})
    err = model.Topics(map[string]interface{}{"campaign_id":111})
    if err != nil { panic(err) }

    results := model.LISTS
    if len(results) != 9 {
        t.Errorf("%v", results)
	}
	if results[0]["attrname_id"].(int) != 81 ||
		results[0]["campaign_id"].(int) != 111 ||
		results[0]["value_id"].(int) != 1 {
        t.Errorf("%v", results)
    }

	db.Close()
}

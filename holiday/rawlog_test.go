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

	err = model.ExecSQL(`drop table if exists rawlog_111_4`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_111_5`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_111_6`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_222_4`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_222_5`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_222_6`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_333_4`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_333_5`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog_333_6`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`drop table if exists rawlog`)
	if err != nil { panic(err) }
	err = model.ExecSQL(`CREATE TABLE rawlog (raw_id timestamp, user_id timestamp, ip32 int, pzua int, tag0 int, tag1 int, tag2 int, tag3 int, tag4 int, tag5 int, tag6 int, tag7 int, tag8 int, tag9 int) TAGS (pub_id int, site_id int)`)
	if err != nil { panic(err) }

	h1 := map[string]interface{}{"ip32":44,"pzua":444,"tag0":4444,"tag1":44444}
	h2 := map[string]interface{}{"ip32":55,"pzua":555,"tag0":5555,"tag1":55555}
	h3 := map[string]interface{}{"ip32":66,"pzua":666,"tag0":6666,"tag1":66666}

	for _, pid := range []int{111,222,333} {
		for _, uid := range []int{777,888,999} {
			ARGS := map[string]interface{}{"user_id":uid, "pub_id":pid}
			ARGS["site_id"] = 4
			model.ARGS = ARGS
			err = model.Insert(h1)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
			ARGS["site_id"] = 5
			model.ARGS = ARGS
			err = model.Insert(h2)
			if err != nil { panic(err) }
			time.Sleep(1 * time.Millisecond)
			ARGS["site_id"] = 6
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
	if results[0]["user_id"].(int64) != 777 ||
		results[0]["pub_id"].(int) != 111 ||
		results[0]["site_id"].(int) != 4 ||
		results[0]["tag0"].(int) != 4444 ||
		results[0]["tag1"].(int) != 44444 {
        t.Errorf("%v", results[0])
        t.Errorf("%v", results[1])
    }

	db.Close()
}

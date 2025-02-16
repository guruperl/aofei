package weight

import (
	"database/sql"
	"net/url"
	"strconv"

	"github.com/genelet/winter/genelet"
	_ "github.com/go-sql-driver/mysql"
)

func SimulateWeight(config, component string) {
	c := genelet.NewConfig(config)
	db, err := sql.Open(c.Db[0], c.Db[1])
	if err != nil {
		panic(err)
	}

	model := new(Model)
	model.Db = db
	model.Current_table = "pub_weight"
	model.Current_key = "weight_id"
	model.Initialize(genelet.NewComponent(component))

	add := new(Model)
	add.Initialize(genelet.NewComponent(component))
	storage := map[string]interface{}{"address": add}

	args := make(url.Values)
	other := make(map[string]interface{})
	extra := []url.Values{url.Values{}}

	err = model.MakeViewsForSlotItem()
	if err != nil {
		panic(err)
	}

	err = model.Do_sql(`TRUNCATE pub_weight`)
	for i := 1; i <= 250; i++ {
		lists := make([]map[string]interface{}, 0)
		args.Set("slot_id", strconv.Itoa(i))
		model.Set_defaults(args, &lists, &other, storage)
		if err := model.Startnew(extra...); err != nil {
			panic(err)
		}
		for j, item := range lists {
			if err := model.Do_sql(`
INSERT INTO pub_weight (slot_id, item_id, weight, created) VALUES 
(?, ?, ?, NOW())`, args.Get("slot_id"), item["item_id"], j+1); err != nil {
				panic(err)
			}
		}
	}

	db.Close()
}

package main

import (
	"database/sql"
	"net/url"

	"github.com/genelet/winter/genelet"
	"github.com/genelet/winter/summer/weight"

	_ "github.com/go-sql-driver/mysql"
)

func getModel(db *sql.DB, component string) (*weight.Model, map[string]interface{}, error) {
	model := new(weight.Model)
	model.Db = db
	model.Initialize(genelet.NewComponent(component))
	add := new(weight.Model)
	add.Initialize(genelet.NewComponent(component))
	storage := map[string]interface{}{"weight": add}

	return model, storage, nil
}

func main() {
	config := "../../../conf/summer.json"
	c, err := genelet.NewConfig(config)
	if err != nil {
		panic(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		panic(err)
	}
	defer db.Close()
	component := "../component.json"
	err = weight.SimulateWeight(db, component)
	if err != nil {
		panic(err)
	}

	model, storage, err := getModel(db, component)
	if err != nil {
		panic(err)
	}
	args := make(url.Values)
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	extra := []url.Values{{}}
	model.Set_defaults(args, &lists, &other, storage)
	epars := model.Edit_pars
	if model.Current_key != "weight_id" ||
		epars[0] != "weight_id" ||
		epars[1] != "slot_id" ||
		epars[2] != "item_id" ||
		epars[3] != "weight" {
		panic(model)
	}

	args["slot_id"] = []string{"125"}
	err = model.Topics(extra...)
	if err != nil {
		panic(err)
	}
	end := lists[1]
	if end["weight_id"].(int64) != 2 ||
		end["slot_id"].(int64) != 1 ||
		end["item_id"].(int64) != 11 ||
		end["qa_campaign"].(int64) != 149796 ||
		end["campaign_id"].(int64) != 3 {
		panic(end)
	}

	lists = make([]map[string]interface{}, 0)
	args["weight_id"] = []string{"36"}
	err = model.Edit(extra...)
	if err != nil {
		panic(err)
	}
	one := lists[0]
	if one["weight_id"].(int64) != 36 ||
		one["slot_id"].(int64) != 3 ||
		one["item_id"].(int64) != 93 {
		panic(one)
	}

	lists = make([]map[string]interface{}, 0)
	extra[0].Set("slot_id", "125")
	err = model.Topics(extra...)
	if err != nil {
		panic(err)
	}
	grule := lists[1]
	if grule["weight_id"].(int64) != 1554 ||
		grule["slot_id"].(int64) != 125 ||
		grule["campaign_id"].(int64) != 3 ||
		grule["item_id"].(int64) != 15 {
		panic(grule)
	}
}

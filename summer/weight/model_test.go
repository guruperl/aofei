package weight

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/genelet/winter/genelet"
)

func getModel(config, component string) (*Model, map[string]interface{}) {
	c := genelet.NewConfig(config)
	db, err := sql.Open(c.Db[0], c.Db[1])
	if err != nil {
		panic(err)
	}

	model := new(Model)
	model.Db = db
	model.Initialize(genelet.NewComponent(component))
	add := new(Model)
	add.Initialize(genelet.NewComponent(component))
	storage := map[string]interface{}{"weight": add}

	return model, storage
}

func TestModel(t *testing.T) {
	config := "../../conf/summer.json"
	component := "component.json"
	SimulateWeight(config, component)
	model, storage := getModel(config, component)
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
		t.Errorf("%v", model)
	}

	args["slot_id"] = []string{"125"}
	err := model.Topics(extra...)
	if err != nil {
		t.Fatal(err)
	}
	end := lists[1]
	if end["weight_id"].(int64) != 2 ||
		end["slot_id"].(int64) != 1 ||
		end["item_id"].(int64) != 11 ||
		end["qa_campaign"].(int64) != 149796 ||
		end["campaign_id"].(int64) != 3 {
		t.Errorf("%v", end)
	}

	/*
		startnew := other["weight_startnew"].([]map[string]interface{})
		if startnew[0]["cost_type"].(string) != "CPM" ||
			startnew[0]["item_id"].(int64) != 5 ||
			startnew[1]["item_id"].(int64) != 15 ||
			startnew[1]["cost_type"].(string) != "CPC" {
			t.Errorf("%v", startnew[0])
			t.Errorf("%v", startnew[1])
		}
	*/

	lists = make([]map[string]interface{}, 0)
	args["weight_id"] = []string{"36"}
	err = model.Edit(extra...)
	if err != nil {
		t.Fatal(err)
	}
	one := lists[0]
	if one["weight_id"].(int64) != 36 ||
		one["slot_id"].(int64) != 3 ||
		one["item_id"].(int64) != 93 {
		t.Errorf("%v", one)
	}

	lists = make([]map[string]interface{}, 0)
	extra[0].Set("slot_id", "125")
	err = model.Topics(extra...)
	if err != nil {
		t.Fatal(err)
	}
	grule := lists[1]
	if grule["weight_id"].(int64) != 1554 ||
		grule["slot_id"].(int64) != 125 ||
		grule["campaign_id"].(int64) != 3 ||
		grule["item_id"].(int64) != 15 {
		t.Errorf("%v", grule)
	}
}

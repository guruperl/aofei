package weight

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestModel(t *testing.T) {
	c, err := genelet.NewConfig("../../etc/summer.local.json")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var weightCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM pub_weight").Scan(&weightCount); err != nil {
		t.Fatal(err)
	}
	if weightCount == 0 {
		t.Skip("pub_weight has no sample rows in the active Docker baseline")
	}

	model := new(Model)
	comp := genelet.NewComponent("component.json")
	model.DB = db
	model.Initialize(comp)
	add := new(Model)
	add.DB = db
	add.Initialize(comp)
	storage := map[string]interface{}{"slot": add}

	args := make(url.Values)
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	extra := []url.Values{{}}
	model.SetDefaults(args, &lists, &other, storage)

	args["slot_id"] = []string{"125"}
	err = model.Topics(extra...)
	if err != nil {
		t.Fatal(err)
	}
	end := lists[1]
	if end["weight_id"].(int64) != 2 ||
		end["slot_id"].(int64) != 1 ||
		end["item_id"].(int64) != 11 ||
		end["qa_item"].(int64) != 149796 ||
		end["campaign_id"].(int64) != 3 {
		t.Errorf("%#v", end)
	}

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
		t.Errorf("%#v", one)
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
		t.Errorf("%#v", grule)
	}
}

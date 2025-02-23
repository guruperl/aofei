package weight

import (
	"database/sql"
	"net/url"
	"strconv"

	"github.com/genelet/winter/genelet"
)

func SimulateWeight(db *sql.DB, component string) error {
	model := new(Model)
	model.Db = db
	model.Initialize(genelet.NewComponent(component))

	add := new(Model)
	add.Initialize(genelet.NewComponent(component))
	storage := map[string]interface{}{"weight": add}

	args := make(url.Values)
	other := make(map[string]interface{})
	extra := []url.Values{{}}

	err := model.MakeViewsForSlotItem()
	if err != nil {
		return err
	}

	err = model.Do_sql(`TRUNCATE pub_weight`)
	if err != nil {
		return err
	}
	for i := 1; i <= 250; i++ {
		lists := make([]map[string]interface{}, 0)
		args.Set("slot_id", strconv.Itoa(i))
		model.Set_defaults(args, &lists, &other, storage)
		if err := model.Startnew(extra...); err != nil {
			return err
		}
		for j, item := range lists {
			if err := model.Do_sql(`
INSERT INTO pub_weight (slot_id, item_id, weight, created) VALUES 
(?, ?, ?, NOW())`, args.Get("slot_id"), item["item_id"], j+1); err != nil {
				return err
			}
		}
	}

	return nil
}

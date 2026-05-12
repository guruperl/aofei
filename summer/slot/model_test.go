package slot

import (
	"net/url"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestModel(t *testing.T) {
	model := new(Model)
	comp := genelet.NewComponent("component.json")
	model.Initialize(comp)
	add := new(Model)
	add.Initialize(comp)
	storage := map[string]interface{}{"slot": add}

	args := make(url.Values)
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	//    extra := []url.Values{url.Values{}}
	model.SetDefaults(args, &lists, &other, storage)

	if model.Nextpages["edit"][0]["model"] != "chac" {
		t.Errorf("%v\n", model.Nextpages)
	}
}

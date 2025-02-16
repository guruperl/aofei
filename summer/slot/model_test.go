package slot

import (
    "testing"
    "net/url"
	"github.com/genelet/winter/genelet"
)

func TestModel(t *testing.T) {
	model := new(Model)
	comp := genelet.NewComponent("component.json")
	model.Initialize(comp)
    add := new(Model)
	add.Initialize(comp)
    storage := map[string]interface{}{ "slot":add }

    args := make(url.Values)
    lists := make([]map[string]interface{},0)
    other := make(map[string]interface{})
//    extra := []url.Values{url.Values{}}
    model.Set_defaults(args, &lists, &other, storage)

	if model.Nextpages["edit"][0]["model"] != "chbelong" {
		t.Errorf("%v\n", model.Nextpages)
	}
}

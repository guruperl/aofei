package slot

import (
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestFilter(t *testing.T) {
	filter := new(Filter)
	comp := genelet.NewComponent("component.json")
	filter.Initialize(comp)
	filter.Action = "insert"
	filter.Component = "slot"

	filter.Base.C = testSummerConfig(t)
	fks := filter.Fks
	actions := filter.Actions
	if fks["pub"][0] != "site_id" {
		t.Errorf("%v\n", filter.Fks)
	}
	if actions["edit"]["validate"][0] != "slot_id" {
		t.Errorf("%v\n", filter.Actions)
	}
}

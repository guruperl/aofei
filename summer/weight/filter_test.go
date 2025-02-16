package weight

import (
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestFilter(t *testing.T) {
	filter := new(Filter)
	filter.Initialize(genelet.NewComponent("component.json"))
	filter.Action = "insert"
	filter.Component = "weight"
	actions := filter.Actions
	fks := filter.Fks
	if actions["delete"]["validate"][0] != "weight_id" {
		t.Errorf("%v", actions)
	}
	if fks["pub"][0] != "slot_id" || fks["pub"][1] != "slot_md5" {
		t.Errorf("%v", fks)
	}
}

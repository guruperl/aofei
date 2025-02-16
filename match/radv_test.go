package match

import (
	"testing"
)

func TestRAdv(t *testing.T) {
	entry := RAdv{1, 2, 3, 4, 5.6}
	packed, err := entry.Pack()
	if err != nil {
		t.Fatal(err)
	}
	entry1, err := UnpackRAdv(packed)
	if err != nil {
		t.Fatal(err)
	}
	if entry1.AdvID != 1 ||
		entry1.CampaignID != 2 ||
		entry1.ItemID != 3 ||
		entry1.CreativeID != 4 ||
		entry1.Price != 5.6 {
		t.Errorf("%v", entry)
		t.Errorf("%v", entry1)
	}
}

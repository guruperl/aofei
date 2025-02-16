package match

import (
	"testing"
)

func TestRPub(t *testing.T) {
	entry := RPub{1, 2, 3, 4}
	packed, err := entry.Pack()
	if err != nil {
		t.Fatal(err)
	}
	entry1, err := UnpackRPub(packed)
	if err != nil {
		t.Fatal(err)
	}
	if entry1.PubID != 1 ||
		entry1.SiteID != 2 ||
		entry1.SlotID != 3 ||
		entry1.SizeID != 4 {
		t.Errorf("%v", entry)
		t.Errorf("%v", entry1)
	}
}

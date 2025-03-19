package match

import (
	"testing"
)

// TestDemandPack tests the Pack and Unpack functions of Demand.
func TestDemandPack(t *testing.T) {
	demand := Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4}
	packed, err := demand.PackString()
	if err != nil {
		t.Error(err)
	}
	demand2, err := UnpackDemandString(packed)
	if err != nil {
		t.Error(err)
	}
	if demand.AdvID != demand2.AdvID {
		t.Errorf("AdvID %d != %d", demand.AdvID, demand2.AdvID)
	}
	if demand.CampaignID != demand2.CampaignID {
		t.Errorf("CampaignID %d != %d", demand.CampaignID, demand2.CampaignID)
	}
	if demand.ItemID != demand2.ItemID {
		t.Errorf("ItemID %d != %d", demand.ItemID, demand2.ItemID)
	}
	if demand.CreativeID != demand2.CreativeID {
		t.Errorf("CreativeID %d != %d", demand.CreativeID, demand2.CreativeID)
	}
}

func TestRAdvSelectOne(t *testing.T) {
	weights := []float32{0.1, 0.2, 0.3, 0.4}
	probs := make(map[int]int)
	for i := 0; i < 100000; i++ {
		k := selectOne(weights)
		probs[k]++
	}
	if probs[0] < 9000 || probs[0] > 11000 ||
		probs[1] < 19000 || probs[1] > 21000 ||
		probs[2] < 29000 || probs[1] > 31000 ||
		probs[3] < 39000 || probs[1] > 41000 {
		t.Errorf("%v", probs)
	}
}

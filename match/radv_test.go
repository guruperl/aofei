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

// TestRAdvPack tests the Pack and Unpack functions of RAdv.
func TestRAdvPack(t *testing.T) {
	demand := Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4}
	cap := Cap{
		CapNumber:   1,
		CapPeriod:   2,
		CapThrottle: 3,
		ClickNumber: 4,
		ClickPeriod: 5,
	}
	radv := RAdv{
		Demand:   demand,
		Weight:   0.1,
		Cost:     0.2,
		CostType: 3,
		Cap:      cap,
	}
	radv2 := RAdv{}
	radvs := RAdvs([]RAdv{radv, radv2})
	packed, err := radvs.Pack()
	if err != nil {
		t.Error(err)
	}
	radvs2, err := UnpackRAdvs(packed)
	if err != nil {
		t.Error(err)
	}
	radv11 := radvs[0]
	radv12 := radvs2[0]
	if radv11.Demand.AdvID != radv12.Demand.AdvID {
		t.Errorf("AdvID %d != %d", radv11.Demand.AdvID, radv12.Demand.AdvID)
	}
	if radv11.Demand.CampaignID != radv12.Demand.CampaignID {
		t.Errorf("CampaignID %d != %d", radv11.Demand.CampaignID, radv12.Demand.CampaignID)
	}
	if radv11.Demand.ItemID != radv12.Demand.ItemID {
		t.Errorf("ItemID %d != %d", radv11.Demand.ItemID, radv12.Demand.ItemID)
	}
	if radv11.Demand.CreativeID != radv12.Demand.CreativeID {
		t.Errorf("CreativeID %d != %d", radv11.Demand.CreativeID, radv12.Demand.CreativeID)
	}
	if radv11.Weight != radv12.Weight {
		t.Errorf("Weight %f != %f", radv11.Weight, radv12.Weight)
	}
	if radv11.Cost != radv12.Cost {
		t.Errorf("Cost %f != %f", radv11.Cost, radv12.Cost)
	}
	if radv11.CostType != radv12.CostType {
		t.Errorf("CostType %d != %d", radv11.CostType, radv12.CostType)
	}
	if radv11.Cap.CapNumber != radv12.Cap.CapNumber {
		t.Errorf("CapNumber %d != %d", radv11.Cap.CapNumber, radv12.Cap.CapNumber)
	}
	if radv11.Cap.CapPeriod != radv12.Cap.CapPeriod {
		t.Errorf("CapPeriod %d != %d", radv11.Cap.CapPeriod, radv12.Cap.CapPeriod)
	}
	if radv11.Cap.CapThrottle != radv12.Cap.CapThrottle {
		t.Errorf("CapThrottle %d != %d", radv11.Cap.CapThrottle, radv12.Cap.CapThrottle)
	}
	if radv11.Cap.ClickNumber != radv12.Cap.ClickNumber {
		t.Errorf("ClickNumber %d != %d", radv11.Cap.ClickNumber, radv12.Cap.ClickNumber)
	}
	if radv11.Cap.ClickPeriod != radv12.Cap.ClickPeriod {
		t.Errorf("ClickPeriod %d != %d", radv11.Cap.ClickPeriod, radv12.Cap.ClickPeriod)
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

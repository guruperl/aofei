package match

import (
	"database/sql"
	"math"
	"testing"

	"github.com/guruperl/aofei/accounting"
)

func TestRAdvUpdateRowRejectsCapTruncationAndMissingPeriod(t *testing.T) {
	base := RAdv{Demand: Demand{ItemID: 7}}
	null := sql.NullInt64{}
	validCost := sql.NullFloat64{Float64: 1, Valid: true}
	cpm := sql.NullString{String: "CPM", Valid: true}
	for _, test := range []struct {
		name                             string
		capNumber, clickNumber           sql.NullInt64
		capPeriod, clickPeriod, throttle sql.NullInt64
	}{
		{name: "number without period", capNumber: sql.NullInt64{Int64: 1, Valid: true}},
		{name: "number overflow", capNumber: sql.NullInt64{Int64: 256, Valid: true}},
		{name: "period overflow", capPeriod: sql.NullInt64{Int64: 65536, Valid: true}},
		{name: "negative throttle", throttle: sql.NullInt64{Int64: -1, Valid: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := base.updateRow(validCost, test.capNumber, test.clickNumber, test.capPeriod, test.clickPeriod, test.throttle, cpm)
			if err == nil {
				t.Fatal("invalid database cap compiled")
			}
		})
	}
	if _, err := base.updateRow(validCost, sql.NullInt64{Int64: 1, Valid: true}, null, sql.NullInt64{Int64: 60, Valid: true}, null, null, cpm); err != nil {
		t.Fatalf("valid database cap rejected: %v", err)
	}
}

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
		CostCPM:  200_000,
		CostType: CostTypeCPM,
		Cap:      cap,
	}
	radvs := RAdvs([]RAdv{radv})
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
		probs[2] < 29000 || probs[2] > 31000 ||
		probs[3] < 39000 || probs[3] > 41000 {
		t.Errorf("%v", probs)
	}
}

func TestSelectOneAllZeroReturnsNoMatch(t *testing.T) {
	if got := selectOne([]float32{0, 0}); got != -1 {
		t.Fatalf("selectOne all zero = %d, want -1", got)
	}
}

func TestSelectOneFallsBackToLastPositiveWeight(t *testing.T) {
	weights := []float32{0.1, 0, 0.2}
	if got := selectOneAt(weights, 0.3, 2); got != 2 {
		t.Fatalf("selectOneAt = %d, want final positive index 2", got)
	}
	before := append([]float32(nil), weights...)
	_ = selectOne(weights)
	for i := range weights {
		if weights[i] != before[i] {
			t.Fatalf("weights mutated: got %v want %v", weights, before)
		}
	}
}

func BenchmarkSelectOneParallel(b *testing.B) {
	weights := []float32{0.5, 1, 2, 4, 8, 3, 1.5, 0.25}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if selectOne(weights) < 0 {
				b.Error("selectOne returned no match")
			}
		}
	})
}

func TestRAdvECPMAndCurrencySelection(t *testing.T) {
	radvs := RAdvs{
		{Demand: Demand{CampaignID: 1, ItemID: 1}, Weight: 100, CostType: CostTypeCPM, Cost: 1.5},
		{Demand: Demand{CampaignID: 2, ItemID: 2}, Weight: 0.01, CostType: CostTypeCPM, Cost: 3.0},
		{Demand: Demand{CampaignID: 3, ItemID: 3}, Weight: 1, CostType: CostTypeCPC, Cost: 400},
	}

	index, price := radvs.PickIndexPrice(2.0, "USD")
	if index != 1 || price != 3.0 {
		t.Fatalf("winner = %d at %f, want highest CPM item 2 at 3.0", index, price)
	}

	index, price = radvs.PickIndexPrice(0, "EUR")
	if index != -1 || price != 0 {
		t.Fatalf("unsupported currency result = %d, %f; want no match", index, price)
	}

	index, price = RAdvs{{Weight: 1, CostType: CostTypeCPM, Cost: 0.5}}.PickIndexPrice(1.0, "")
	if index != -1 || price != 0 {
		t.Fatalf("below floor result = %d, %f; want no match", index, price)
	}
}

func TestRAdvCommercialAuctionUsesWeightOnlyWithinWinningDemandUnit(t *testing.T) {
	radvs := RAdvs{
		{Demand: Demand{AdvID: 1, CampaignID: 20, ItemID: 200, CreativeID: 1}, Weight: 1_000, CostType: CostTypeCPM, Cost: 2},
		{Demand: Demand{AdvID: 1, CampaignID: 10, ItemID: 100, CreativeID: 2}, Weight: 1, CostType: CostTypeCPM, Cost: 3},
		{Demand: Demand{AdvID: 1, CampaignID: 10, ItemID: 100, CreativeID: 3}, Weight: 3, CostType: CostTypeCPM, Cost: 3},
	}

	index, price := radvs.pickIndexPriceAt(0, "USD", 0.10)
	if index != 1 || price != 3 {
		t.Fatalf("first rotation point = %d at %f, want creative 2 at 3 CPM", index, price)
	}
	index, price = radvs.pickIndexPriceAt(0, "USD", 0.90)
	if index != 2 || price != 3 {
		t.Fatalf("second rotation point = %d at %f, want creative 3 at 3 CPM", index, price)
	}
}

func TestRAdvCommercialAuctionTieIsDeterministic(t *testing.T) {
	radvs := RAdvs{
		{Demand: Demand{AdvID: 1, CampaignID: 20, ItemID: 1, CreativeID: 1}, Weight: 1, CostType: CostTypeCPM, Cost: 2},
		{Demand: Demand{AdvID: 2, CampaignID: 10, ItemID: 9, CreativeID: 2}, Weight: 1, CostType: CostTypeCPM, Cost: 2},
		{Demand: Demand{AdvID: 3, CampaignID: 10, ItemID: 8, CreativeID: 3}, Weight: 1, CostType: CostTypeCPM, Cost: 2},
	}
	for _, point := range []float32{0, 0.25, 0.99} {
		index, price := radvs.pickIndexPriceAt(2, "", point)
		if index != 2 || price != 2 {
			t.Fatalf("point %f winner = %d at %f, want campaign 10 item 8", point, index, price)
		}
	}
}

func TestRAdvCommercialAuctionKeepsSubFloat32CPMOrdering(t *testing.T) {
	radvs := RAdvs{
		{Demand: Demand{CampaignID: 1, ItemID: 1}, Weight: 1, CostType: CostTypeCPM, Cost: 100, CostCPM: 100_000_001},
		{Demand: Demand{CampaignID: 2, ItemID: 2}, Weight: 1, CostType: CostTypeCPM, Cost: 100, CostCPM: 100_000_002},
	}
	if radvs[0].CostCPM.Float32() != radvs[1].CostCPM.Float32() {
		t.Fatal("test values must collide in the float32 compatibility projection")
	}
	index, _ := radvs.PickIndexPrice(100.000002, "USD")
	if index != 1 {
		t.Fatalf("winner = %d, want exact higher CPM candidate 1", index)
	}
	index, exact := radvs.PickIndexExact(100.000002, "USD")
	if index != 1 || exact != 100_000_002 {
		t.Fatalf("exact winner = %d at %s, want candidate 1 at 100.000002", index, exact.String())
	}
	if index, _ := radvs.PickIndexPrice(100.0000001, "USD"); index != -1 {
		t.Fatalf("over-scale floor selected candidate %d", index)
	}
}

func TestRAdvCommercialAuctionRejectsLegacyAndInvalidPrices(t *testing.T) {
	for _, candidate := range []RAdv{
		{Weight: 1, CostType: CostTypeROI, Cost: 1},
		{Weight: 1, CostType: CostTypeCPC, Cost: 1},
		{Weight: 1, CostType: CostTypeCPA, Cost: 1},
		{Weight: 1, CostType: CostTypeCPM, Cost: 0},
		{Weight: 1, CostType: CostTypeCPM, Cost: -1},
		{Weight: 1, CostType: CostTypeCPM, Cost: float32(math.NaN())},
		{Weight: 1, CostType: CostTypeCPM, Cost: float32(math.Inf(1))},
		{Weight: 1, CostType: CostTypeCPM, Cost: 1, CostCPM: -1},
		{Weight: 1, CostType: CostTypeCPM, Cost: 1, CostCPM: accounting.MaxCPM + 1},
		{Weight: 0, CostType: CostTypeCPM, Cost: 1},
	} {
		if index, price := (RAdvs{candidate}).PickIndexPrice(0, "USD"); index != -1 || price != 0 {
			t.Fatalf("invalid candidate %+v selected as %d at %f", candidate, index, price)
		}
	}
	valid := RAdvs{{Weight: 1, CostType: CostTypeCPM, Cost: 1}}
	for _, floor := range []float64{-1, math.NaN(), math.Inf(1)} {
		if index, _ := valid.PickIndexPrice(floor, "USD"); index != -1 {
			t.Fatalf("invalid floor %v selected index %d", floor, index)
		}
	}
}

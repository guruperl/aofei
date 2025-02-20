package match

import (
	"database/sql"
	"testing"

	"github.com/genelet/winter/pzutil"
	_ "github.com/go-sql-driver/mysql"
)

func TestWeight(t *testing.T) {
	weights := GetWeightSamples()
	bs, err := PackNWeights(weights)
	if err != nil {
		t.Fatal(err)
	}
	newWeights, err := UnpackNWeights(bs)
	if err != nil {
		t.Fatal(err)
	}

	w1 := weights[0]

	newW1 := newWeights[0]
	if w1.CapNumber != newW1.CapNumber || w1.Weight != newW1.Weight {
		t.Errorf("%v", weights)
		t.Errorf("%v", newWeights)
	}

	slot1 := &Slot{uint32(9), uint32(10), weights}
	bs, err = slot1.Pack()
	if err != nil {
		t.Fatal(err)
	}
	newSlot1, err := UnpackSlot(bs)
	if err != nil {
		t.Fatal(err)
	}
	newS1 := newSlot1.Weights[0]
	if newSlot1.SizeID != uint32(10) || w1.CapNumber != newS1.CapNumber || w1.Weight != newS1.Weight {
		t.Errorf("%v", slot1)
		t.Errorf("%v", newSlot1)
	}

	c, err := pzutil.NewConfig("../conf/gotest.conf")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ws, err := DBGetNWeights(db, uint32(125))
	if err != nil {
		t.Fatal(err)
	}
	if ws.SlotID != 125 ||
		ws.SizeID != 5 ||
		ws.Weights[1].WeightID != 1554 ||
		ws.Weights[1].ItemID != 15 ||
		ws.Weights[1].CampaignID != 3 {
		t.Errorf("%d %d %v", ws.SlotID, ws.SizeID, ws.Weights[1])
	}
}

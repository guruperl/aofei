package acl

import (
	"bytes"
	"math"
	"testing"

	"github.com/guruperl/aofei/accounting"
)

func validPublisherWrite() *Pub {
	return &Pub{
		AccountingVersion: accounting.ExactMoneyContract,
		PubID:             42,
		Active:            true,
		Sites:             map[string]uint32{"example.com": 7},
		Slots:             map[uint32]map[string]uint32{7: {"leaderboard": 99}},
		SlotFloors:        map[uint32]map[uint32]float64{7: {99: 1.25}},
		SlotFloorCPMs:     map[uint32]map[uint32]accounting.CPM{7: {99: 1_250_000}},
	}
}

func TestPublisherPackRejectsInvalidMonetaryProvenanceBeforeWriting(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Pub)
	}{
		{name: "missing marker", mutate: func(pub *Pub) { pub.AccountingVersion = "" }},
		{name: "unknown marker", mutate: func(pub *Pub) { pub.AccountingVersion = "future-version" }},
		{name: "missing exact floor", mutate: func(pub *Pub) { delete(pub.SlotFloorCPMs[7], 99) }},
		{name: "missing compatibility floor", mutate: func(pub *Pub) { delete(pub.SlotFloors[7], 99) }},
		{name: "mismatched compatibility floor", mutate: func(pub *Pub) { pub.SlotFloors[7][99] = 1.5 }},
		{name: "negative-zero compatibility floor", mutate: func(pub *Pub) { pub.SlotFloorCPMs[7][99] = 0; pub.SlotFloors[7][99] = math.Copysign(0, -1) }},
		{name: "NaN compatibility floor", mutate: func(pub *Pub) { pub.SlotFloors[7][99] = math.NaN() }},
		{name: "infinite compatibility floor", mutate: func(pub *Pub) { pub.SlotFloors[7][99] = math.Inf(1) }},
		{name: "out-of-range compatibility floor", mutate: func(pub *Pub) { pub.SlotFloors[7][99] = accounting.MaxCPM.Float64() + 1 }},
		{name: "negative exact floor", mutate: func(pub *Pub) { pub.SlotFloorCPMs[7][99] = -1 }},
		{name: "out-of-range exact floor", mutate: func(pub *Pub) { pub.SlotFloorCPMs[7][99] = accounting.MaxCPM + 1 }},
		{name: "extra exact floor", mutate: func(pub *Pub) { pub.SlotFloorCPMs[7][100] = 1 }},
		{name: "extra compatibility floor", mutate: func(pub *Pub) { pub.SlotFloors[7][100] = 0.000001 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pub := validPublisherWrite()
			test.mutate(pub)
			data, err := pub.Pack()
			if err == nil || len(data) != 0 {
				t.Fatalf("Pack = %d bytes, %v; want no bytes and an error", len(data), err)
			}
			var output bytes.Buffer
			if err := pub.PackIO(&output); err == nil || output.Len() != 0 {
				t.Fatalf("PackIO = %d bytes, %v; want untouched output and an error", output.Len(), err)
			}
		})
	}
}

func TestPublisherPackAcceptsMatchingV3Floors(t *testing.T) {
	pub := validPublisherWrite()
	data, err := pub.Pack()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnpackPub(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.AccountingVersion != accounting.ExactMoneyContract || decoded.SlotFloorCPMs[7][99] != 1_250_000 || decoded.SlotFloors[7][99] != 1.25 {
		t.Fatalf("decoded publisher = %#v", decoded)
	}
}

func TestDirectPublisherPackRequiresEmbeddedV3Parity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*DirectPub)
	}{
		{name: "missing direct marker", mutate: func(pub *DirectPub) { pub.AccountingVersion = "" }},
		{name: "unknown direct marker", mutate: func(pub *DirectPub) { pub.AccountingVersion = "future-version" }},
		{name: "missing embedded marker", mutate: func(pub *DirectPub) { pub.Pub.AccountingVersion = "" }},
		{name: "unknown embedded marker", mutate: func(pub *DirectPub) { pub.Pub.AccountingVersion = "future-version" }},
		{name: "missing direct exact floor", mutate: func(pub *DirectPub) { delete(pub.SlotFloorCPMs[7], 99) }},
		{name: "mismatched direct exact floor", mutate: func(pub *DirectPub) { pub.SlotFloorCPMs[7][99] = 1_500_000; pub.SlotFloors[7][99] = 1.5 }},
		{name: "mismatched direct compatibility floor", mutate: func(pub *DirectPub) { pub.SlotFloors[7][99] = 1.5 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			direct := NewDirectPub("pub.example", validPublisherWrite())
			test.mutate(direct)
			data, err := direct.Pack()
			if err == nil || len(data) != 0 {
				t.Fatalf("Pack = %d bytes, %v; want no bytes and an error", len(data), err)
			}
		})
	}
}

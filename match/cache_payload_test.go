package match

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"testing"
)

func TestCachePayloadVersionedAndLegacyDecode(t *testing.T) {
	delivery := Delivery{GeneratedAtUnix: 123, ItemTotal: DeliveryBalance{ID: 9, LimitImp: 10}}
	if err := delivery.SetTimezone("UTC"); err != nil {
		t.Fatal(err)
	}
	radvs := RAdvs{{Demand: Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4}, Weight: 1, CostType: CostTypeCPM, Cost: 1.5, CostCPM: 1_500_000, Delivery: delivery}}
	versioned, err := radvs.Pack()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(versioned, cachePayloadMagic) {
		t.Fatalf("RAdvs payload missing cache magic")
	}
	gotRAdvs, err := UnpackRAdvs(versioned)
	if err != nil {
		t.Fatal(err)
	}
	if gotRAdvs[0].CreativeID != 4 || gotRAdvs[0].Delivery != delivery {
		t.Fatalf("versioned RAdvs decode = %+v", gotRAdvs[0])
	}

	legacy := []legacyRAdv{{
		Demand:   radvs[0].Demand,
		Weight:   radvs[0].Weight,
		CostType: radvs[0].CostType,
		Cost:     radvs[0].Cost,
		Cap:      radvs[0].Cap,
	}}
	var legacyRAdvs bytes.Buffer
	if err := binary.Write(&legacyRAdvs, binary.LittleEndian, legacy); err != nil {
		t.Fatal(err)
	}
	gotRAdvs, err = UnpackRAdvs(legacyRAdvs.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if gotRAdvs[0].CreativeID != 4 {
		t.Fatalf("legacy RAdvs decode = %+v", gotRAdvs[0])
	}

	audience := &Audience{}
	versioned, err = audience.Pack()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(versioned, cachePayloadMagic) {
		t.Fatalf("Audience payload missing cache magic")
	}
	if _, err := UnpackAudience(versioned); err != nil {
		t.Fatal(err)
	}
	var legacyAudience bytes.Buffer
	if err := gob.NewEncoder(&legacyAudience).Encode(audience); err != nil {
		t.Fatal(err)
	}
	if _, err := UnpackAudience(legacyAudience.Bytes()); err != nil {
		t.Fatal(err)
	}

	creative := &Creative{CreativeName: "creative", SizeID: 300250}
	versioned, err = creative.Pack()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(versioned, cachePayloadMagic) {
		t.Fatalf("Creative payload missing cache magic")
	}
	if _, err := UnpackCreative(versioned); err != nil {
		t.Fatal(err)
	}
	var legacyCreative bytes.Buffer
	if err := gob.NewEncoder(&legacyCreative).Encode(creative); err != nil {
		t.Fatal(err)
	}
	if _, err := UnpackCreative(legacyCreative.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestCachePayloadRejectsUnknownVersion(t *testing.T) {
	data := packCachePayload(cachePayloadKindCreative, 99, []byte("body"))
	if _, err := UnpackCreative(data); err == nil {
		t.Fatal("expected unknown creative cache version to fail")
	}
}

func TestRAdvsDecodeVersionOnePayload(t *testing.T) {
	legacy := []legacyRAdv{{
		Demand:   Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4},
		Weight:   1,
		CostType: 2,
		Cost:     1.5,
	}}
	var body bytes.Buffer
	if err := binary.Write(&body, binary.LittleEndian, legacy); err != nil {
		t.Fatal(err)
	}
	got, err := UnpackRAdvs(packCachePayload(cachePayloadKindRAdvs, 1, body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CreativeID != 4 || got[0].Delivery.HasPolicy() {
		t.Fatalf("version-one decode = %+v", got)
	}
	got, err = UnpackRAdvs(body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CreativeID != 4 || got[0].Delivery.HasPolicy() {
		t.Fatalf("unversioned legacy decode = %+v", got)
	}
}

func TestRAdvsDecodeVersionTwoFloatPayloadAsCompatibilityOnly(t *testing.T) {
	legacy := []legacyRAdvV2{{
		Demand: Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4},
		Weight: 1, CostType: CostTypeCPM, Cost: 2.5,
		Delivery: legacyDeliveryV2{ItemTotal: legacyDeliveryBalanceV2{ID: 7, LimitSpend: 1.25, CurrentSpend: 0.0025}},
	}}
	var body bytes.Buffer
	if err := binary.Write(&body, binary.LittleEndian, legacy); err != nil {
		t.Fatal(err)
	}
	got, err := UnpackRAdvs(packCachePayload(cachePayloadKindRAdvs, 2, body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("version-two record count = %d, want 1", len(got))
	}
	exact, ok := got[0].ExactCPM()
	if got[0].CostCPM != 0 || !ok || exact != 2_500_000 || got[0].Delivery.ItemTotal.LimitSpendNano != 1_250_000_000 || got[0].Delivery.ItemTotal.CurrentSpendNano != 2_500_000 {
		t.Fatalf("version-two exact compatibility decode = %+v", got)
	}
	if _, err := got.Pack(); err == nil {
		t.Fatal("version-two compatibility record was promoted into a v3 cache payload")
	}
}

func TestLegacyRAdvsWithUnsetCostTypeRemainReadableButIneligible(t *testing.T) {
	legacyV1 := []legacyRAdv{{
		Demand: Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4},
		Weight: 1,
		Cost:   1.5,
	}}
	var versionOne bytes.Buffer
	if err := binary.Write(&versionOne, binary.LittleEndian, legacyV1); err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{
		"headerless":  versionOne.Bytes(),
		"version-one": packCachePayload(cachePayloadKindRAdvs, 1, versionOne.Bytes()),
	} {
		got, err := UnpackRAdvs(payload)
		if err != nil {
			t.Fatalf("%s decode: %v", name, err)
		}
		if len(got) != 1 {
			t.Fatalf("%s record count = %d, want 1", name, len(got))
		}
		if _, ok := got[0].ExactCPM(); ok {
			t.Fatalf("%s unset cost type became exact CPM authority", name)
		}
		if index, _ := got.PickIndexExact(0, "USD"); index != -1 {
			t.Fatalf("%s unset cost type selected candidate %d", name, index)
		}
	}

	legacyV2 := []legacyRAdvV2{{
		Demand: Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4},
		Weight: 1,
		Cost:   1.5,
	}}
	var versionTwo bytes.Buffer
	if err := binary.Write(&versionTwo, binary.LittleEndian, legacyV2); err != nil {
		t.Fatal(err)
	}
	got, err := UnpackRAdvs(packCachePayload(cachePayloadKindRAdvs, 2, versionTwo.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("version-two record count = %d, want 1", len(got))
	}
	if _, ok := got[0].ExactCPM(); ok {
		t.Fatal("version-two unset cost type became exact CPM authority")
	}
	if index, _ := got.PickIndexExact(0, "USD"); index != -1 {
		t.Fatalf("version-two unset cost type selected candidate %d", index)
	}
}

func TestUnversionedRAdvsAlwaysDecodeAsLegacy(t *testing.T) {
	legacySize := binary.Size(legacyRAdv{})
	currentSize := binary.Size(RAdv{})
	count := currentSize / gcdForTest(currentSize, legacySize)
	legacy := make([]legacyRAdv, count)
	for i := range legacy {
		legacy[i] = legacyRAdv{
			Demand: Demand{AdvID: uint32(i + 1), CampaignID: 2, ItemID: 3, CreativeID: 4},
			Weight: 1,
		}
	}
	var body bytes.Buffer
	if err := binary.Write(&body, binary.LittleEndian, legacy); err != nil {
		t.Fatal(err)
	}
	if body.Len()%currentSize != 0 {
		t.Fatalf("test payload length %d is not ambiguous with current record size %d", body.Len(), currentSize)
	}
	got, err := UnpackRAdvs(body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(legacy) {
		t.Fatalf("unversioned record count = %d, want %d legacy records", len(got), len(legacy))
	}
	for i := range got {
		if got[i].AdvID != uint32(i+1) || got[i].Delivery.HasPolicy() {
			t.Fatalf("unversioned record %d = %+v", i, got[i])
		}
	}
}

func gcdForTest(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

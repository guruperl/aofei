package match

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestCachePayloadVersionedAndLegacyDecode(t *testing.T) {
	radvs := RAdvs{{Demand: Demand{AdvID: 1, CampaignID: 2, ItemID: 3, CreativeID: 4}, Weight: 1, CostType: 2, Cost: 1.5}}
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
	if gotRAdvs[0].CreativeID != 4 {
		t.Fatalf("versioned RAdvs decode = %+v", gotRAdvs[0])
	}

	var legacyRAdvs bytes.Buffer
	if err := radvs.packLegacy(&legacyRAdvs); err != nil {
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

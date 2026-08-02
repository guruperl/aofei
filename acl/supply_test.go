package acl

import (
	"bytes"
	"encoding/gob"
	"testing"
)

type legacyPubCache struct {
	PubID      uint32
	Active     bool
	Sites      map[string]uint32
	SiteTypes  map[uint32]SiteType
	Slots      map[uint32]map[string]uint32
	SlotSizes  map[uint32]map[uint32]uint32
	SlotFloors map[uint32]map[uint32]float64
}

type legacyDirectPubCache struct {
	Domain     string
	Pub        *legacyPubCache
	Sites      map[uint32]string
	SiteTypes  map[uint32]SiteType
	Slots      map[uint32]map[uint32]string
	SlotSizes  map[uint32]map[uint32]uint32
	SlotFloors map[uint32]map[uint32]float64
}

func TestSupplyMetadataValidation(t *testing.T) {
	seller := SellerMetadata{ID: "seller-7", Type: "Publisher", ASI: "w8m.com", Domain: "example.com", Authorized: true}
	if err := seller.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (SellerMetadata{ID: "<script>", Type: "Publisher"}).Validate(); err == nil {
		t.Fatal("hostile seller id was accepted")
	}
	if err := (SellerMetadata{ID: "seller-7", Type: "Publisher", ASI: "127.0.0.1", Authorized: true}).Validate(); err == nil {
		t.Fatal("seller ASI IP literal was accepted")
	}
	if err := (SellerMetadata{ID: "seller-7", Type: "Publisher", ASI: "w8m.com", Name: "bad\nname", Authorized: true}).Validate(); err == nil {
		t.Fatal("seller name control character was accepted")
	}
	if err := (SiteSupplyMetadata{Environment: "Web", CanonicalIdentity: "https://example.com", IntegrationMode: "BrowserTag"}).Validate(); err == nil {
		t.Fatal("URL was accepted as a canonical web domain")
	}
	if err := (SiteSupplyMetadata{Environment: "Web", CanonicalIdentity: "example.com", StoreURL: "javascript:alert(1)", IntegrationMode: "BrowserTag"}).Validate(); err == nil {
		t.Fatal("executable store URL was accepted")
	}
	if err := (SlotSupplyMetadata{RefreshMode: "Timed", RefreshSeconds: 14}).Validate(); err == nil {
		t.Fatal("too-frequent timed refresh was accepted")
	}
}

func TestPublisherGobCacheIsAdditiveInBothDirections(t *testing.T) {
	legacy := legacyPubCache{
		PubID: 7, Active: true, Sites: map[string]uint32{"example.com": 8},
		SiteTypes:  map[uint32]SiteType{8: SiteTypeWeb},
		Slots:      map[uint32]map[string]uint32{8: {"top": 9}},
		SlotSizes:  map[uint32]map[uint32]uint32{8: {9: 10}},
		SlotFloors: map[uint32]map[uint32]float64{8: {9: 1.25}},
	}
	var old bytes.Buffer
	if err := gob.NewEncoder(&old).Encode(&legacy); err != nil {
		t.Fatal(err)
	}
	decoded, err := UnpackPub(old.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.SupplyFor(8, 9); got.Site.Environment != "Unknown" || got.Slot.Placement != "Unknown" {
		t.Fatalf("old cache did not decode to explicit unknown taxonomy: %#v", got)
	}
	decoded.Seller = SellerMetadata{ID: "pending-7", Type: "Publisher", ASI: "w8m.com"}
	if got := decoded.SupplyFor(8, 9).Seller; got.ID != "" || got.ASI != "" || got.Authorized {
		t.Fatalf("unapproved seller reached runtime supply metadata: %#v", got)
	}

	decoded.Seller = SellerMetadata{ID: "seller-7", Type: "Publisher", ASI: "w8m.com", Authorized: true}
	decoded.SiteSupply = map[uint32]SiteSupplyMetadata{8: {Environment: "Web", CanonicalIdentity: "example.com", IntegrationMode: "BrowserTag"}}
	decoded.SlotSupply = map[uint32]map[uint32]SlotSupplyMetadata{8: {9: {MediaIntent: "Banner", Placement: "AboveFold", RefreshMode: "None"}}}
	packed, err := decoded.Pack()
	if err != nil {
		t.Fatal(err)
	}
	var oldReader legacyPubCache
	if err := gob.NewDecoder(bytes.NewReader(packed)).Decode(&oldReader); err != nil {
		t.Fatal(err)
	}
	if oldReader.PubID != 7 || oldReader.Slots[8]["top"] != 9 || oldReader.SlotFloors[8][9] != 1.25 {
		t.Fatalf("old reader lost its pre-P02 fields: %#v", oldReader)
	}
}

func TestDirectPublisherGobCacheIsAdditiveInBothDirections(t *testing.T) {
	legacyPub := &legacyPubCache{
		PubID: 7, Active: true, Sites: map[string]uint32{"example.com": 8},
		SiteTypes:  map[uint32]SiteType{8: SiteTypeWeb},
		Slots:      map[uint32]map[string]uint32{8: {"top": 9}},
		SlotSizes:  map[uint32]map[uint32]uint32{8: {9: 10}},
		SlotFloors: map[uint32]map[uint32]float64{8: {9: 1.25}},
	}
	legacy := legacyDirectPubCache{
		Domain: "example.com", Pub: legacyPub,
		Sites: map[uint32]string{8: "example.com"}, SiteTypes: map[uint32]SiteType{8: SiteTypeWeb},
		Slots: map[uint32]map[uint32]string{8: {9: "top"}}, SlotSizes: map[uint32]map[uint32]uint32{8: {9: 10}},
		SlotFloors: map[uint32]map[uint32]float64{8: {9: 1.25}},
	}
	var old bytes.Buffer
	if err := gob.NewEncoder(&old).Encode(&legacy); err != nil {
		t.Fatal(err)
	}
	decoded, err := UnpackDirectPub(old.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Pub.SupplyFor(8, 9); got.Site.Environment != "Unknown" || got.Slot.MediaIntent != "Unknown" {
		t.Fatalf("old direct cache did not decode to explicit unknown taxonomy: %#v", got)
	}

	decoded.Pub.Seller = SellerMetadata{ID: "seller-7", Type: "Publisher", ASI: "w8m.com", Authorized: true}
	decoded.Pub.SiteSupply = map[uint32]SiteSupplyMetadata{8: {Environment: "Web", CanonicalIdentity: "example.com", IntegrationMode: "BrowserTag"}}
	decoded.Pub.SlotSupply = map[uint32]map[uint32]SlotSupplyMetadata{8: {9: {MediaIntent: "Banner", Placement: "AboveFold", RefreshMode: "None"}}}
	packed, err := decoded.Pack()
	if err != nil {
		t.Fatal(err)
	}
	var oldReader legacyDirectPubCache
	if err := gob.NewDecoder(bytes.NewReader(packed)).Decode(&oldReader); err != nil {
		t.Fatal(err)
	}
	if oldReader.Domain != "example.com" || oldReader.Pub.PubID != 7 || oldReader.Slots[8][9] != "top" || oldReader.SlotFloors[8][9] != 1.25 {
		t.Fatalf("old direct reader lost its pre-P02 fields: %#v", oldReader)
	}
}

package dsp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestApprovedSellerBuildsTruthfulSupplyChain(t *testing.T) {
	for _, test := range []struct {
		name     string
		seller   acl.SellerMetadata
		complete int8
	}{
		{"owner", acl.SellerMetadata{ID: "seller-1", Type: "Publisher", ASI: "w8m.com", Authorized: true}, 1},
		{"reseller", acl.SellerMetadata{ID: "seller-2", Type: "Intermediary", ASI: "w8m.com", Authorized: true}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := sourceFromApprovedSeller(test.seller)
			if source == nil || source.SChain == nil || source.SChain.Complete != test.complete || len(source.SChain.Nodes) != 1 {
				t.Fatalf("source = %#v", source)
			}
			if node := source.SChain.Nodes[0]; node.ASI != "w8m.com" || node.SID != test.seller.ID || node.HP == nil || *node.HP != 1 {
				t.Fatalf("node = %#v", node)
			}
		})
	}
	if source := sourceFromApprovedSeller(acl.SellerMetadata{ID: "browser-claim", Type: "Publisher", ASI: "w8m.com"}); source != nil {
		t.Fatalf("unapproved seller generated source: %#v", source)
	}
}

func TestDirectSSPUsesOnlyCachedSellerState(t *testing.T) {
	parsed, err := ParseSSPRequest([]byte(`{
		"site":"ignored-here","source":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"evil.example","sid":"claim"}]}},
		"adUnits":[{"code":"slot-one","mediaTypes":{"banner":{"size":[300,250]}}}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	pub := &acl.DirectPub{Domain: "publisher.example", Pub: &acl.Pub{
		Seller: acl.SellerMetadata{ID: "approved-7", Type: "Publisher", ASI: "w8m.com", Authorized: true},
	}}
	unit := SSPValidatedUnit{Site: "site-token", SiteStr: "publisher.example", SlotStr: "top", RPub: match.RPub{PubID: 7, SiteID: 8, SlotID: 9, SizeID: match.SizeID2To1(300, 250)}}
	bid, err := (&Controller{}).openRTBFromValidatedSSP(httptest.NewRequest("POST", "/pz", nil), parsed, pub, []SSPValidatedUnit{unit}, "")
	if err != nil {
		t.Fatal(err)
	}
	if bid.Source == nil || bid.Source.SChain == nil || bid.Source.SChain.Nodes[0].SID != "approved-7" {
		t.Fatalf("direct source was not rebuilt from cache: %#v", bid.Source)
	}
	if bid.Source.SChain.Nodes[0].SID == "claim" {
		t.Fatal("browser seller claim reached OpenRTB")
	}
}

func TestMiddlemanPreservesOnlyValidatedSupplyChain(t *testing.T) {
	hp := int8(1)
	bid := openrtb2.BidRequest{
		ID: "request-1", Device: &openrtb2.Device{}, Imp: []openrtb2.Imp{{ID: "imp-1"}},
		Source: &openrtb2.Source{PChain: "private-payment-chain", SChain: &openrtb2.SupplyChain{
			Ver: "1.0", Complete: 1,
			Nodes: []openrtb2.SupplyChainNode{{ASI: "w8m.com", SID: "seller-1", HP: &hp}},
		}},
	}
	raw, err := json.Marshal(&bid)
	if err != nil {
		t.Fatal(err)
	}
	out, err := middlemanRequestBodyForAssignment(raw, "exchange.example")
	if err != nil {
		t.Fatal(err)
	}
	var forwarded openrtb2.BidRequest
	if err := json.Unmarshal(out, &forwarded); err != nil {
		t.Fatal(err)
	}
	if forwarded.Source == nil || forwarded.Source.SChain == nil || forwarded.Source.SChain.Nodes[0].SID != "seller-1" {
		t.Fatalf("validated chain was not preserved: %s", out)
	}
	if forwarded.Source.PChain != "" {
		t.Fatalf("private payment chain was forwarded: %q", forwarded.Source.PChain)
	}

	bid.Source.SChain.Nodes[0].ASI = "https://evil.example/path"
	raw, _ = json.Marshal(&bid)
	if _, err := middlemanRequestBodyForAssignment(raw, "exchange.example"); err == nil {
		t.Fatal("invalid browser/partner supply-chain claim was accepted")
	}

	bid.Source.SChain.Nodes[0].ASI = "w8m.com"
	bid.Source.SChain.Nodes[0].HP = nil
	raw, _ = json.Marshal(&bid)
	if _, err := middlemanRequestBodyForAssignment(raw, "exchange.example"); err == nil {
		t.Fatal("supply-chain node without required hp was accepted")
	}
}

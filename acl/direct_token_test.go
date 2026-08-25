package acl

import (
	"encoding/base32"
	"errors"
	"strings"
	"testing"

	"github.com/guruperl/aofei/accounting"
)

func TestDirectTokenMatchesHistoricalSamples(t *testing.T) {
	tests := []struct {
		token string
		x     uint32
		y     uint32
	}{
		{token: "AAAACAH774AAA", x: 65536, y: 65535},
		{token: "AAAACAAUAMAAA", x: 65536, y: 788},
		{token: "AAAACAH677776", x: 65536, y: 4294967294},
		{token: "CUBQAAH774AAA", x: 789, y: 65535},
	}

	for _, tt := range tests {
		x, y, err := UnpackDirectToken(tt.token)
		if err != nil {
			t.Fatalf("UnpackDirectToken(%q): %v", tt.token, err)
		}
		if x != tt.x || y != tt.y {
			t.Fatalf("UnpackDirectToken(%q) = %d, %d; want %d, %d", tt.token, x, y, tt.x, tt.y)
		}
		packed, err := PackDirectToken(x, y)
		if err != nil {
			t.Fatal(err)
		}
		if packed != tt.token {
			t.Fatalf("PackDirectToken(%d, %d) = %q, want %q", x, y, packed, tt.token)
		}
	}
}

func TestUnpackDirectTokenRejectsMalformedLengths(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "short",
			token: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte{1, 2, 3, 4}),
		},
		{
			name:  "long",
			token: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}),
		},
		{
			name:  "invalid-base32",
			token: "not valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := UnpackDirectToken(tt.token); err == nil {
				t.Fatalf("UnpackDirectToken(%q) succeeded, want error", tt.token)
			}
		})
	}
}

func TestDirectTokenV2BindsCompleteInventoryTuple(t *testing.T) {
	codec, err := NewDirectTokenCodec(DirectTokenKey{ID: "primary", Epoch: 7, Secret: directTokenTestKey(1)}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	site, err := codec.PackSite(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	slot, err := codec.PackSlot(42, 9, 100, 19661050)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(site, "pz2.site.primary.7.") || !strings.HasPrefix(slot, "pz2.slot.primary.7.") {
		t.Fatalf("unexpected v2 tokens: site=%q slot=%q", site, slot)
	}
	pubID, siteID, version, err := codec.UnpackSite(site)
	if err != nil {
		t.Fatal(err)
	}
	if pubID != 42 || siteID != 9 || version != DirectTokenV2 {
		t.Fatalf("site = %d/%d/%s, want 42/9/v2", pubID, siteID, version)
	}
	slotID, sizeID, version, err := codec.UnpackSlot(slot, pubID, siteID)
	if err != nil {
		t.Fatal(err)
	}
	if slotID != 100 || sizeID != 19661050 || version != DirectTokenV2 {
		t.Fatalf("slot = %d/%d/%s, want 100/19661050/v2", slotID, sizeID, version)
	}
	if _, _, _, err := codec.UnpackSlot(slot, pubID, siteID+1); !errors.Is(err, ErrInvalidDirectToken) {
		t.Fatalf("cross-site slot error = %v, want invalid token", err)
	}
	if _, err := codec.PackSite(0, siteID); err == nil {
		t.Fatal("zero publisher site locator was emitted")
	}
	if _, err := codec.PackSlot(pubID, siteID, 0, sizeID); err == nil {
		t.Fatal("zero slot locator was emitted")
	}
}

func TestDirectTokenV2RejectsTamperAndUnknownVersions(t *testing.T) {
	codec, err := NewDirectTokenCodec(DirectTokenKey{ID: "primary", Epoch: 7, Secret: directTokenTestKey(1)}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	site, err := codec.PackSite(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(site, ".")
	for _, test := range []struct {
		name  string
		token string
	}{
		{name: "payload", token: strings.Join(append(append([]string(nil), parts[:4]...), mutateTokenText(parts[4]), parts[5]), ".")},
		{name: "signature", token: strings.Join(append(append([]string(nil), parts[:5]...), mutateTokenText(parts[5])), ".")},
		{name: "epoch", token: strings.Replace(site, ".7.", ".8.", 1)},
		{name: "key id", token: strings.Replace(site, ".primary.", ".unknown.", 1)},
		{name: "unknown version", token: strings.Replace(site, "pz2.", "pz3.", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := codec.UnpackSite(test.token); !errors.Is(err, ErrInvalidDirectToken) {
				t.Fatalf("UnpackSite(%q) error = %v, want invalid token", test.token, err)
			}
		})
	}
}

func TestDirectTokenV2RotationAndLegacyGate(t *testing.T) {
	oldKey := DirectTokenKey{ID: "inventory", Epoch: 10, Secret: directTokenTestKey(1)}
	oldCodec, err := NewDirectTokenCodec(oldKey, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	oldSite, err := oldCodec.PackSite(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	oldSlot, err := oldCodec.PackSlot(42, 9, 100, 19661050)
	if err != nil {
		t.Fatal(err)
	}

	newKey := DirectTokenKey{ID: "inventory", Epoch: 11, Secret: directTokenTestKey(2)}
	rotating, err := NewDirectTokenCodec(newKey, &oldKey, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := rotating.UnpackSite(oldSite); err != nil {
		t.Fatalf("previous site during overlap: %v", err)
	}
	if _, _, _, err := rotating.UnpackSlot(oldSlot, 42, 9); err != nil {
		t.Fatalf("previous slot during overlap: %v", err)
	}
	newSite, err := rotating.PackSite(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(newSite, ".inventory.11.") {
		t.Fatalf("new token did not use current epoch: %q", newSite)
	}

	withdrawn, err := NewDirectTokenCodec(newKey, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := withdrawn.UnpackSite(oldSite); !errors.Is(err, ErrInvalidDirectToken) {
		t.Fatalf("withdrawn epoch error = %v, want invalid token", err)
	}
	legacy, err := PackDirectToken(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, version, err := withdrawn.UnpackSite(legacy); version != DirectTokenLegacy || !errors.Is(err, ErrLegacyDirectTokenDisabled) {
		t.Fatalf("legacy gate = version %s error %v", version, err)
	}
	if _, _, version, err := rotating.UnpackSite(legacy); err != nil || version != DirectTokenLegacy {
		t.Fatalf("legacy overlap = version %s error %v", version, err)
	}
}

func TestDirectTokenCodecRejectsUnsafeKeyRings(t *testing.T) {
	valid := DirectTokenKey{ID: "primary", Epoch: 1, Secret: directTokenTestKey(1)}
	tests := []struct {
		name     string
		current  DirectTokenKey
		previous *DirectTokenKey
	}{
		{name: "partial current", current: DirectTokenKey{ID: "primary"}},
		{name: "bad key id", current: DirectTokenKey{ID: "bad.id", Epoch: 1, Secret: directTokenTestKey(1)}},
		{name: "zero epoch", current: DirectTokenKey{ID: "primary", Secret: directTokenTestKey(1)}},
		{name: "short secret", current: DirectTokenKey{ID: "primary", Epoch: 1, Secret: []byte("short")}},
		{name: "previous without current", previous: &valid},
		{name: "duplicate selector", current: valid, previous: &DirectTokenKey{ID: "primary", Epoch: 1, Secret: directTokenTestKey(2)}},
		{name: "reused secret", current: valid, previous: &DirectTokenKey{ID: "primary", Epoch: 2, Secret: directTokenTestKey(1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewDirectTokenCodec(test.current, test.previous, true); err == nil {
				t.Fatal("unsafe key ring was accepted")
			}
		})
	}
}

func directTokenTestKey(fill byte) []byte {
	return []byte(strings.Repeat(string([]byte{fill}), 32))
}

func mutateTokenText(text string) string {
	if text[0] == 'A' {
		return "B" + text[1:]
	}
	return "A" + text[1:]
}

func TestDirectPubMapValidatesAndReconstructsSupplyStrings(t *testing.T) {
	pub := &Pub{
		PubID:     42,
		Active:    true,
		Sites:     map[string]uint32{"example.com": 7},
		SlotSizes: map[uint32]map[uint32]uint32{7: {99: 12345}},
		Slots: map[uint32]map[string]uint32{
			7: map[string]uint32{"leaderboard": 99},
		},
	}

	byID := DirectPubMapFromPubMap(PubMap{"pub.example": pub})
	direct := byID.PubByID(42)
	if direct == nil {
		t.Fatal("direct publisher lookup is nil")
	}
	if direct.Domain != "pub.example" {
		t.Fatalf("domain = %q, want pub.example", direct.Domain)
	}
	siteStr, slotStr, ok := direct.Validate(7, 99, 12345)
	if !ok {
		t.Fatal("expected direct supply to validate")
	}
	if siteStr != "example.com" || slotStr != "leaderboard" {
		t.Fatalf("reverse strings = %q, %q; want example.com, leaderboard", siteStr, slotStr)
	}
	if _, _, ok := direct.Validate(7, 100, 12345); ok {
		t.Fatal("wrong slot validated")
	}
	if _, _, ok := direct.Validate(8, 99, 12345); ok {
		t.Fatal("wrong site validated")
	}
	if _, _, ok := direct.Validate(7, 99, 54321); ok {
		t.Fatal("wrong size validated")
	}
}

func TestDirectPubMapOmitsInactiveAndLimitedPublishers(t *testing.T) {
	pubmap := PubMap{
		"active.example": {
			PubID:  1,
			Active: true,
			Sites:  map[string]uint32{"active.example": 10},
			Slots:  map[uint32]map[string]uint32{10: map[string]uint32{"slot": 100}},
		},
		"inactive.example": {
			PubID:  2,
			Active: false,
			Sites:  map[string]uint32{"inactive.example": 20},
			Slots:  map[uint32]map[string]uint32{20: map[string]uint32{"slot": 200}},
		},
		"limited.example": {
			PubID:       3,
			Active:      true,
			LimitImps:   10,
			CurrentImps: 10,
			Sites:       map[string]uint32{"limited.example": 30},
			Slots:       map[uint32]map[string]uint32{30: map[string]uint32{"slot": 300}},
		},
	}

	byID := DirectPubMapFromPubMap(pubmap)
	if byID.PubByID(1) == nil {
		t.Fatal("active publisher missing from direct by-id lookup")
	}
	if byID.PubByID(2) != nil {
		t.Fatal("inactive publisher present in direct by-id lookup")
	}
	if byID.PubByID(3) != nil {
		t.Fatal("limited publisher present in direct by-id lookup")
	}
}

func TestDirectPubPackRoundTrip(t *testing.T) {
	pub := &Pub{
		PubID:      42,
		Active:     true,
		Sites:      map[string]uint32{"example.com": 7},
		SiteTypes:  map[uint32]SiteType{7: SiteTypeWeb},
		Slots:      map[uint32]map[string]uint32{7: map[string]uint32{"leaderboard": 99}},
		SlotSizes:  map[uint32]map[uint32]uint32{7: map[uint32]uint32{99: 12345}},
		SlotFloors: map[uint32]map[uint32]float64{7: map[uint32]float64{99: 1.75}},
	}
	direct := NewDirectPub("pub.example", pub)
	data, err := direct.Pack()
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnpackDirectPub(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Pub.PubID != 42 || got.Sites[7] != "example.com" || got.SiteTypes[7] != SiteTypeWeb ||
		got.Slots[7][99] != "leaderboard" || got.SlotSizes[7][99] != 12345 || got.SlotFloors[7][99] != 1.75 {
		t.Fatalf("round trip = %#v", got)
	}
	if _, _, siteType, floor, ok := got.CommercialSlot(7, 99, 12345); !ok || siteType != SiteTypeWeb || floor != 1.75 {
		t.Fatalf("commercial slot = type %v floor %v ok %v", siteType, floor, ok)
	}
}

func TestValidateCommercialPubMapFailsClosedOnIncompletePolicy(t *testing.T) {
	valid := func() *Pub {
		return &Pub{
			AccountingVersion: accounting.ExactMoneyContract,
			PubID:             42,
			Active:            true,
			Sites:             map[string]uint32{"example.com": 7},
			SiteTypes:         map[uint32]SiteType{7: SiteTypeWeb},
			Slots:             map[uint32]map[string]uint32{7: {"leaderboard": 99}},
			SlotSizes:         map[uint32]map[uint32]uint32{7: {99: (300 << 16) | 250}},
			SlotFloors:        map[uint32]map[uint32]float64{7: {99: 1.25}},
			SlotFloorCPMs:     map[uint32]map[uint32]accounting.CPM{7: {99: 1_250_000}},
		}
	}
	if err := ValidateCommercialPubMap(PubMap{"pub.example": valid()}); err != nil {
		t.Fatalf("valid commercial inventory: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*Pub)
	}{
		{name: "missing site type", mutate: func(pub *Pub) { pub.SiteTypes = nil }},
		{name: "missing exact floor", mutate: func(pub *Pub) { pub.SlotFloorCPMs = nil }},
		{name: "empty size", mutate: func(pub *Pub) { pub.SlotSizes[7][99] = 0 }},
		{name: "negative exact floor", mutate: func(pub *Pub) { pub.SlotFloorCPMs[7][99] = -1 }},
		{name: "missing accounting marker", mutate: func(pub *Pub) { pub.AccountingVersion = "" }},
		{name: "unknown accounting marker", mutate: func(pub *Pub) { pub.AccountingVersion = "future-version" }},
		{name: "unknown site type", mutate: func(pub *Pub) { pub.SiteTypes[7] = SiteType(99) }},
		{name: "web identity is URL", mutate: func(pub *Pub) { pub.Sites = map[string]uint32{"https://example.com/path": 7} }},
		{name: "web identity has empty label", mutate: func(pub *Pub) { pub.Sites = map[string]uint32{"www..example.com": 7} }},
		{name: "duplicate site id", mutate: func(pub *Pub) { pub.Sites["other.example"] = 7 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pub := valid()
			test.mutate(pub)
			if err := ValidateCommercialPubMap(PubMap{"pub.example": pub}); err == nil {
				t.Fatal("incomplete commercial inventory validated")
			}
		})
	}
	inactive := valid()
	inactive.Active = false
	inactive.SiteTypes = nil
	if err := ValidateCommercialPubMap(PubMap{"inactive.example": inactive}); err != nil {
		t.Fatalf("inactive inventory blocked an unrelated generation: %v", err)
	}
	second := valid()
	second.Sites = map[string]uint32{"other.example": 7}
	if err := ValidateCommercialPubMap(PubMap{"one.example": valid(), "two.example": second}); err == nil {
		t.Fatal("duplicate publisher id across domains validated")
	}
}

func TestCurrentPublisherFloorRequiresExactAuthority(t *testing.T) {
	pub := &Pub{
		AccountingVersion: accounting.ExactMoneyContract,
		SlotFloors:        map[uint32]map[uint32]float64{7: {99: 9.75}},
		SlotFloorCPMs:     map[uint32]map[uint32]accounting.CPM{7: {99: 1_234_567}},
	}
	if floor, ok := pub.ExactSlotFloor(7, 99); !ok || floor != 1_234_567 {
		t.Fatalf("exact floor = %s, %v; want 1.234567, true", floor.String(), ok)
	}
	delete(pub.SlotFloorCPMs[7], 99)
	if floor, ok := pub.ExactSlotFloor(7, 99); ok {
		t.Fatalf("current generation fell back to float floor %s", floor.String())
	}

	pub.AccountingVersion = ""
	pub.SlotFloorCPMs[7][99] = 1_234_567
	if floor, ok := pub.ExactSlotFloor(7, 99); !ok || floor != 9_750_000 {
		t.Fatalf("legacy floor = %s, %v; want 9.750000, true", floor.String(), ok)
	}
	pub.AccountingVersion = "future-version"
	if floor, ok := pub.ExactSlotFloor(7, 99); ok {
		t.Fatalf("unknown accounting version supplied floor %s", floor.String())
	}
}

func TestDirectPublisherFloorResolvesAccountingProvenanceFirst(t *testing.T) {
	pub := &Pub{
		Active:        true,
		Sites:         map[string]uint32{"example.com": 7},
		SiteTypes:     map[uint32]SiteType{7: SiteTypeWeb},
		Slots:         map[uint32]map[string]uint32{7: {"slot": 99}},
		SlotSizes:     map[uint32]map[uint32]uint32{7: {99: 300250}},
		SlotFloors:    map[uint32]map[uint32]float64{7: {99: 9.75}},
		SlotFloorCPMs: map[uint32]map[uint32]accounting.CPM{7: {99: 1_234_567}},
	}
	direct := NewDirectPub("pub.example", pub)
	if version, ok := direct.AccountingContract(); !ok || version != accounting.LegacyMoneyContract {
		t.Fatalf("legacy direct contract = %q, %v", version, ok)
	}
	if _, _, _, floor, ok := direct.CommercialSlotExact(7, 99, 300250); !ok || floor != 9_750_000 {
		t.Fatalf("legacy direct floor = %s, %v; want float-derived 9.750000", floor.String(), ok)
	}
	direct.AccountingVersion = "future-version"
	if _, ok := direct.AccountingContract(); ok {
		t.Fatal("unknown direct accounting marker was accepted")
	}
	if _, _, _, floor, ok := direct.CommercialSlotExact(7, 99, 300250); ok {
		t.Fatalf("unknown direct accounting marker supplied floor %s", floor.String())
	}
}

package acl

import (
	"encoding/base32"
	"testing"
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
			PubID:      42,
			Active:     true,
			Sites:      map[string]uint32{"example.com": 7},
			SiteTypes:  map[uint32]SiteType{7: SiteTypeWeb},
			Slots:      map[uint32]map[string]uint32{7: {"leaderboard": 99}},
			SlotSizes:  map[uint32]map[uint32]uint32{7: {99: (300 << 16) | 250}},
			SlotFloors: map[uint32]map[uint32]float64{7: {99: 1.25}},
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
		{name: "missing floor", mutate: func(pub *Pub) { pub.SlotFloors = nil }},
		{name: "empty size", mutate: func(pub *Pub) { pub.SlotSizes[7][99] = 0 }},
		{name: "negative floor", mutate: func(pub *Pub) { pub.SlotFloors[7][99] = -1 }},
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

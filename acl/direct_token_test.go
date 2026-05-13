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
		PubID:     42,
		Active:    true,
		Sites:     map[string]uint32{"example.com": 7},
		Slots:     map[uint32]map[string]uint32{7: map[string]uint32{"leaderboard": 99}},
		SlotSizes: map[uint32]map[uint32]uint32{7: map[uint32]uint32{99: 12345}},
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
	if got.Pub.PubID != 42 || got.Sites[7] != "example.com" || got.Slots[7][99] != "leaderboard" || got.SlotSizes[7][99] != 12345 {
		t.Fatalf("round trip = %#v", got)
	}
}

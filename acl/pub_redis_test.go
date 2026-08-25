package acl

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/accounting"
	"github.com/mediocregopher/radix/v4"
)

type redisCommandRecorder struct {
	commands []string
}

func (r *redisCommandRecorder) Addr() net.Addr {
	return redisRecorderAddr("redis-recorder")
}

func (r *redisCommandRecorder) Do(_ context.Context, action radix.Action) error {
	r.commands = append(r.commands, fmt.Sprint(action))
	return nil
}

func (r *redisCommandRecorder) Close() error {
	return nil
}

type redisRecorderAddr string

func (a redisRecorderAddr) Network() string { return "test" }
func (a redisRecorderAddr) String() string  { return string(a) }

func TestPubToRedisUpdatesPubmapAndDirectPubByID(t *testing.T) {
	pub := &Pub{
		AccountingVersion: accounting.ExactMoneyContract,
		PubID:             42,
		Active:            true,
		Sites:             map[string]uint32{"example.com": 7},
		Slots:             map[uint32]map[string]uint32{7: {"leaderboard": 99}},
		SlotFloors:        map[uint32]map[uint32]float64{7: {99: 1.75}},
		SlotFloorCPMs:     map[uint32]map[uint32]accounting.CPM{7: {99: 1_750_000}},
	}
	redis := &redisCommandRecorder{}

	if err := pub.ToRedis(context.Background(), redis, "pub.example"); err != nil {
		t.Fatal(err)
	}
	if len(redis.commands) != 2 {
		t.Fatalf("commands = %#v, want 2 commands", redis.commands)
	}
	if !strings.Contains(redis.commands[0], `"HSET" "pubmap" "pub.example"`) {
		t.Fatalf("pubmap command = %s", redis.commands[0])
	}
	if !strings.Contains(redis.commands[1], `"HSET" "pubmap:by-id" "42"`) {
		t.Fatalf("direct by-id command = %s", redis.commands[1])
	}
}

func TestPublisherRedisAndSpreadWritersPreflightBeforeMutation(t *testing.T) {
	invalid := validPublisherWrite()
	invalid.SlotFloors[7][99] = 9.75
	redis := &redisCommandRecorder{}
	if err := invalid.ToRedis(context.Background(), redis, "invalid.example"); err == nil {
		t.Fatal("invalid publisher ToRedis succeeded")
	}
	if len(redis.commands) != 0 {
		t.Fatalf("invalid publisher emitted Redis commands: %#v", redis.commands)
	}

	pubmap := PubMap{
		"delete.example":  {PubID: 7},
		"valid.example":   validPublisherWrite(),
		"invalid.example": invalid,
	}
	if err := pubmap.ToRedisKeys(context.Background(), redis, "pubmap:next", "pubmap:by-id:next"); err == nil {
		t.Fatal("invalid publisher map ToRedisKeys succeeded")
	}
	if len(redis.commands) != 0 {
		t.Fatalf("publisher-map preflight emitted Redis commands: %#v", redis.commands)
	}
	if err := pubmap.ToSpread(nil); err == nil {
		t.Fatal("invalid publisher map ToSpread succeeded")
	}
}

func TestPubMapRedisRoundTripPreservesCommercialSlotPolicy(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	const sizeID uint32 = 19661050
	pubmap := PubMap{
		"pub.example": {
			AccountingVersion: accounting.ExactMoneyContract,
			PubID:             42,
			Active:            true,
			Sites:             map[string]uint32{"example.com": 7},
			SiteTypes:         map[uint32]SiteType{7: SiteTypeWeb},
			Slots:             map[uint32]map[string]uint32{7: {"leaderboard": 99}},
			SlotSizes:         map[uint32]map[uint32]uint32{7: {99: sizeID}},
			SlotFloors:        map[uint32]map[uint32]float64{7: {99: 1.75}},
			SlotFloorCPMs:     map[uint32]map[uint32]accounting.CPM{7: {99: 1_750_000}},
		},
	}
	if err := ValidateCommercialPubMap(pubmap); err != nil {
		t.Fatal(err)
	}
	if err := pubmap.ToRedis(ctx, client); err != nil {
		t.Fatal(err)
	}
	pub, err := PubFromRedis(ctx, client, "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if pub.AccountingVersion != accounting.ExactMoneyContract || pub.SlotFloorCPMs[7][99] != 1_750_000 || pub.SlotFloors[7][99] != 1.75 {
		t.Fatalf("publisher cache lost v3/compatibility floor shapes: %#v", pub)
	}
	direct, err := PubByIDFromRedis(ctx, client, 42)
	if err != nil {
		t.Fatal(err)
	}
	site, slot, siteType, floor, ok := direct.CommercialSlot(7, 99, sizeID)
	if !ok || site != "example.com" || slot != "leaderboard" || siteType != SiteTypeWeb || floor != 1.75 {
		t.Fatalf("CommercialSlot = (%q, %q, %v, %v, %v)", site, slot, siteType, floor, ok)
	}
	if direct.AccountingVersion != accounting.ExactMoneyContract || direct.SlotFloorCPMs[7][99] != 1_750_000 || direct.SlotFloors[7][99] != 1.75 {
		t.Fatalf("direct publisher cache lost v3/compatibility floor shapes: %#v", direct)
	}
}

func TestPubToRedisDeletesPubmapAndDirectPubByID(t *testing.T) {
	tests := []struct {
		name string
		pub  *Pub
	}{
		{
			name: "inactive",
			pub:  &Pub{PubID: 42, Active: false},
		},
		{
			name: "limited",
			pub:  &Pub{PubID: 43, Active: true, LimitImps: 10, CurrentImps: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redis := &redisCommandRecorder{}
			if err := tt.pub.ToRedis(context.Background(), redis, "pub.example"); err != nil {
				t.Fatal(err)
			}
			if len(redis.commands) != 2 {
				t.Fatalf("commands = %#v, want 2 commands", redis.commands)
			}
			if !strings.Contains(redis.commands[0], `"HDEL" "pubmap" "pub.example"`) {
				t.Fatalf("pubmap delete command = %s", redis.commands[0])
			}
			wantByID := fmt.Sprintf(`"HDEL" "pubmap:by-id" "%d"`, tt.pub.PubID)
			if !strings.Contains(redis.commands[1], wantByID) {
				t.Fatalf("direct by-id delete command = %s, want %s", redis.commands[1], wantByID)
			}
		})
	}
}

func TestDBGetPubMapFiltersInactiveSitesAndSlots(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"domain", "pub_id", "active", "foreign_id", "site_id", "site_type", "slot_name", "slot_id", "size_id", "bidfloor", "limit_imp", "current_imp",
		"seller_id", "seller_type", "seller_asi", "seller_name", "seller_domain", "seller_authorized",
		"inventory_environment", "canonical_identity", "store_url", "integration_mode",
		"media_intent", "placement", "render_context", "refresh_mode", "refresh_seconds",
		"ad_density", "traffic_quality", "source_quality", "management_control",
	}).
		AddRow("pub.example", 42, "Yes", "example.com", 7, "Web", "leaderboard", 99, 300250, 1.75, nil, nil,
			"seller-42", "Publisher", "w8m.com", "Example", "example.com", "Yes",
			"Web", "example.com", "https://example.com", "BrowserTag",
			"Banner", "AboveFold", "WebPage", "None", 0, "Standard", "Reviewed", "OwnedOperated", "Publisher")
	mock.ExpectQuery(`(?s)WHERE s\.active='Yes' AND t\.active='Yes'`).
		WillReturnRows(rows)

	pubmap, err := DBGetPubMap(db)
	if err != nil {
		t.Fatal(err)
	}
	if pubmap["pub.example"].Slots[7]["leaderboard"] != 99 {
		t.Fatalf("pubmap = %#v", pubmap)
	}
	if pubmap["pub.example"].SiteTypes[7] != SiteTypeWeb || pubmap["pub.example"].SlotFloors[7][99] != 1.75 {
		t.Fatalf("commercial metadata = %#v", pubmap["pub.example"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

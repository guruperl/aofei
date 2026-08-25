package match

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/guruperl/aofei/acl"
	"github.com/mediocregopher/radix/v4"
)

func TestWriteMiddlemanRouteCacheKeysPublishesBothVersions(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	cache := &MiddlemanRouteCache{
		Version: MiddlemanRouteCacheVersion,
		Entries: []MiddlemanRouteEntry{{AccountingVersion: MiddlemanRouteAccountingVersion, BidderID: 7, TriggerMode: "Fallback"}},
	}
	if err := writeMiddlemanRouteCacheKeys(ctx, client, cache, "routes:legacy", "routes:v2"); err != nil {
		t.Fatal(err)
	}
	for key, wantVersion := range map[string]int{"routes:legacy": MiddlemanRouteCacheLegacyVersion, "routes:v2": MiddlemanRouteCacheVersion} {
		data, err := server.Get(key)
		if err != nil {
			t.Fatal(err)
		}
		var got MiddlemanRouteCache
		if err := json.Unmarshal([]byte(data), &got); err != nil {
			t.Fatal(err)
		}
		if got.Version != wantVersion {
			t.Fatalf("%s version = %d, want %d", key, got.Version, wantVersion)
		}
	}
	if err := writeMiddlemanRouteCacheKeys(ctx, client, cache, "same", "same"); err == nil {
		t.Fatal("identical route cache keys were accepted")
	}
}

func TestDBValidateMiddlemanActivationChecksEveryTopologyBoundary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for range 5 {
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	}
	if err := DBValidateMiddlemanActivation(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBValidateMiddlemanActivationRejectsEnabledSyntheticDemand(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for range 3 {
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	err = DBValidateMiddlemanActivation(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "synthetic reporting rows enabled") {
		t.Fatalf("activation error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestMiddlemanRouteEntryEligibility(t *testing.T) {
	entityType, entityID := EntityPointer(32, 333)
	sizeID := Uint32Pointer(4194368)
	entry := MiddlemanRouteEntry{
		EntityTypeID: entityType,
		EntityID:     entityID,
		SizeID:       sizeID,
		Audience: &acl.ACLAudience{
			SiteTypes: acl.SiteTypeWeb,
			BPub:      []string{"blocked.example"},
		},
	}
	attr := &Attribute{
		RPub: RPub{
			PubID:  111,
			SiteID: 222,
			SlotID: 333,
			SizeID: 4194368,
		},
		ACL: &acl.ACL{
			PubStr:   "allowed.example",
			SiteType: acl.SiteTypeWeb,
			SiteStr:  "site.example",
		},
	}
	if !entry.Eligible(attr) {
		t.Fatalf("entry should be eligible")
	}

	attr.SlotID = 334
	if entry.Eligible(attr) {
		t.Fatalf("slot-specific entry matched the wrong slot")
	}
	attr.SlotID = 333
	attr.ACL.PubStr = "blocked.example"
	if entry.Eligible(attr) {
		t.Fatalf("blocked publisher should fail ACL eligibility")
	}

	partial := MiddlemanRouteEntry{EntityTypeID: entityType}
	if partial.Eligible(attr) {
		t.Fatalf("partial route target should not match globally")
	}
}

func TestMiddlemanRouteCacheMetadataIsAdditive(t *testing.T) {
	raw := []byte(`{"version":1,"entries":[]}`)
	var cache MiddlemanRouteCache
	if err := json.Unmarshal(raw, &cache); err != nil {
		t.Fatal(err)
	}
	if cache.Metadata != nil {
		t.Fatalf("metadata = %#v, want nil for old cache payload", cache.Metadata)
	}

	cache.Entries = []MiddlemanRouteEntry{{TargetID: 1, GroupID: 2, RouteBidderID: 3, BidderID: 4}}
	checksum := cache.RouteChecksum()
	if checksum == "" {
		t.Fatal("checksum is empty")
	}
	cache.Metadata = &MiddlemanRouteCacheMetadata{
		GeneratedAt:      "2026-05-13T00:00:00Z",
		EntryCount:       len(cache.Entries),
		Source:           "mysql",
		RouteDBHighWater: "2026-05-13T00:00:00Z",
		Checksum:         checksum,
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("metadata payload is invalid JSON: %s", data)
	}
}

func TestMiddlemanRouteCurrentCacheRejectsMissingOrUnknownAccountingVersion(t *testing.T) {
	server := miniredis.RunT(t)
	ctx := context.Background()
	client, err := (radix.PoolConfig{Size: 1}).New(ctx, "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, version := range []string{"", "future-version"} {
		cache := &MiddlemanRouteCache{
			Version: MiddlemanRouteCacheVersion,
			Entries: []MiddlemanRouteEntry{{AccountingVersion: version}},
		}
		data, err := json.Marshal(cache)
		if err != nil {
			t.Fatal(err)
		}
		server.Set(HashNameMiddlemanRoutesV2, string(data))
		if _, err := MiddlemanRouteCacheFromRedis(ctx, client); err == nil {
			t.Fatalf("current route entry accounting version %q was accepted", version)
		}
	}
}

func TestMiddlemanRouteLegacyFallbackCacheExcludesAlways(t *testing.T) {
	cache := &MiddlemanRouteCache{
		Version: MiddlemanRouteCacheVersion,
		Entries: []MiddlemanRouteEntry{
			{AccountingVersion: MiddlemanRouteAccountingVersion, TargetID: 1, GroupID: 1, RouteBidderID: 1, BidderID: 1, TriggerMode: "Fallback"},
			{AccountingVersion: MiddlemanRouteAccountingVersion, TargetID: 2, GroupID: 2, RouteBidderID: 2, BidderID: 2, TriggerMode: "Always"},
		},
		Metadata: &MiddlemanRouteCacheMetadata{
			GeneratedAt:      "2026-05-13T00:00:00Z",
			RouteDBHighWater: "2026-05-13T00:00:00Z",
		},
	}
	legacy := cache.legacyFallbackCache()
	if legacy.Version != MiddlemanRouteCacheLegacyVersion {
		t.Fatalf("legacy version = %d, want %d", legacy.Version, MiddlemanRouteCacheLegacyVersion)
	}
	if len(legacy.Entries) != 1 || legacy.Entries[0].TargetID != 1 {
		t.Fatalf("legacy entries = %#v, want only fallback entry", legacy.Entries)
	}
	if legacy.Entries[0].TriggerMode != "" {
		t.Fatalf("legacy trigger mode = %q, want omitted fallback mode", legacy.Entries[0].TriggerMode)
	}
	if legacy.Metadata == nil || legacy.Metadata.EntryCount != 1 || legacy.Metadata.RouteDBHighWater == "" {
		t.Fatalf("legacy metadata = %#v", legacy.Metadata)
	}
}

func TestMiddlemanRouteEntrySpecificity(t *testing.T) {
	slotType, slotID := EntityPointer(32, 333)
	siteType, siteID := EntityPointer(31, 222)

	global := MiddlemanRouteEntry{}
	site := MiddlemanRouteEntry{EntityTypeID: siteType, EntityID: siteID}
	slotWithSize := MiddlemanRouteEntry{EntityTypeID: slotType, EntityID: slotID, SizeID: Uint32Pointer(4194368)}

	if !(slotWithSize.Specificity() > site.Specificity() && site.Specificity() > global.Specificity()) {
		t.Fatalf("specificity order = slot+size %d, site %d, global %d", slotWithSize.Specificity(), site.Specificity(), global.Specificity())
	}
}

func TestMiddlemanRouteEntryValidatesOpenRTB25PartnerProfile(t *testing.T) {
	valid := MiddlemanRouteEntry{
		BidderID: 1, AdvID: 2,
		SyntheticCampaignID: 3, SyntheticItemID: 4, SyntheticCreativeID: 5,
		EndpointURL: "https://bidder.example/openrtb", OpenRTBVersion: "2.5",
		CredentialRef: "MIDDLEMAN_BIDDER_HEADERS", GroupTimeoutMS: 200, BidderTimeoutMS: 100,
	}
	if err := valid.ValidatePartnerProfile(); err != nil {
		t.Fatalf("valid profile: %v", err)
	}
	for name, mutate := range map[string]func(*MiddlemanRouteEntry){
		"version":         func(e *MiddlemanRouteEntry) { e.OpenRTBVersion = "2.6" },
		"relative URL":    func(e *MiddlemanRouteEntry) { e.EndpointURL = "/bid" },
		"credential":      func(e *MiddlemanRouteEntry) { e.CredentialRef = "" },
		"credential name": func(e *MiddlemanRouteEntry) { e.CredentialRef = "secret/ref" },
		"timeout":         func(e *MiddlemanRouteEntry) { e.BidderTimeoutMS = 0 },
		"synthetic id":    func(e *MiddlemanRouteEntry) { e.SyntheticCreativeID = 0 },
		"seat control":    func(e *MiddlemanRouteEntry) { e.Seat = "seat\nother" },
	} {
		t.Run(name, func(t *testing.T) {
			entry := valid
			mutate(&entry)
			if err := entry.ValidatePartnerProfile(); err == nil {
				t.Fatal("invalid profile was accepted")
			}
		})
	}
}

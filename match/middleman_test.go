package match

import (
	"encoding/json"
	"testing"

	"github.com/genelet/winter/acl"
)

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

func TestMiddlemanRouteLegacyFallbackCacheExcludesAlways(t *testing.T) {
	cache := &MiddlemanRouteCache{
		Version: MiddlemanRouteCacheVersion,
		Entries: []MiddlemanRouteEntry{
			{TargetID: 1, GroupID: 1, RouteBidderID: 1, BidderID: 1, TriggerMode: "Fallback"},
			{TargetID: 2, GroupID: 2, RouteBidderID: 2, BidderID: 2, TriggerMode: "Always"},
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

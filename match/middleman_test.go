package match

import (
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

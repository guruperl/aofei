package acl

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestNewOpenRTBACLSetsSiteTypeAndImpSlot(t *testing.T) {
	req := &openrtb2.BidRequest{
		App: &openrtb2.App{
			Bundle: "com.example.app",
		},
		Imp: []openrtb2.Imp{
			{ID: "one", TagID: "slot-one"},
			{ID: "two", TagID: "slot-two"},
		},
	}

	got := NewOpenRTBACLForImp(req, 1, "pub.example")
	if got.SiteType != SiteTypeAPP {
		t.Fatalf("SiteType = %d, want app", got.SiteType)
	}
	if got.SiteStr != "com.example.app" {
		t.Fatalf("SiteStr = %q, want bundle", got.SiteStr)
	}
	if got.SlotStr != "slot-two" {
		t.Fatalf("SlotStr = %q, want per-impression tag", got.SlotStr)
	}
}

func TestNewOpenRTBACLWebSiteType(t *testing.T) {
	req := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "site.example"},
		Imp:  []openrtb2.Imp{{ID: "one", TagID: "slot-one"}},
	}

	got := NewOpenRTBACL(req, "pub.example")
	if got.SiteType != SiteTypeWeb {
		t.Fatalf("SiteType = %d, want web", got.SiteType)
	}
	if got.SiteStr != "site.example" {
		t.Fatalf("SiteStr = %q, want site domain", got.SiteStr)
	}
}

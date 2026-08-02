package managementapi

import (
	"math"
	"strings"
	"testing"
)

func TestDeliveryAndTargetingValidation(t *testing.T) {
	schedule := strings.Repeat("1", 168)
	campaign := campaignWrite{Name: " Campaign ", Delivery: DeliveryPolicy{Timezone: "Asia/Shanghai", WeeklySchedule: &schedule, Pacing: "Even"}}
	if err := validateCampaignWrite(&campaign); err != nil {
		t.Fatal(err)
	}
	if campaign.Name != "Campaign" {
		t.Fatalf("name was not normalized: %q", campaign.Name)
	}
	bad := schedule[:167]
	campaign.Delivery.WeeklySchedule = &bad
	if err := validateCampaignWrite(&campaign); err == nil {
		t.Fatal("short weekly schedule accepted")
	}
	item := itemWrite{Name: "item", LandingURL: "https://example.invalid", PriceCPMUSD: 1, Delivery: DeliveryPolicy{Timezone: "UTC"}}
	if err := validateItemWrite(&item); err == nil {
		t.Fatal("item accepted a caller-supplied campaign timezone")
	}
	item.Delivery.Timezone = ""
	item.LandingURL = "  https://example.invalid/landing  "
	item.ImpressionURLs = []string{" https://tracker.example.invalid/imp "}
	if err := validateItemWrite(&item); err != nil {
		t.Fatal(err)
	}
	if item.LandingURL != "https://example.invalid/landing" || item.ImpressionURLs[0] != "https://tracker.example.invalid/imp" {
		t.Fatalf("item URLs were not normalized: %#v", item)
	}
	item.ImpressionURLs = []string{"https://tracker.example.invalid/imp,a"}
	if err := validateItemWrite(&item); err == nil {
		t.Fatal("comma-delimited tracker storage accepted a URL containing comma")
	}
	tooMany := uint64(math.MaxUint32) + 1
	item.ImpressionURLs = nil
	item.Delivery.TotalLimits.Imps = &tooMany
	if err := validateItemWrite(&item); err == nil {
		t.Fatal("database-overflowing impression limit was accepted")
	}

	targeting := targetingWrite{
		SiteTypes: []string{"Web", "Web", "App"}, Languages: []string{"ZH", "EN"},
		DeviceTypes: []string{"2", "1"}, Positions: []string{"3", "1"},
	}
	if err := validateTargetingWrite(&targeting); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(targeting.SiteTypes, ","); got != "App,Web" {
		t.Fatalf("site type normalization = %q", got)
	}
}

func TestCreativeValidationUsesSourceOnlyContract(t *testing.T) {
	banner := creativeWrite{Name: "banner", Width: 300, Height: 250, MediaType: "Banner", SourceURL: "https://cdn.example/ad.png", Weight: 1}
	if err := validateCreativeWrite(&banner); err != nil {
		t.Fatal(err)
	}
	banner.SourceURL = "javascript:alert(1)"
	if err := validateCreativeWrite(&banner); err == nil {
		t.Fatal("executable source URL accepted")
	}
	native := creativeWrite{Name: "native", Width: 300, Height: 250, MediaType: "Native", Weight: 1, Native: &NativeCreative{Version: "1", Title: "Title", Description: "Description", MainImageURL: "https://cdn.example/main.webp"}}
	if err := validateCreativeWrite(&native); err != nil {
		t.Fatal(err)
	}
}

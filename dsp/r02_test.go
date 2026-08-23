package dsp

import (
	"net/url"
	"testing"
	"time"

	"github.com/guruperl/aofei/acl"
	"github.com/guruperl/aofei/advice"
	"github.com/guruperl/aofei/match"
	"github.com/guruperl/aofei/maxmind"
)

func TestReportingDimensionsUseOnlyCoarseNumericClassifications(t *testing.T) {
	attribute := &match.Attribute{
		When: time.Now(),
		Geo:  &maxmind.Geo{CountryID: 44, StateID: 55, CityID: 66, ZipID: 77},
		PzUa: &advice.PzUa{OS: advice.DeviceOS(3), Device: advice.DeviceType(2)},
		Supply: acl.SupplyMetadata{
			Seller: acl.SellerMetadata{ID: "seller-7", Type: "Publisher", ASI: "w8m.com", Authorized: true},
			Site:   acl.SiteSupplyMetadata{Environment: "Web", IntegrationMode: "BrowserTag"},
			Slot:   acl.SlotSupplyMetadata{MediaIntent: "Banner", Placement: "AboveFold", RenderContext: "WebPage", RefreshMode: "Timed", RefreshSeconds: 60, AdDensity: "Standard", TrafficQuality: "Reviewed", SourceQuality: "OwnedOperated", ManagementControl: "Publisher"},
		},
	}
	dimensions := reportingDimensionsFromAttribute(attribute)
	if dimensions.CountryID != 44 || dimensions.StateID != 55 || dimensions.DeviceOS != 3 || dimensions.DeviceType != 2 {
		t.Fatalf("reporting dimensions = %#v", dimensions)
	}
	values, err := (&WinLoss{Reporting: dimensions}).packURLValues(true)
	if err != nil {
		t.Fatalf("pack tracking values: %v", err)
	}
	for key, want := range map[string]string{
		"report_country_id": "44", "report_state_id": "55",
		"report_device_os": "3", "report_device_type": "2",
		"report_environment": "Web", "report_integration": "BrowserTag",
		"report_media_intent": "Banner", "report_placement": "AboveFold",
		"report_refresh_mode": "Timed", "report_refresh_seconds": "60",
		"report_seller_type": "Publisher", "report_seller_id": "seller-7",
	} {
		if got := values.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for _, forbidden := range []string{"city", "zip", "ip", "ua", "ifa", "consent"} {
		if values.Get(forbidden) != "" {
			t.Errorf("tracking values contain forbidden dimension %q", forbidden)
		}
	}
}

func TestReportingDimensionsFromTrackingRejectsOverflow(t *testing.T) {
	dimensions, err := reportingDimensionsFromTracking(url.Values{
		"report_country_id": {"44"}, "report_device_type": {"2"},
	})
	if err != nil || dimensions.CountryID != 44 || dimensions.DeviceType != 2 {
		t.Fatalf("valid dimensions = %#v, %v", dimensions, err)
	}
	if _, err := reportingDimensionsFromTracking(url.Values{"report_device_type": {"256"}}); err == nil {
		t.Fatal("overflowing reporting device type was accepted")
	}
	if _, err := reportingDimensionsFromTracking(url.Values{"report_environment": {"<script>"}}); err == nil {
		t.Fatal("hostile supply reporting dimension was accepted")
	}
	if _, err := reportingDimensionsFromTracking(url.Values{"report_seller_type": {"Unknown"}, "report_seller_id": {"seller-1"}}); err == nil {
		t.Fatal("unapproved seller id was accepted")
	}
	if _, err := reportingDimensionsFromTracking(url.Values{"report_refresh_mode": {"Timed"}, "report_refresh_seconds": {"14"}}); err == nil {
		t.Fatal("invalid reporting refresh interval was accepted")
	}
}

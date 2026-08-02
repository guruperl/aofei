package acl

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDBGetPubSetsDefaultAppAndWebSlots(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"pub_id", "active", "foreign_id", "site_id", "site_type", "slot_name", "slot_id", "size_id", "bidfloor", "limit_imp", "current_imp",
		"seller_id", "seller_type", "seller_asi", "seller_name", "seller_domain", "seller_authorized",
		"inventory_environment", "canonical_identity", "store_url", "integration_mode",
		"media_intent", "placement", "render_context", "refresh_mode", "refresh_seconds",
		"ad_density", "traffic_quality", "source_quality", "management_control",
	}).
		AddRow(7, "Yes", SITEDefaultApp, 11, "App", SLOTDefault, 101, 300250, 1.25, nil, nil,
			"seller-7", "Publisher", "w8m.com", "Example", "example.com", "Yes",
			"App", "com.example.app", "https://example.com/app", "SDK",
			"Banner", "InFeed", "InApp", "None", 0, "Standard", "Reviewed", "OwnedOperated", "Publisher").
		AddRow(7, "Yes", SITEDefaultWeb, 12, "Web", SLOTDefault, 102, 300250, 2.5, nil, nil,
			"seller-7", "Publisher", "w8m.com", "Example", "example.com", "Yes",
			"Web", "example.com", "https://example.com", "BrowserTag",
			"Banner", "AboveFold", "WebPage", "Timed", 30, "Low", "Reviewed", "OwnedOperated", "Publisher")
	mock.ExpectQuery(`(?s)SELECT p\.pub_id.*WHERE domain = \?`).
		WithArgs("pub.example").
		WillReturnRows(rows)

	pub, err := DBGetPub(db, "pub.example")
	if err != nil {
		t.Fatal(err)
	}
	if pub.DefaultAppSiteID != 11 || pub.DefaultAppSlotID != 101 {
		t.Fatalf("app defaults = site %d slot %d, want 11/101", pub.DefaultAppSiteID, pub.DefaultAppSlotID)
	}
	if pub.DefaultWebSiteID != 12 || pub.DefaultWebSlotID != 102 {
		t.Fatalf("web defaults = site %d slot %d, want 12/102", pub.DefaultWebSiteID, pub.DefaultWebSlotID)
	}
	if pub.SiteTypes[11] != SiteTypeAPP || pub.SiteTypes[12] != SiteTypeWeb || pub.SlotFloors[12][102] != 2.5 {
		t.Fatalf("commercial metadata = types %#v floors %#v", pub.SiteTypes, pub.SlotFloors)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBGetPubFiltersInactiveSitesAndSlots(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"pub_id", "active", "foreign_id", "site_id", "site_type", "slot_name", "slot_id", "size_id", "bidfloor", "limit_imp", "current_imp",
		"seller_id", "seller_type", "seller_asi", "seller_name", "seller_domain", "seller_authorized",
		"inventory_environment", "canonical_identity", "store_url", "integration_mode",
		"media_intent", "placement", "render_context", "refresh_mode", "refresh_seconds",
		"ad_density", "traffic_quality", "source_quality", "management_control",
	}).
		AddRow(7, "Yes", "example.com", 11, "Web", "leaderboard", 101, 300250, 0, nil, nil,
			"", "Publisher", "", "", "", "No",
			"Unknown", "", "", "Unknown",
			"Unknown", "Unknown", "Unknown", "Unknown", 0, "Unknown", "Unknown", "Unknown", "Unknown")
	mock.ExpectQuery(`(?s)WHERE domain = \? AND s\.active='Yes' AND t\.active='Yes'`).
		WithArgs("pub.example").
		WillReturnRows(rows)

	if _, err := DBGetPub(db, "pub.example"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

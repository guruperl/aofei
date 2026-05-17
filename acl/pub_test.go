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
		"pub_id", "active", "foreign_id", "site_id", "site_type", "slot_name", "slot_id", "size_id", "limit_imp", "current_imp",
	}).
		AddRow(7, "Yes", SITEDefaultApp, 11, "App", SLOTDefault, 101, 300250, nil, nil).
		AddRow(7, "Yes", SITEDefaultWeb, 12, "Web", SLOTDefault, 102, 300250, nil, nil)
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
		"pub_id", "active", "foreign_id", "site_id", "site_type", "slot_name", "slot_id", "size_id", "limit_imp", "current_imp",
	}).
		AddRow(7, "Yes", "example.com", 11, "Web", "leaderboard", 101, 300250, nil, nil)
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

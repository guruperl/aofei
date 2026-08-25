package acl

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/accounting"
)

func TestDBAddNewContextRejectsCanceledInventoryMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pub := &Pub{
		PubID: 7,
		Sites: map[string]uint32{"site.example": 11},
		Slots: map[uint32]map[string]uint32{11: {}},
	}
	_, err = (PubMap{"pub.example": pub}).DBAddNewContext(ctx, db, "pub.example", "site.example", "Web", "new-slot")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DBAddNewContext error = %v, want context canceled", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBAddNewContextRollsBackCompoundSiteCreation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pub := &Pub{PubID: 7, Sites: make(map[string]uint32)}
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO pub_site`).
		WithArgs(uint32(7), "site.example", "site.example", "site.example", "Web").
		WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec(`INSERT INTO pub_slot`).
		WithArgs(uint32(11), "slot.example").
		WillReturnError(errors.New("slot insert failed"))
	mock.ExpectRollback()

	_, err = (PubMap{"pub.example": pub}).DBAddNewContext(
		context.Background(), db, "pub.example", "site.example", "Web", "slot.example",
	)
	if err == nil {
		t.Fatal("DBAddNewContext error = nil, want slot failure")
	}
	if _, ok := pub.Sites["site.example"]; ok {
		t.Fatal("rolled-back site was installed in the in-memory publisher")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDBAddNewContextCommitsSiteAndSlotTogether(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pub := &Pub{PubID: 7, Sites: make(map[string]uint32)}
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO pub_site`).
		WithArgs(uint32(7), "site.example", "site.example", "site.example", "Web").
		WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec(`INSERT INTO pub_slot`).
		WithArgs(uint32(11), "slot.example").
		WillReturnResult(sqlmock.NewResult(13, 1))
	mock.ExpectCommit()

	got, err := (PubMap{"pub.example": pub}).DBAddNewContext(
		context.Background(), db, "pub.example", "site.example", "Web", "slot.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != pub || pub.Sites["site.example"] != 11 || pub.Slots[11]["slot.example"] != 13 {
		t.Fatalf("committed inventory = %+v", pub)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAddPubContextRollsBackIncompleteDefaults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO pub \(pub_id, domain, email, passwd, address_id, active, created\)`).
		WithArgs(sqlmock.AnyArg(), "pub.example", "pub.example").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO pub_site`).
		WithArgs(sqlmock.AnyArg(), SITEDefaultApp, SITEDefaultApp, SITEDefaultApp, "App").
		WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec(`INSERT INTO pub_site`).
		WithArgs(sqlmock.AnyArg(), SITEDefaultWeb, SITEDefaultWeb, SITEDefaultWeb, "Web").
		WillReturnError(errors.New("web default insert failed"))
	mock.ExpectRollback()

	pub, err := AddPubContext(context.Background(), db, "pub.example")
	if err == nil || pub != nil {
		t.Fatalf("AddPubContext pub=%+v error=%v, want rolled-back failure", pub, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertPublisherRetriesPrimaryKeyCollision(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insert := `INSERT INTO pub \(pub_id, domain, email, passwd, address_id, active, created\)`
	mock.ExpectExec(insert).WithArgs(uint32(7), "pub.example", "pub.example").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate PRIMARY"})
	mock.ExpectQuery(`SELECT email FROM pub WHERE pub_id=\? LIMIT 1`).WithArgs(uint32(7)).
		WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("collision.example"))
	mock.ExpectExec(insert).WithArgs(uint32(8), "pub.example", "pub.example").
		WillReturnResult(sqlmock.NewResult(0, 1))
	ids := []uint32{7, 8}
	got, err := insertPublisher(db, "pub.example", func() (uint32, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 8 {
		t.Fatalf("publisher id = %d, want 8", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInsertPublisherDoesNotRetryDuplicateEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(`INSERT INTO pub \(pub_id, domain, email, passwd, address_id, active, created\)`).
		WithArgs(uint32(7), "pub.example", "pub.example").
		WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate email"})
	mock.ExpectQuery(`SELECT email FROM pub WHERE pub_id=\? LIMIT 1`).WithArgs(uint32(7)).
		WillReturnRows(sqlmock.NewRows([]string{"email"}))
	_, err = insertPublisher(db, "pub.example", func() (uint32, error) { return 7, nil })
	if err == nil {
		t.Fatal("duplicate publisher email succeeded")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

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
	if pub.AccountingVersion != accounting.ExactMoneyContract || pub.SiteTypes[11] != SiteTypeAPP || pub.SiteTypes[12] != SiteTypeWeb ||
		pub.SlotFloors[12][102] != 2.5 || pub.SlotFloorCPMs[12][102] != 2_500_000 {
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

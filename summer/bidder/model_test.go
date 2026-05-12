package bidder

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestApproveCreatesSyntheticChain(t *testing.T) {
	db := openSummerTestDB(t)
	defer db.Close()

	name := uniqueBidderName("create")
	bidderID := insertTestBidder(t, db, 1, name, 0, 0, 0)
	defer cleanupBidderChain(t, db, bidderID, 0, 0, 0)

	lists := make([]map[string]interface{}, 0)
	model := approvalModel(db, bidderID, "secret/ref/create", &lists)
	if err := model.Approve(); err != nil {
		t.Fatal(err)
	}

	campaignID, itemID, creativeID := approvedSyntheticIDs(t, db, bidderID)
	defer cleanupBidderChain(t, db, bidderID, campaignID, itemID, creativeID)
	assertSyntheticChain(t, db, 1, campaignID, itemID, creativeID)
	assertBidderApproved(t, db, bidderID, "secret/ref/create")
	if len(lists) != 1 || lists[0]["active"] != "Yes" {
		t.Fatalf("approval lists = %#v", lists)
	}
}

func TestApproveReusesExistingSyntheticChain(t *testing.T) {
	db := openSummerTestDB(t)
	defer db.Close()

	campaignID, itemID, creativeID := insertSyntheticChain(t, db, 1, uniqueBidderName("reuse-chain"))
	name := uniqueBidderName("reuse")
	bidderID := insertTestBidder(t, db, 1, name, campaignID, itemID, creativeID)
	defer cleanupBidderChain(t, db, bidderID, campaignID, itemID, creativeID)

	lists := make([]map[string]interface{}, 0)
	model := approvalModel(db, bidderID, "secret/ref/reuse", &lists)
	if err := model.Approve(); err != nil {
		t.Fatal(err)
	}

	gotCampaignID, gotItemID, gotCreativeID := approvedSyntheticIDs(t, db, bidderID)
	if gotCampaignID != campaignID || gotItemID != itemID || gotCreativeID != creativeID {
		t.Fatalf("synthetic IDs = %d/%d/%d, want %d/%d/%d",
			gotCampaignID, gotItemID, gotCreativeID, campaignID, itemID, creativeID)
	}
	assertBidderApproved(t, db, bidderID, "secret/ref/reuse")
}

func TestApproveRejectsPartialSyntheticChain(t *testing.T) {
	db := openSummerTestDB(t)
	defer db.Close()

	campaignID, itemID, creativeID := insertSyntheticChain(t, db, 1, uniqueBidderName("partial-chain"))
	name := uniqueBidderName("partial")
	bidderID := insertTestBidder(t, db, 1, name, campaignID, 0, 0)
	defer cleanupBidderChain(t, db, bidderID, campaignID, itemID, creativeID)

	lists := make([]map[string]interface{}, 0)
	model := approvalModel(db, bidderID, "secret/ref/partial", &lists)
	if err := model.Approve(); err == nil || !strings.Contains(err.Error(), "partial synthetic") {
		t.Fatalf("Approve error = %v, want partial synthetic error", err)
	}
}

func TestApproveRejectsWrongAdvertiserChain(t *testing.T) {
	db := openSummerTestDB(t)
	defer db.Close()

	otherAdvID := insertTestAdvertiser(t, db)
	defer deleteTestAdvertiser(t, db, otherAdvID)

	campaignID, itemID, creativeID := insertSyntheticChain(t, db, otherAdvID, uniqueBidderName("wrong-chain"))
	name := uniqueBidderName("wrong")
	bidderID := insertTestBidder(t, db, 1, name, campaignID, itemID, creativeID)
	defer cleanupBidderChain(t, db, bidderID, campaignID, itemID, creativeID)

	lists := make([]map[string]interface{}, 0)
	model := approvalModel(db, bidderID, "secret/ref/wrong", &lists)
	if err := model.Approve(); err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("Approve error = %v, want advertiser-chain error", err)
	}
}

func approvalModel(db *sql.DB, bidderID int64, credentialRef string, lists *[]map[string]interface{}) *Model {
	other := map[string]interface{}{}
	model := &Model{}
	model.DB = db
	model.SetDefaults(url.Values{
		"bidder_id":      {fmt.Sprintf("%d", bidderID)},
		"credential_ref": {credentialRef},
	}, lists, &other, nil)
	return model
}

func uniqueBidderName(prefix string) string {
	return fmt.Sprintf("adv_bidder_test_%s_%d", prefix, time.Now().UnixNano())
}

func insertTestBidder(t *testing.T, db *sql.DB, advID int64, name string, campaignID, itemID, creativeID int64) int64 {
	t.Helper()

	result, err := db.Exec(`
INSERT INTO adv_bidder
	(adv_id, synthetic_campaign_id, synthetic_item_id, synthetic_creative_id,
		bidder_name, endpoint_url, openrtb_version, timeout_ms)
VALUES (?, NULLIF(?, 0), NULLIF(?, 0), NULLIF(?, 0), ?, ?, '2.5', 100)`,
		advID, campaignID, itemID, creativeID, name, "https://bidder.example/openrtb")
	if err != nil {
		t.Fatal(err)
	}
	bidderID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return bidderID
}

func insertSyntheticChain(t *testing.T, db *sql.DB, advID int64, name string) (int64, int64, int64) {
	t.Helper()

	result, err := db.Exec(`
INSERT INTO adv_campaign
	(adv_id, campaign_name, foreign_id, access_order, active, created)
VALUES (?, ?, ?, 'Inherit', 'No', NOW())`, advID, name, name)
	if err != nil {
		t.Fatal(err)
	}
	campaignID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	result, err = db.Exec(`
INSERT INTO adv_item
	(campaign_id, item_name, item_click, cost_type, cost, fl_sitetypes,
		access_order, channel_order, active, created)
VALUES (?, ?, 'about:blank', 'CPM', 0, 'App,Web',
	'Inherit', 'Black', 'No', NOW())`, campaignID, name)
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	result, err = db.Exec(`
INSERT INTO adv_creative
	(creative_name, item_id, size_id, media_type, active, created)
VALUES (?, ?, ?, 'Banner', 'No', NOW())`, name, itemID, syntheticCreativeSizeID)
	if err != nil {
		t.Fatal(err)
	}
	creativeID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return campaignID, itemID, creativeID
}

func approvedSyntheticIDs(t *testing.T, db *sql.DB, bidderID int64) (int64, int64, int64) {
	t.Helper()

	var campaignID, itemID, creativeID int64
	err := db.QueryRow(`
SELECT synthetic_campaign_id, synthetic_item_id, synthetic_creative_id
FROM adv_bidder
WHERE bidder_id=?`, bidderID).Scan(&campaignID, &itemID, &creativeID)
	if err != nil {
		t.Fatal(err)
	}
	return campaignID, itemID, creativeID
}

func assertSyntheticChain(t *testing.T, db *sql.DB, advID, campaignID, itemID, creativeID int64) {
	t.Helper()

	var campaignActive, itemClick, itemCostType, itemSiteTypes, itemAccess, itemChannel, itemActive, creativeMedia, creativeActive string
	var campaignAdvID, itemCampaignID, creativeItemID, creativeSizeID int64
	var itemCost float64
	err := db.QueryRow(`
SELECT c.adv_id, c.active, i.campaign_id, i.item_click, i.cost_type, i.cost,
	i.fl_sitetypes, i.access_order, i.channel_order, i.active,
	cr.item_id, cr.size_id, cr.media_type, cr.active
FROM adv_campaign c
INNER JOIN adv_item i ON i.campaign_id=c.campaign_id
INNER JOIN adv_creative cr ON cr.item_id=i.item_id
WHERE c.campaign_id=? AND i.item_id=? AND cr.creative_id=?`,
		campaignID, itemID, creativeID).Scan(
		&campaignAdvID, &campaignActive, &itemCampaignID, &itemClick, &itemCostType, &itemCost,
		&itemSiteTypes, &itemAccess, &itemChannel, &itemActive,
		&creativeItemID, &creativeSizeID, &creativeMedia, &creativeActive,
	)
	if err != nil {
		t.Fatal(err)
	}
	if campaignAdvID != advID || campaignActive != "No" ||
		itemCampaignID != campaignID || itemClick != "about:blank" || itemCostType != "CPM" ||
		itemCost != 0 || itemSiteTypes != "App,Web" || itemAccess != "Inherit" ||
		itemChannel != "Black" || itemActive != "No" ||
		creativeItemID != itemID || creativeSizeID != syntheticCreativeSizeID ||
		creativeMedia != "Banner" || creativeActive != "No" {
		t.Fatalf("synthetic chain defaults mismatch")
	}
}

func assertBidderApproved(t *testing.T, db *sql.DB, bidderID int64, credentialRef string) {
	t.Helper()

	var gotRef, status, active string
	err := db.QueryRow(`
SELECT credential_ref, credential_status, active
FROM adv_bidder
WHERE bidder_id=?`, bidderID).Scan(&gotRef, &status, &active)
	if err != nil {
		t.Fatal(err)
	}
	if gotRef != credentialRef || status != "Active" || active != "Yes" {
		t.Fatalf("bidder approval = %q/%q/%q", gotRef, status, active)
	}
}

func insertTestAdvertiser(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	advID := time.Now().UnixNano() % 1000000000
	_, err := db.Exec(`
INSERT INTO adv
	(adv_id, email, passwd, domain, address_id, active, access_order, created)
VALUES (?, ?, SHA1('test'), ?, 1, 'Yes', 'Black', NOW())`,
		advID, fmt.Sprintf("bidder-test-%d@example.test", advID), fmt.Sprintf("bidder-test-%d", advID))
	if err != nil {
		t.Fatal(err)
	}
	return advID
}

func deleteTestAdvertiser(t *testing.T, db *sql.DB, advID int64) {
	t.Helper()

	if _, err := db.Exec(`DELETE FROM adv WHERE adv_id=?`, advID); err != nil {
		t.Fatalf("delete test advertiser %d: %v", advID, err)
	}
}

func cleanupBidderChain(t *testing.T, db *sql.DB, bidderID, campaignID, itemID, creativeID int64) {
	t.Helper()

	if _, err := db.Exec(`DELETE FROM adv_bidder WHERE bidder_id=?`, bidderID); err != nil {
		t.Fatalf("delete bidder %d: %v", bidderID, err)
	}
	if creativeID != 0 {
		if _, err := db.Exec(`DELETE FROM adv_creative WHERE creative_id=?`, creativeID); err != nil {
			t.Fatalf("delete creative %d: %v", creativeID, err)
		}
	}
	if itemID != 0 {
		if _, err := db.Exec(`DELETE FROM adv_item WHERE item_id=?`, itemID); err != nil {
			t.Fatalf("delete item %d: %v", itemID, err)
		}
	}
	if campaignID != 0 {
		if _, err := db.Exec(`DELETE FROM adv_campaign WHERE campaign_id=?`, campaignID); err != nil {
			t.Fatalf("delete campaign %d: %v", campaignID, err)
		}
	}
}

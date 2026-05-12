package bidder

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	"github.com/genelet/winter/summer"
)

const syntheticCreativeSizeID = 4194368

type Model struct {
	summer.Model
}

type bidderApproval struct {
	BidderID    int64
	AdvID       int64
	CampaignID  sql.NullInt64
	ItemID      sql.NullInt64
	CreativeID  sql.NullInt64
	BidderName  string
	EndpointURL string
}

func (self *Model) Approve(extra ...url.Values) error {
	if self.DB == nil {
		return fmt.Errorf("database is not set")
	}

	bidderID, err := strconv.ParseInt(self.ARGS.Get("bidder_id"), 10, 64)
	if err != nil || bidderID <= 0 {
		return fmt.Errorf("bidder_id is required")
	}
	credentialRef := self.ARGS.Get("credential_ref")
	if credentialRef == "" {
		return fmt.Errorf("credential_ref is required")
	}

	tx, err := self.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bidder, err := loadBidderForApproval(tx, bidderID)
	if err != nil {
		return err
	}

	campaignID, itemID, creativeID, err := self.approvalSyntheticIDs(tx, bidder)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
UPDATE adv_bidder
SET synthetic_campaign_id=?, synthetic_item_id=?, synthetic_creative_id=?,
	credential_ref=?, credential_status='Active', active='Yes'
WHERE bidder_id=?`,
		campaignID, itemID, creativeID, credentialRef, bidderID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	*self.LISTS = []map[string]interface{}{{
		"bidder_id":             strconv.FormatInt(bidderID, 10),
		"adv_id":                strconv.FormatInt(bidder.AdvID, 10),
		"synthetic_campaign_id": strconv.FormatInt(campaignID, 10),
		"synthetic_item_id":     strconv.FormatInt(itemID, 10),
		"synthetic_creative_id": strconv.FormatInt(creativeID, 10),
		"credential_ref":        credentialRef,
		"credential_status":     "Active",
		"active":                "Yes",
	}}
	return self.ProcessAfter("approve", extra...)
}

func loadBidderForApproval(tx *sql.Tx, bidderID int64) (bidderApproval, error) {
	var bidder bidderApproval
	err := tx.QueryRow(`
SELECT bidder_id, adv_id, synthetic_campaign_id, synthetic_item_id,
	synthetic_creative_id, bidder_name, endpoint_url
FROM adv_bidder
WHERE bidder_id=?
FOR UPDATE`, bidderID).Scan(
		&bidder.BidderID,
		&bidder.AdvID,
		&bidder.CampaignID,
		&bidder.ItemID,
		&bidder.CreativeID,
		&bidder.BidderName,
		&bidder.EndpointURL,
	)
	if err == sql.ErrNoRows {
		return bidder, fmt.Errorf("bidder_id %d not found", bidderID)
	}
	return bidder, err
}

func (self *Model) approvalSyntheticIDs(tx *sql.Tx, bidder bidderApproval) (int64, int64, int64, error) {
	idsSet := 0
	for _, id := range []sql.NullInt64{bidder.CampaignID, bidder.ItemID, bidder.CreativeID} {
		if id.Valid && id.Int64 > 0 {
			idsSet++
		}
	}
	switch idsSet {
	case 0:
		campaignID, itemID, creativeID, err := createSyntheticChain(tx, bidder)
		return campaignID, itemID, creativeID, err
	case 3:
		campaignID := bidder.CampaignID.Int64
		itemID := bidder.ItemID.Int64
		creativeID := bidder.CreativeID.Int64
		if err := validateSyntheticChain(tx, bidder.AdvID, campaignID, itemID, creativeID); err != nil {
			return 0, 0, 0, err
		}
		return campaignID, itemID, creativeID, nil
	default:
		return 0, 0, 0, fmt.Errorf("partial synthetic reporting chain on bidder_id %d", bidder.BidderID)
	}
}

func createSyntheticChain(tx *sql.Tx, bidder bidderApproval) (int64, int64, int64, error) {
	name := "adv_bidder:" + strconv.FormatInt(bidder.BidderID, 10)
	campaignResult, err := tx.Exec(`
INSERT INTO adv_campaign
	(adv_id, campaign_name, foreign_id, access_order, active, created)
VALUES (?, ?, ?, 'Inherit', 'No', NOW())`, bidder.AdvID, name, name)
	if err != nil {
		return 0, 0, 0, err
	}
	campaignID, err := campaignResult.LastInsertId()
	if err != nil {
		return 0, 0, 0, err
	}

	itemResult, err := tx.Exec(`
INSERT INTO adv_item
	(campaign_id, item_name, item_click, cost_type, cost, fl_sitetypes,
		access_order, channel_order, active, created)
VALUES (?, ?, 'about:blank', 'CPM', 0, 'App,Web',
	'Inherit', 'Black', 'No', NOW())`, campaignID, name)
	if err != nil {
		return 0, 0, 0, err
	}
	itemID, err := itemResult.LastInsertId()
	if err != nil {
		return 0, 0, 0, err
	}

	creativeResult, err := tx.Exec(`
INSERT INTO adv_creative
	(creative_name, item_id, size_id, media_type, active, created)
VALUES (?, ?, ?, 'Banner', 'No', NOW())`, name, itemID, syntheticCreativeSizeID)
	if err != nil {
		return 0, 0, 0, err
	}
	creativeID, err := creativeResult.LastInsertId()
	if err != nil {
		return 0, 0, 0, err
	}
	return campaignID, itemID, creativeID, nil
}

func validateSyntheticChain(tx *sql.Tx, advID, campaignID, itemID, creativeID int64) error {
	var found int
	err := tx.QueryRow(`
SELECT COUNT(*)
FROM adv_campaign c
INNER JOIN adv_item i ON i.campaign_id=c.campaign_id
INNER JOIN adv_creative cr ON cr.item_id=i.item_id
WHERE c.adv_id=? AND c.campaign_id=? AND i.item_id=? AND cr.creative_id=?`,
		advID, campaignID, itemID, creativeID).Scan(&found)
	if err != nil {
		return err
	}
	if found != 1 {
		return fmt.Errorf("synthetic reporting chain does not belong to bidder advertiser")
	}
	return nil
}

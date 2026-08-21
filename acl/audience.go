package acl

import (
	"bytes"
	"database/sql"
	"encoding/gob"
)

type ACLAudience struct {
	// This should be the adv domain
	AdvDomain string
	// this should be the foriegn id of the campaign
	CampaignForeignID string
	// Simply target Web or App. 0: all, 1: web, 2: app
	SiteTypes SiteType
	// adv or campaign, black list publisher. because of proc_slot, this is redundant in matching
	BPub []string
	// adv or campaign, white list publisher. because of proc_slot, this is redundant in matching
	WPub []string
	// adv or campaign, black list site/app. because of proc_slot, this is redundant in matching
	BApp []string
	// adv or campaign, white list site/app. because of proc_slot, this is redundant in matching
	WApp []string
	// campaign's category
	Categories []string
	// item white list category
	White []string
	// item black list category
	Black []string
}

// Pack serializes the audience into a byte slice.
func (self *ACLAudience) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(self)
	return buf.Bytes(), err
}

// UnpackACLAudience deserializes the audience from a byte slice.
func UnpackACLAudience(data []byte) (*ACLAudience, error) {
	audience := new(ACLAudience)
	buf := bytes.NewReader(data)
	err := gob.NewDecoder(buf).Decode(audience)
	return audience, err
}

func (self *ACLAudience) Has(a *ACL) bool {
	if self == nil || a == nil {
		return false
	}

	grepString := func(a []string, s string) bool {
		for _, v := range a {
			if v == s {
				return true
			}
		}
		return false
	}
	grepStrings := func(a []string, s []string) bool {
		for _, v := range s {
			if grepString(a, v) {
				return true
			}
		}
		return false
	}

	// adv simple blocks
	if self.SiteTypes != 0 {
		if self.SiteTypes != a.SiteType {
			return false
		}
	}

	// adv block pub
	if self.BPub != nil {
		if grepString(self.BPub, a.PubStr) {
			return false
		}
	} else if self.WPub != nil {
		if !grepString(self.WPub, a.PubStr) {
			return false
		}
	}

	// campaig block site
	if self.BApp != nil {
		if grepString(self.BApp, a.SiteStr) {
			return false
		}
	} else if self.WApp != nil {
		if !grepString(self.WApp, a.SiteStr) {
			return false
		}
	}

	// pub block adv
	if a.BAdv != nil && grepString(a.BAdv, self.AdvDomain) {
		return false
	}
	// app block item
	if a.BApp != nil && grepString(a.BApp, self.CampaignForeignID) {
		return false
	}

	// item category bw slot category
	if self.White != nil && !grepStrings(self.White, a.Categories) {
		return false
	}
	if self.Black != nil && grepStrings(self.Black, a.Categories) {
		return false
	}

	// slot category bw item category
	// first if item categories is null
	if self.Categories == nil {
		if a.White != nil {
			return false
		} else {
			return true
		}
	}
	// now self.Categories != nil
	if a.White != nil && !grepStrings(a.White, self.Categories) {
		return false
	}
	if a.Black != nil && grepStrings(a.Black, self.Categories) {
		return false
	}

	return true
}

// DBGetACLAudience retrieves category audience from the database.
func DBGetACLAudience(db *sql.DB, itemID uint32) (*ACLAudience, error) {
	aud := new(ACLAudience)

	err := dbGetPubAppAudience(db, itemID, aud)
	if err != nil {
		return nil, err
	}
	err = dbGetCategoryAudience(db, itemID, aud)
	return aud, err
}

// dbGetPubAppAudience retrieves black/white list publisher and app/site audience
// see proc_slot and proc_creative for more details
func dbGetPubAppAudience(db *sql.DB, itemID uint32, aud *ACLAudience) error {
	var aOrder, cOrder, iOrder string
	var advID, campaignID uint32
	var sitetypes string
	err := db.QueryRow(`
SELECT a.domain, c.foreign_id, a.adv_id, a.access_order, c.campaign_id, c.access_order, i.fl_sitetypes, i.access_order
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv a USING (adv_id)
WHERE i.item_id=?`, itemID).Scan(&aud.AdvDomain, &aud.CampaignForeignID, &advID, &aOrder, &campaignID, &cOrder, &sitetypes, &iOrder)
	if err != nil {
		return err
	}
	if sitetypes == "Web" {
		aud.SiteTypes = SiteTypeWeb
	} else if sitetypes == "App" {
		aud.SiteTypes = SiteTypeAPP
	}

	var pubSQL, appSQL string
	var id uint32
	var order string
	switch {
	case iOrder == "Inherit" && cOrder == "Inherit":
		pubSQL = `
SELECT p.domain
FROM ac ac
INNER JOIN pub p ON (ac.othertype_id=3 AND p.pub_id=ac.other_id)
WHERE ac.entitytype_id=4 AND ac.entity_id=?`
		appSQL = `
SELECT s.foreign_id
FROM ac ac
INNER JOIN pub_site s ON (ac.othertype_id=31 AND s.site_id=ac.other_id)
WHERE ac.entitytype_id=4 AND ac.entity_id=?`
		id = advID
		order = aOrder
	case iOrder == "Inherit":
		pubSQL = `
SELECT p.domain
FROM ac ac
INNER JOIN pub p ON (ac.othertype_id=3 AND p.pub_id=ac.other_id)
WHERE (ac.entitytype_id=41 AND ac.entity_id=?)`
		appSQL = `
SELECT s.foreign_id
FROM ac ac
INNER JOIN pub_site s ON (ac.othertype_id=31 AND s.site_id=ac.other_id)
WHERE (ac.entitytype_id=41 AND ac.entity_id=?)`
		id = campaignID
		order = cOrder
	default:
		appSQL = `
SELECT foreign_id
FROM ac ac
INNER JOIN pub_site s ON (ac.othertype_id=31 AND s.site_id=ac.other_id)
WHERE (ac.entitytype_id=42 AND ac.entity_id=?)`
		id = itemID
		order = iOrder
	}

	if iOrder == "Inherit" {
		rows, err := db.Query(pubSQL, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var domain string
			err = rows.Scan(&domain)
			if err != nil {
				return err
			}
			if order == "White" {
				aud.WPub = append(aud.WPub, domain)
			} else if order == "Black" {
				aud.BPub = append(aud.BPub, domain)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	rows, err := db.Query(appSQL, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var foreignID string
		err = rows.Scan(&foreignID)
		if err != nil {
			return err
		}
		if order == "White" {
			aud.WApp = append(aud.WApp, foreignID)
		} else if order == "Black" {
			aud.BApp = append(aud.BApp, foreignID)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	return nil
}

// dbGetCategoryAudience retrieves category audience from the database.
func dbGetCategoryAudience(db *sql.DB, itemID uint32, aud *ACLAudience) error {
	rows, err := db.Query(`
SELECT c.channel_name
FROM ch_belong b
INNER JOIN def_channel c USING (channel_id)
INNER JOIN adv_item i ON (b.entitytype_id=41 AND b.entity_id=i.campaign_id)
WHERE i.item_id=?`, itemID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var channelName string
		err = rows.Scan(&channelName)
		if err != nil {
			return err
		}
		aud.Categories = append(aud.Categories, channelName)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	var channelOrder string
	err = db.QueryRow(`
SELECT channel_order
FROM adv_item
WHERE item_id=?`, itemID).Scan(&channelOrder)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	rows, err = db.Query(`
SELECT c.channel_name
FROM ch_ac a
INNER JOIN def_channel c USING (channel_id)
INNER JOIN adv_item i ON (a.entitytype_id=42 AND a.entity_id=i.item_id)
WHERE i.item_id=?`, itemID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var channelName string
		err = rows.Scan(&channelName)
		if err != nil {
			return err
		}
		cats = append(cats, channelName)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	if cats != nil {
		if channelOrder == "White" {
			aud.White = cats
		} else {
			aud.Black = cats
		}
	}

	return nil
}

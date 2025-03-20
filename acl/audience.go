package acl

import (
	"bytes"
	"database/sql"
	"encoding/gob"
)

type ACLAudience struct {
	AdvStr string
	// this should be the foriegn id of the campaign
	AppStr string
	// adv or campaign, black list publisher
	BPub []string
	// adv or campaign, white list publisher
	WPub []string
	// adv or campaign, black list site/app
	BApp []string
	// adv or campaign, white list site/app
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
	if a.BAdv != nil && grepString(a.BAdv, self.AdvStr) {
		return false
	}
	// app block item
	if a.BApp != nil && grepString(a.BApp, self.AppStr) {
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

	var aOrder, cOrder string
	var advID, campaignID uint32
	err := db.QueryRow(`
SELECT a.domain, c.foreign_id, a.adv_id, a.access_order, c.campaign_id, c.access_order
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv a USING (adv_id)
WHERE i.item_id=?`, itemID).Scan(&aud.AdvStr, &aud.AppStr, &advID, &aOrder, &campaignID, &cOrder)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
SELECT entitytype_id, entity_id, othertype_id, other_id, p.domain, s.site_url
FROM ac ac
LEFT JOIN pub p          ON (ac.othertype_id=3 AND p.pub_id=ac.other_id)
LEFT JOIN pub_site s     ON (ac.othertype_id=31 AND s.site_id=ac.other_id)
LEFT JOIN adv a          ON (ac.entitytype_id=4 AND a.adv_id=ac.entity_id)
LEFT JOIN adv_campaign c ON (ac.entitytype_id=41 AND c.campaign_id=ac.entity_id)
WHERE (entitytype_id=4 AND entity_id=?)
OR (entitytype_id=41 AND entity_id=? AND c.access_order != "Inherit")`, advID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entityType, entityID, otherType, otherID uint32
		var pubDomain, siteURL sql.NullString
		err = rows.Scan(&entityType, &entityID, &otherType, &otherID, &pubDomain, &siteURL)
		if err != nil {
			return nil, err
		}
		if (entityType == 4 && aOrder == "White") || (entityType == 41 && cOrder == "White") {
			if otherType == 3 && pubDomain.Valid {
				aud.WPub = append(aud.WPub, pubDomain.String)
			} else if otherType == 31 && siteURL.Valid {
				aud.WApp = append(aud.WApp, siteURL.String)
			}
		} else if (entityType == 4 && aOrder == "Black") || (entityType == 41 && cOrder == "Black") {
			if otherType == 3 && pubDomain.Valid {
				aud.BPub = append(aud.BPub, pubDomain.String)
			} else if otherType == 31 && siteURL.Valid {
				aud.BApp = append(aud.BApp, siteURL.String)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	rows, err = db.Query(`
SELECT c.channel_name
FROM ch_belong b
INNER JOIN def_channel c USING (channel_id)
INNER JOIN adv_item i ON (b.entitytype_id=41 AND b.entity_id=i.campaign_id)
WHERE i.item_id=?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var channelName string
		err = rows.Scan(&channelName)
		if err != nil {
			return nil, err
		}
		aud.Categories = append(aud.Categories, channelName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	var channelOrder string
	err = db.QueryRow(`
SELECT channel_order
FROM adv_item
WHERE item_id=?`, itemID).Scan(&channelOrder)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	rows, err = db.Query(`
SELECT c.channel_name
FROM ch_ac a
INNER JOIN def_channel c USING (channel_id)
INNER JOIN adv_item i ON (a.entitytype_id=42 AND a.entity_id=i.item_id)
WHERE i.item_id=?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var channelName string
		err = rows.Scan(&channelName)
		if err != nil {
			return nil, err
		}
		cats = append(cats, channelName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if cats != nil {
		if channelOrder == "White" {
			aud.White = cats
		} else {
			aud.Black = cats
		}
	}

	return aud, nil
}

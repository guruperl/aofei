package weight

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) StartnewFast(extra ...url.Values) error {
	ARGS := self.ARGS

	str, err := GetSlotWeightsString(self.DB, ARGS.Get("slot_id"))
	if err != nil {
		return err
	}
	return self.SelectSQL(self.LISTS, str)
}

func GetSlotWeightsString(db *sql.DB, slotID string) (string, error) {
	var ao1, ao2, chOrder, qaDevice, qaPosition, qaLanguage, flCreative, flExpnd, flMime string
	var pubID, siteID, sizeID string
	var flItem, qaSlot uint32

	err := db.QueryRow(`
SELECT p.pub_id, p.access_order AS ao1,
	s.site_id, s.access_order AS ao2,
	t.slot_id, t.size_id, t.channel_order, 
	t.qa_device, t.qa_position, t.qa_language,
	t.fl_creative, t.fl_expnd, t.fl_mime,
	(0+t.fl_item) AS fl_item, (0+t.qa_slot) AS qa_slot
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub p USING (pub_id)
WHERE t.active='Yes' AND s.active='Yes' AND t.slot_id=?`, slotID).Scan(
		&pubID, &ao1, &siteID, &ao2, &slotID, &sizeID, &chOrder, &qaDevice, &qaPosition, &qaLanguage, &flCreative, &flExpnd, &flItem, &qaSlot)
	if err != nil {
		return "", err
	}

	entitytypeID := "31"
	entityID := siteID
	acOrder := ao2
	if acOrder == "Inherit" {
		entitytypeID = "3"
		entityID = pubID
		acOrder = ao1
	}

	var acStr, chStr string
	if acOrder == "Black" {
		acStr = "ac.entity_id IS NULL"
	} else {
		acStr = "ac.entity_id IS NOT NULL"
	}
	if chOrder == "Black" {
		chStr = "(ca.entity_id IS NULL OR cb.entity_id IS NULL)"
	} else {
		chStr = "ca.entity_id IS NOT NULL AND cb.entity_id IS NOT NULL)"
	}

	cam := summer.UnpackItem(flItem)
	sit := summer.UnpackSlot(qaSlot)

	rightPub := `(ac.othertype_id=3 AND ac.other_id=` + pubID + `) OR (ac.othertype_id=31 AND ac.other_id=` + siteID + `)`
	rightAdv := `(ac.othertype_id=4 AND ac.other_id=a.adv_id) OR (ac.othertype_id=41 AND ac.other_id=c.campaign_id)`
	// advertiser 4, can block pub 3, site 31
	// advertiser's campaign 41, can block pub 3, site 31
	// publisher 3, can block advertiser 4, campaign 41
	// publisher's site 31, can block advertiser 4, campaign 41

	str := `-- main gives the list of adv and camaigns that ALLOW this site!
SELECT main.adv_id, main.campaign_id, main.campaign_name, main.channel_order,
	i.startx, i.endx, i.item_name, i.item_id, i.cost_type, i.cost
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv a USING (adv_id)
	SELECT adv_id, campaign_id, channel_order, campaign_name
	LEFT JOIN ac ON
		(ac.entitytype_id=4 AND ac.entity_id=c.adv_id AND (` + rightPub + `)

-- here is restriction by this site's ac
LEFT JOIN ac ON
	(ac.entitytype_id=` + entitytypeID + ` AND ac.entity_id=` + entityID + ` AND (` + rightAdv + `)

-- channel WHITE and BLACK are ANY
-- here is channel restriction by campaign
LEFT JOIN ch_ac a ON
	(a.entitytype_id=42 and a.entity_id=i.item_id)
LEFT JOIN ch_belong b ON
	(b.channel_id=a.channel_id AND b.entitytype_id=32 AND b.entity_id=` + slotID + `)

-- here is restriction by site/slot on campaign
LEFT JOIN ch_belong cb ON
	(cb.entitytype_id=42 AND cb.entity_id=i.item_id)
LEFT JOIN ch_ac ca ON
	(ca.channel_id=cb.channel_id AND ca.entitytype_id=32 AND ca.entity_id=` + slotID + `)

WHERE i.active='Yes' AND i.size_id =` + sizeID + `

AND (
	(	c.access_order="Inherit"
		AND (
		(a.access_order="White" AND ac.other_id IS NOT NULL) OR (a.access_order="Black" AND ac.other_id IS NULL)
		)
	) OR (
		c.access_order!="Inherit"
		AND (
		(c.access_order="White" AND ac.other_id IS NOT NULL) OR (c.access_order="Black" AND ac.other_id IS NULL)
		)
	)
)

AND ` + acStr + `

-- here is channel restriction by campaign
AND (
	(i.channel_order='Black' AND (a.entity_id IS NULL OR b.entity_id IS NULL)) OR
	(i.channel_order='White' AND a.entity_id IS NOT NULL AND b.entity_id IS NOT NULL)
)

-- here is restriction by site/slot on campaign
AND ` + chStr + `

AND ((i.qa_item>>0) &7)>=` + strconv.Itoa(int(cam.Content)) + `
AND ((i.qa_item>>3) &7)>=` + strconv.Itoa(int(cam.Visual)) + `
AND ((i.qa_item>>6) &7)>=` + strconv.Itoa(int(cam.Act)) + `
AND ((i.qa_item>>9) &7)>=` + strconv.Itoa(int(cam.Download)) + `
AND ((i.qa_item>>12)&7)>=` + strconv.Itoa(int(cam.Speed)) + `
AND ((i.qa_item>>15)&7)>=` + strconv.Itoa(int(cam.Postclick)) + `
AND ((i.fl_slot>>0) &3)<=` + strconv.Itoa(int(sit.Internet)) + `
AND ((i.fl_slot>>2) &3)<=` + strconv.Itoa(int(sit.World)) + `
AND ((i.fl_slot>>4) &3)<=` + strconv.Itoa(int(sit.Local)) + `
AND ((i.fl_slot>>6) &3)<=` + strconv.Itoa(int(sit.Domain)) + `
AND ((i.fl_slot>>8) &3)<=` + strconv.Itoa(int(sit.Age)) + `
AND ((i.fl_slot>>10)&3)<=` + strconv.Itoa(int(sit.Visual)) + `
AND ((i.fl_slot>>12)&3)<=` + strconv.Itoa(int(sit.Popup)) + `
AND ((i.fl_slot>>14)&3)<=` + strconv.Itoa(int(sit.Crowd)) + `
AND ((i.fl_slot>>16)&3)<=` + strconv.Itoa(int(sit.Traffic)) + `
AND ((i.fl_slot>>18)&3)<=` + strconv.Itoa(int(sit.Source)) + `
AND ((i.fl_slot>>20)&3)<=` + strconv.Itoa(int(sit.Control)) + `
AND FIND_IN_SET("` + qaDevice + `", i.fl_device) > 0
AND FIND_IN_SET("` + qaPosition + `", i.fl_position) > 0
AND FIND_IN_SET("` + qaLanguage + `", i.fl_language) > 0
AND FIND_IN_SET(i.qa_creative, "` + flCreative + `") > 0
AND FIND_IN_SET(i.qa_expnd, "` + flExpnd + `") > 0
AND FIND_IN_SET(i.qa_mime, "` + flMime + `") > 0

AND i.size_id=` + sizeID + `
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (i.endx >= NOW() OR (i.endx IS NULL))
`
	return str, nil
}

func (self *Model) Insupd(extra ...url.Values) error {
	ARGS := self.ARGS
	slotID := ARGS.Get("slot_id")

	var err error
	// if err = self.StartnreFast(extra...); err != nil { return err }
	if err = self.Startnew(extra...); err != nil {
		return err
	}
	str := ``
	n := 0
	for _, item := range *self.LISTS {
		if item["cost"] == nil || item["cost_type"] == nil {
			continue
		}
		cost := item["cost"].(float64)
		switch item["cost_type"].(string) {
		case "CPM":
			cost *= 1.0
		case "CPC":
			cost *= 10.0
		case "CPA":
			cost *= 0.01
		default:
			cost *= 1.0
		}
		weight := cost * cost * cost
		if weight < 0 {
			weight *= -1.0
		}
		str += `(` + slotID + `, ` + strconv.FormatInt(item["item_id"].(int64), 10) + `, ` + strconv.FormatFloat(weight, 'f', -1, 32) + `, NOW()),`
		n++
	}

	if err = self.DoSQL("DELETE FROM pub_weight WHERE slot_id=?", slotID); err == nil {
		if n == 0 {
			return nil
		} // in case no new item, we still delete the old items
		err = self.DoSQL("INSERT INTO pub_weight (slot_id, item_id, weight, created) VALUES " + str[:len(str)-1])
	}
	return err
}

// c.access_order  ='Inherit',  4, a.adv_id      -- a.access_order
// c.access_order !='Inherit', 41, c.campaign_id -- c.access_order
// s.access_order  ='Inherit',  3, p.pub_id      -- p.access_order
// s.access_order !='Inherit', 31, s.site_id     -- s.access_order

func (self *Model) StartnewOld(extra ...url.Values) error {
	ARGS := self.ARGS
	slotID := ARGS.Get("slot_id")

	err := self.GetArgs(ARGS,
		`SELECT s.access_order, t.mychannel, t.channel_order
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
WHERE slot_id=?`, slotID)
	if err != nil {
		return err
	}

	sa := (ARGS.Get("access_order") == "Inherit")

	return self.SelectSQL(self.LISTS,
		`SELECT DISTINCT item_id, cost_type, cost
FROM `+SlotItemViewName(true, sa)+`
WHERE slot_id=?
UNION DISTINCT
SELECT DISTINCT item_id, cost_type, cost
FROM `+SlotItemViewName(false, sa)+`
WHERE slot_id=?`, slotID, slotID)
}

func (self *Model) Startnew(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS,
		`SELECT DISTINCT item_id, item_name, campaign_id, campaign_name, cost_type, cost
FROM ViewSlot
WHERE slot_id=?`, self.ARGS.Get("slot_id"))
}

func WhiteViewName(ca, sa bool) string {
	return fmt.Sprintf("WHITE%t%t", ca, sa)
}

func SlotItemViewName(ca, sa bool) string {
	return fmt.Sprintf("VIEW%t%t", ca, sa)
}

func (self *Model) MakeViewsForWhite() error {
	var err error
	for _, ca := range []bool{true, false} {
		for _, sa := range []bool{true, false} {
			name := WhiteViewName(ca, sa)
			str := SlotItemSelectString(ca, sa, true)
			if err = self.DoSQL(`DROP VIEW IF EXISTS ` + name); err == nil {
				err = self.DoSQL(`CREATE VIEW ` + name + ` AS ` + str)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *Model) MakeViewsForSlotItem() error {
	var err error
	for _, ca := range []bool{true, false} {
		for _, sa := range []bool{true, false} {
			name := SlotItemViewName(ca, sa)
			str := SlotItemSelectString(ca, sa, false)
			if err = self.DoSQL(`DROP VIEW IF EXISTS ` + name); err == nil {
				err = self.DoSQL(`CREATE VIEW ` + name + ` AS ` + str)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SlotItemSelectString has
// ca: campaign access_control is inherit or not inherit
// sa: site     access_control is inherit or not inherit
func SlotItemSelectString(ca, sa, future bool) string {
	startx := ``
	if !future {
		startx = `AND (i.startx <= NOW() OR (i.startx IS NULL))`
	}

	CaMap := map[bool][]string{
		true:  {"4", "a.adv_id", "c.access_order ='Inherit'", "a.access_order"},
		false: {"41", "c.campaign_id", "c.access_order!='Inherit'", "c.access_order"},
	}
	SaMap := map[bool][]string{
		true:  {"3", "p.pub_id", "s.access_order ='Inherit'", "p.access_order"},
		false: {"31", "s.site_id", "s.access_order!='Inherit'", "s.access_order"},
	}

	rightPub := `(ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)`
	rightAdv := `(bc.othertype_id=4 AND bc.other_id=a.adv_id) OR (bc.othertype_id=41 AND bc.other_id=c.campaign_id)`

	return `SELECT t.slot_id, i.item_id, i.cost_type, i.cost
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
INNER JOIN adv_item i USING (size_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv      a USING (adv_id)

-- adv/camp to restrict site
LEFT JOIN ac ON
	(ac.entitytype_id=` + CaMap[ca][0] + ` AND ac.entity_id=` + CaMap[ca][1] + ` AND (` + rightPub + `))

-- site/pub to restrict adv
LEFT JOIN ac bc ON
	(bc.entitytype_id=` + SaMap[sa][0] + ` AND bc.entity_id=` + SaMap[sa][1] + ` AND (` + rightAdv + `))

-- campaign to restrict site/slot
LEFT JOIN ch_ac ha ON
	(ha.entitytype_id=42 and ha.entity_id=i.item_id)
LEFT JOIN ch_belong hb ON
	(hb.entitytype_id=32 AND hb.entity_id=t.slot_id AND hb.channel_id=ha.channel_id)

-- site/slot to resitrct campaign
LEFT JOIN ch_ac ca ON
	(ca.entitytype_id=32 AND ca.entity_id=t.slot_id)
LEFT JOIN ch_belong cb ON
	(cb.entitytype_id=42 AND cb.entity_id=i.item_id AND ca.channel_id=cb.channel_id)

WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
AND   a.active="Yes" AND c.active="Yes" AND i.active="Yes"

AND FIND_IN_SET(i.qa_mime,     t.fl_mime)>0
AND FIND_IN_SET(t.qa_language, i.fl_language)>0
AND FIND_IN_SET(t.qa_device,   i.fl_device)>0
AND FIND_IN_SET(t.qa_position, i.fl_position)>0
AND FIND_IN_SET(i.qa_expnd,    t.fl_expnd)>0
AND FIND_IN_SET(i.qa_creative, t.fl_creative)>0
AND ((c.qa_item>>0)&7) >=((s.fl_item>>0 )&7)
AND ((c.qa_item>>3)&7) >=((s.fl_item>>3 )&7)
AND ((c.qa_item>>6)&7) >=((s.fl_item>>6 )&7)
AND ((c.qa_item>>9)&7) >=((s.fl_item>>9 )&7)
AND ((c.qa_item>>12)&7)>=((s.fl_item>>12)&7)
AND ((c.qa_item>>15)&7)>=((s.fl_item>>15)&7)
AND ((c.fl_slot>>0)&3) <=((s.qa_slot>> 0)&3)
AND ((c.fl_slot>>2)&3) <=((s.qa_slot>> 2)&3)
AND ((c.fl_slot>>4)&3) <=((s.qa_slot>> 4)&3)
AND ((c.fl_slot>>6)&3) <=((s.qa_slot>> 6)&3)
AND ((c.fl_slot>>8)&3) <=((s.qa_slot>> 8)&3)
AND ((c.fl_slot>>10)&3)<=((s.qa_slot>>10)&3)
AND ((c.fl_slot>>12)&3)<=((s.qa_slot>>12)&3)
AND ((c.fl_slot>>14)&3)<=((s.qa_slot>>14)&3)
AND ((c.fl_slot>>16)&3)<=((s.qa_slot>>16)&3)
AND ((c.fl_slot>>18)&3)<=((s.qa_slot>>18)&3)
AND ((c.fl_slot>>20)&3)<=((s.qa_slot>>20)&3)
` + startx + `
AND (  i.endx >= NOW() OR (  i.endx IS NULL))

-- adv/camp to restrict site
AND ( ` + CaMap[ca][2] + ` AND (
		(` + CaMap[ca][3] + `="White" AND ac.entity_id IS NOT NULL) OR
		(` + CaMap[ca][3] + `="Black" AND ac.entity_id IS NULL)
	)
)

-- pub/site to restrict adv
AND ( ` + SaMap[sa][2] + ` AND (
		(` + SaMap[sa][3] + `="White" AND bc.entity_id IS NOT NULL) OR
		(` + SaMap[sa][3] + `="Black" AND bc.entity_id IS NULL)
	)
)

-- to restrict site/slot
AND (
		(i.channel_order='Black' AND (ha.entity_id IS NULL OR hb.entity_id IS NULL)) OR
		(i.channel_order='White' AND ha.entity_id IS NOT NULL AND hb.entity_id IS NOT NULL)
)

-- to restrict campaign
AND (
		(t.channel_order='Black' AND (ca.entity_id IS NULL OR cb.entity_id IS NULL)) OR
		(t.channel_order='White' AND ca.entity_id IS NOT NULL AND cb.entity_id IS NOT NULL)
)
`
}

func allInheritExample() string {
	rightPub := `(ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)`
	rightAdv := `(bc.othertype_id=4 AND bc.other_id=a.adv_id) OR (bc.othertype_id=41 AND bc.other_id=c.campaign_id)`
	return `SELECT t.slot_id, i,item_id
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
INNER JOIN adv_item i USING (size_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv      a USING (adv_id)

-- adv/camp to restrict site
LEFT JOIN ac ON
	(ac.entitytype_id=4 AND ac.entity_id=a.adv_id AND (` + rightPub + `))

-- site/pub to restrict adv
LEFT JOIN ac bc ON
	(bc.entitytype_id=3 AND bc.entity_id=p.pub_id AND (` + rightAdv + `))

-- campaign to restrict site/slot
LEFT JOIN ch_ac ha ON
	(ha.entitytype_id=42 and ha.entity_id=i.item_id)
LEFT JOIN ch_belong hb ON
	(hb.channel_id=ha.channel_id AND hb.entitytype_id=32 AND hb.entity_id=s.slot_id)

-- site/slot to resitrct campaign
LEFT JOIN ch_ac ca ON
	(ca.entitytype_id=32 AND ca.entity_id=s.slot_id)
LEFT JOIN ch_belong cb ON
	(ca.channel_id=cb.channel_id AND cb.entitytype_id=42 AND cb.entity_id=i.item_id)

WHERE t.active="Yes" AND i.active="Yes"

-- adv/camp to restrict site
AND ( c.access_order="Inherit" AND (
		(a.access_order="White" AND ac.entity_id IS NOT NULL) OR
		(a.access_order="Black" AND ac.entiry_id IS NULL)
	)
)

-- pub/site to restrict adv
AND ( s.access_order="Inherit" AND (
		(p.access_order="White" AND bc.entity_id IS NOT NULL) OR
		(p.access_order="Black" AND bc.entity_id IS NULL)
	)
)

-- to restrict site/slot
AND (
		(i.channel_order='Black' AND (ha.entity_id IS NULL OR hb.entity_id IS NULL)) OR
		(i.channel_order='White' AND ha.entity_id IS NOT NULL AND hb.entity_id IS NOT NULL)
)

-- to restrict campaign
AND ( 
		(t.channel_order='Black' AND (ca.entity_id IS NULL OR cb.entity_id IS NULL)) OR
		(t.channel_order='White' AND ca.entity_id IS NOT NULL AND cb.entity_id IS NOT NULL)
)

AND FIND_IN_SET(i.qa_mime,     t.fl_mime)>0
AND FIND_IN_SET(t.qa_language, i.fl_language)>0
AND FIND_IN_SET(t.qa_device,   i.fl_device)>0
AND FIND_IN_SET(t.qa_position, i.fl_position)>0
AND FIND_IN_SET(i.qa_expnd,    t.fl_expnd)>0
AND FIND_IN_SET(i.qa_creative, t.fl_creative)>0
AND ((c.qa_item>>0)&7)>=((s.fl_item>>0)&7)
AND ((c.qa_item>>3)&7)>=((s.fl_item>>3)&7)
AND ((c.qa_item>>6)&7)>=((s.fl_item>>6)&7)
AND ((c.qa_item>>9)&7)>=((s.fl_item>>9)&7)
AND ((c.qa_item>>12)&7)>=((s.fl_item>>12)&7)
AND ((c.qa_item>>15)&7)>=((s.fl_item>>15)&7)
AND ((c.fl_slot>>0)&3)<=((s.qa_slot>>0)&3)
AND ((c.fl_slot>>2)&3)<=((s.qa_slot>>2)&3)
AND ((c.fl_slot>>4)&3)<=((s.qa_slot>>4)&3)
AND ((c.fl_slot>>6)&3)<=((s.qa_slot>>6)&3)
AND ((c.fl_slot>>8)&3)<=((s.qa_slot>>8)&3)
AND ((c.fl_slot>>10)&3)<=((s.qa_slot>>10)&3)
AND ((c.fl_slot>>12)&3)<=((s.qa_slot>>12)&3)
AND ((c.fl_slot>>14)&3)<=((s.qa_slot>>14)&3)
AND ((c.fl_slot>>16)&3)<=((s.qa_slot>>16)&3)
AND ((c.fl_slot>>18)&3)<=((s.qa_slot>>18)&3)
AND ((c.fl_slot>>20)&3)<=((s.qa_slot>>20)&3)
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (i.endx >= NOW() OR (i.endx IS NULL))
`
}

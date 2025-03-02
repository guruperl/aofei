package main

import (
	"context"
	"database/sql"
	"fmt"
)

func makeViewsForSlotItem(ctx context.Context, db *sql.DB) error {
	var err error
	for _, ca := range []bool{true, false} {
		for _, sa := range []bool{true, false} {
			name1 := fmt.Sprintf("VIEW%t%t", ca, sa)
			name2 := fmt.Sprintf("WHITE%t%t", ca, sa)
			str1 := slotItemSelectString(ca, sa, false)
			str2 := slotItemSelectString(ca, sa, true)
			if _, err = db.ExecContext(ctx, `DROP VIEW IF EXISTS `+name1); err == nil {
				if _, err = db.ExecContext(ctx, `CREATE VIEW `+name1+` AS `+str1); err == nil {
					if _, err = db.ExecContext(ctx, `DROP VIEW IF EXISTS `+name2); err == nil {
						_, err = db.ExecContext(ctx, `CREATE VIEW `+name2+` AS `+str2)
					}
				}
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Function slotItemSelectString has
// ca: campaign access_control is inherit or not inherit
// sa: site     access_control is inherit or not inherit
// c.access_order  ='Inherit',  4, a.adv_id      -- a.access_order
// c.access_order !='Inherit', 41, c.campaign_id -- c.access_order
// s.access_order  ='Inherit',  3, p.pub_id      -- p.access_order
// s.access_order !='Inherit', 31, s.site_id     -- s.access_order
func slotItemSelectString(ca, sa, future bool) string {
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
AND ((i.qa_item>>0)&7) >=((t.fl_item>>0 )&7)
AND ((i.qa_item>>3)&7) >=((t.fl_item>>3 )&7)
AND ((i.qa_item>>6)&7) >=((t.fl_item>>6 )&7)
AND ((i.qa_item>>9)&7) >=((t.fl_item>>9 )&7)
AND ((i.qa_item>>12)&7)>=((t.fl_item>>12)&7)
AND ((i.qa_item>>15)&7)>=((t.fl_item>>15)&7)
AND ((i.fl_slot>>0)&3) <=((t.qa_slot>> 0)&3)
AND ((i.fl_slot>>2)&3) <=((t.qa_slot>> 2)&3)
AND ((i.fl_slot>>4)&3) <=((t.qa_slot>> 4)&3)
AND ((i.fl_slot>>6)&3) <=((t.qa_slot>> 6)&3)
AND ((i.fl_slot>>8)&3) <=((t.qa_slot>> 8)&3)
AND ((i.fl_slot>>10)&3)<=((t.qa_slot>>10)&3)
AND ((i.fl_slot>>12)&3)<=((t.qa_slot>>12)&3)
AND ((i.fl_slot>>14)&3)<=((t.qa_slot>>14)&3)
AND ((i.fl_slot>>16)&3)<=((t.qa_slot>>16)&3)
AND ((i.fl_slot>>18)&3)<=((t.qa_slot>>18)&3)
AND ((i.fl_slot>>20)&3)<=((t.qa_slot>>20)&3)
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

/*
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
AND ((i.qa_item>>0)&7)>=((t.fl_item>>0)&7)
AND ((i.qa_item>>3)&7)>=((t.fl_item>>3)&7)
AND ((i.qa_item>>6)&7)>=((t.fl_item>>6)&7)
AND ((i.qa_item>>9)&7)>=((t.fl_item>>9)&7)
AND ((i.qa_item>>12)&7)>=((t.fl_item>>12)&7)
AND ((i.qa_item>>15)&7)>=((t.fl_item>>15)&7)
AND ((i.fl_slot>>0)&3)<=((t.qa_slot>>0)&3)
AND ((i.fl_slot>>2)&3)<=((t.qa_slot>>2)&3)
AND ((i.fl_slot>>4)&3)<=((t.qa_slot>>4)&3)
AND ((i.fl_slot>>6)&3)<=((t.qa_slot>>6)&3)
AND ((i.fl_slot>>8)&3)<=((t.qa_slot>>8)&3)
AND ((i.fl_slot>>10)&3)<=((t.qa_slot>>10)&3)
AND ((i.fl_slot>>12)&3)<=((t.qa_slot>>12)&3)
AND ((i.fl_slot>>14)&3)<=((t.qa_slot>>14)&3)
AND ((i.fl_slot>>16)&3)<=((t.qa_slot>>16)&3)
AND ((i.fl_slot>>18)&3)<=((t.qa_slot>>18)&3)
AND ((i.fl_slot>>20)&3)<=((t.qa_slot>>20)&3)
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (i.endx >= NOW() OR (i.endx IS NULL))
`
}
*/

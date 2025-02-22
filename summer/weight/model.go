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

	str, err := GetSlotWeightsString(self.Db, ARGS.Get("slot_id"))
	if err != nil {
		return err
	}
	return self.Select_sql(self.LISTS, str)
}

func GetSlotWeightsString(db *sql.DB, slot_id string) (string, error) {
	var ao1, ao2, co1, co2, mychannel, qa_device, fl_mime, qa_language string
	var pub_id, site_id, size_id string
	var fl_campaign, qa_site uint32

	err := db.QueryRow(`
SELECT p.pub_id, p.access_order AS ao1,
	s.site_id, s.channel_order AS co1, s.access_order AS ao2,
	t.qa_device, t.fl_mime, t.qa_language,
	(0+s.fl_campaign) AS fl_campaign, (0+s.qa_site) AS qa_site,
	t.slot_id, t.size_id, t.channel_order AS co2, t.mychannel
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub p USING (pub_id)
WHERE t.active='Yes' AND s.active='Yes'
AND t.slot_id=?`, slot_id).Scan(&pub_id, &ao1, &site_id, &co1, &ao2, &qa_device, &fl_mime, &qa_language, &fl_campaign, &qa_site, &slot_id, &size_id, &co2, &mychannel)
	if err != nil {
		return "", err
	}

	entitytype_id := "31"
	entity_id := site_id
	ac_order := ao2
	if ac_order == "Inherit" {
		entitytype_id = "3"
		entity_id = pub_id
		ac_order = ao1
	}

	ch_entitytype_id := "32"
	ch_entity_id := slot_id
	ch_order := co2
	if ch_order == "Inherit" {
		ch_entitytype_id = "31"
		ch_entity_id = site_id
		ch_order = co1
	}

	mych_entitytype_id := "32"
	mych_entity_id := slot_id
	if mychannel == "Inherit" {
		mych_entitytype_id = "31"
		mych_entity_id = site_id
	}

	var ac_str, ch_str string
	if ac_order == "Black" {
		ac_str = "ac.entity_id IS NULL"
	} else {
		ac_str = "ac.entity_id IS NOT NULL"
	}
	if ch_order == "Black" {
		ch_str = "(ca.entity_id IS NULL OR cb.entity_id IS NULL)"
	} else {
		ch_str = "ca.entity_id IS NOT NULL AND cb.entity_id IS NOT NULL)"
	}

	cam := summer.UnpackCampaign(fl_campaign)
	sit := summer.UnpackSite(qa_site)

	and := `
AND a.active='Yes' AND c.active='Yes'
AND ((c.qa_campaign>>0) &7)>=` + strconv.Itoa(int(cam.Content)) + `
AND ((c.qa_campaign>>3) &7)>=` + strconv.Itoa(int(cam.Visual)) + `
AND ((c.qa_campaign>>6) &7)>=` + strconv.Itoa(int(cam.Act)) + `
AND ((c.qa_campaign>>9) &7)>=` + strconv.Itoa(int(cam.Download)) + `
AND ((c.qa_campaign>>12)&7)>=` + strconv.Itoa(int(cam.Speed)) + `
AND ((c.qa_campaign>>15)&7)>=` + strconv.Itoa(int(cam.Postclick)) + `
AND ((c.fl_site>>0) &3)<=` + strconv.Itoa(int(sit.Internet)) + `
AND ((c.fl_site>>2) &3)<=` + strconv.Itoa(int(sit.World)) + `
AND ((c.fl_site>>4) &3)<=` + strconv.Itoa(int(sit.Local)) + `
AND ((c.fl_site>>6) &3)<=` + strconv.Itoa(int(sit.Domain)) + `
AND ((c.fl_site>>8) &3)<=` + strconv.Itoa(int(sit.Age)) + `
AND ((c.fl_site>>10)&3)<=` + strconv.Itoa(int(sit.Visual)) + `
AND ((c.fl_site>>12)&3)<=` + strconv.Itoa(int(sit.Popup)) + `
AND ((c.fl_site>>14)&3)<=` + strconv.Itoa(int(sit.Crowd)) + `
AND ((c.fl_site>>16)&3)<=` + strconv.Itoa(int(sit.Traffic)) + `
AND ((c.fl_site>>18)&3)<=` + strconv.Itoa(int(sit.Source)) + `
AND ((c.fl_site>>20)&3)<=` + strconv.Itoa(int(sit.Control)) + `
`

	right_pub := `(ac.othertype_id=3 AND ac.other_id=` + pub_id + `) OR (ac.othertype_id=31 AND ac.other_id=` + site_id + `)`
	right_adv := `(ac.othertype_id=4 AND ac.other_id=main.adv_id) OR (ac.othertype_id=41 AND ac.other_id=main.campaign_id)`
	// advertiser 4, can block pub 3, site 31
	// advertiser's campaign 41, can block pub 3, site 31
	// publisher 3, can block advertiser 4, campaign 41
	// publisher's site 31, can block advertiser 4, campaign 41

	str := `-- main gives the list of adv and camaigns that ALLOW this site!
SELECT main.adv_id, main.campaign_id, main.campaign_name, main.channel_order,
	i.startx, i.endx, i.item_name, i.item_id, i.cost_type, i.cost
FROM adv_item i
INNER JOIN (
	SELECT adv_id, campaign_id, channel_order, campaign_name
	FROM adv_campaign c
	INNER JOIN adv a USING (adv_id)
	LEFT JOIN ac ON
		(ac.entitytype_id=4 AND ac.entity_id=c.adv_id AND (` + right_pub + `)
	WHERE c.access_order="Inherit"
	AND (
		(a.access_order="White" AND ac.other_id IS NOT NULL) OR
		(a.access_order="Black" AND ac.other_id IS NULL)
	)
	` + and + `
	UNION
	SELECT adv_id, campaign_id, channel_order, campaign_name
	FROM adv_campaign c
	INNER JOIN adv a USING (adv_id)
	LEFT JOIN ac ON
		(ac.entitytype_id=41 AND ac.entity_id=c.campaign_id AND (` + right_pub + `)
	WHERE c.access_order!="Inherit"
	AND (
		(c.access_order="White" AND ac.other_id IS NOT NULL) OR
		(c.access_order="Black" AND ac.other_id IS NULL)
	)
	` + and + `
) main ON (i.campaign_id=main.campaign_id)

-- here is restriction by this site's ac
LEFT JOIN ac ON
	(ac.entitytype_id=` + entitytype_id + ` AND ac.entity_id=` + entity_id + ` AND (` + right_adv + `)

-- channel WHITE and BLACK are ANY
-- here is channel restriction by campaign
LEFT JOIN ch_ac a ON
	(a.entitytype_id=41 and a.entity_id=main.campaign_id)
LEFT JOIN ch_belong b ON
	(b.channel_id=a.channel_id AND b.entitytype_id=` + mych_entitytype_id + ` AND b.entity_id=` + mych_entity_id + `)

-- here is restriction by site/slot on campaign
LEFT JOIN ch_belong cb ON
	(cb.entitytype_id=41 AND cb.entity_id=main.campaign_id)
LEFT JOIN ch_ac ca ON
	(ca.channel_id=cb.channel_id AND ca.entitytype_id=` + ch_entitytype_id + ` AND ca.entity_id=` + ch_entity_id + `)

WHERE i.active='Yes' AND i.size_id =` + size_id + `

-- here is restriction by this site's ac
AND ` + ac_str + `

-- here is channel restriction by campaign
AND (
	(main.channel_order='Black' AND (a.entity_id IS NULL OR b.entity_id IS NULL)) OR
	(main.channel_order='White' AND a.entity_id IS NOT NULL AND b.entity_id IS NOT NULL)
)

-- here is restriction by site/slot on campaign
AND ` + ch_str + `

AND i.size_id=` + size_id + `
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (i.endx >= NOW() OR (i.endx IS NULL))
AND FIND_IN_SET("` + qa_device + `", i.fl_device) > 0
AND FIND_IN_SET("` + qa_language + `", i.fl_language) > 0
AND FIND_IN_SET(i.qa_mime, "` + fl_mime + `") > 0
`
	return str, nil
}

func (self *Model) Insupd(extra ...url.Values) error {
	ARGS := self.ARGS
	slot_id := ARGS.Get("slot_id")

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
		str += `(` + slot_id + `, ` + strconv.FormatInt(item["item_id"].(int64), 10) + `, ` + strconv.FormatFloat(weight, 'f', -1, 32) + `, NOW()),`
		n++
	}

	if err = self.Do_sql("DELETE FROM pub_weight WHERE slot_id=?", slot_id); err == nil {
		if n == 0 {
			return nil
		} // in case no new item, we still delete the old items
		err = self.Do_sql("INSERT INTO pub_weight (slot_id, item_id, weight, created) VALUES " + str[:len(str)-1])
	}
	return err
}

// c.access_order  ='Inherit',  4, a.adv_id      -- a.access_order
// c.access_order !='Inherit', 41, c.campaign_id -- c.access_order
// s.access_order  ='Inherit',  3, p.pub_id      -- p.access_order
// s.access_order !='Inherit', 31, s.site_id     -- s.access_order
// t.mychannel     ='Inherit', 31, s.site_id     --
// t.mychannel    !='Inherit', 32, t.slot_id     --
// t.channel_order ='Interit', 31, s.site_id     -- s.channel_order
// t.channel_order!='Interit', 32, t.slot_id     -- t.channel_order

func (self *Model) StartnewOld(extra ...url.Values) error {
	ARGS := self.ARGS
	slot_id := ARGS.Get("slot_id")

	err := self.Get_args(ARGS,
		`SELECT s.access_order, t.mychannel, t.channel_order
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
WHERE slot_id=?`, slot_id)
	if err != nil {
		return err
	}

	sa := (ARGS.Get("access_order") == "Inherit")
	tm := (ARGS.Get("mychannel") == "Inherit")
	tc := (ARGS.Get("channel_order") == "Inherit")

	return self.Select_sql(self.LISTS,
		`SELECT DISTINCT item_id, cost_type, cost
FROM `+SlotItemViewName(true, sa, tm, tc)+`
WHERE slot_id=?
UNION DISTINCT
SELECT DISTINCT item_id, cost_type, cost
FROM `+SlotItemViewName(false, sa, tm, tc)+`
WHERE slot_id=?`, slot_id, slot_id)
}

func (self *Model) Startnew(extra ...url.Values) error {
	return self.Select_sql(self.LISTS,
		`SELECT DISTINCT item_id, item_name, campaign_id, campaign_name, cost_type, cost
FROM ViewSlot
WHERE slot_id=?`, self.ARGS.Get("slot_id"))
}

func WhiteViewName(ca, sa, tm, tc bool) string {
	return fmt.Sprintf("WHITE%t%t%t%t", ca, sa, tm, tc)
}

func SlotItemViewName(ca, sa, tm, tc bool) string {
	return fmt.Sprintf("VIEW%t%t%t%t", ca, sa, tm, tc)
}

func (self *Model) MakeViewsForWhite() error {
	var err error
	for _, ca := range []bool{true, false} {
		for _, sa := range []bool{true, false} {
			for _, tm := range []bool{true, false} {
				for _, tc := range []bool{true, false} {
					name := WhiteViewName(ca, sa, tm, tc)
					str := SlotItemSelectString(ca, sa, tm, tc, true)
					if err = self.Do_sql(`DROP VIEW IF EXISTS ` + name); err == nil {
						err = self.Do_sql(`CREATE VIEW ` + name + ` AS ` + str)
					}
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (self *Model) MakeViewsForSlotItem() error {
	var err error
	for _, ca := range []bool{true, false} {
		for _, sa := range []bool{true, false} {
			for _, tm := range []bool{true, false} {
				for _, tc := range []bool{true, false} {
					name := SlotItemViewName(ca, sa, tm, tc)
					str := SlotItemSelectString(ca, sa, tm, tc, false)
					if err = self.Do_sql(`DROP VIEW IF EXISTS ` + name); err == nil {
						err = self.Do_sql(`CREATE VIEW ` + name + ` AS ` + str)
					}
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// SlotItemSelectString has
// ca: campaign access_control is inherit or not inherit
// sa: site     access_control is inherit or not inherit
func SlotItemSelectString(ca, sa, tm, tc, future bool) string {
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
	TmMap := map[bool][]string{
		true:  {"31", "s.site_id", "t.mychannel     ='Inherit'"},
		false: {"32", "t.slot_id", "t.mychannel    !='Inherit'"},
	}
	TcMap := map[bool][]string{
		true:  {"31", "s.site_id", "t.channel_order ='Inherit'", "s.channel_order"},
		false: {"32", "t.slot_id", "t.channel_order!='Inherit'", "t.channel_order"},
	}

	right_pub := `(ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)`
	right_adv := `(bc.othertype_id=4 AND bc.other_id=a.adv_id) OR (bc.othertype_id=41 AND bc.other_id=c.campaign_id)`

	return `SELECT t.slot_id, i.item_id, i.cost_type, i.cost
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
INNER JOIN adv_item i USING (size_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv      a USING (adv_id)

-- adv/camp to restrict site
LEFT JOIN ac ON
	(ac.entitytype_id=` + CaMap[ca][0] + ` AND ac.entity_id=` + CaMap[ca][1] + ` AND (` + right_pub + `))

-- site/pub to restrict adv
LEFT JOIN ac bc ON
	(bc.entitytype_id=` + SaMap[sa][0] + ` AND bc.entity_id=` + SaMap[sa][1] + ` AND (` + right_adv + `))

-- campaign to restrict site/slot
LEFT JOIN ch_ac ha ON
	(ha.entitytype_id=41 and ha.entity_id=c.campaign_id)
LEFT JOIN ch_belong hb ON
	(hb.entitytype_id=` + TmMap[tm][0] + ` AND hb.entity_id=` + TmMap[tm][1] + ` AND hb.channel_id=ha.channel_id)

-- site/slot to resitrct campaign
LEFT JOIN ch_ac ca ON
	(ca.entitytype_id=` + TcMap[tc][0] + ` AND ca.entity_id=` + TcMap[tc][1] + `)
LEFT JOIN ch_belong cb ON
	(cb.entitytype_id=41 AND cb.entity_id=c.campaign_id AND ca.channel_id=cb.channel_id)

WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
AND   a.active="Yes" AND c.active="Yes" AND i.active="Yes"

AND FIND_IN_SET(i.qa_mime,     t.fl_mime)>0
AND FIND_IN_SET(t.qa_language, i.fl_language)>0
AND FIND_IN_SET(t.qa_device,   i.fl_device)>0
AND FIND_IN_SET(t.qa_position, i.fl_position)>0
AND FIND_IN_SET(t.qa_content,  i.fl_content)>0
AND FIND_IN_SET(i.qa_creative, t.fl_creative)>0
AND ((c.qa_campaign>>0)&7) >=((s.fl_campaign>>0 )&7)
AND ((c.qa_campaign>>3)&7) >=((s.fl_campaign>>3 )&7)
AND ((c.qa_campaign>>6)&7) >=((s.fl_campaign>>6 )&7)
AND ((c.qa_campaign>>9)&7) >=((s.fl_campaign>>9 )&7)
AND ((c.qa_campaign>>12)&7)>=((s.fl_campaign>>12)&7)
AND ((c.qa_campaign>>15)&7)>=((s.fl_campaign>>15)&7)
AND ((c.fl_site>>0)&3) <=((s.qa_site>> 0)&3)
AND ((c.fl_site>>2)&3) <=((s.qa_site>> 2)&3)
AND ((c.fl_site>>4)&3) <=((s.qa_site>> 4)&3)
AND ((c.fl_site>>6)&3) <=((s.qa_site>> 6)&3)
AND ((c.fl_site>>8)&3) <=((s.qa_site>> 8)&3)
AND ((c.fl_site>>10)&3)<=((s.qa_site>>10)&3)
AND ((c.fl_site>>12)&3)<=((s.qa_site>>12)&3)
AND ((c.fl_site>>14)&3)<=((s.qa_site>>14)&3)
AND ((c.fl_site>>16)&3)<=((s.qa_site>>16)&3)
AND ((c.fl_site>>18)&3)<=((s.qa_site>>18)&3)
AND ((c.fl_site>>20)&3)<=((s.qa_site>>20)&3)
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
AND ( ` + TmMap[tm][2] + ` AND (
		(c.channel_order='Black' AND (ha.entity_id IS NULL OR hb.entity_id IS NULL)) OR
		(c.channel_order='White' AND ha.entity_id IS NOT NULL AND hb.entity_id IS NOT NULL)
	)
)

-- to restrict campaign
AND ( ` + TcMap[tc][2] + ` AND (
		(` + TcMap[tc][3] + `='Black' AND (ca.entity_id IS NULL OR cb.entity_id IS NULL)) OR
		(` + TcMap[tc][3] + `='White' AND ca.entity_id IS NOT NULL AND cb.entity_id IS NOT NULL)
	)
)
`
}

func allInheritExample() string {
	right_pub := `(ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)`
	right_adv := `(bc.othertype_id=4 AND bc.other_id=a.adv_id) OR (bc.othertype_id=41 AND bc.other_id=c.campaign_id)`
	return `SELECT t.slot_id, i,item_id
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
INNER JOIN adv_item i USING (size_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv      a USING (adv_id)

-- adv/camp to restrict site
LEFT JOIN ac ON
	(ac.entitytype_id=4 AND ac.entity_id=a.adv_id AND (` + right_pub + `))

-- site/pub to restrict adv
LEFT JOIN ac bc ON
	(bc.entitytype_id=3 AND bc.entity_id=p.pub_id AND (` + right_adv + `))

-- campaign to restrict site/slot
LEFT JOIN ch_ac ha ON
	(ha.entitytype_id=41 and ha.entity_id=c.campaign_id)
LEFT JOIN ch_belong hb ON
	(hb.channel_id=ha.channel_id AND hb.entitytype_id=31 AND hb.entity_id=s.site_id)

-- site/slot to resitrct campaign
LEFT JOIN ch_ac ca ON
	(ca.entitytype_id=31 AND ca.entity_id=s.site_id)
LEFT JOIN ch_belong cb ON
	(ca.channel_id=cb.channel_id AND cb.entitytype_id=41 AND cb.entity_id=c.campaign_id)

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
AND ( t.mychannel="Inherit" AND (
		(c.channel_order='Black' AND (ha.entity_id IS NULL OR hb.entity_id IS NULL)) OR
		(c.channel_order='White' AND ha.entity_id IS NOT NULL AND hb.entity_id IS NOT NULL)
	)
)

-- to restrict campaign
AND ( t.channel_order="Inherit" AND (
		(s.channel_order='Black' AND (ca.entity_id IS NULL OR cb.entity_id IS NULL)) OR
		(s.channel_order='White' AND ca.entity_id IS NOT NULL AND cb.entity_id IS NOT NULL)
	)
)

AND FIND_IN_SET(i.qa_mime,     t.fl_mime)>0
AND FIND_IN_SET(t.qa_language, i.fl_language)>0
AND FIND_IN_SET(t.qa_device,   i.fl_device)>0
AND FIND_IN_SET(t.qa_position, i.fl_position)>0
AND FIND_IN_SET(t.qa_content,  i.fl_content)>0
AND FIND_IN_SET(i.qa_creative, t.fl_creative)>0
AND ((c.qa_campaign>>0)&7)>=((s.fl_campaign>>0)&7)
AND ((c.qa_campaign>>3)&7)>=((s.fl_campaign>>3)&7)
AND ((c.qa_campaign>>6)&7)>=((s.fl_campaign>>6)&7)
AND ((c.qa_campaign>>9)&7)>=((s.fl_campaign>>9)&7)
AND ((c.qa_campaign>>12)&7)>=((s.fl_campaign>>12)&7)
AND ((c.qa_campaign>>15)&7)>=((s.fl_campaign>>15)&7)
AND ((c.fl_site>>0)&3)<=((s.qa_site>>0)&3)
AND ((c.fl_site>>2)&3)<=((s.qa_site>>2)&3)
AND ((c.fl_site>>4)&3)<=((s.qa_site>>4)&3)
AND ((c.fl_site>>6)&3)<=((s.qa_site>>6)&3)
AND ((c.fl_site>>8)&3)<=((s.qa_site>>8)&3)
AND ((c.fl_site>>10)&3)<=((s.qa_site>>10)&3)
AND ((c.fl_site>>12)&3)<=((s.qa_site>>12)&3)
AND ((c.fl_site>>14)&3)<=((s.qa_site>>14)&3)
AND ((c.fl_site>>16)&3)<=((s.qa_site>>16)&3)
AND ((c.fl_site>>18)&3)<=((s.qa_site>>18)&3)
AND ((c.fl_site>>20)&3)<=((s.qa_site>>20)&3)
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (i.endx >= NOW() OR (i.endx IS NULL))
`
}

package ssp

import (
	"database/sql"
	"fmt"

	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/summer/weight"

	"github.com/mediocregopher/radix.v2/pool"
)

func DBMakeHalfHour(c *pzutil.Config, db *sql.DB, p *pool.Pool, min int) error {
	if _, err := db.Exec(`
UPDATE adv_item SET active="Yes"
WHERE active = "New" AND ( ((UNIX_TIMESTAMP(NOW())-UNIX_TIMESTAMP(startx))) BETWEEN 0 AND ` + fmt.Sprintf("%d", min) + `)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
UPDATE adv_item SET active="No"
WHERE active = "Yes" AND ( ((UNIX_TIMESTAMP(endx )-UNIX_TIMESTAMP(NOW() ))) BETWEEN 0 AND ` + fmt.Sprintf("%d", min) + `)`); err != nil {
		return err
	}
	if _, err := db.Exec(
		`UPDATE cron_halfhour SET status='processing' WHERE status='new'`); err != nil {
		return err
	}

	str := ``
	var err error
	for _, ca := range []bool{true, false} {
		for _, sa := range []bool{true, false} {
			for _, tm := range []bool{true, false} {
				for _, tc := range []bool{true, false} {
					name := weight.SlotItemViewName(ca, sa, tm, tc)
					str += `SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN adv_creative c ON (h.entitytype_id=43 AND h.entity_id=c.creative_id)
INNER JOIN ` + name + ` v USING (item_id)
INNER JOIN pub_weight w ON (v.slot_id=w.slot_id)
WHERE h.status="processing" AND h.why IN ("content","creative","creative_")
UNION
`
				}
			}
		}
	}

	str +=
		`SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN adv a ON (h.entitytype_id=4 AND h.entity_id=a.adv_id)
INNER JOIN adv_campaign c USING (adv_id)
INNER JOIN adv_item i USING (campaign_id)
INNER JOIN pub_weight w USING (item_id)
WHERE h.status="processing" AND h.why IN ("Yes","No","bw","ac","ac_")
UNION
SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN adv_campaign c ON (h.entitytype_id=41 AND h.entity_id=c.campaign_id)
INNER JOIN adv_item i USING (campaign_id)
INNER JOIN pub_weight w USING (item_id)
WHERE h.status="processing" AND h.why IN ("Yes","No","bw","channel","ac","ac_","ch_ac","ch_ac_","ch_belong","ch_belong_")
UNION
SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN adv_item i ON (h.entitytype_id=42 AND h.entity_id=i.item_id)
INNER JOIN pub_weight w USING (item_id)
WHERE h.status="processing" AND h.why IN ("Yes","No")
UNION
SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN pub p ON (h.entitytype_id=3 AND h.entity_id=p.pub_id)
INNER JOIN pub_site s USING (pub_id) 
INNER JOIN pub_slot l USING (site_id) 
INNER JOIN pub_weight w USING (slot_id) 
WHERE p.active="No" AND h.status="processing" AND h.why IN ("No","bw","ac","ac_")
UNION
SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN pub_site s ON (h.entitytype_id=31 AND h.entity_id=s.site_id)
INNER JOIN pub_slot l USING (site_id) 
INNER JOIN pub_weight w USING (slot_id) 
WHERE s.active="No" AND h.status="processing" AND h.why IN ("No","bw","channel","ac","ac_","ch_ac","ch_ac_","ch_belong","ch_belong_")
UNION
SELECT DISTINCT w.slot_id
FROM cron_halfhour h
INNER JOIN pub_slot l ON (h.entitytype_id=32 AND h.entity_id=l.slot_id)
INNER JOIN pub_weight w USING (slot_id) 
WHERE l.active="No" AND h.status="processing" AND h.why IN ("No","channel","ch_ac","ch_ac_","ch_belong","ch_belong_")
`

	rows, err := db.Query(str)
	if err != nil {
		return err
	}

	ids := make([]uint32, 0)
	for rows.Next() {
		var id uint32
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	conn, err := p.Get()
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		str := ``
		for _, id := range ids {
			idstr := pzutil.IDStr(id)
			str += idstr + `,`
			conn.Cmd("HDEL", c.SLOT, idstr)
		}
		str = str[:len(str)-1]
		_, err = db.Exec(`DELETE FROM pub_weight WHERE slot_id IN (` + str + `)`)
		if err != nil {
			return err
		}
	}
	p.Put(conn)
	_, err = db.Exec(`UPDATE cron_halfhour SET status='done' WHERE status='processing'`)
	if err != nil {
		return err
	}
	return nil
}

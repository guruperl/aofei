package main

import (
	"database/sql"
)

// InsertLedger inserts ledger data into database.
// myDay is the date of the ledger datetime
// slots is a map of slot_id to site_id and pub_id
// items is a map of item_id to campaign_id and adv_id
// imps is a map of slot_id to item_id to impressions
// clis is a map of slot_id to item_id to clicks
// spes is a map of slot_id to item_id to spend
func InsertLedger(db *sql.DB, myDay string, slots, items map[int][]int, imps, clis map[int]map[int]int, spes map[int]map[int]float32) error {
	ins_log := `INSERT INTO ledger_log (timely, created) VALUES (?, NOW())`
	ins_pub := `INSERT INTO ledger_pub (log_id, slot_id, site_id, pub_id) VALUES (?,?,?,?)`
	ins_adv := `INSERT INTO ledger_adv (log_id, item_id, campaign_id, adv_id) VALUES (?,?,?,?)`
	ins_pub_adv := `INSERT INTO ledger_pub_adv (lp_id, la_id, imps, clis, spend) VALUES (?,?,?,?,?)`
	upd_pub := `UPDATE ledger_pub lp
INNER JOIN (
	SELECT pa.lp_id AS lp_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_pub p USING (lp_id)
	WHERE p.log_id=?
	GROUP BY pa.lp_id
) tmp ON (lp.lp_id=tmp.lp_id)
SET lp.imps=tmp.imps, lp.clis=tmp.clis, lp.spend=tmp.spend`
	upd_adv := `UPDATE ledger_adv la
INNER JOIN (
	SELECT pa.la_id AS la_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_adv a USING (la_id)
	WHERE a.log_id=?
	GROUP BY pa.la_id
) tmp ON (la.la_id=tmp.la_id)
SET la.imps=tmp.imps, la.clis=tmp.clis, la.spend=tmp.spend`
	upd_log := `UPDATE ledger_log SET imps=?, clis=?, spend=? WHERE log_id=?`

	var logID int64
	lpIds := make(map[int]int64)
	laIds := make(map[int]int64)

	var res sql.Result
	var err error

	if res, err = db.Exec(ins_log, myDay); err != nil {
		return err
	}
	if logID, err = res.LastInsertId(); err != nil {
		return err
	}

	for slotID, arr := range slots {
		var lpID int64
		if res, err = db.Exec(ins_pub, logID, slotID, arr[0], arr[1]); err != nil {
			return err
		}
		if lpID, err = res.LastInsertId(); err != nil {
			return err
		}
		lpIds[slotID] = lpID
	}

	for itemID, arr := range items {
		var laID int64
		if res, err = db.Exec(ins_adv, logID, itemID, arr[0], arr[1]); err != nil {
			return err
		}
		if laID, err = res.LastInsertId(); err != nil {
			return err
		}
		laIds[itemID] = laID
	}

	is := 0
	cs := 0
	ss := float32(0)
	for slotID, lpID := range lpIds {
		for itemID, laID := range laIds {
			i := imps[slotID][itemID]
			c := clis[slotID][itemID]
			s := spes[slotID][itemID]
			if _, err = db.Exec(ins_pub_adv, lpID, laID, i, c, s); err != nil {
				return err
			}
			is += i
			cs += c
			ss += s
		}
	}

	if _, err = db.Exec(upd_pub, logID); err != nil {
		return err
	}
	if _, err = db.Exec(upd_adv, logID); err != nil {
		return err
	}
	if _, err = db.Exec(upd_log, is, cs, ss, logID); err != nil {
		return err
	}

	return nil
}

// InsertDaily inserts daily data into database.
// myDay is the date. The ledger data of that day will be aggregated into daily data.
func InsertDaily(db *sql.DB, myDay string) error {
	ins_log := `
INSERT  INTO daily_log (daily, imps, clis, spend, created)
SELECT DATE(timely) AS daily, SUM(imps), SUM(clis), SUM(spend), NOW()
FROM ledger_log
WHERE DATE(timely)=? GROUP BY daily`
	ins_pub := `
INSERT INTO daily_pub (log_id, slot_id, site_id, pub_id, imps, clis, spend)
SELECT dl.log_id, tmp.slot_id, tmp.site_id, tmp.pub_id, tmp.tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT ANY_VALUE(DATE(timely)) AS daily, slot_id, ANY_VALUE(site_id) AS site_id, ANY_VALUE(pub_id) AS pub_id, SUM(p.imps) AS imps, SUM(p.clis) AS clis, SUM(p.spend) AS spend
	FROM ledger_pub p
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(timely)=? GROUP BY slot_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`
	ins_adv := `
INSERT INTO daily_adv (log_id, item_id, campaign_id, adv_id, imps, clis, spend)
SELECT dl.log_id, tmp.item_id, tmp.campaign_id, tmp.adv_id, tmp.tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT ANY_VALUE(DATE(timely)) AS daily, item_id, ANY_VALUE(campaign_id) AS campaign_id, ANY_VALUE(adv_id) AS adv_id, SUM(a.imps) AS imps, SUM(a.clis) AS clis, SUM(a.spend) AS spend
	FROM ledger_adv a
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(timely)=? GROUP BY item_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`
	ins_pub_adv := `
INSERT INTO daily_pub_adv (lp_id, la_id, imps, clis, spend)
SELECT dp.lp_id, da.la_id, tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT DATE(timely) AS daily, slot_id, item_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_pub p USING (lp_id)
	INNER JOIN ledger_adv a USING (la_id)
	INNER JOIN ledger_log l ON (p.log_id=l.log_id)
	WHERE DATE(timely)=? GROUP BY daily, p.slot_id, a.item_id
) tmp
INNER JOIN daily_pub dp ON (tmp.slot_id=dp.slot_id)
INNER JOIN daily_log ldp ON (dp.log_id=ldp.log_id)
INNER JOIN daily_adv da ON (tmp.item_id=da.item_id)
INNER JOIN daily_log lda ON (da.log_id=lda.log_id)
WHERE ldp.daily=? AND lda.daily=?`

	if _, err := db.Exec(ins_log, myDay); err != nil {
		return err
	}
	if _, err := db.Exec(ins_pub, myDay); err != nil {
		return err
	}
	if _, err := db.Exec(ins_adv, myDay); err != nil {
		return err
	}
	if _, err := db.Exec(ins_pub_adv, myDay, myDay, myDay); err != nil {
		return err
	}

	return nil
}

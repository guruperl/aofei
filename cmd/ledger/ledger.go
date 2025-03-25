// Description: The ledger package provides the ledger data structure and methods to insert ledger data into database.
// The ledger data structure contains the database connection, the directory of the winloss file, the interval in minutes,
// the active time, the current timestamp, the slots, and the items existing in the database.
//
// StatisticsToLedger should run very interval, e.g. 10 minutes, to insert the statistics of the winloss file into the ledger database.
// InsertDaily should run daily to aggregate the ledger data pf the previous day, or a given day, into daily ledger.
package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/genelet/winter/dsp"
)

type Ledger struct {
	DB         *sql.DB
	LogWinLoss string
	Interval   int
	Active     time.Time
	current    int64
	slots      map[uint32][2]uint32
	creatives  map[uint32][3]uint32
}

// NewLedger creates a new Ledger with the given database and interval in minutes.
// The active time is the time of the previous timestamp of the current time by default,
// unless the timestamp is passed as an argument.
// If the active time is already existing in the database, nil will be returned.
func NewLedger(db *sql.DB, dir string, interval int, stamp ...int) (*Ledger, error) {
	var current int64
	if len(stamp) > 0 {
		current = int64(stamp[0])
	} else {
		current = time.Now().Unix()/int64(interval*60) - 1
	}
	active := time.Unix(current*int64(interval*60), 0)
	myDay := active.Format("2006-01-02 15:04:05")
	err := db.QueryRow(`SELECT 1 FROM ledger_log WHERE timely=?`, myDay).Scan(new(int))
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		return nil, nil
	}

	rows, err := db.Query(`
SELECT slot_id, s.site_id, s.pub_id
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slots := make(map[uint32][2]uint32)
	for rows.Next() {
		var slotID, siteID, pubID uint32
		if err := rows.Scan(&slotID, &siteID, &pubID); err != nil {
			return nil, err
		}
		slots[slotID] = [2]uint32{siteID, pubID}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.Query(`
SELECT creative_id, i.item_id, i.campaign_id, c.adv_id
FROM adv_creative t
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	creatives := make(map[uint32][3]uint32)
	for rows.Next() {
		var creativeID, itemID, campaignID, advID uint32
		if err := rows.Scan(&creativeID, &itemID, &campaignID, &advID); err != nil {
			return nil, err
		}
		creatives[creativeID] = [3]uint32{itemID, campaignID, advID}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &Ledger{
		DB:         db,
		LogWinLoss: dir,
		Interval:   interval,
		Active:     active,
		current:    current,
		slots:      slots,
		creatives:  creatives,
	}, nil
}

// Statistics returns the statistics of the winloss file of the specific timestamp.
func (self *Ledger) Statistics() (map[uint32][2]uint32, map[uint32][3]uint32, map[uint32]map[uint32]int, map[uint32]map[uint32]int, map[uint32]map[uint32]float32, error) {
	imps := make(map[uint32]map[uint32]int)
	clis := make(map[uint32]map[uint32]int)
	spes := make(map[uint32]map[uint32]float32)

	name := fmt.Sprintf("%s/winloss.%d", self.LogWinLoss, self.current)
	fh, err := os.Open(name)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	defer fh.Close()

	slots := make(map[uint32][2]uint32)
	creatives := make(map[uint32][3]uint32)

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		var wl dsp.WinLoss
		if err := json.Unmarshal(scanner.Bytes(), &wl); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		slotID := wl.RPub.SlotID
		creativeID := wl.RAdv.ItemID
		var found bool
		switch wl.Status {
		case dsp.StatusTrackImp:
			if imps[slotID] == nil {
				imps[slotID] = make(map[uint32]int)
			}
			imps[slotID][creativeID] += 1
			if spes[slotID] == nil {
				spes[slotID] = make(map[uint32]float32)
			}
			spes[slotID][creativeID] += wl.RAdv.Cost
			found = true
		case dsp.StatusTrackClk:
			if clis[slotID] == nil {
				clis[slotID] = make(map[uint32]int)
			}
			clis[slotID][creativeID] += 1
			found = true
		default:
		}
		if found {
			slots[slotID] = self.slots[slotID]
			creatives[creativeID] = self.creatives[creativeID]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return slots, creatives, imps, clis, spes, nil
}

// StatisticsToLedger calculates the statistics of the winloss file of the specific timestamp
// and inserts them into the ledger database.
func (self *Ledger) StatisticsToLedger() error {
	slots, creatives, imps, clis, spes, err := self.Statistics()
	if err != nil {
		return err
	}
	myDay := self.Active.Format("2006-01-02 15:04:05")
	return insertLedger(self.DB, myDay, slots, creatives, imps, clis, spes)
}

// insertLedger inserts ledger data into database.
// myDay is the date of the ledger datetime
// slots is a map of slot_id to site_id and pub_id
// creatives is a map of creative_id to item_id, campaign_id and adv_id
// imps is a map of slot_id to creative_id to impressions
// clis is a map of slot_id to creative_id to clicks
// spes is a map of slot_id to creative_id to spend
func insertLedger(db *sql.DB, myDay string, slots map[uint32][2]uint32, creatives map[uint32][3]uint32, imps, clis map[uint32]map[uint32]int, spes map[uint32]map[uint32]float32) error {
	insLog := `INSERT INTO ledger_log (timely, created) VALUES (?, NOW())`
	insPub := `INSERT INTO ledger_pub (log_id, slot_id, site_id, pub_id) VALUES (?,?,?,?)`
	insAdv := `INSERT INTO ledger_adv (log_id, creative_id, item_id, campaign_id, adv_id) VALUES (?,?,?,?,?)`
	insPubAdv := `INSERT INTO ledger_pub_adv (lp_id, la_id, imps, clis, spend) VALUES (?,?,?,?,?)`
	updPub := `UPDATE ledger_pub lp
INNER JOIN (
	SELECT pa.lp_id AS lp_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_pub p USING (lp_id)
	WHERE p.log_id=?
	GROUP BY pa.lp_id
) tmp ON (lp.lp_id=tmp.lp_id)
SET lp.imps=tmp.imps, lp.clis=tmp.clis, lp.spend=tmp.spend`
	updAdv := `UPDATE ledger_adv la
INNER JOIN (
	SELECT pa.la_id AS la_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_adv a USING (la_id)
	WHERE a.log_id=?
	GROUP BY pa.la_id
) tmp ON (la.la_id=tmp.la_id)
SET la.imps=tmp.imps, la.clis=tmp.clis, la.spend=tmp.spend`
	updLog := `UPDATE ledger_log SET imps=?, clis=?, spend=? WHERE log_id=?`

	var logID int64
	lpIds := make(map[uint32]int64)
	laIds := make(map[uint32]int64)

	var res sql.Result
	var err error

	if res, err = db.Exec(insLog, myDay); err != nil {
		return err
	}
	if logID, err = res.LastInsertId(); err != nil {
		return err
	}

	for slotID, ids := range slots {
		var lpID int64
		if res, err = db.Exec(insPub, logID, slotID, ids[0], ids[1]); err != nil {
			return err
		}
		if lpID, err = res.LastInsertId(); err != nil {
			return err
		}
		lpIds[slotID] = lpID
	}

	for creativeID, ids := range creatives {
		var laID int64
		if res, err = db.Exec(insAdv, logID, creativeID, ids[0], ids[1], ids[2]); err != nil {
			return err
		}
		if laID, err = res.LastInsertId(); err != nil {
			return err
		}
		laIds[creativeID] = laID
	}

	is := 0
	cs := 0
	ss := float32(0)
	for slotID, lpID := range lpIds {
		for creativeID, laID := range laIds {
			i, ok1 := imps[slotID][creativeID]
			c, ok2 := clis[slotID][creativeID]
			s, ok3 := spes[slotID][creativeID]
			if !ok1 && !ok2 && !ok3 {
				continue
			}
			if _, err = db.Exec(insPubAdv, lpID, laID, i, c, s); err != nil {
				return err
			}
			is += i
			cs += c
			ss += s
		}
	}

	if _, err = db.Exec(updPub, logID); err != nil {
		return err
	}
	if _, err = db.Exec(updAdv, logID); err != nil {
		return err
	}
	if _, err = db.Exec(updLog, is, cs, ss, logID); err != nil {
		return err
	}

	_, err = db.Exec(`
UPDATE adv_balance b
INNER JOIN (
	SELECT total_balance_id, imps, clis, spend
	FROM ledger_pub l
	INNER JOIN pub p USING (pub_id)
	WHERE l.log_id=?
) tmp ON (b.balance_id=tmp.total_balance_id)
SET b.current_imps=tmp.imps, b.current_clis=tmp.clis, b.current_spend=tmp.spend`, logID)
	return err
}

// InsertDaily aggregates daily data of the previous day, or the given day into daily ledger.
func InsertDaily(db *sql.DB, myDays ...string) error {
	var myDay string
	if len(myDays) == 0 {
		myDay = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	} else {
		myDay = myDays[0]
	}

	err := db.QueryRow(`SELECT 1 FROM daily_log WHERE daily=?`, myDay).Scan(new(int))
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else {
		return nil
	}

	insLog := `
INSERT INTO daily_log (daily, imps, clis, spend, created)
SELECT DATE(timely) AS daily, SUM(imps), SUM(clis), SUM(spend), NOW()
FROM ledger_log
WHERE DATE(timely)=? GROUP BY daily`
	insPub := `
INSERT INTO daily_pub (log_id, slot_id, site_id, pub_id, imps, clis, spend)
SELECT dl.log_id, tmp.slot_id, tmp.site_id, tmp.pub_id, tmp.tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT ANY_VALUE(DATE(timely)) AS daily, slot_id, ANY_VALUE(site_id) AS site_id, ANY_VALUE(pub_id) AS pub_id, SUM(p.imps) AS imps, SUM(p.clis) AS clis, SUM(p.spend) AS spend
	FROM ledger_pub p
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(timely)=? GROUP BY slot_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`
	insAdv := `
INSERT INTO daily_adv (log_id, creative_id, item_id, campaign_id, adv_id, imps, clis, spend)
SELECT dl.log_id, tmp.creative_id, tmp.item_id, tmp.campaign_id, tmp.adv_id, tmp.tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT ANY_VALUE(DATE(timely)) AS daily, creative_id, ANY_VALUE(item_id) AS item_id, ANY_VALUE(campaign_id) AS campaign_id, ANY_VALUE(adv_id) AS adv_id, SUM(a.imps) AS imps, SUM(a.clis) AS clis, SUM(a.spend) AS spend
	FROM ledger_adv a
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(timely)=? GROUP BY creative_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`
	insPubAdv := `
INSERT INTO daily_pub_adv (lp_id, la_id, imps, clis, spend)
SELECT dp.lp_id, da.la_id, tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT DATE(timely) AS daily, slot_id, creative_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_pub p USING (lp_id)
	INNER JOIN ledger_adv a USING (la_id)
	INNER JOIN ledger_log l ON (p.log_id=l.log_id)
	WHERE DATE(timely)=? GROUP BY daily, p.slot_id, a.creative_id
) tmp
INNER JOIN daily_pub dp ON (tmp.slot_id=dp.slot_id)
INNER JOIN daily_log ldp ON (dp.log_id=ldp.log_id)
INNER JOIN daily_adv da ON (tmp.creative_id=da.creative_id)
INNER JOIN daily_log lda ON (da.log_id=lda.log_id)
WHERE ldp.daily=? AND lda.daily=?`

	if _, err := db.Exec(insLog, myDay); err != nil {
		return err
	}
	if _, err := db.Exec(insPub, myDay); err != nil {
		return err
	}
	if _, err := db.Exec(insAdv, myDay); err != nil {
		return err
	}
	if _, err := db.Exec(insPubAdv, myDay, myDay, myDay); err != nil {
		return err
	}

	_, err = db.Exec(`
UPDATE adv_balance b
INNER JOIN (
	SELECT daily_balance_id, l.imps, l.clis, l.spend
	FROM daily_pub l
	INNER JOIN pub p USING (pub_id)
	INNER JOIN daily_log dl USING (log_id)
	WHERE dl.daily=?
) tmp ON (b.balance_id=tmp.daily_balance_id)
SET b.daily_imps=tmp.imps, b.daily_clis=tmp.clis, b.daily_spend=tmp.spend`, myDay)
	return err
}

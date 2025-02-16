package match

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"time"

	"github.com/genelet/winter/pzutil"
)

// Imp is packed as 10*32 + 16 total 42 bytes
type Imp struct {
	Status16 uint16
	Nano64   int64
	IP32     uint32
	PzUa32   uint32
	Pid
	PubID  uint32
	SiteID uint32
}

// Win is packaged as 6 * 32 total 24 bytes
type Win struct {
	SlotID uint32
	RAdv   // 5*32
}

type Record struct {
	Imp
	Wins []Win
}

func (self *Record) PackIO(buf io.Writer) error {
	n := uint8(len(self.Wins))
	err := binary.Write(buf, binary.LittleEndian, self.Imp)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.LittleEndian, n)
	if err != nil {
		return err
	}
	return binary.Write(buf, binary.LittleEndian, self.Wins)
}

func (self *Record) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := self.PackIO(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

/*
func (self *Record)Sendlog(nc *nats.Conn, subj string) error {
	data, err := self.Pack()
	if err != nil { return err }
    nc.Publish(subj, data)
    return nil
}
*/

func UnpackRecordIO(buf io.Reader) (*Record, error) {
	imp := Imp{}
	err := binary.Read(buf, binary.LittleEndian, &imp)
	if err != nil {
		return nil, err
	}
	var n uint8
	err = binary.Read(buf, binary.LittleEndian, &n)
	if err != nil {
		return nil, err
	}
	wins := make([]Win, n)
	err = binary.Read(buf, binary.LittleEndian, &wins)
	if err != nil {
		return nil, err
	}

	return &Record{imp, wins}, nil
}

func UnpackRecord(bs []byte) (*Record, error) {
	buf := bytes.NewReader(bs)
	return UnpackRecordIO(buf)
}

func ParseRecordsFromFile(fn string) ([]*Record, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records := make([]*Record, 0)
	for {
		record, err := UnpackRecordIO(f)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		records = append(records, record)
	}

	return records, nil
}

type passing struct {
	total  []interface{}
	sites  map[uint32]uint32
	pubs   map[uint32]uint32
	camps  map[uint32]uint32
	advs   map[uint32]uint32
	lpubs  map[uint32][]interface{}
	ladvs  map[uint32][]interface{}
	ledger map[uint32]map[uint32][]interface{}
}

func makePassing() *passing {
	return &passing{
		[]interface{}{float32(0.0), uint32(0), uint32(0)},
		make(map[uint32]uint32),
		make(map[uint32]uint32),
		make(map[uint32]uint32),
		make(map[uint32]uint32),
		make(map[uint32][]interface{}),
		make(map[uint32][]interface{}),
		make(map[uint32]map[uint32][]interface{})}
}

func (self *Record) AddLedger(pass *passing) {
	if _, ok := pass.pubs[self.SiteID]; !ok {
		pass.pubs[self.SiteID] = self.PubID
	}
	status := pzutil.UnpackStatus(self.Status16)
	for _, win := range self.Wins {
		win.unitLedger(self.SiteID, status.Request == pzutil.CLIC, pass)
	}
}

func (self *Win) unitLedger(siteID uint32, isClick bool, pass *passing) {
	t := pass.total
	sites := pass.sites
	//pubs  := pass.pubs
	camps := pass.camps
	advs := pass.advs
	lpubs := pass.lpubs
	ladvs := pass.ladvs
	ledger := pass.ledger

	slotID := self.SlotID
	itemID := self.ItemID
	if _, ok := sites[slotID]; !ok {
		sites[slotID] = siteID
	}
	if _, ok := camps[itemID]; !ok {
		camps[itemID] = self.CampaignID
	}
	if _, ok := advs[self.CampaignID]; !ok {
		advs[self.CampaignID] = self.AdvID
	}

	if lpubs[slotID] == nil {
		lpubs[slotID] = []interface{}{float32(0.0), uint32(0), uint32(0)}
	}
	if ladvs[itemID] == nil {
		ladvs[itemID] = []interface{}{float32(0.0), uint32(0), uint32(0)}
	}

	if ledger[slotID] == nil {
		ledger[slotID] = make(map[uint32][]interface{})
	}
	if ledger[slotID][itemID] == nil {
		ledger[slotID][itemID] = []interface{}{float32(0.0), uint32(0), uint32(0)}
	}

	if t == nil {
		t = []interface{}{float32(0.0), uint32(0), uint32(0)}
	}

	if isClick {
		price := -1.0 * self.Price
		lpubs[slotID][0] = lpubs[slotID][0].(float32) + price
		lpubs[slotID][2] = lpubs[slotID][2].(uint32) + 1
		ladvs[itemID][0] = ladvs[itemID][0].(float32) + price
		ladvs[itemID][2] = ladvs[itemID][2].(uint32) + 1
		ledger[slotID][itemID][0] = ledger[slotID][itemID][0].(float32) + price
		ledger[slotID][itemID][2] = ledger[slotID][itemID][2].(uint32) + 1
		t[0] = t[0].(float32) + price
		t[2] = t[2].(uint32) + 1
	} else {
		price := 0.001 * self.Price
		lpubs[slotID][0] = lpubs[slotID][0].(float32) + price
		lpubs[slotID][1] = lpubs[slotID][1].(uint32) + 1
		ladvs[itemID][0] = ladvs[itemID][0].(float32) + price
		ladvs[itemID][1] = ladvs[itemID][1].(uint32) + 1
		ledger[slotID][itemID][0] = ledger[slotID][itemID][0].(float32) + price
		ledger[slotID][itemID][1] = ledger[slotID][itemID][1].(uint32) + 1
		t[0] = t[0].(float32) + price
		t[1] = t[1].(uint32) + 1
	}
}

func WhenToString(when time.Time, sep int) string {
	t := when.Truncate(time.Duration(sep) * time.Minute).Format(time.RFC3339)
	t = strings.Replace(t[:len(t)-6], "T", " ", 1)
	return t
}

func SetLedger(db *sql.DB, t string, pass *passing) error {
	total := pass.total
	sites := pass.sites
	pubs := pass.pubs
	camps := pass.camps
	advs := pass.advs
	lpubs := pass.lpubs
	ladvs := pass.ladvs
	ledger := pass.ledger

	var logID, lpID, laID, lpaID int64

	err := db.QueryRow("SELECT log_id FROM ledger_log WHERE timely=?", t).Scan(&logID)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO ledger_log (timely, spend, imps, clis, created) VALUES (?,?,?,?,NOW())", t, total[0], total[1], total[2])
		if err != nil {
			return err
		}
		logID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	} else {
		_, err := db.Exec("UPDATE ledger_log SET spend=spend+?, imps=imps+?, clis=clis+? WHERE log_id=?", total[0], total[1], total[2], logID)
		if err != nil {
			return err
		}
	}

	sthpub, _ := db.Prepare("SELECT lp_id  FROM ledger_pub WHERE log_id=? AND pub_id=?")
	sthadv, _ := db.Prepare("SELECT la_id  FROM ledger_adv WHERE log_id=? AND adv_id=?")
	sthpa, _ := db.Prepare("SELECT lpa_id FROM ledger_pub_adv WHERE lp_id=? AND la_id=?")

	inspub, _ := db.Prepare("INSERT INTO ledger_pub (log_id, pub_id, site_id, slot_id, spend, imps, clis) VALUES (?,?,?,?,?,?,?)")
	insadv, _ := db.Prepare("INSERT INTO ledger_adv (log_id, adv_id, campaign_id, item_id, spend, imps, clis) VALUES (?,?,?,?,?,?,?)")
	inspa, _ := db.Prepare("INSERT INTO ledger_pub_adv (lp_id, la_id, spend, imps, clis) VALUES (?,?,?,?,?)")

	updpub, _ := db.Prepare("UPDATE ledger_pub SET spend=spend+?, imps=imps+?, clis=clis+? WHERE lp_id=?")
	updadv, _ := db.Prepare("UPDATE ledger_adv SET spend=spend+?, imps=imps+?, clis=clis+? WHERE la_id=?")
	updpa, _ := db.Prepare("UPDATE ledger_pub_adv SET spend=spend+?, imps=imps+?, clis=clis+? WHERE lpa_id=?")

	lpids := make(map[uint32]uint32)
	for slotID, item := range lpubs {
		siteID := sites[slotID]
		pubID := pubs[siteID]
		err = sthpub.QueryRow(logID, pubID).Scan(&lpID)
		if err == sql.ErrNoRows {
			res, err := inspub.Exec(logID, pubID, siteID, slotID, item[0], item[1], item[2])
			if err != nil {
				return err
			}
			lpID, err = res.LastInsertId()
			if err != nil {
				return err
			}
		} else {
			_, err = updpub.Exec(item[0], item[1], item[2], lpID)
			if err != nil {
				return err
			}
		}
		lpids[slotID] = uint32(lpID)
	}
	sthpub.Close()
	inspub.Close()
	updpub.Close()

	laids := make(map[uint32]uint32)
	for itemID, item := range ladvs {
		campaignID := camps[itemID]
		advID := advs[campaignID]
		err = sthadv.QueryRow(logID, advID).Scan(&laID)
		if err == sql.ErrNoRows {
			res, err := insadv.Exec(logID, advID, campaignID, itemID, item[0], item[1], item[2])
			if err != nil {
				return err
			}
			laID, err = res.LastInsertId()
			if err != nil {
				return err
			}
		} else {
			_, err = updadv.Exec(item[0], item[1], item[2], laID)
			if err != nil {
				return err
			}
		}
		laids[itemID] = uint32(laID)
	}
	sthadv.Close()
	insadv.Close()
	updadv.Close()

	for slotID, values := range ledger {
		for itemID, item := range values {
			lpID := lpids[slotID]
			laID := laids[itemID]
			err = sthpa.QueryRow(lpID, laID).Scan(&lpaID)
			if err == sql.ErrNoRows {
				_, err = inspa.Exec(lpID, laID, item[0], item[1], item[2])
			} else {
				_, err = updpa.Exec(item[0], item[1], item[2], lpaID)
			}
			if err != nil {
				return err
			}
		}
	}
	sthpa.Close()
	inspa.Close()
	updpa.Close()

	return nil
}

func SetRecordFromFile(db *sql.DB, sep int, fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	pass := makePassing()
	marker := make(map[string]bool)
	var t64 int64
	var record *Record

	i := 0
	for {
		record, err = UnpackRecordIO(f)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		t64 = record.Nano64
		t := WhenToString(time.Unix((t64)/int64(time.Second), 0), sep)
		if _, ok := marker[t]; !ok {
			// any records between 00:00:00 to 00:04:59 are recorded as 00:05:00
			if i > 0 {
				if err := SetLedger(db, t, pass); err != nil {
					return err
				}
				marker[t] = true
			}

			pass = makePassing()
			i++
		}
		record.AddLedger(pass)
	}
	if (pass.total)[1].(uint32) > 0 || (pass.total)[2].(uint32) > 0 {
		t := WhenToString(time.Unix((t64+int64(sep)*int64(time.Minute))/int64(time.Second), 0), sep)
		return SetLedger(db, t, pass)
	}

	return nil
}

/*
func NewRecord(imp Imp, rpub RPub, radv RAdv) *Record {
	return &Record{imp, rpub, radv}
}

func GetBasicRecord(imp Imp, site *Site, slot *Slot) *Record {
	return NewRecord(imp, GetBasicRPub(site, slot), RAdv{})
}

func GetPublicRecord(imp Imp, site *Site, slot *Slot) *Record {
	return NewRecord(imp, GetBasicRPub(site, slot), GetPublicRAdv())
}

func GetRecord(imp Imp, site *Site, slot *Slot, item *Item, weight *Weight, creative *Creative) *Record {
	return NewRecord(imp, GetBasicRPub(site, slot), GetRAdv(item, weight, creative))
}
*/

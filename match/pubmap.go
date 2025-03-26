package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"

	"github.com/mediocregopher/radix/v4"
)

const (
	PUBDefault  = "default"
	SITEDefault = "default"
	SLOTDefault = "default"
)

type Pub struct {
	PubID            uint32
	Imps             int
	DefaultWebSiteID uint32
	DefaultAppSiteID uint32
	DefaultWebSlotID uint32
	DefaultAppSlotID uint32
	Sites            map[string]uint32
	Slots            map[uint32]map[string]uint32
}

// Pack encodes a Pub object into a byte slice.
func (self *Pub) Pack() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(self)
	return buf.Bytes(), err
}

// UnpackPub decodes a byte slice into a Pub object.
func UnpackPub(data []byte) (*Pub, error) {
	var p Pub
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&p)
	return &p, err
}

// GetRPub returns the Pub object from the bid request.
func (self *Pub) GetRPub(siteStr, slotStr string, isApp bool) RPub {
	defaultRPub := func() RPub {
		if isApp {
			return RPub{
				PubID:  self.PubID,
				SiteID: self.DefaultAppSiteID,
				SlotID: self.DefaultAppSlotID,
			}
		}
		return RPub{
			PubID:  self.PubID,
			SiteID: self.DefaultWebSiteID,
			SlotID: self.DefaultWebSlotID,
		}
	}

	siteID, ok := self.Sites[siteStr]
	if !ok {
		return defaultRPub()
	}
	slots, ok := self.Slots[siteID]
	if !ok {
		return defaultRPub()
	}
	slotID, ok := slots[slotStr]
	if !ok {
		return defaultRPub()
	}

	return RPub{
		PubID:  self.PubID,
		SiteID: siteID,
		SlotID: slotID,
	}
}

// ToRedis put Pub to redis
func (self *Pub) ToRedis(ctx context.Context, conn radix.Client, domain string) error {
	bs, err := self.Pack()
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "HSET", "pub", domain, string(bs)))
}

// RedisGetPub retrieves Pub from redis
func RedisGetPub(ctx context.Context, conn radix.Client, domain string) (*Pub, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", "pub", domain))
	if err != nil {
		return nil, err
	}
	return UnpackPub(bs)
}

// DBGetPub retrieves the Pub from the database using domain
func DBGetPub(db *sql.DB, domain string) (*Pub, error) {
	rows, err := db.Query(`
SELECT p.pub_id, foreign_id, s.site_id, t.slot_name, t.slot_id
FROM pub p
INNER JOIN pub_site s USING (pub_id)
INNER JOIN pub_slot t USING (slot_id)
WHERE domain = ?`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	p := &Pub{
		Sites: make(map[string]uint32),
		Slots: make(map[uint32]map[string]uint32),
	}
	for rows.Next() {
		var siteID, slotID uint32
		var slotName, foreignID string
		err = rows.Scan(&p.PubID, &foreignID, &siteID, &slotName, &slotID)
		if err != nil {
			return nil, err
		}
		if _, ok := p.Sites[foreignID]; !ok {
			p.Sites[foreignID] = siteID
		}
		if _, ok := p.Slots[siteID]; !ok {
			p.Slots[siteID] = make(map[string]uint32)
		}
		if _, ok := p.Slots[siteID][slotName]; !ok {
			p.Slots[siteID][slotName] = slotID
		}
		if foreignID == SITEDefault && slotName == SLOTDefault {
			p.DefaultWebSiteID = siteID
			p.DefaultWebSlotID = slotID
		}
	}
	return p, rows.Err()
}

func (self *Pub) addSlot(db *sql.DB, siteID uint32, slotStr string) (uint32, error) {
	row, err := db.Exec(`
INSERT INTO pub_slot (site_id, slot_name, active, created)
VALUES (?, ?, 'Yes', NOW())`, siteID, slotStr)
	if err != nil {
		return 0, err
	}
	id, err := row.LastInsertId()
	if err != nil {
		return 0, err
	}
	slotID := uint32(id)
	if self.Slots == nil {
		self.Slots = make(map[uint32]map[string]uint32)
	}
	if self.Slots[siteID] == nil {
		self.Slots[siteID] = make(map[string]uint32)
	}
	self.Slots[siteID][slotStr] = slotID
	return slotID, nil
}

func (self *Pub) addSite(db *sql.DB, siteStr string, site_type string) (uint32, error) {
	row, err := db.Exec(`
INSERT INTO pub_site (pub_id, site_name, foreign_id, site_type, active, created)
VALUES (?, ?, ?, ?, 'Yes', NOW())`, self.PubID, siteStr, siteStr, site_type)
	if err != nil {
		return 0, err
	}
	id, err := row.LastInsertId()
	if err != nil {
		return 0, err
	}
	siteID := uint32(id)
	if self.Sites == nil {
		self.Sites = make(map[string]uint32)
	}
	self.Sites[siteStr] = siteID
	return siteID, nil
}

func addPub(db *sql.DB, pubStr string) (*Pub, error) {
	row, err := db.Exec(`
INSERT INTO pub (domain, foreign_id, active, created) VALUES (?, ?, 'Yes', NOW())`, pubStr, pubStr)
	if err != nil {
		return nil, err
	}
	id, err := row.LastInsertId()
	if err != nil {
		return nil, err
	}
	pubID := uint32(id)
	pub := &Pub{
		PubID: pubID,
		Sites: make(map[string]uint32),
		Slots: make(map[uint32]map[string]uint32),
	}

	if pub.DefaultAppSiteID, err = pub.addSite(db, SITEDefault, "App"); err == nil {
		if pub.DefaultWebSiteID, err = pub.addSite(db, SITEDefault, "Web"); err == nil {
			if pub.DefaultAppSlotID, err = pub.addSlot(db, pub.DefaultAppSiteID, SLOTDefault); err == nil {
				_, err = pub.addSlot(db, pub.DefaultWebSiteID, SLOTDefault)
			}
		}
	}

	return pub, err
}

type PubMap map[string]*Pub

// DBGetPubMap retrieves the PubMap from the database
func DBGetPubMap(db *sql.DB) (PubMap, error) {
	rows, err := db.Query(`
SELECT domain, p.pub_id, foreign_id, s.site_id, t.slot_name, t.slot_id
FROM pub p
INNER JOIN pub_site s USING (pub_id)
INNER JOIN pub_slot t USING (slot_id)
WHERE p.active = 'Yes' AND s.active = 'Yes' AND t.active = 'Yes'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pubMap := make(map[string]*Pub)
	for rows.Next() {
		var pubID, siteID, slotID uint32
		var domain, slotName, foreignID string
		err = rows.Scan(&domain, &pubID, &foreignID, &siteID, &slotName, &slotID)
		if err != nil {
			return nil, err
		}
		if _, ok := pubMap[domain]; !ok {
			pubMap[domain] = &Pub{
				PubID: pubID,
				Sites: make(map[string]uint32),
				Slots: make(map[uint32]map[string]uint32),
			}
		}
		if _, ok := pubMap[domain].Sites[foreignID]; !ok {
			pubMap[domain].Sites[foreignID] = siteID
		}
		if _, ok := pubMap[domain].Slots[siteID]; !ok {
			pubMap[domain].Slots[siteID] = make(map[string]uint32)
		}
		if _, ok := pubMap[domain].Slots[siteID][slotName]; !ok {
			pubMap[domain].Slots[siteID][slotName] = slotID
		}
		if foreignID == SITEDefault && slotName == SLOTDefault {
			pubMap[domain].DefaultWebSiteID = siteID
			pubMap[domain].DefaultWebSlotID = slotID
		}
	}
	return pubMap, rows.Err()
}

// DBAddNew adds a new RPub object to the database.
func (self PubMap) DBAddNew(db *sql.DB, pubStr, siteStr, siteType, slotStr string) (*Pub, uint32, uint32, error) {
	var pub *Pub
	var siteID, slotID uint32
	var ok1, ok2, ok3 bool
	var err error
	pub, ok1 = self[pubStr]
	if ok1 {
		siteID, ok2 = pub.Sites[siteStr]
		if ok2 {
			slotID, ok3 = pub.Slots[siteID][slotStr]
			if !ok3 {
				slotID, err = pub.addSlot(db, siteID, slotStr)
			}
		} else {
			if siteID, err = pub.addSite(db, siteStr, siteType); err == nil {
				slotID, err = pub.addSlot(db, siteID, slotStr)
			}
		}
	} else {
		if pub, err = addPub(db, pubStr); err == nil {
			if siteID, err = pub.addSite(db, siteStr, siteType); err == nil {
				slotID, err = pub.addSlot(db, siteID, slotStr)
			}
		}
	}

	return pub, siteID, slotID, err
}

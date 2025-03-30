package acl

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"io"
	"math/rand"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
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

// PackIO packs a Pub object into a byte slice in IO writer.
func (self *Pub) PackIO(w io.Writer) error {
	enc := gob.NewEncoder(w)
	return enc.Encode(self)
}

// UnpackPub decodes a byte slice into a Pub object.
func UnpackPub(data []byte) (*Pub, error) {
	var p Pub
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&p)
	return &p, err
}

// UnpackPubIO decodes a byte slice from an IO reader into a Pub object.
func UnpackPubIO(r io.Reader) (*Pub, error) {
	var p Pub
	dec := gob.NewDecoder(r)
	err := dec.Decode(&p)
	return &p, err
}

// GetRPub returns three IDs from the bid request.
// This is the main purpose of the Pub object.
func (self *Pub) GetRPub(siteStr, slotStr string, isApp bool) (uint32, uint32, uint32) {
	defaultRPub := func() (uint32, uint32, uint32) {
		if isApp {
			return self.PubID, self.DefaultAppSiteID, self.DefaultAppSlotID
		} else {
			return self.PubID, self.DefaultWebSiteID, self.DefaultWebSlotID
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

	return self.PubID, siteID, slotID
}

// ToSpread put Pub to spread
func (self *Pub) ToSpread(conn *nats.Conn, channel, domain string) error {
	bs, err := self.Pack()
	if err != nil {
		return err
	}
	return conn.Publish(channel+":"+domain, bs)
}

// SpreadGetPub retrieves Pub from nats
func SpreadGetPub(m *nats.Msg, channel, top string) error {
	subject := m.Subject
	domain := strings.TrimPrefix(subject, channel+":")
	w, err := os.OpenFile(top+"/"+channel+"/"+domain, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = w.Write(m.Data)
	return err
}

// DBGetPub retrieves the Pub from the database using domain
func DBGetPub(db *sql.DB, domain string) (*Pub, error) {
	rows, err := db.Query(`
SELECT p.pub_id, foreign_id, s.site_id, s.site_type, t.slot_name, t.slot_id
FROM pub p
INNER JOIN pub_site s USING (pub_id)
INNER JOIN pub_slot t USING (site_id)	"log"
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
		var slotName, foreignID, siteType string
		err = rows.Scan(&p.PubID, &foreignID, &siteID, &siteType, &slotName, &slotID)
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
		if foreignID == SITEDefaultApp && slotName == SLOTDefault {
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

func (self *Pub) addSite(db *sql.DB, siteStr string, siteType string) (uint32, error) {
	row, err := db.Exec(`
INSERT INTO pub_site (pub_id, site_name, foreign_id, site_url, site_type, active, created)
VALUES (?, ?, ?, ?, ?, 'Yes', NOW())`, self.PubID, siteStr, siteStr, siteStr, siteType)
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
	pubID := rand.Uint32()
	_, err := db.Exec(`
INSERT INTO pub (pub_id, domain, email, passwd, address_id, active, created)
VALUES (?, ?, ?, '123456789', 1, 'Yes', NOW())`, pubID, pubStr, pubStr)
	if err != nil {
		return nil, err
	}
	pub := &Pub{
		PubID: pubID,
		Sites: make(map[string]uint32),
		Slots: make(map[uint32]map[string]uint32),
	}

	if pub.DefaultAppSiteID, err = pub.addSite(db, SITEDefaultApp, "App"); err == nil {
		if pub.DefaultWebSiteID, err = pub.addSite(db, SITEDefaultWeb, "Web"); err == nil {
			if pub.DefaultAppSlotID, err = pub.addSlot(db, pub.DefaultAppSiteID, SLOTDefault); err == nil {
				pub.DefaultWebSlotID, err = pub.addSlot(db, pub.DefaultWebSiteID, SLOTDefault)
			}
		}
	}

	return pub, err
}

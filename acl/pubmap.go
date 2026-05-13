package acl

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"os"
	"strconv"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type PubMap map[string]*Pub

type DirectPub struct {
	Domain    string
	Pub       *Pub
	Sites     map[uint32]string
	Slots     map[uint32]map[uint32]string
	SlotSizes map[uint32]map[uint32]uint32
}

type DirectPubMap map[uint32]*DirectPub

// ToRedis encodes the PubMap to a byte slice and stores it in Redis
func (self PubMap) ToRedis(ctx context.Context, conn radix.Client) error {
	arr := []string{HashNamePubmap}
	byID := []string{HashNamePubByID}
	for k, v := range self {
		var bs []byte
		var err error
		if v != nil && v.Active && (v.LimitImps == 0 || v.CurrentImps < v.LimitImps) {
			if bs, err = v.Pack(); err == nil {
				arr = append(arr, k, string(bs))
			}
			if err == nil {
				var direct []byte
				direct, err = NewDirectPub(k, v).Pack()
				if err == nil {
					byID = append(byID, strconv.FormatUint(uint64(v.PubID), 10), string(direct))
				}
			}
		} else {
			// delete old pubmap
			err = conn.Do(ctx, radix.Cmd(nil, "HDEL", HashNamePubmap, k))
			if err == nil && v != nil {
				err = conn.Do(ctx, radix.Cmd(nil, "HDEL", HashNamePubByID, strconv.FormatUint(uint64(v.PubID), 10)))
			}
		}
		if err != nil {
			return err
		}
	}
	if len(arr) > 1 {
		if err := conn.Do(ctx, radix.Cmd(nil, "HMSET", arr...)); err != nil {
			return err
		}
	}
	if len(byID) > 1 {
		return conn.Do(ctx, radix.Cmd(nil, "HMSET", byID...))
	}
	return nil
}

// ToSpread encodes the PubMap to a byte slice and publish it to nats
func (self PubMap) ToSpread(conn *nats.Conn) error {
	for k, v := range self {
		err := v.ToSpread(conn, k)
		if err != nil {
			return err
		}
	}
	return nil
}

// PubMapFromRedis retrieves the PubMap from Redis and decodes it from a byte slice
func PubMapFromRedis(ctx context.Context, conn radix.Client) (PubMap, error) {
	name := HashNamePubmap
	arr := make([]string, 0)
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", name))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil // No PubMap found in Redis
	}
	pubmap := make(PubMap)
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows // Invalid format
	}
	// Decode each key-value pair into the PubMap
	for i := 0; i < len(arr); i += 2 {
		pubStr := arr[i]
		data := []byte(arr[i+1])
		pub, err := UnpackPub(data)
		if err != nil {
			return nil, err
		}
		pubmap[pubStr] = pub
	}
	return pubmap, nil
}

var HashNamePubmap = "pubmap"

var HashNamePubByID = "pubmap:by-id"

func NewDirectPub(domain string, pub *Pub) *DirectPub {
	direct := &DirectPub{
		Domain:    domain,
		Pub:       pub,
		Sites:     make(map[uint32]string),
		Slots:     make(map[uint32]map[uint32]string),
		SlotSizes: make(map[uint32]map[uint32]uint32),
	}
	if pub == nil {
		return direct
	}
	for siteStr, siteID := range pub.Sites {
		if _, ok := direct.Sites[siteID]; !ok {
			direct.Sites[siteID] = siteStr
		}
	}
	for siteID, slots := range pub.Slots {
		if direct.Slots[siteID] == nil {
			direct.Slots[siteID] = make(map[uint32]string)
		}
		for slotStr, slotID := range slots {
			if _, ok := direct.Slots[siteID][slotID]; !ok {
				direct.Slots[siteID][slotID] = slotStr
			}
		}
	}
	for siteID, slots := range pub.SlotSizes {
		if direct.SlotSizes[siteID] == nil {
			direct.SlotSizes[siteID] = make(map[uint32]uint32)
		}
		for slotID, sizeID := range slots {
			if _, ok := direct.SlotSizes[siteID][slotID]; !ok {
				direct.SlotSizes[siteID][slotID] = sizeID
			}
		}
	}
	return direct
}

func DirectPubMapFromPubMap(pubmap PubMap) DirectPubMap {
	if pubmap == nil {
		return nil
	}
	byID := make(DirectPubMap)
	for domain, pub := range pubmap {
		if pub == nil || !pub.Active || (pub.LimitImps != 0 && pub.CurrentImps >= pub.LimitImps) {
			continue
		}
		byID[pub.PubID] = NewDirectPub(domain, pub)
	}
	return byID
}

func (self *DirectPub) Pack() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(self)
	return buf.Bytes(), err
}

func UnpackDirectPub(data []byte) (*DirectPub, error) {
	var pub DirectPub
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&pub)
	return &pub, err
}

func (self *DirectPub) Validate(siteID, slotID, sizeID uint32) (string, string, bool) {
	if self == nil || self.Pub == nil {
		return "", "", false
	}
	siteStr, ok := self.Sites[siteID]
	if !ok {
		return "", "", false
	}
	slots := self.Slots[siteID]
	if slots == nil {
		return "", "", false
	}
	slotStr, ok := slots[slotID]
	if !ok {
		return "", "", false
	}
	sizes := self.SlotSizes[siteID]
	if sizes == nil {
		return "", "", false
	}
	configuredSizeID, ok := sizes[slotID]
	if !ok || configuredSizeID != sizeID {
		return "", "", false
	}
	return siteStr, slotStr, true
}

func (self DirectPubMap) PubByID(pubID uint32) *DirectPub {
	if self == nil {
		return nil
	}
	return self[pubID]
}

func PubByIDFromRedis(ctx context.Context, conn radix.Client, pubID uint32) (*DirectPub, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNamePubByID, strconv.FormatUint(uint64(pubID), 10)))
	if err == nil && len(bs) > 2 {
		return UnpackDirectPub(bs)
	}
	return nil, err
}

func DirectPubMapFromRedis(ctx context.Context, conn radix.Client) (DirectPubMap, error) {
	arr := make([]string, 0)
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", HashNamePubByID))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil
	}
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows
	}
	byID := make(DirectPubMap)
	for i := 0; i < len(arr); i += 2 {
		pubID64, err := strconv.ParseUint(arr[i], 10, 32)
		if err != nil {
			return nil, err
		}
		pub, err := UnpackDirectPub([]byte(arr[i+1]))
		if err != nil {
			return nil, err
		}
		byID[uint32(pubID64)] = pub
	}
	return byID, nil
}

// PubMapFromIO retrieves the PubMap from IO
func PubMapFromIO(top string) (PubMap, error) {
	files, err := os.ReadDir(top + "/" + HashNamePubmap)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No files found
		}
		return nil, err // Error reading directory
	}

	pubmap := make(PubMap)
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}
		name := file.Name()
		pub, err := PubFromIO(top, name)
		if err != nil {
			return nil, err // Error unpacking Pub
		}
		if pub == nil {
			continue // Skip nil Pub objects
		}
		pubmap[name] = pub
	}

	return pubmap, nil
}

// DBGetPubMap retrieves the PubMap from the database
func DBGetPubMap(db *sql.DB) (PubMap, error) {
	rows, err := db.Query(`
	SELECT domain, p.pub_id, p.active, foreign_id, s.site_id, t.slot_name, t.slot_id, t.size_id, b.limit_imp, b.current_imp
	FROM pub p
	INNER JOIN pub_site s USING (pub_id)
	INNER JOIN pub_slot t USING (site_id)
LEFT JOIN adv_balance b ON (p.total_balance_id=b.balance_id)
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pubMap := make(map[string]*Pub)
	for rows.Next() {
		var pubID, siteID, slotID, sizeID uint32
		var limitImps, currentImps sql.NullInt64
		var active sql.NullString
		var domain, slotName, foreignID string
		err = rows.Scan(&domain, &pubID, &active, &foreignID, &siteID, &slotName, &slotID, &sizeID, &limitImps, &currentImps)
		if err != nil {
			return nil, err
		}
		if _, ok := pubMap[domain]; !ok {
			pubMap[domain] = &Pub{
				PubID:     pubID,
				Sites:     make(map[string]uint32),
				Slots:     make(map[uint32]map[string]uint32),
				SlotSizes: make(map[uint32]map[uint32]uint32),
			}
		}
		if active.Valid && active.String == "Yes" {
			pubMap[domain].Active = true
		}
		if limitImps.Valid {
			pubMap[domain].LimitImps = uint32(limitImps.Int64)
		}
		if currentImps.Valid {
			pubMap[domain].CurrentImps = uint32(currentImps.Int64)
		}
		if _, ok := pubMap[domain].Sites[foreignID]; !ok {
			pubMap[domain].Sites[foreignID] = siteID
		}
		if _, ok := pubMap[domain].Slots[siteID]; !ok {
			pubMap[domain].Slots[siteID] = make(map[string]uint32)
		}
		if _, ok := pubMap[domain].SlotSizes[siteID]; !ok {
			pubMap[domain].SlotSizes[siteID] = make(map[uint32]uint32)
		}
		if _, ok := pubMap[domain].Slots[siteID][slotName]; !ok {
			pubMap[domain].Slots[siteID][slotName] = slotID
		}
		if _, ok := pubMap[domain].SlotSizes[siteID][slotID]; !ok {
			pubMap[domain].SlotSizes[siteID][slotID] = sizeID
		}
		if foreignID == SITEDefaultApp && slotName == SLOTDefault {
			pubMap[domain].DefaultAppSiteID = siteID
			pubMap[domain].DefaultAppSlotID = slotID
		}
		if foreignID == SITEDefaultWeb && slotName == SLOTDefault {
			pubMap[domain].DefaultWebSiteID = siteID
			pubMap[domain].DefaultWebSlotID = slotID
		}
	}
	return pubMap, rows.Err()
}

// DBAddNew adds a new RPub object to the database.
func (self PubMap) DBAddNew(db *sql.DB, pubStr, siteStr, siteType, slotStr string) (*Pub, error) {
	var pub *Pub
	var siteID, slotID uint32
	var ok1, ok2, ok3 bool
	var err error
	pub, ok1 = self[pubStr]
	if ok1 {
		siteID, ok2 = pub.Sites[siteStr]
		if ok2 {
			if _, ok3 = pub.Slots[siteID][slotStr]; !ok3 {
				slotID, err = pub.addSlot(db, siteID, slotStr)
				pub.Slots[siteID][slotStr] = slotID
			}
		} else {
			if siteID, err = pub.addSite(db, siteStr, siteType); err == nil {
				pub.Sites[siteStr] = siteID
				if slotID, err = pub.addSlot(db, siteID, slotStr); err == nil {
					pub.Slots[siteID] = make(map[string]uint32)
					pub.Slots[siteID][slotStr] = slotID
				}
			}
		}
	} else {
		pub, err = AddPub(db, pubStr)
	}

	return pub, err
}

// PubFromRedis returns the Pub object by pub string.
// This is the main purpose of PubMap, used in Controller.
func PubFromRedis(ctx context.Context, conn radix.Client, pubStr string) (*Pub, error) {
	var bs []byte
	name := HashNamePubmap
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", name, pubStr))
	if err == nil && len(bs) > 2 {
		return UnpackPub(bs)
	}

	return nil, err
}

// PubFromIO returns the Pub object by pub string.
// This is the main purpose of PubMap, used in Controller.
func PubFromIO(top, pubStr string) (*Pub, error) {
	fullname := top + "/" + HashNamePubmap + "/" + pubStr
	f, err := os.Open(fullname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No file found
		}
		return nil, err // Error opening file
	}
	defer f.Close()
	return UnpackPubIO(f)
}

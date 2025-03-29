package acl

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/mediocregopher/radix/v4"
)

type PubMap map[string]*Pub

// ToRedis encodes the PubMap to a byte slice and stores it in Redis
func (self PubMap) ToRedis(ctx context.Context, conn radix.Client, name string) error {
	arr := []string{name}
	for k, v := range self {
		bs, err := v.Pack()
		if err != nil {
			return err
		}
		arr = append(arr, k, string(bs))
	}
	return conn.Do(ctx, radix.Cmd(nil, "HMSET", arr...))
}

// PubMapFromRedis retrieves the PubMap from Redis and decodes it from a byte slice
func PubMapFromRedis(ctx context.Context, conn radix.Client, name string) (PubMap, error) {
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

// DBGetPubMap retrieves the PubMap from the database
func DBGetPubMap(db *sql.DB) (PubMap, error) {
	rows, err := db.Query(`
SELECT domain, p.pub_id, foreign_id, s.site_id, t.slot_name, t.slot_id
FROM pub p
INNER JOIN pub_site s USING (pub_id)
INNER JOIN pub_slot t USING (site_id)
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
		pub, err = addPub(db, pubStr)
	}

	return pub, err
}

// PubRedisName return the redis name for the PubMap.
func PubRedisName(fullname string) string {
	arr := strings.SplitN(filepath.Base(fullname), ".", -1)
	return arr[0]
}

// PubFromRedis returns the Pub object pub string.
// This is the main purpose of PubMap, used in Controller.
func (self PubMap) PubFromRedis(ctx context.Context, conn radix.Client, fullname, pubStr string) (*Pub, error) {
	var pubObj *Pub
	var bs []byte
	name := PubRedisName(fullname)
	err := conn.Do(ctx, radix.Cmd(&bs, "MGET", name, pubStr))
	if err == nil && len(bs) > 2 {
		pubObj, err = UnpackPub(bs)
	}
	if err != nil {
		return nil, err
	}
	if pubObj == nil {
		pubObj = self[pubStr]
		if pubObj == nil {
			pubObj = self[PUBDefault]
		}
	}
	return pubObj, nil
}

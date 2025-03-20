package match

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/binary"

	"github.com/genelet/winter/acl"
)

const (
	PUBDefault    = "default"
	SITEDefault   = "default"
	SLOTDefault   = "default"
	PUBDefaultID  = uint32(1)
	SITEDefaultID = uint32(2)
	SLOTDefaultID = uint32(6)
)

type RPub struct {
	PubID  uint32
	SiteID uint32
	SlotID uint32
	SizeID uint32
}

// PackString serializes the RPub object to a RawURL string
func (self RPub) PackString() (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// UnpackRPubString deserializes the RPub object from a RawURL string
func UnpackRPubString(text string) (RPub, error) {
	rp := RPub{}
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return rp, err
	}
	buf := bytes.NewReader([]byte(data))
	err = binary.Read(buf, binary.LittleEndian, &rp)
	if err != nil {
		return RPub{}, err
	}
	return rp, nil
}

type RPubMap struct {
	PubMap  map[string]uint32                       `json:"pub_map,omitempty"`
	SiteMap map[uint32]map[string]uint32            `json:"site_map,omitempty"`
	SlotMap map[uint32]map[uint32]map[string]uint32 `json:"slot_map,omitempty"`
}

// DefaultRPubMap returns the default RPubMap with preconfigured mappings
// for publishers, sites, and slots. It initializes the map with default
// values for PUBDefault, SITEDefault, and SLOTDefault.
func DefaultRPubMap() *RPubMap {
	return &RPubMap{
		PubMap: map[string]uint32{
			PUBDefault: PUBDefaultID,
		},
		SiteMap: map[uint32]map[string]uint32{
			PUBDefaultID: {
				SITEDefault: SITEDefaultID,
			},
		},
		SlotMap: map[uint32]map[uint32]map[string]uint32{
			PUBDefaultID: {
				SITEDefaultID: {
					SLOTDefault: SLOTDefaultID,
				},
			},
		},
	}
}

// GetRPub returns the RPub object from the bid request.
func (self *RPubMap) GetRPub(a *acl.ACL) RPub {
	var pubID, siteID, slotID uint32
	var ok bool
	pubID, ok = self.PubMap[a.PubStr]
	if !ok {
		pubID = PUBDefaultID
	}
	if self.SiteMap[pubID] == nil {
		pubID = PUBDefaultID
		siteID = SITEDefaultID
	} else {
		siteID, ok = self.SiteMap[pubID][a.SiteStr]
		if !ok {
			siteID = self.SiteMap[pubID][SITEDefault]
		}
	}
	if self.SlotMap[pubID] == nil || self.SlotMap[pubID][siteID] == nil {
		pubID = PUBDefaultID
		siteID = SITEDefaultID
		slotID = SLOTDefaultID
	} else {
		slotID, ok = self.SlotMap[pubID][siteID][a.SlotStr]
		if !ok {
			slotID = self.SlotMap[pubID][siteID][SLOTDefault]
		}
	}

	return RPub{
		PubID:  pubID,
		SiteID: siteID,
		SlotID: slotID,
	}
}

// DBGetRPubMap returns the RPubMap object from the database.
func DBGetRPubMap(db *sql.DB) (*RPubMap, error) {
	rpubMap := DefaultRPubMap()
	rows, err := db.Query(`
SELECT p.pub_id, s.site_id, t.slot_id, domain, foreign_id, slot_name
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub p USING (pub_id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pubID, siteID, slotID uint32
		var pubStr, siteStr, slotStr string
		err = rows.Scan(&pubID, &siteID, &slotID, &pubStr, &siteStr, &slotStr)
		if err != nil {
			return nil, err
		}
		if rpubMap.PubMap == nil {
			rpubMap.PubMap = make(map[string]uint32)
		}
		rpubMap.PubMap[pubStr] = pubID
		if rpubMap.SiteMap == nil {
			rpubMap.SiteMap = make(map[uint32]map[string]uint32)
		}
		if rpubMap.SiteMap[pubID] == nil {
			rpubMap.SiteMap[pubID] = make(map[string]uint32)
		}
		rpubMap.SiteMap[pubID][siteStr] = siteID
		if rpubMap.SlotMap == nil {
			rpubMap.SlotMap = make(map[uint32]map[uint32]map[string]uint32)
		}
		if rpubMap.SlotMap[pubID] == nil {
			rpubMap.SlotMap[pubID] = make(map[uint32]map[string]uint32)
		}
		if rpubMap.SlotMap[pubID][siteID] == nil {
			rpubMap.SlotMap[pubID][siteID] = make(map[string]uint32)
		}
		rpubMap.SlotMap[pubID][siteID][slotStr] = slotID
	}
	return rpubMap, rows.Err()
}

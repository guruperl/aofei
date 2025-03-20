package match

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/binary"

	"github.com/genelet/winter/acl"
)

const (
	PUBDefault  = "default"
	SITEDefault = "default"
	SLOTDefault = "default"
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
	PUBDefaultID uint32                                  `json:"pub_default_id,omitempty"`
	SITEWebID    uint32                                  `json:"site_web_id,omitempty"`
	SITEAppID    uint32                                  `json:"site_app_id,omitempty"`
	SLOTWebID    uint32                                  `json:"slot_web_id,omitempty"`
	SLOTAppID    uint32                                  `json:"slot_app_id,omitempty"`
	PubMap       map[string]uint32                       `json:"pub_map,omitempty"`
	SiteMap      map[uint32]map[string]uint32            `json:"site_map,omitempty"`
	SlotMap      map[uint32]map[uint32]map[string]uint32 `json:"slot_map,omitempty"`
}

// GetRPub returns the RPub object from the bid request.
func (self *RPubMap) GetRPub(a *acl.ACL, isApp bool) RPub {
	var pubID, siteID, slotID uint32
	var ok bool
	pubID, ok = self.PubMap[a.PubStr]
	if !ok {
		pubID = self.PUBDefaultID
	}
	if self.SiteMap[pubID] == nil {
		pubID = self.PUBDefaultID
		siteID = self.SITEWebID
		if isApp {
			siteID = self.SITEAppID
		}
	} else {
		siteID, ok = self.SiteMap[pubID][a.SiteStr]
		if !ok {
			siteID = self.SiteMap[pubID][SITEDefault]
		}
	}
	if self.SlotMap[pubID] == nil || self.SlotMap[pubID][siteID] == nil {
		pubID = self.PUBDefaultID
		siteID = self.SITEWebID
		slotID = self.SLOTWebID
		if isApp {
			siteID = self.SITEAppID
			slotID = self.SLOTAppID
		}
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
	rpubMap := new(RPubMap)
	rows, err := db.Query(`
SELECT p.pub_id, s.site_type, s.site_id, t.slot_id, domain, foreign_id, slot_name
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
		var siteType sql.NullString
		err = rows.Scan(&pubID, &siteType, &siteID, &slotID, &pubStr, &siteStr, &slotStr)
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
		if pubStr == PUBDefault {
			rpubMap.PUBDefaultID = pubID
			if siteStr == SITEDefault {
				if siteType.Valid && siteType.String == "App" {
					rpubMap.SITEAppID = siteID
					if slotStr == SLOTDefault {
						rpubMap.SLOTAppID = slotID
					}
				} else {
					rpubMap.SITEWebID = siteID
					if slotStr == SLOTDefault {
						rpubMap.SLOTWebID = slotID
					}
				}
			}
		}
	}
	return rpubMap, rows.Err()
}

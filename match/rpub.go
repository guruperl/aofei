package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"

	"github.com/genelet/winter/acl"

	openrtb2 "github.com/prebid/openrtb/v20/openrtb2"
)

const (
	PUBDefault    = "default"
	SITEDefault   = "default"
	SLOTDefault   = "default"
	PUBDefaultID  = uint32(1)
	SITEDefaultID = uint32(1)
	SLOTDefaultID = uint32(1)
)

// PubMap is a map of publisher name to ID
var PubMap map[string]uint32 = map[string]uint32{
	PUBDefault: PUBDefaultID,
}

// SiteMap is a map of site name to ID
var SiteMap map[uint32]map[string]uint32 = map[uint32]map[string]uint32{
	PUBDefaultID: {
		SITEDefault: SITEDefaultID,
	},
}

// SlotMap is a map of slot name to ID
var SlotMap map[uint32]map[uint32]map[string]uint32 = map[uint32]map[uint32]map[string]uint32{
	PUBDefaultID: {
		SITEDefaultID: {
			SLOTDefault: SLOTDefaultID,
		},
	},
}

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

// getRPub returns the RPub object from the bid request.
func getRPub(bidRequest *openrtb2.BidRequest, a *acl.ACL) RPub {
	var pubID, siteID, slotID uint32
	var ok bool
	pubID, ok = PubMap[a.PubStr]
	if !ok {
		pubID = PUBDefaultID
	}
	if SiteMap[pubID] == nil {
		pubID = PUBDefaultID
		siteID = SITEDefaultID
	} else {
		siteID, ok = SiteMap[pubID][a.SiteStr]
		if !ok {
			siteID = SiteMap[pubID][SITEDefault]
		}
	}
	if SlotMap[pubID] == nil || SlotMap[pubID][siteID] == nil {
		pubID = PUBDefaultID
		siteID = SITEDefaultID
		slotID = SLOTDefaultID
	} else {
		slotID, ok = SlotMap[pubID][siteID][a.SlotStr]
		if !ok {
			slotID = SlotMap[pubID][siteID][SLOTDefault]
		}
	}

	return RPub{
		PubID:  pubID,
		SiteID: siteID,
		SlotID: slotID,
	}
}

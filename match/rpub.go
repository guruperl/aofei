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
	PUBIDDefault  = 1
	SITEIDDefault = 1
	SLOTIDDefault = 1
)

// PubMap is a map of publisher name to ID
var PubMap map[string]uint32 = map[string]uint32{
	PUBDefault: PUBIDDefault,
}

// SiteMap is a map of site name to ID
var SiteMap map[string]map[string]uint32 = map[string]map[string]uint32{
	PUBDefault: {
		SITEDefault: SITEIDDefault,
	},
}

// SlotMap is a map of slot name to ID
var SlotMap map[string]map[string]map[string]uint32 = map[string]map[string]map[string]uint32{
	PUBDefault: {
		SITEDefault: {
			SLOTDefault: SLOTIDDefault,
		},
	},
}

type RPub struct {
	PubID  uint32
	SiteID uint32
	SlotID uint32
	SizeID uint32
}

// getRPub returns the RPub object from the bid request.
func getRPub(bidRequest *openrtb2.BidRequest, a *acl.ACL) RPub {
	pubStr := a.PubStr
	siteStr := a.SiteStr
	slotStr := a.SlotStr

	rpub := RPub{}

	if _, ok := PubMap[pubStr]; !ok {
		pubStr = PUBDefault
	}
	if _, ok := SiteMap[pubStr][siteStr]; !ok {
		siteStr = SITEDefault
	}
	if _, ok := SlotMap[pubStr][siteStr][slotStr]; !ok {
		slotStr = SLOTDefault
	}

	if id, ok := PubMap[pubStr]; ok {
		rpub.PubID = id
		if id, ok := SiteMap[pubStr][siteStr]; ok {
			rpub.SiteID = id
			if id, ok := SlotMap[pubStr][siteStr][slotStr]; ok {
				rpub.SlotID = id
			}
		}
	}

	return rpub
}

// the followings are for ssp

func (self RPub) Pack1() (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self.SlotID)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

func (self RPub) Pack() (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

func UnpackRPub(text string) (RPub, error) {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return RPub{}, err
	}
	rp := RPub{}
	buf := bytes.NewReader([]byte(data))
	err = binary.Read(buf, binary.LittleEndian, &rp)
	if err != nil {
		return RPub{}, err
	}

	return rp, nil
}

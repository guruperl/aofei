package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

type RPub struct {
	PubID  uint32
	SiteID uint32
	SlotID uint32
	SizeID uint32
}

func GetBasicRPub(site *Site, slot *Slot) RPub {
	return RPub{
		PubID:  site.PubID,
		SiteID: site.SiteID,
		SlotID: slot.SlotID,
		SizeID: slot.SizeID}
}

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

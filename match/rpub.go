package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

type RPub struct {
	PubID  uint32 `json:"pub_id,omitempty"`
	SiteID uint32 `json:"site_id,omitempty"`
	SlotID uint32 `json:"slot_id,omitempty"`
	SizeID uint32 `json:"size_id,omitempty"`
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

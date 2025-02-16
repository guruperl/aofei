package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

type RAdv struct {
	AdvID      uint32
	CampaignID uint32
	ItemID     uint32
	CreativeID uint32
	Price      float32
}

func GetPublicRAdv() RAdv {
	return RAdv{0, 0, 0, 0, 0.0}
}

func GetRAdv(item *Item, weight *Weight, creative *Creative) RAdv {
	return RAdv{item.AdvID, weight.CampaignID, weight.ItemID, creative.CreativeID, weight.Price}
}

func (self RAdv) Pack() (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

func UnpackRAdv(text string) (RAdv, error) {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return RAdv{}, err
	}
	ra := RAdv{}
	buf := bytes.NewReader([]byte(data))
	err = binary.Read(buf, binary.LittleEndian, &ra)
	if err != nil {
		return RAdv{}, err
	}

	return ra, nil
}

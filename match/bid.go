package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"time"
)

type Bid struct {
	ItemID     uint32
	CreativeID uint32
	When       int64
}

// Pack packs the Bid to binary.
func (self Bid) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// UnpackBid unpacks the Bid from binary.
func UnpackBid(data []byte) (Bid, error) {
	buf := bytes.NewReader(data)
	var b Bid
	err := binary.Read(buf, binary.LittleEndian, &b)
	return b, err
}

func NewBid(when time.Time, adv RAdv) Bid {
	return Bid{
		ItemID:     adv.ItemID,
		CreativeID: adv.CreativeID,
		When:       when.UnixNano(),
	}
}

func (self Bid) BidID() (string, error) {
	bs, err := self.Pack()
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(bs)
	return encoded, nil
}

func UnpackBidID(encoded string) (Bid, error) {
	bs, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Bid{}, err
	}
	return UnpackBid(bs)
}

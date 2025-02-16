package match

import (
	//"log"
	"bytes"
	"encoding/binary"
)

// Weight is saved as 4+4+4+1+1+2+2+1+2+4+4
type Weight struct {
	WeightID    uint32
	ItemID      uint32
	CampaignID  uint32
	Endx        uint32
	Mime8       uint8
	CapNumber   uint8
	CapPeriod   uint16
	CapThrottle uint16
	ClickNumber uint8
	ClickPeriod uint16
	Weight      float32
	Price       float32
}

func PackNWeights(weights []Weight) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, weights)
	return buf.Bytes(), err
}

func UnpackNWeights(data []byte) ([]Weight, error) {
	n := len(data) / 28
	weights := make([]Weight, n)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, weights)
	return weights, err
}

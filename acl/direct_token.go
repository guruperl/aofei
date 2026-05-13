package acl

import (
	"bytes"
	"encoding/base32"
	"encoding/binary"
	"fmt"
)

// PackDirectToken serializes two uint32 IDs using the historical SSP base32
// little-endian token format.
func PackDirectToken(x, y uint32) (string, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, x); err != nil {
		return "", err
	}
	if err := binary.Write(buf, binary.LittleEndian, y); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf.Bytes()), nil
}

func UnpackDirectToken(text string) (uint32, uint32, error) {
	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(text)
	if err != nil {
		return 0, 0, err
	}
	if len(data) != 8 {
		return 0, 0, fmt.Errorf("invalid direct token length %d", len(data))
	}
	var ids [2]uint32
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &ids); err != nil {
		return 0, 0, err
	}
	return ids[0], ids[1], nil
}

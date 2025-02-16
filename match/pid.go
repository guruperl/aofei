package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
)

type Pid struct {
	StartNano int64
	StartIP   uint32
	StartUa   uint32
}

func (self Pid) PackBytes() []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	if err != nil {
		return nil
	}

	return buf.Bytes()
}

func UnpackBytesPid(data []byte) (Pid, error) {
	buf := bytes.NewReader(data)
	v := Pid{}
	err := binary.Read(buf, binary.LittleEndian, &v)
	return v, err
}

func (self Pid) PackHex() string {
	bs := self.PackBytes()
	return hex.EncodeToString(bs)
}

func UnpackHexPid(text string) (Pid, error) {
	data, err := hex.DecodeString(text)
	if err != nil {
		return Pid{}, err
	}

	return UnpackBytesPid(data)
}

func (self Pid) Pack() (string, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, self); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

func UnpackPid(text string) (Pid, error) {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return Pid{}, err
	}

	buf := bytes.NewReader(data)
	v := Pid{}
	err = binary.Read(buf, binary.LittleEndian, &v)
	return v, err
}

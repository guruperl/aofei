package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"time"
	// "encoding/ascii85"
)

type Fcap struct {
	Total    uint8
	Startym  uint8
	Startdhm uint16
	Last     uint16
}

func NewFcap(when time.Time) Fcap {
	years := when.Year() - 2017
	months := int(when.Month())
	days := when.Day()
	hours := when.Hour()
	minutes := when.Minute()

	return Fcap{Total: uint8(1), Startym: uint8(years<<4 + months), Startdhm: uint16(days<<11 + hours<<6 + minutes), Last: uint16(0)}
}

func RefreshFcap(fcap Fcap, when time.Time) Fcap {
	return Fcap{fcap.Total + uint8(1), fcap.Startym, fcap.Startdhm, uint16(when.Sub(fcap.GetStart(when.Location())) / time.Minute)}
	// DO NOT exceeds 45 days!
}

func (self Fcap) GetStart(loc *time.Location) time.Time {
	return time.Date(2017+int(self.Startym>>4), time.Month(15&self.Startym), int(self.Startdhm>>11), int(31&(self.Startdhm>>6)), int(63&self.Startdhm), 0, 0, loc)
}

func (self Fcap) GetLast(loc *time.Location) time.Time {
	s := self.GetStart(loc)
	return s.Add(time.Duration(self.Last) * time.Minute)
}

func (self Fcap) SinceStart(when time.Time) uint16 {
	return uint16(when.Sub(self.GetStart(when.Location())) / time.Minute)
}

func (self Fcap) SinceLast(when time.Time) uint16 {
	return uint16(when.Sub(self.GetLast(when.Location())) / time.Minute)
}

type Cap struct {
	Fcap
	Cid uint32
}

func UpdateFcaps(fcaps *map[uint32]Fcap, id uint32, when time.Time) {
	if fcap, ok := (*fcaps)[id]; ok {
		(*fcaps)[id] = RefreshFcap(fcap, when)
	} else {
		(*fcaps)[id] = NewFcap(when)
	}
}

func PackFcaps(fcaps map[uint32]Fcap) (string, error) {
	buf := new(bytes.Buffer)
	for cid, fcap := range fcaps {
		c := Cap{fcap, cid}
		err := binary.Write(buf, binary.LittleEndian, c)
		if err != nil {
			return "", err
		}
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
	/*
	   src := buf.Bytes()
	   dbuf := make([]byte, ascii85.MaxEncodedLen(len(src)))
	   n := ascii85.Encode(dbuf, src)
	   return string(dbuf[0:n]), nil
	*/
}

func UnpackFcaps(current time.Time, text string) (map[uint32]Fcap, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	/*
		dbuf := make([]byte, 4*len(text))
		ndst, _, err := ascii85.Decode(dbuf, []byte(text), true)
		if err != nil { return nil, err }
		data := dbuf[0:ndst]
	*/

	caps := make([]Cap, len(data)/10)
	buf := bytes.NewReader([]byte(data))
	err = binary.Read(buf, binary.LittleEndian, caps)
	if err != nil {
		return nil, err
	}

	fcaps := make(map[uint32]Fcap)
	for _, c := range caps {
		// DO NOT exceed 45 days
		if current.Sub(c.GetStart(current.Location())) < 45*24*60*time.Minute {
			fcap := c.Fcap
			fcaps[c.Cid] = fcap
		}
	}
	return fcaps, nil
}

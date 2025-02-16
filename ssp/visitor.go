package ssp

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"time"

	"github.com/genelet/winter/ipsearch"
	"github.com/genelet/winter/match"
	"github.com/genelet/winter/uadevice"
)

type Visitor struct {
	match.Pid
	*ipsearch.PzGeo
	TotalDays   uint8
	LastYear    int16
	LastYearDay int16
	lastIP      uint32
}

func NewVisitor(current time.Time, ip32 uint32, ua *uadevice.PzUa, geo *ipsearch.PzGeo) *Visitor {
	pid := match.Pid{
		StartNano: current.UnixNano(),
		StartIP:   ip32,
		StartUa:   ua.Pack(),
	}
	return &Visitor{
		Pid:         pid,
		PzGeo:       geo,
		TotalDays:   0,
		LastYear:    int16(current.Year()),
		LastYearDay: int16(current.YearDay()),
		lastIP:      ip32,
	}
}

func (self *Visitor) Pack() (string, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(*self)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func UnpackVisitor(text string) (*Visitor, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	v := new(Visitor)
	err = dec.Decode(v)
	return v, err
}

func RetrieveVisitor(ucookie string, ips *ipsearch.IPSearch, ua *uadevice.PzUa, current time.Time, ipstr string, ip32 uint32) (*Visitor, bool, bool, error) {
	var err error
	var visitor *Visitor
	var geo *ipsearch.PzGeo
	needCookie := false
	isNew := false

	if ucookie == "" {
		//geo = ips.CreatePzGeo(ip_str)
		visitor = NewVisitor(current, ip32, ua, geo)
		needCookie = true
		isNew = true
	} else {
		visitor, err = UnpackVisitor(ucookie)
		if err != nil { // corrupted cookie, reset
			geo = ips.CreatePzGeo(ipstr)
			visitor = NewVisitor(current, ip32, ua, geo)
			needCookie = true
			isNew = true
		} else {
			if ip32 == visitor.lastIP {
				//geo = visitor.PzGeo
			} else {
				geo = ips.CreatePzGeo(ipstr)
				visitor.lastIP = ip32
				visitor.PzGeo = geo
				needCookie = true
			}

			if visitor.LastYear != int16(current.Year()) || visitor.LastYearDay != int16(current.YearDay()) {
				visitor.TotalDays += 1
				visitor.LastYear = int16(current.Year())
				visitor.LastYearDay = int16(current.YearDay())
				needCookie = true
			}
		}
	}

	return visitor, needCookie, isNew, nil
}

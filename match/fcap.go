package match

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/mediocregopher/radix/v4"
)

const (
	FCAPStartYear = 2025
	FCAPImp       = 1
	FCAPCli       = 2
)

// Fcap is frequecy cap class
// Total is total number of access since the starting time
// StartYM and StartDHM are for starting time
// Last is the minutes passed since the last time
type Fcap struct {
	Total    uint8  `json:"total"`
	StartYM  uint8  `json:"ym"`
	StartDHM uint16 `json:"dhm"`
	Last     uint16 `json:"ls"`
}

// CreateFcap creates a new Fcap instance from time
// DO NOT exceeds 45 days!
func CreateFcap(when time.Time) Fcap {
	years := when.Year() - FCAPStartYear
	months := int(when.Month())
	days := when.Day()
	hours := when.Hour()
	minutes := when.Minute()

	return Fcap{
		Total:    uint8(0),
		StartYM:  uint8(years<<4 + months),
		StartDHM: uint16(days<<11 + hours<<6 + minutes),
		Last:     uint16(0),
	}
}

// Refresh adds one more count and update the last access time
func (self Fcap) Refresh(when time.Time) {
	self.Total += 1
	self.Last = uint16(when.Sub(self.GetStart()) / time.Minute)
	return
}

// GetStart gets starting time in time
func (self Fcap) GetStart() time.Time {
	return time.Date(FCAPStartYear+int(self.StartYM>>4), time.Month(15&self.StartYM), int(self.StartDHM>>11), int(31&(self.StartDHM>>6)), int(63&self.StartDHM), 0, 0, time.Local)
}

// GetLast gets last access time in time
func (self Fcap) GetLast() time.Time {
	s := self.GetStart()
	return s.Add(time.Duration(self.Last) * time.Minute)
}

// SinceStart reports minutes passed since the start
func (self Fcap) SinceStart(when time.Time) uint16 {
	return uint16(when.Sub(self.GetStart()) / time.Minute)
}

// SinceLast reports minutes passed since the last access
func (self Fcap) SinceLast(when time.Time) uint16 {
	return uint16(when.Sub(self.GetLast()) / time.Minute)
}

type BothCap struct {
	Imp Fcap
	Cli Fcap
}

func (self BothCap) Refresh(ctx context.Context, conn radix.Client, when time.Time, pid string, item *Item, act int) error {
	imp := self.Imp
	cli := self.Cli
	if act&FCAPImp == 1 {
		if !item.Cap.ValidPeriodImp(when, imp) {
			imp = CreateFcap(when)
		}
		imp.Refresh(when)
	}
	if act&FCAPCli == 1 {
		if !item.Cap.ValidPeriodImp(when, imp) {
			cli = CreateFcap(when)
		}
		cli.Refresh(when)
	}

	buf := new(bytes.Buffer)
	object := BothCap{Imp: imp, Cli: cli}
	err := binary.Write(buf, binary.LittleEndian, object)
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "HSET", "fcap", fmt.Sprintf("%s:%d", pid, item.ItemID), string(buf.Bytes())))
}

func NewSFcaps(ctx context.Context, conn radix.Client, pid string, itemIDs []uint32) (map[uint32]BothCap, error) {
	names := []string{"fcap"}
	for _, itemID := range itemIDs {
		names = append(names, fmt.Sprintf("%s:%d", pid, itemID))
	}
	sdata := make([][]byte, len(itemIDs))
	err := conn.Do(ctx, radix.Cmd(&sdata, "HMGET", names...))
	if err != nil {
		return nil, err
	}
	sfcaps := make(map[uint32]BothCap)
	i := 0
	for _, data := range sdata {
		if len(data) < 1 {
			i++
			continue
		}
		buf := bytes.NewReader(data)
		sfcap := BothCap{}
		err := binary.Read(buf, binary.LittleEndian, &sfcap)
		if err != nil {
			return nil, err
		}
		sfcaps[itemIDs[i]] = sfcap
		i++
	}
	return sfcaps, nil
}

// GetCapped reports which campaigns are denied giving targeting caps requriment from advertisers,
func GetCapped(when time.Time, sfcaps map[uint32]BothCap, caps map[uint32]Cap) map[uint32]bool {
	denies := make(map[uint32]bool)
	for cid, thisCap := range caps {
		sfcap, ok := sfcaps[cid]
		if !ok {
			continue
		}
		if thisCap.CanServeImp(when, sfcap.Imp) &&
			thisCap.CanServeCli(when, sfcap.Cli) {
			continue
		}
		denies[cid] = true
	}
	return denies
}

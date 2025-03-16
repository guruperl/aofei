package match

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/mediocregopher/radix/v4"
)

const (
	FCAPStartYear = 2025
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

// Pack packs the BothCap into bytes
func (self BothCap) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// UnpackBothCap unpacks the BothCap from bytes
func UnpackBothCap(data []byte) (BothCap, error) {
	buf := bytes.NewReader(data)
	bothcap := BothCap{}
	err := binary.Read(buf, binary.LittleEndian, &bothcap)
	return bothcap, err
}

func HashNameBothCap(pid string) string {
	return fmt.Sprintf("bothcap:%s", pid)
}

func BothCapsToRedis(ctx context.Context, conn radix.Client, pid string, bothcaps map[uint32]BothCap) error {
	var arr []string
	for itemID, bothcap := range bothcaps {
		data, err := bothcap.Pack()
		if err != nil {
			return err
		}
		arr = append(arr, fmt.Sprintf("%d", itemID), string(data))
	}
	return conn.Do(ctx, radix.FlatCmd(nil, "HMSET", HashNameBothCap(pid), arr))
}

func BothCapsFromRedis(ctx context.Context, conn radix.Client, pid string, slotIDs []string) (map[uint32]BothCap, error) {
	var data map[string]string
	err := conn.Do(ctx, radix.FlatCmd(&data, "HMGET", HashNameBothCap(pid), slotIDs))
	if err != nil {
		return nil, err
	}
	bothcaps := make(map[uint32]BothCap)
	for str, sdata := range data {
		slotID, err := strconv.ParseUint(str, 10, 32)
		if err != nil {
			return nil, err
		}
		if sdata == "" {
			continue
		}
		bothcap, err := UnpackBothCap([]byte(sdata))
		if err != nil {
			return nil, err
		}
		bothcaps[uint32(slotID)] = bothcap
	}
	return bothcaps, nil
}

func (self BothCap) Refresh(when time.Time, block RAdv, isImp bool, isCli bool) {
	imp := self.Imp
	cli := self.Cli
	if isImp {
		if !block.Cap.ValidPeriodImp(when, imp) {
			imp = CreateFcap(when)
		}
		imp.Refresh(when)
	}
	if isCli {
		if !block.Cap.ValidPeriodImp(when, imp) {
			cli = CreateFcap(when)
		}
		cli.Refresh(when)
	}
}

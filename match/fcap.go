package match

import (
	"bytes"
	"context"
	"encoding/base64"
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
	StartYM  uint8  `json:"start_ym"`
	StartDHM uint16 `json:"start_dhm"`
	Last     uint16 `json:"last"`
}

// NewFcap creates a new Fcap instance from time
// DO NOT exceeds 45 days!
func NewFcap(when time.Time) Fcap {
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
func (self *Fcap) Refresh(when time.Time) {
	(*self).Total = self.Total + 1
	(*self).Last = uint16(when.Sub(self.GetStart()) / time.Minute)
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

// NewBothCap creates a new BothCap instance
func NewBothCap(when time.Time) BothCap {
	return BothCap{
		Imp: NewFcap(when),
		Cli: NewFcap(when),
	}
}

// Pack packs the BothCap into bytes
func (self BothCap) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// PackString packs the BothCap into RawURL string
func (self BothCap) PackString() (string, error) {
	data, err := self.Pack()
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// UnpackBothCap unpacks the BothCap from bytes
func UnpackBothCap(data []byte) (BothCap, error) {
	buf := bytes.NewReader(data)
	bothcap := BothCap{}
	err := binary.Read(buf, binary.LittleEndian, &bothcap)
	return bothcap, err
}

// UnpackBothCapString unpacks the BothCap from RawURL string
func UnpackBothCapString(text string) (BothCap, error) {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return BothCap{}, err
	}
	return UnpackBothCap(data)
}

func (self *BothCap) Refresh(when time.Time, block RAdv, isImp bool, isCli bool) {
	imp := self.Imp
	cli := self.Cli
	if isImp {
		if !block.Cap.ValidPeriodImp(when, imp) {
			imp = NewFcap(when)
		}
		imp.Refresh(when)
	}
	if isCli {
		if !block.Cap.ValidPeriodCli(when, cli) {
			cli = NewFcap(when)
		}
		cli.Refresh(when)
	}
	(*self).Imp = imp
	(*self).Cli = cli
}

func HashNameBothCap(pid string) string {
	return fmt.Sprintf("bothcap:%s", pid)
}

// MustRefreshBothCap reads bothcap from Redis and refreshes it. And write it back to Redis.
func MustRefreshBothCap(ctx context.Context, conn radix.Client, when time.Time, pid string, itemID uint32, cap Cap, isImp bool, isCli bool) error {
	if !isImp && !isCli {
		return nil
	}
	var data []byte
	key := HashNameBothCap(pid)
	itemIDStr := fmt.Sprintf("%d", itemID)
	err := conn.Do(ctx, radix.Cmd(&data, "HGET", key, itemIDStr))
	if err != nil {
		return err
	}

	var bothcap BothCap
	if len(data) > 0 {
		bothcap, err = UnpackBothCap(data)
		if err != nil {
			return err
		}
	} else {
		bothcap = NewBothCap(when)
	}
	bothcap.Refresh(when, RAdv{Cap: cap}, isImp, isCli)

	if data, err = bothcap.Pack(); err == nil {
		err = conn.Do(ctx, radix.Cmd(nil, "HSET", key, itemIDStr, string(data)))
	}
	return err
}

func BothCapsToRedis(ctx context.Context, conn radix.Client, pid string, bothcaps map[uint32]BothCap) error {
	arr := []string{HashNameBothCap(pid)}
	for itemID, bothcap := range bothcaps {
		data, err := bothcap.Pack()
		if err != nil {
			return err
		}
		arr = append(arr, fmt.Sprintf("%d", itemID), string(data))
	}
	return conn.Do(ctx, radix.Cmd(nil, "HMSET", arr...))
}

// BothCapsFromRedis retrieves bothcaps from Redis.
func BothCapsFromRedis(ctx context.Context, conn radix.Client, pid string, itemIDs []string) (map[uint32]BothCap, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}

	data := make([]string, len(itemIDs))
	arr := append([]string{HashNameBothCap(pid)}, itemIDs...)
	err := conn.Do(ctx, radix.Cmd(&data, "HMGET", arr...))
	if err != nil {
		return nil, err
	}
	bothcaps := make(map[uint32]BothCap)
	for i, sdata := range data {
		if len(sdata) == 0 {
			continue
		}
		itemID, err := strconv.Atoi(itemIDs[i])
		if err != nil {
			return nil, err
		}
		bothcap, err := UnpackBothCap([]byte(sdata))
		if err != nil {
			return nil, err
		}
		bothcaps[uint32(itemID)] = bothcap
	}
	if len(bothcaps) == 0 {
		return nil, nil
	}
	return bothcaps, nil
}

// BothCapsCleanupExpired removes expired bothcaps from Redis.
func BothCapsCleanupExpired(ctx context.Context, conn radix.Client, pid string, itemIDs []string) error {
	arr := append([]string{HashNameBothCap(pid)}, itemIDs...)
	return conn.Do(ctx, radix.Cmd(nil, "HDEL", arr...))
}

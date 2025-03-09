package holiday

import (
	"time"
)

type Dayhour struct {
    Current time.Time
}

func (self *Dayhour) GetTags() *Tags {
	format := "2006-01-02 15:04:05"
    zero, _ := time.Parse(format, "1970-01-01 00:00:00")
	fullhour := uint32(self.Current.Sub(zero).Hours())
	fullday  := uint32(fullhour/24)

	ref := map[uint32][]uint32{
		1001:[]uint32{fullday},
		1002:[]uint32{fullhour},
		1003:[]uint32{uint32(self.Current.Weekday())},
		1004:[]uint32{uint32(self.Current.Hour())}}
	return &Tags{TagHashArray:ref}
}

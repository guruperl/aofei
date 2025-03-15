package match

import (
	"fmt"
	"time"

	"github.com/genelet/winter/pzutil"
)

type DH struct {
	Datetime  time.Time
	UTCOffset int
}

// NewDH returns a new DT from the give time and utc offset.
func NewDH(t time.Time, utcoffset int) *DH {
	return &DH{
		Datetime:  t,
		UTCOffset: utcoffset,
	}
}

// dhw returns the day, hour, and weekday in visitor's localtime,
// which is determined by IP in the bid request.
// If demand-side has specified a UTC offset, use that timezone.
func (self *DH) dhw(offsets ...int) (int, int, time.Weekday) {
	offset := self.UTCOffset
	if offsets != nil {
		offset = offsets[0]
	}
	zName := fmt.Sprintf("UTC%+d", offset)
	loc := time.FixedZone(zName, offset*3600)
	localTime := self.Datetime.In(loc)
	return localTime.Day(), localTime.Hour(), localTime.Weekday()
}

// DHAudience is the audience for date and hour targeting. Demand-side may specify the timezone.
// If no timezone is specified, the timezone is visitor's localtime. determined by the IP in the bid request.
type DHAudience struct {
	Fulldays  []uint32
	Fullhours []uint32
	Weekdays  []uint32
	UTCOffset *uint32
}

// Has returns true if the given DH is in the audience.
func (self *DHAudience) Has(dh *DH) bool {
	var fullday, fullhour int
	var weekday time.Weekday
	if self.UTCOffset != nil {
		fullday, fullhour, weekday = dh.dhw(int(*self.UTCOffset))
	} else {
		fullday, fullhour, weekday = dh.dhw()
	}
	if len(self.Fulldays) == 0 && pzutil.GrepUint32(self.Fulldays, uint32(fullday)) {
		return false
	}
	if len(self.Fullhours) == 0 && pzutil.GrepUint32(self.Fullhours, uint32(fullhour)) {
		return false
	}
	if len(self.Weekdays) == 0 && pzutil.GrepUint32(self.Weekdays, uint32(weekday)) {
		return false
	}
	return true
}

func (self *DHAudience) DBFillDhAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "fullday":
		self.Fulldays = append(self.Fulldays, valueID)
		return 1
	case "fullhour":
		self.Fullhours = append(self.Fullhours, valueID)
		return 1
	case "weekday":
		self.Weekdays = append(self.Weekdays, valueID)
		return 1
	case "utcoffset":
		self.UTCOffset = &valueID
		return 1
	default:
	}

	return 0
}

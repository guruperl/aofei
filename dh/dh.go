package dh

import (
	"fmt"
	"time"
)

type DH struct {
	Datetime      time.Time
	FlyOffset     uint8
	OffsetSeconds int
}

// NewDH returns a new DT from the given time and stored UTC offset enum.
func NewDH(t time.Time, utcoffset uint8) *DH {
	seconds := offsetEnumSeconds(utcoffset)
	return &DH{
		Datetime:      t,
		FlyOffset:     utcoffset,
		OffsetSeconds: seconds,
	}
}

// NewDHFromMinutes returns a new DH from an OpenRTB UTC offset in minutes.
func NewDHFromMinutes(t time.Time, utcOffsetMinutes int64) *DH {
	return &DH{
		Datetime:      t,
		OffsetSeconds: int(utcOffsetMinutes) * 60,
	}
}

// dhw returns the day, hour, and weekday in visitor's localtime,
// which is determined by IP in the bid request.
// If demand-side has specified a UTC offset, use that timezone.
func (self *DH) dhw(offsets ...uint8) (int, int, int) {
	offsetSeconds := self.OffsetSeconds
	if offsets != nil && offsets[0] != 0 {
		offsetSeconds = offsetEnumSeconds(offsets[0])
	}
	zName := fmt.Sprintf("UTC%+g", float64(offsetSeconds)/3600.0)
	loc := time.FixedZone(zName, offsetSeconds)
	localTime := self.Datetime.In(loc)
	fullday := localTime.Day()
	// fullhour is stored as a legacy 1-based bucket: 1 means 00:00 and 24 means 23:00.
	fullhour := localTime.Hour() + 1
	weekday := int(localTime.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return fullday, fullhour, weekday
}

func offsetEnumSeconds(fly uint8) int {
	if fly >= 1 && fly <= 13 {
		return int(fly-1) * 3600
	} else if fly >= 14 && fly <= 24 {
		return int(fly-25) * 3600
	}
	return 0
}

// dhNames returns the day, hour, and weekday names.
func dhNames() map[string]map[uint32]string {
	names := make(map[string]map[uint32]string)
	names["weekday"] = map[uint32]string{
		0: "All",
		1: "Monday",
		2: "Tuesday",
		3: "Wednesday",
		4: "Thursday",
		5: "Friday",
		6: "Saturday",
		7: "Sunday",
	}
	names["fullday"] = map[uint32]string{
		0: "All",
	}
	for i := 1; i < 32; i++ {
		names["fullday"][uint32(i)] = fmt.Sprintf("%02d", i)
	}
	names["fullhour"] = map[uint32]string{
		0:  "All",
		1:  "00:00",
		2:  "01:00",
		3:  "02:00",
		4:  "03:00",
		5:  "04:00",
		6:  "05:00",
		7:  "06:00",
		8:  "07:00",
		9:  "08:00",
		10: "09:00",
		11: "10:00",
		12: "11:00",
		13: "12:00",
		14: "13:00",
		15: "14:00",
		16: "15:00",
		17: "16:00",
		18: "17:00",
		19: "18:00",
		20: "19:00",
		21: "20:00",
		22: "21:00",
		23: "22:00",
		24: "23:00",
	}
	names["utcoffset"] = map[uint32]string{
		14: "UTC-11",
		15: "UTC-10",
		16: "UTC-09",
		17: "UTC-08",
		18: "UTC-07",
		19: "UTC-06",
		20: "UTC-05",
		21: "UTC-04",
		22: "UTC-03",
		23: "UTC-02",
		24: "UTC-01",
		0:  "Visitor's Local Time",
		1:  "UTC+00",
		2:  "UTC+01",
		3:  "UTC+02",
		4:  "UTC+03",
		5:  "UTC+04",
		6:  "UTC+05",
		7:  "UTC+06",
		8:  "UTC+07",
		9:  "UTC+08",
		10: "UTC+09",
		11: "UTC+10",
		12: "UTC+11",
		13: "UTC+12",
	}
	return names
}

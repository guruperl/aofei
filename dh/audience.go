// Package dh handles date and hour targeting
package dh

import (
	"net/url"
	"strconv"
)

// DHAudience is the audience for date and hour targeting. Demand-side may specify the timezone.
// If no timezone is specified, the timezone is visitor's localtime. determined by the IP in the bid request.
type DHAudience struct {
	Fulldays  uint32
	Fullhours uint32
	Weekdays  uint32
	UTCOffset uint32
}

func (self *DHAudience) hasDay(day uint8, display ...bool) bool {
	if self == nil || ((display == nil || !display[0]) && self.Fulldays == 0) {
		return true
	}
	if day == 0 {
		return false
	}
	if self.Fulldays&(1<<day) != 0 {
		return true
	}
	return false
}

func (self *DHAudience) hasHour(hour uint8, display ...bool) bool {
	if self == nil || ((display == nil || !display[0]) && self.Fullhours == 0) {
		return true
	}
	if hour == 0 {
		return false
	}
	if self.Fullhours&(1<<hour) != 0 {
		return true
	}
	return false
}

func (self *DHAudience) hasWeekday(weekday uint8, display ...bool) bool {
	if self == nil || ((display == nil || !display[0]) && self.Weekdays == 0) {
		return true
	}
	if weekday == 0 {
		return false
	}
	if self.Weekdays&(1<<weekday) != 0 {
		return true
	}
	return false
}

// Has returns true if the given DH is in the audience.
func (self *DHAudience) Has(dh *DH) bool {
	if self == nil {
		return true
	}

	fullday, fullhour, weekday := dh.dhw(uint8(self.UTCOffset))
	if !self.hasDay(uint8(fullday)) {
		return false
	}
	if !self.hasHour(uint8(fullhour)) {
		return false
	}
	if !self.hasWeekday(uint8(weekday)) {
		return false
	}

	return true
}

func newDHAudience(days []uint32, hours []uint32, weeks []uint32, offsets []uint32) *DHAudience {
	aud := new(DHAudience)
	for _, day := range days {
		aud.Fulldays += (1 << day)
	}
	for _, hour := range hours {
		aud.Fullhours += (1 << hour)
	}
	for _, week := range weeks {
		aud.Weekdays += (1 << week)
	}
	if len(offsets) > 0 {
		aud.UTCOffset = offsets[0]
	}
	return aud
}

// DHResetArgs resets the ARGS to the values in the DHAudience, ready to be inserted or updated in the database.
func DHResetArgs(ARGS url.Values) error {
	pars := make(map[string][]uint32)
	for _, item := range []string{"fullday", "fullhour", "weekday", "utcoffset"} {
		if values, ok := ARGS[item]; ok {
			for _, value := range values {
				if value == "0" {
					pars[item] = nil
					break
				}
				if value != "" {
					v, err := strconv.ParseInt(value, 10, 32)
					if err != nil {
						return err
					}
					pars[item] = append(pars[item], uint32(v))
				}
			}
		}
	}
	if len(pars) == 0 {
		return nil
	}

	aud := newDHAudience(pars["fullday"], pars["fullhour"], pars["weekday"], pars["utcoffset"])

	ARGS.Del("fullday")
	ARGS.Del("fullhour")
	ARGS.Del("weekday")
	ARGS.Del("utcoffset")
	if aud.Fulldays != 0 {
		ARGS.Set("fullday", strconv.FormatInt(int64(aud.Fulldays), 10))
	}
	if aud.Fullhours != 0 {
		ARGS.Set("fullhour", strconv.FormatInt(int64(aud.Fullhours), 10))
	}
	if aud.Weekdays != 0 {
		ARGS.Set("weekday", strconv.FormatInt(int64(aud.Weekdays), 10))
	}
	if aud.UTCOffset != 0 {
		ARGS.Set("utcoffset", strconv.FormatInt(int64(aud.UTCOffset), 10))
	}
	return nil
}

// Tmpls returns the map of attribute name and valueID ready to use on web page.
func (self *DHAudience) Tmpls() map[string]map[int][]interface{} {
	other := make(map[string]map[int][]interface{})
	for attrname, val := range dhNames() {
		item := make(map[int][]interface{})
		for valueID, name := range val {
			switch attrname {
			case "fullday":
				item[int(valueID)] = []interface{}{name, self.hasDay(uint8(valueID), true)}
			case "fullhour":
				item[int(valueID)] = []interface{}{name, self.hasHour(uint8(valueID), true)}
			case "weekday":
				item[int(valueID)] = []interface{}{name, self.hasWeekday(uint8(valueID), true)}
			case "utcoffset":
				item[int(valueID)] = []interface{}{name, self.UTCOffset == valueID}
			default:
			}
		}
		other[attrname] = item
	}

	return other
}

func (self *DHAudience) DBFillDhAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "fullday":
		self.Fulldays = valueID
		return 1
	case "fullhour":
		self.Fullhours = valueID
		return 1
	case "weekday":
		self.Weekdays = valueID
		return 1
	case "utcoffset":
		self.UTCOffset = valueID
		return 1
	default:
	}

	return 0
}

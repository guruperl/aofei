package demo

import (
	"net/url"
	"strconv"
)

type DemoAudience struct {
	Genders   uint32
	Yobs      uint32
	Languages uint32
}

// has returns true if gender is in the DemoAudience set.
func (self *DemoAudience) hasGender(gender GENDER, display ...bool) bool {
	if (display == nil || !display[0]) && self.Genders == 0 {
		return true
	}
	if gender == GENDERUndefined {
		return false
	}
	if self.Genders&(1<<gender) != 0 {
		return true
	}

	return false
}

func (self *DemoAudience) hasYob(yob YOB, display ...bool) bool {
	if (display == nil || !display[0]) && self.Yobs == 0 {
		return true
	}
	if yob == YOBUndefined {
		return false
	}
	if self.Yobs&(1<<yob) != 0 {
		return true
	}

	return false
}

func (self *DemoAudience) hasLanguage(lang LANGUAGE, display ...bool) bool {
	if (display == nil || !display[0]) && self.Languages == 0 {
		return true
	}
	if lang == LanguageOther {
		return false
	}
	if self.Languages&(1<<lang) != 0 {
		return true
	}

	return false
}

// hasLanguages returns true if one of the languages is in the DemoAudience set.
func (self *DemoAudience) hasLanguages(langs WLangs) bool {
	if self.Languages == 0 {
		return true
	}
	if langs == 0 {
		return false
	}
	if self.Languages&uint32(langs) != 0 {
		return true
	}

	return false
}

// Has returns true if the DemoAudience has the given Demo.
func (self *DemoAudience) Has(dmo *Demo) bool {
	// if self == nil, it means no audience is set, so it should be true.
	if self == nil {
		return true
	}

	if !self.hasGender(dmo.Gender) {
		return false
	}
	if !self.hasLanguages(dmo.Language) {
		return false
	}
	if !self.hasYob(dmo.Yob) {
		return false
	}

	return true
}

func newDemoAudience(genders []uint32, yobs []uint32, langs []uint32) *DemoAudience {
	aud := new(DemoAudience)
	for _, gender := range genders {
		aud.Genders += (1 << gender)
	}
	for _, yob := range yobs {
		aud.Yobs += (1 << yob)
	}
	for _, lang := range langs {
		aud.Languages += (1 << lang)
	}
	return aud
}

// DemoResetArgs resets the ARGS to the values in the DemoAudience, ready to be inserted or updated in the database.
func DemoResetArgs(ARGS url.Values) error {
	pars := make(map[string][]uint32)
	for _, item := range []string{"gender", "yob", "language"} {
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

	aud := newDemoAudience(pars["gender"], pars["yob"], pars["language"])

	ARGS.Del("gender")
	ARGS.Del("yob")
	ARGS.Del("language")
	if aud.Genders != 0 {
		ARGS.Set("gender", strconv.FormatInt(int64(aud.Genders), 10))
	}
	if aud.Languages != 0 {
		ARGS.Set("language", strconv.FormatInt(int64(aud.Languages), 10))
	}
	if aud.Yobs != 0 {
		ARGS.Add("yob", strconv.FormatInt(int64(aud.Yobs), 10))
	}

	return nil
}

// Tmpls returns the map of attribute name and valueID ready to use on web page.
func (self *DemoAudience) Tmpls() map[string]map[int][]interface{} {
	demos := make(map[string]map[int][]interface{})
	for attrname, val := range demoNames() {
		item := make(map[int][]interface{})
		for valueID, name := range val {
			switch attrname {
			case "gender":
				item[int(valueID)] = []interface{}{name, self.hasGender(GENDER(valueID), true)}
			case "yob":
				item[int(valueID)] = []interface{}{name, self.hasYob(YOB(valueID), true)}
			case "language":
				item[int(valueID)] = []interface{}{name, self.hasLanguage(LANGUAGE(valueID), true)}
			}
		}
		demos[attrname] = item
	}
	return demos
}

// DBFillDemoAudience fills the DemoAudience with the attribute name and valueID, derived from the database.
func (self *DemoAudience) DBFillDemoAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "gender":
		self.Genders = valueID
		return 1
	case "yob":
		self.Yobs = valueID
		return 1
	case "language":
		self.Languages = valueID
		return 1
	default:
	}

	return 0
}

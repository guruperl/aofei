package demo

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/pzutil"
)

type DemoAudience struct {
	Genders     []uint32
	Yobs        []uint32
	Marrieds    []uint32
	Incomes     []uint32
	Childs      []uint32
	Households  []uint32
	Ethnicitys  []uint32
	Educations  []uint32
	Occupations []uint32
}

func (self *DemoAudience) MatchDemo(dmo *Demo) bool {
	if len(self.Genders) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Genders, dmo.Gender)) {
		return false
	}
	if len(self.Yobs) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Yobs, dmo.Yob)) {
		return false
	}
	if len(self.Marrieds) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Marrieds, dmo.Married)) {
		return false
	}
	if len(self.Incomes) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Incomes, dmo.Income)) {
		return false
	}
	if len(self.Childs) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Childs, dmo.Child)) {
		return false
	}
	if len(self.Households) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Households, dmo.Household)) {
		return false
	}
	if len(self.Ethnicitys) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Ethnicitys, dmo.Ethnicity)) {
		return false
	}
	if len(self.Educations) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Educations, dmo.Education)) {
		return false
	}
	if len(self.Occupations) > 0 && (dmo == nil || !pzutil.GrepUint32(self.Occupations, dmo.Occupation)) {
		return false
	}

	return true
}

func DemoAudienceFromArgs(ARGS url.Values) *DemoAudience {
	g := func(args url.Values, name string, which *[]uint32) {
		values := ARGS[name]
		if values != nil && len(values) > 0 {
			for _, value := range values {
				v, err := strconv.ParseUint(value, 10, 32)
				if err == nil && v > 0 {
					*which = append(*which, uint32(v))
				}
			}
		}
	}

	aud := new(DemoAudience)

	g(ARGS, "gender", &aud.Genders)
	g(ARGS, "yob", &aud.Yobs)
	g(ARGS, "married", &aud.Marrieds)
	g(ARGS, "income", &aud.Incomes)
	g(ARGS, "child", &aud.Childs)
	g(ARGS, "household", &aud.Households)
	g(ARGS, "ethnicity", &aud.Ethnicitys)
	g(ARGS, "education", &aud.Educations)
	g(ARGS, "occupation", &aud.Occupations)

	return aud
}

func (self *DemoAudience) ToArgs(ARGS url.Values) {
	g := func(args url.Values, name string, values []uint32) {
		if values != nil && len(values) > 0 {
			for _, value := range values {
				args.Add(name, strconv.FormatUint(uint64(value), 10))
			}
		}
	}

	g(ARGS, "gender", self.Genders)
	g(ARGS, "yob", self.Yobs)
	g(ARGS, "married", self.Marrieds)
	g(ARGS, "income", self.Incomes)
	g(ARGS, "child", self.Childs)
	g(ARGS, "household", self.Households)
	g(ARGS, "ethnicity", self.Ethnicitys)
	g(ARGS, "education", self.Educations)
	g(ARGS, "occupation", self.Occupations)
}

func (aud *DemoAudience) DBFillDemoAudience(attrname string, value_id uint32) {
	switch attrname {
	case "gender":
		aud.Genders = append(aud.Genders, value_id)
	case "yob":
		aud.Yobs = append(aud.Yobs, value_id)
	case "married":
		aud.Marrieds = append(aud.Marrieds, value_id)
	case "income":
		aud.Incomes = append(aud.Incomes, value_id)
	case "child":
		aud.Childs = append(aud.Childs, value_id)
	case "household":
		aud.Households = append(aud.Households, value_id)
	case "ethnicity":
		aud.Ethnicitys = append(aud.Ethnicitys, value_id)
	case "education":
		aud.Educations = append(aud.Educations, value_id)
	case "occupation":
		aud.Occupations = append(aud.Occupations, value_id)

	default:
	}
}

func (aud *DemoAudience) DbLineDemoAudience(attrname string, value_ids string) {
	if value_ids == "" {
		return
	}
	for _, id := range strings.Split(value_ids, ",") {
		if value_id, err := strconv.ParseUint(id, 10, 32); err == nil {
			aud.DBFillDemoAudience(attrname, uint32(value_id))
		}
	}
}

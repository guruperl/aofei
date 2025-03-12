package demo

import (
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

func (self *DemoAudience) Has(dmo *Demo) bool {
	if len(self.Genders) > 0 && (dmo == nil || dmo.Gender == 0 || !pzutil.GrepUint32(self.Genders, uint32(dmo.Gender))) {
		return false
	}
	if len(self.Yobs) > 0 && (dmo == nil || dmo.Yob == 0 || !pzutil.GrepUint32(self.Yobs, dmo.Yob)) {
		return false
	}
	if len(self.Marrieds) > 0 && (dmo == nil || dmo.Married == 0 || !pzutil.GrepUint32(self.Marrieds, dmo.Married)) {
		return false
	}
	if len(self.Incomes) > 0 && (dmo == nil || dmo.Income == 0 || !pzutil.GrepUint32(self.Incomes, dmo.Income)) {
		return false
	}
	if len(self.Childs) > 0 && (dmo == nil || dmo.Child == 0 || !pzutil.GrepUint32(self.Childs, dmo.Child)) {
		return false
	}
	if len(self.Households) > 0 && (dmo == nil || dmo.Household == 0 || !pzutil.GrepUint32(self.Households, dmo.Household)) {
		return false
	}
	if len(self.Ethnicitys) > 0 && (dmo == nil || dmo.Ethnicity == 0 || !pzutil.GrepUint32(self.Ethnicitys, dmo.Ethnicity)) {
		return false
	}
	if len(self.Educations) > 0 && (dmo == nil || dmo.Education == 0 || !pzutil.GrepUint32(self.Educations, dmo.Education)) {
		return false
	}
	if len(self.Occupations) > 0 && (dmo == nil || dmo.Occupation == 0 || !pzutil.GrepUint32(self.Occupations, dmo.Occupation)) {
		return false
	}

	return true
}

/*
	func DemoAudienceFromArgs(ARGS url.Values) *DemoAudience {
		g := func(_ url.Values, name string, which *[]uint32) {
			values := ARGS[name]
			if len(values) > 0 {
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
			if len(values) > 0 {
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
*/

func (self *DemoAudience) DBFillDemoAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "gender":
		self.Genders = append(self.Genders, valueID)
		return 1
	case "yob":
		self.Yobs = append(self.Yobs, valueID)
		return 1
	case "married":
		self.Marrieds = append(self.Marrieds, valueID)
		return 1
	case "income":
		self.Incomes = append(self.Incomes, valueID)
		return 1
	case "child":
		self.Childs = append(self.Childs, valueID)
		return 1
	case "household":
		self.Households = append(self.Households, valueID)
		return 1
	case "ethnicity":
		self.Ethnicitys = append(self.Ethnicitys, valueID)
		return 1
	case "education":
		self.Educations = append(self.Educations, valueID)
		return 1
	case "occupation":
		self.Occupations = append(self.Occupations, valueID)
		return 1
	default:
	}

	return 0
}

/*
func (self *DemoAudience) DBLineDemoAudience(attrname string, valueIDs string) {
	if valueIDs == "" {
		return
	}
	for _, id := range strings.Split(valueIDs, ",") {
		if valueID, err := strconv.ParseUint(id, 10, 32); err == nil {
			self.DBFillDemoAudience(attrname, uint32(valueID))
		}
	}
}
*/

// Package demo handles user demographic data
package demo

import "strconv"

// year starts at 1930
const (
	STARTYEAR uint32 = 1930
)

type DEMOGender uint32

const (
	DEMOGenderUndefined DEMOGender = iota
	DEMOGenderM
	DEMOGenderF
	DEMOGenderO
)

var String2Gender = map[string]DEMOGender{
	"UNDEFINED": DEMOGenderUndefined,
	"M":         DEMOGenderM,
	"F":         DEMOGenderF,
	"O":         DEMOGenderO,
	"m":         DEMOGenderM,
	"f":         DEMOGenderF,
	"o":         DEMOGenderO,
	"Male":      DEMOGenderM,
	"Female":    DEMOGenderF,
	"Other":     DEMOGenderO,
	"male":      DEMOGenderM,
	"female":    DEMOGenderF,
	"other":     DEMOGenderO,
}

var Gender2String = map[DEMOGender]string{
	DEMOGenderUndefined: "UNDEFINED",
	DEMOGenderM:         "M",
	DEMOGenderF:         "F",
	DEMOGenderO:         "O",
}

/*
var GenderTypeName = map[uint32]string{
	0: "UNDEFINED",
	1: "M",
	2: "F",
	3: "O",
}
var GenderTypeValue = map[string]uint32{
	"UNDEFINED": 0,
	"M":         1,
	"F":         2,
	"O":         3,
}
var MarriedTypeName = map[uint32]string{
	0: "UNDEFINED",
	1: "Married",
	2: "Single",
}
var MarriedTypeValue = map[string]uint32{
	"UNDEFINED": 0,
	"Married":   1,
	"Single":    2,
}
var IncomeTypeName = map[uint32]string{
	0:  "UNDEFINED",
	1:  "10K",
	2:  "20K",
	3:  "40K",
	4:  "60K",
	5:  "80K",
	6:  "100K",
	7:  "150K",
	8:  "200K",
	9:  "300K",
	10: "500K",
	11: "1M",
	12: "2M",
	13: "5M",
	14: "10M",
	15: "20M",
}
var IncomeTypeValue = map[string]uint32{
	"UNDEFINED": 0,
	"10K":       1,
	"20K":       2,
	"40K":       3,
	"60K":       4,
	"80K":       5,
	"100K":      6,
	"150K":      7,
	"200K":      8,
	"300K":      9,
	"500K":      10,
	"1M":        11,
	"2M":        12,
	"5M":        13,
	"10M":       14,
	"20M":       15,
}
var ChildTypeName = map[uint32]string{
	0: "UNDEFINED",
	1: "Yes",
	2: "No",
}
var ChildTypeValue = map[string]uint32{
	"UNDEFINE": 0,
	"Yes":      1,
	"No":       2,
}
var EthnicityTypeName = map[uint32]string{
	0: "UNDEFINED",
	1: "African",
	2: "Asian",
	3: "Caucasian",
	4: "Hispanic",
	5: "Native",
}
var EthnicityTypeValue = map[string]uint32{
	"UNDEFINED": 0,
	"African":   1,
	"Asian":     2,
	"Caucasian": 3,
	"Hispanic":  4,
	"Native":    5,
}
var EducationTypeName = map[uint32]string{
	0: "UNDEFINED",
	1: "Below9",
	2: "Below12",
	3: "Highschool",
	4: "College",
	5: "Bachelor",
	6: "Master",
	7: "PhD",
}
var EducationTypeValue = map[string]uint32{
	"UNDEFINED":  0,
	"Below9":     1,
	"Below12":    2,
	"Highschool": 3,
	"College":    4,
	"Bachelor":   5,
	"Master":     6,
	"PhD":        7,
}
var ReligionTypeName = map[uint32]string{
	0: "UNDEFINED",
	1: "Buddhism",
	2: "Christianity",
	3: "Islam",
	4: "Judaism",
	5: "Atheism",
	6: "Other",
}
var ReligionTypeValue = map[string]uint32{
	"UNDEFINED":    0,
	"Buddhism":     1,
	"Christianity": 2,
	"Islam":        3,
	"Judaism":      4,
	"Atheism":      5,
	"Other":        6,
}
var OccupationTypeName = map[uint32]string{
	0:  "UNDEFINED",
	1:  "少年儿童",
	2:  "无业人员",
	3:  "离退休",
	4:  "家庭主妇",
	5:  "机关团体",
	6:  "农业",
	7:  "畜牧业",
	8:  "渔业",
	9:  "木材森林业",
	10: "矿业采石业",
	11: "交通运输业",
	12: "餐旅业",
	13: "建筑工程",
	14: "制造业",
	15: "媒体",
	16: "医疗保健",
	17: "影视娱乐",
	18: "教育机构",
	19: "宗教人士",
	20: "邮政",
	21: "电信电力",
	22: "商业买卖",
	23: "银行保险",
	24: "自由职业",
	25: "家庭管理",
	26: "警察",
	27: "其他执法",
	28: "军人",
	29: "IT业",
	30: "运动员",
}
var OccupationTypeValue = map[string]uint32{
	"UNDEFINED": 0,
	"少年儿童":      1,
	"无业人员":      2,
	"离退休":       3,
	"家庭主妇":      4,
	"机关团体":      5,
	"农业":        6,
	"畜牧业":       7,
	"渔业":        8,
	"木材森林业":     9,
	"矿业采石业":     10,
	"交通运输业":     11,
	"餐旅业":       12,
	"建筑工程":      13,
	"制造业":       14,
	"媒体":        15,
	"医疗保健":      16,
	"影视娱乐":      17,
	"教育机构":      18,
	"宗教人士":      19,
	"邮政":        20,
	"电信电力":      21,
	"商业买卖":      22,
	"银行保险":      23,
	"自由职业":      24,
	"家庭管理":      25,
	"警察":        26,
	"其他执法":      27,
	"军人":        28,
	"IT业":       29,
	"运动员":       30,
}
*/

/*
•	性别，      2比特 （总数4；0表示未知，以下类推）
•	出生年，    7比特 （总数128）
•	婚否，      2比特 （总数4）
•	收入，      5比特 （总数32）
•	是否有孩子，2比特 （总数4）
•	家庭人总数，3比特 （总数8）
•	种族，      3比特 （总数8）
•	教育程度，  3比特 （总数8）
•	职业，      5比特 （总数 32）
*/

type Demo struct {
	Gender     DEMOGender
	Yob        uint32
	Married    uint32
	Income     uint32
	Child      uint32
	Household  uint32
	Ethnicity  uint32
	Education  uint32
	Occupation uint32
}

// NewDemo creates a new Demo
func NewDemo(gender string, yob int) *Demo {
	if gender == "" && yob == 0 {
		return nil
	}
	demo := new(Demo)
	if gender != "" {
		demo.Gender = String2Gender[gender]
	}
	if yob > 0 {
		demo.Yob = uint32(yob)
	}
	return demo
}

func (self *Demo) Pack() uint32 {
	yob := self.Yob
	if yob > 0 {
		if yob < STARTYEAR {
			yob = STARTYEAR
		}
		if yob >= 128 {
			yob = 127
		}
		yob -= STARTYEAR
	}
	if self.Gender >= 4 {
		self.Gender = DEMOGenderUndefined
	}
	if self.Married >= 4 {
		self.Married = 0
	}
	if self.Income >= 32 {
		self.Income = 31
	}
	if self.Child >= 4 {
		self.Child = 0
	}
	if self.Household >= 8 {
		self.Household = 7
	}
	if self.Ethnicity >= 8 {
		self.Ethnicity = 0
	}
	if self.Education >= 8 {
		self.Ethnicity = 0
	}
	if self.Occupation >= 32 {
		self.Occupation = 0
	}

	return ((uint32(self.Gender) & 3) << 0) +
		((yob & 127) << 2) +
		((self.Married & 3) << 9) +
		((self.Income & 31) << 11) +
		((self.Child & 3) << 16) +
		((self.Household & 7) << 18) +
		((self.Ethnicity & 7) << 21) +
		((self.Education & 7) << 24) +
		((self.Occupation & 31) << 27)
}

// UnpackDemo unpacks a 32 bits to the original object
func UnpackDemo(demo uint32) *Demo {
	gender := DEMOGender((demo >> 0) & 3)
	yob := (demo >> 2) & 127
	if yob > 0 {
		yob += STARTYEAR
	}
	married := (demo >> 9) & 3
	income := (demo >> 11) & 31
	child := (demo >> 16) & 3
	household := (demo >> 18) & 7
	ethnicity := (demo >> 21) & 7
	education := (demo >> 24) & 7
	occupation := (demo >> 27) & 31

	return &Demo{gender, yob, married, income, child, household, ethnicity, education, occupation}
}

func DemoAttrs() map[string]string {
	return map[string]string{
		"yob":        "Year of Birth",
		"household":  "Household",
		"gender":     "Gender",
		"married":    "Married Status",
		"income":     "Income",
		"occupation": "Occupation"}
}

func DemoNames() map[string]map[uint32]string {
	out := map[string]map[uint32]string{
		"gender": {
			0: "UNDEFINED",
			1: "Male",
			2: "Female",
		},
	}
	yobs := make(map[uint32]string)
	for yob := STARTYEAR; yob < 2020; yob++ {
		yobs[uint32(yob)] = strconv.Itoa(int(yob))
	}
	out["yob"] = yobs

	return out
}

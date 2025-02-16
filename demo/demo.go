package demo

import (
	"net/url"
	"strconv"
	"github.com/genelet/winter/pzutil"
)

type Demo struct {
    Gender  uint32
    Yob     uint32
    Married uint32
    Income  uint32
    Child   uint32
    Household   uint32
    Ethnicity   uint32
    Education   uint32
    Occupation  uint32
}

// year starts at 1930
const (
	StartYear uint32 = 1930
)

var GenderType_name  = map[uint32]string{
	0:"UNDEFINED",
	1:"M",
	2:"F",
	3:"O"}
var GenderType_value = map[string]uint32{
	"UNDEFINED":0,
	"M":1,
	"F":2,
	"O":3}

var MarriedType_name  = map[uint32]string{
	0:"UNDEFINED",
	1:"Married",
	2:"Single"}
var MarriedType_value = map[string]uint32{
	"UNDEFINED":0,
	"Married":1,
	"Single":2}

var IncomeType_name  = map[uint32]string{
	 0:"UNDEFINED",
	 1:"10K",
	 2:"20K",
	 3:"40K",
	 4:"60K",
	 5:"80K",
	 6:"100K",
	 7:"150K",
	 8:"200K",
	 9:"300K",
	10:"500K",
	11:"1M",
	12:"2M",
	13:"5M",
	14:"10M",
	15:"20M"}
var IncomeType_value = map[string]uint32{
    "UNDEFINED"  : 0,
    "10K"     : 1,
    "20K"     : 2,
    "40K"     : 3,
    "60K"     : 4,
    "80K"     : 5,
    "100K"    : 6,
    "150K"    : 7,
    "200K"    : 8,
    "300K"    : 9,
    "500K"    : 10,
    "1M"      : 11,
    "2M"      : 12,
    "5M"      : 13,
    "10M"     : 14,
    "20M"     : 15}

var ChildType_name  = map[uint32]string{
	0:"UNDEFINED",
	1:"Yes",
	2:"No"}
var ChildType_value = map[string]uint32{
	"UNDEFINE":0,
	"Yes":1,
	"No":2}

var EthnicityType_name  = map[uint32]string{
	0:"UNDEFINED",
	1:"African",
	2:"Asian",
	3:"Caucasian",
	4:"Hispanic",
	5:"Native"}
var EthnicityType_value = map[string]uint32{
	"UNDEFINED":0,
	"African"  :1,
	"Asian"    :2,
	"Caucasian":3,
	"Hispanic" :4,
	"Native"   :5}

var EducationType_name  = map[uint32]string{
	0:"UNDEFINED",
	1:"Below9",
	2:"Below12",
	3:"Highschool",
	4:"College",
	5:"Bachelor",
	6:"Master",
	7:"PhD"}
var EducationType_value = map[string]uint32{
	"UNDEFINED" :0,
	"Below9"    :1,
	"Below12"   :2,
	"Highschool":3,
	"College"   :4,
	"Bachelor"  :5,
	"Master"    :6,
	"PhD"       :7}

var ReligionType_name = map[uint32]string{
	0:"UNDEFINED",
	1:"Buddhism",
	2:"Christianity",
	3:"Islam",
	4:"Judaism",
	5:"Atheism",
	6:"Other"}

var ReligionType_value = map[string]uint32{
	"UNDEFINED":0,
	"Buddhism":1,
	"Christianity":2,
	"Islam":3,
	"Judaism":4,
	"Atheism":5,
	"Other":6}

var OccupationType_name  = map[uint32]string{
    0:"UNDEFINED",
    1:"少年儿童",
    2:"无业人员",
    3:"离退休",
    4:"家庭主妇",
    5:"机关团体",
    6:"农业",
    7:"畜牧业",
    8:"渔业",
    9:"木材森林业",
   10:"矿业采石业",
   11:"交通运输业",
   12:"餐旅业",
   13:"建筑工程",
   14:"制造业",
   15:"媒体",
   16:"医疗保健",
   17:"影视娱乐",
   18:"教育机构",
   19:"宗教人士",
   20:"邮政",
   21:"电信电力",
   22:"商业买卖",
   23:"银行保险",
   24:"自由职业",
   25:"家庭管理",
   26:"警察",
   27:"其他执法",
   28:"军人",
   29:"IT业",
   30:"运动员"}
var OccupationType_value = map[string]uint32{
   "UNDEFINED"  :0,
   "少年儿童"   :1,
   "无业人员"   :2,
   "离退休"     :3,
   "家庭主妇"   :4,
   "机关团体"   :5,
   "农业"       :6,
   "畜牧业"     :7,
   "渔业"       :8,
   "木材森林业" :9,
   "矿业采石业" :10,
   "交通运输业" :11,
   "餐旅业"     :12,
   "建筑工程"   :13,
   "制造业"     :14,
   "媒体"       :15,
   "医疗保健"   :16,
   "影视娱乐"   :17,
   "教育机构"   :18,
   "宗教人士"   :19,
   "邮政"       :20,
   "电信电力"   :21,
   "商业买卖"   :22,
   "银行保险"   :23,
   "自由职业"   :24,
   "家庭管理"   :25,
   "警察"       :26,
   "其他执法"   :27,
   "军人"       :28,
   "IT业"       :29,
   "运动员"     :30}

func (self *Demo)Pack() uint32 {
	yob := self.Yob
	if yob > 0 && yob < StartYear {
		yob = StartYear
	}
	yob -= (StartYear-1)
    if self.Gender   >=  4  { self.Gender   = 0  }
    if yob           >=128  { yob           = 127}
    if self.Married  >=  4  { self.Married  = 0  }
    if self.Income   >= 32  { self.Income   = 31 }
    if self.Child    >=  4  { self.Child    = 0  }
    if self.Household>=  8  { self.Household= 7  }
    if self.Ethnicity>=  8  { self.Ethnicity= 0  }
    if self.Education>=  8  { self.Ethnicity= 0  }
    if self.Occupation>=32  { self.Occupation=0  }

    return ( (self.Gender     &   3) << 0 ) +
	( (yob             & 127) <<  2 ) +
    ( (self.Married    &   3) <<  9 ) +
    ( (self.Income     &  31) << 11 ) +
    ( (self.Child      &   3) << 16 ) +
    ( (self.Household  &   7) << 18 ) +
    ( (self.Ethnicity  &   7) << 21 ) +
    ( (self.Education  &   7) << 24 ) +
    ( (self.Occupation &  31) << 27 )
}
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

func UnpackDemo(demo uint32) *Demo {
    return &Demo{UnpackDemoGender(demo), UnpackDemoYob(demo), UnpackDemoMarried(demo), UnpackDemoIncome(demo), UnpackDemoChild(demo), UnpackDemoHousehold(demo), UnpackDemoEthnicity(demo), UnpackDemoEducation(demo), UnpackDemoOccupation(demo)}
}
func UnpackDemoGender(demo uint32) uint32 {
	return (demo >>  0) &  3
}
func UnpackDemoYob(demo uint32) uint32 {
	b := (demo >>  2) &  127
	if b > 0 { b += (StartYear-1) }
	return b
}
func UnpackDemoMarried(demo uint32) uint32 {
	return (demo >>  9) &  3
}
func UnpackDemoIncome(demo uint32) uint32 {
	return (demo >> 11) &  31
}
func UnpackDemoChild(demo uint32) uint32 {
	return (demo >> 16) &  3
}
func UnpackDemoHousehold(demo uint32) uint32 {
	return (demo >> 18) &  7
}
func UnpackDemoEthnicity(demo uint32) uint32 {
	return (demo >> 21) &  7
}
func UnpackDemoEducation(demo uint32) uint32 {
	return (demo >> 24) &  7
}
func UnpackDemoOccupation(demo uint32) uint32 {
	return (demo >> 27) &  31
}

func MatchGender(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoGender), attrvalue)
}
func MatchYob(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoYob), attrvalue)
}
func MatchMarried(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoMarried), attrvalue)
}
func MatchIncome(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoIncome), attrvalue)
}
func MatchChild(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoChild), attrvalue)
}
func MatchHousehold(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoHousehold), attrvalue)
}
func MatchEthnicity(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoEthnicity), attrvalue)
}
func MatchEducation(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoEducation), attrvalue)
}
func MatchOccupation(demos []uint32, attrvalue uint32) bool {
	return pzutil.GrepUint32(pzutil.MapUint32(demos, UnpackDemoOccupation), attrvalue)
}

func DemoAttrs() map[string]string {
	return map[string]string{
		"yob":"Year of Birth", "household":"Household",
		"gender":"Gender", "married":"Married Status", "income":"Income",
		"occupation":"Occupation"}
}

func DemoNames() map[string]map[uint32]string {
	out := map[string]map[uint32]string{
"gender":GenderType_name, "married": MarriedType_name,
"income": IncomeType_name,
"occupation": OccupationType_name}
	for name, items := range out {
		if _, ok := items[uint32(0)]; ok {
			delete(out[name], uint32(0))
		}
	}
	delete(out["gender"],uint32(3))
    yobs := make(map[uint32]string)
    for yob:=1930; yob<2017; yob++ {
        yobs[uint32(yob)] = strconv.Itoa(yob)
    }
	hs := make(map[uint32]string)
	for h:=1; h<8; h++ {
		hs[uint32(h)] = strconv.Itoa(h)
	}
	out["yob"] = yobs
	out["household"] = hs
	return out
}

func CreateDemo(gender, yob_str, married, income, child, household_str, ethnicity, education, occupation string) *Demo {
	yob0, err := strconv.ParseUint(yob_str, 10, 32)
	yob := uint32(0)
	if err == nil {
		yob = uint32(yob0)
	}
	household0, err := strconv.ParseUint(household_str, 10, 32)
	household := uint32(0)
	if err == nil {
		 household = uint32(household0)
	}

	return &Demo{GenderType_value[gender], yob, MarriedType_value[married], IncomeType_value[income], ChildType_value[child], household, EthnicityType_value[ethnicity], EducationType_value[education], OccupationType_value[occupation]}
}

func CreateArgsDemo(ARGS url.Values, gender, yob, married, income, child, household, ethnicity, education, occupation string) *Demo {
	a := ARGS.Get(gender);     if a=="" { a = "UNDEFINED" }
	b := ARGS.Get(yob);        if b=="" { b = "UNDEFINED" }
	c := ARGS.Get(married);    if c=="" { c = "UNDEFINED" }
	d := ARGS.Get(income);     if d=="" { d = "UNDEFINED" }
	e := ARGS.Get(child);      if e=="" { e = "UNDEFINED" }
	f := ARGS.Get(household);  if f=="" { f = "UNDEFINED" }
	g := ARGS.Get(ethnicity);  if g=="" { g = "UNDEFINED" }
	h := ARGS.Get(education);  if h=="" { h = "UNDEFINED" }
	i := ARGS.Get(occupation); if i=="" { i = "UNDEFINED" }
	return CreateDemo(a, b, c, d, e, f, g, h, i)
}

func SetArgsDemos(ARGS url.Values, demography, gender, yob, married, income, child, household, ethnicity, education, occupation string) {
    genders, ok := ARGS[gender]
    if !ok { genders = make([]string,0) }
    yobs, ok := ARGS[yob]
    if !ok { yobs = make([]string,0) }
    marrieds, ok := ARGS[married]
    if !ok { marrieds = make([]string,0) }
    incomes, ok := ARGS[income]
    if !ok { incomes = make([]string,0) }
    childs, ok := ARGS[child]
    if !ok { childs = make([]string,0) }
    households, ok := ARGS[household]
    if !ok { households = make([]string,0) }
    ethnicitys, ok := ARGS[ethnicity]
    if !ok { ethnicitys = make([]string,0) }
    educations, ok := ARGS[education]
    if !ok { educations = make([]string,0) }
    occupations, ok := ARGS[occupation];
    if !ok { occupations = make([]string,0) }

	for _, a := range genders {
	for _, b := range yobs {
	for _, c := range marrieds {
	for _, d := range incomes {
	for _, e := range childs {
	for _, f := range households {
	for _, g := range ethnicitys {
	for _, h := range educations {
	for _, i := range occupations {
		demo := CreateDemo(a, b, c, d, e, f, g, h, i)
		ARGS.Add(demography, strconv.FormatUint(uint64(demo.Pack()), 10))
	} } } } } } } } }
}

func GetDemoIndividuals(lists []*Demo, other map[string]interface{}, gender, yob, married, income, child, household, ethnicity, education, occupation string) {
//lists is for a specific campaign
//targetname_id, attrname_id, attrname, targetvalue_id, value_id
	genders  := make(map[string]bool)
	   yobs  := make(map[string]bool)
	marrieds := make(map[string]bool)
	incomes  := make(map[string]bool)
	childs   := make(map[string]bool)
	households  := make(map[string]bool)
	ethnicitys  := make(map[string]bool)
	educations  := make(map[string]bool)
	occupations := make(map[string]bool)
	for _, name := range GenderType_name {
		if name=="UNDEFINED" { continue }
		genders[name] = false
	}
	for yob:=1930; yob<2017; yob++ {
		yobs[strconv.Itoa(yob)] = false
	}
	for _, name := range MarriedType_name {
		if name=="UNDEFINED" { continue }
		marrieds[name] = false
	}
	for _, name := range IncomeType_name {
		if name=="UNDEFINED" { continue }
		incomes[name] = false
	}
	for _, name := range ChildType_name {
		if name=="UNDEFINED" { continue }
		childs[name] = false
	}
	for household:=1; household<8; household++ {
		households[strconv.Itoa(household)] = false
	}
	for _, name := range EthnicityType_name {
		if name=="UNDEFINED" { continue }
		ethnicitys[name] = false
	}
	for _, name := range EducationType_name {
		if name=="UNDEFINED" { continue }
		educations[name] = false
	}
	for _, name := range OccupationType_name {
		if name=="UNDEFINED" { continue }
		occupations[name] = false
	}

	for _, demo := range lists {
		if demo.Gender > 0  { genders[GenderType_name[demo.Gender]]=true }
		if demo.Yob > 0     { yobs[strconv.Itoa(int(demo.Yob))]=true }
		if demo.Married > 0 { marrieds[MarriedType_name[demo.Married]]=true }
		if demo.Income > 0  { incomes[IncomeType_name[demo.Income]]=true }
		if demo.Child > 0   { childs[ChildType_name[demo.Child]]=true }
		if demo.Household > 0  { households[strconv.Itoa(int(demo.Household))]=true }
		if demo.Ethnicity > 0  { ethnicitys[EthnicityType_name[demo.Ethnicity]]=true }
		if demo.Education > 0  { educations[EducationType_name[demo.Education]]=true }
		if demo.Occupation > 0 { occupations[OccupationType_name[demo.Occupation]]=true }
	}

	other[gender]  = genders
	other[yob]     = yobs
	other[married] = marrieds
	other[income]  = incomes
	other[child]   = childs
	other[household]  = households
	other[ethnicity]  = ethnicitys
	other[education]  = educations
	other[occupation] = occupations
}

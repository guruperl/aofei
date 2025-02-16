package demo

import (
	"testing"
	"github.com/genelet/winter/pzutil"
)

func TestDemo(t *testing.T) {
	for k, v := range GenderType_name {
		if GenderType_value[v] != k {
			t.Errorf("%v %v", GenderType_name, GenderType_value)
		}
	}
	for k, v := range IncomeType_name {
		if IncomeType_value[v] != k {
			t.Errorf("%v %v", IncomeType_name, IncomeType_value)
		}
	}
	for k, v := range ChildType_name {
		if ChildType_value[v] != k {
			t.Errorf("%v %v", ChildType_name, ChildType_value)
		}
	}
	for k, v := range MarriedType_name {
		if MarriedType_value[v] != k {
			t.Errorf("%v %v", MarriedType_name, MarriedType_value)
		}
	}
	for k, v := range EthnicityType_name {
		if EthnicityType_value[v] != k {
			t.Errorf("%v %v", EthnicityType_name, EthnicityType_value)
		}
	}
	for k, v := range EducationType_name {
		if EducationType_value[v] != k {
			t.Errorf("%v %v", EducationType_name, EducationType_value)
		}
	}
	for k, v := range OccupationType_name {
		if OccupationType_value[v] != k {
			t.Errorf("%v %v", OccupationType_name, OccupationType_value)
		}
	}

	int_demos := make([]uint32,0)

	a := []uint32{0, 1}
	b := []uint32{1929, 1939, 1999, 2009}
	c := []uint32{0, 2}
	d := []uint32{0, 3, 13, 31}
	e := []uint32{0, 3}
	f := []uint32{0, 4, 5}
	g := []uint32{0, 5, 6}
	h := []uint32{0, 6, 7}
	i := []uint32{0, 8, 18, 28}
	for _, x1 := range a {
	for _, x2 := range b {
		if x2 < 1930 { x2 = 1930 }
	for _, x3 := range c {
	for _, x4 := range d {
	for _, x5 := range e {
	for _, x6 := range f {
	for _, x7 := range g {
	for _, x8 := range h {
	for _, x9 := range i {
		demo := &Demo{x1,x2,x3,x4,x5,x6,x7,x8,x9}
		pack := demo.Pack()
		int_demos = append(int_demos, pack)
		demo0:= UnpackDemo(pack)
		if x1!=demo0.Gender ||
		x2 != demo0.Yob ||
		x3 != demo0.Married ||
		x4 != demo0.Income ||
		x5 != demo0.Child ||
		x6 != demo0.Household ||
		x7 != demo0.Ethnicity ||
		x8 != demo0.Education ||
		x9 != demo0.Occupation {
			t.Errorf("%v %v", demo, demo0)
		}
	} } } } } } } } }


	for _, x1 := range a {
		if x1 < 1 { continue }
		if !MatchGender(int_demos, x1) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoGender),  x1)
		}
	}
	for _, x2 := range b {
		if x2 < 1 { continue }
		if x2 < 1930 { x2 = 1930 }
		if !MatchYob(int_demos, x2) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoYob),  x2)
		}
	}
	for _, x3 := range c {
		if x3 < 1 { continue }
		if !MatchMarried(int_demos, x3) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoMarried),  x3)
		}
	}
	for _, x4 := range d {
		if x4 < 1 { continue }
		if !MatchIncome(int_demos, x4) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoIncome),  x4)
		}
	}
	for _, x5 := range e {
		if x5 < 1 { continue }
		if !MatchChild(int_demos, x5) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoChild),  x5)
		}
	}
	for _, x6 := range f {
		if x6 < 1 { continue }
		if !MatchHousehold(int_demos, x6) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoHousehold),  x6)
		}
	}
	for _, x7 := range g {
		if x7 < 1 { continue }
		if !MatchEthnicity(int_demos, x7) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoEthnicity),  x7)
		}
	}
	for _, x8 := range h {
		if x8 < 1 { continue }
		if !MatchEducation(int_demos, x8) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoEducation),  x8)
		}
	}
	for _, x9 := range i {
		if x9 < 1 { continue }
		if !MatchOccupation(int_demos, x9) {
			t.Errorf("%v %v", pzutil.MapUint32(int_demos, UnpackDemoOccupation),  x9)
		}
	}


	demo   := &Demo{4,1929+128,4,32,4,8,8,8,32}
	packed := demo.Pack()
	demo0  := UnpackDemo(packed)
	if 0!=demo0.Gender ||
		127!= (demo0.Yob-1929) ||
		 0 != demo0.Married ||
		31 != demo0.Income ||
		 0 != demo0.Child ||
		 7 != demo0.Household ||
		 0 != demo0.Ethnicity ||
		 0 != demo0.Education ||
		 0 != demo0.Occupation {
		t.Errorf("%v", demo0)
	}
	if GenderType_value["UNDEFINED"] != demo0.Gender ||
		    MarriedType_value["UNDEFINED"] != demo0.Married   ||
		      ChildType_value["UNDEFINED"] != demo0.Child     ||
		  EthnicityType_value["UNDEFINED"] != demo0.Ethnicity ||
		  EducationType_value["UNDEFINED"] != demo0.Education ||
		 OccupationType_value["UNDEFINED"] != demo0.Occupation {
		t.Errorf("%v", demo0)
	}
	if "UNDEFINED" != GenderType_name[demo0.Gender] ||
	"UNDEFINED" != MarriedType_name[demo0.Married] ||
	"UNDEFINED" != ChildType_name[demo0.Child] ||
	"UNDEFINED" != EthnicityType_name[demo0.Ethnicity] ||
	"UNDEFINED" != EducationType_name[demo0.Education] ||
	"UNDEFINED" != OccupationType_name[demo0.Occupation] {
		t.Errorf("%s %s %s %s %s %s", GenderType_name[demo0.Gender], MarriedType_name[demo0.Married], ChildType_name[demo0.Child], EthnicityType_name[demo0.Ethnicity], EducationType_name[demo0.Education], OccupationType_name[demo0.Occupation])
	}

	//t.Errorf("%v", GetDemoParameters())
}

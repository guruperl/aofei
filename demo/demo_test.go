package demo

import (
	"testing"
)

func TestDemo(t *testing.T) {
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
			if x2 < 1930 {
				x2 = 1930
			}
			for _, x3 := range c {
				for _, x4 := range d {
					for _, x5 := range e {
						for _, x6 := range f {
							for _, x7 := range g {
								for _, x8 := range h {
									for _, x9 := range i {
										demo := &Demo{DEMOGender(x1), x2, x3, x4, x5, x6, x7, x8, x9}
										pack := demo.Pack()
										demo0 := UnpackDemo(pack)
										if x1 != uint32(demo0.Gender) ||
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
									}
								}
							}
						}
					}
				}
			}
		}
	}

	demo := &Demo{4, 1929 + 128, 4, 32, 4, 8, 8, 8, 32}
	packed := demo.Pack()
	demo0 := UnpackDemo(packed)
	if demo0.Gender != 0 ||
		(demo0.Yob-1929) != 127 ||
		demo0.Married != 0 ||
		demo0.Income != 31 ||
		demo0.Child != 0 ||
		demo0.Household != 7 ||
		demo0.Ethnicity != 0 ||
		demo0.Education != 0 ||
		demo0.Occupation != 0 {
		t.Errorf("%v", demo0)
	}
}

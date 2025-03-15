// Package demo handles user demographic data
package demo

// year starts at 1927
const (
	STARTYEAR int = 1927
)

type YOB uint8

const (
	YOBUndefined YOB = iota
	YOB1930
	YOB1934
	YOB1938
	YOB1942
	YOB1946
	YOB1950
	YOB1954
	YOB1958
	YOB1962
	YOB1966
	YOB1970
	YOB1974
	YOB1978
	YOB1982
	YOB1986
	YOB1990
	YOB1994
	YOB1998
	YOB2002
	YOB2006
	YOB2010
	YOB2014
	YOB2018
	YOB2022
	YOB2026
	YOB2030
	YOB2034
	YOB2038
	YOB2042
	YOB2046
	YOB2050
)

var YOB2String = map[YOB]string{
	YOBUndefined: "All",
	YOB1930:      "1927-1930",
	YOB1934:      "1931-1934",
	YOB1938:      "1935-1938",
	YOB1942:      "1939-1942",
	YOB1946:      "1943-1946",
	YOB1950:      "1947-1950",
	YOB1954:      "1951-1954",
	YOB1958:      "1955-1958",
	YOB1962:      "1959-1962",
	YOB1966:      "1963-1966",
	YOB1970:      "1967-1970",
	YOB1974:      "1971-1974",
	YOB1978:      "1975-1978",
	YOB1982:      "1979-1982",
	YOB1986:      "1983-1986",
	YOB1990:      "1987-1990",
	YOB1994:      "1991-1994",
	YOB1998:      "1995-1998",
	YOB2002:      "1999-2002",
	YOB2006:      "2003-2006",
	YOB2010:      "2007-2010",
	YOB2014:      "2011-2014",
	YOB2018:      "2015-2018",
	YOB2022:      "2019-2022",
	YOB2026:      "2023-2026",
	YOB2030:      "2027-2030",
	YOB2034:      "2031-2034",
	YOB2038:      "2035-2038",
	YOB2042:      "2039-2042",
	YOB2046:      "2043-2046",
	YOB2050:      "2047-2050",
}

type Demo struct {
	Gender GENDER
	Yob    YOB
	//	Married   uint32
	//	Income    uint32
	//	Child     uint32
	//	Household uint32
	//	Ethnicity uint32
	//	Education uint32
	Language WLangs
}

// NewDemo creates a new Demo
func NewDemo(gender string, yobfull uint32, langs []string) *Demo {
	if gender == "" && yobfull == 0 && len(langs) == 0 {
		return nil
	}

	demo := new(Demo)
	if gender != "" {
		demo.Gender = String2Gender[gender]
	}
	if yobfull != 0 {
		yob := int(yobfull) - STARTYEAR
		if yob < 0 {
			yob = 0
		}
		if yob >= 128 {
			yob = 127
		}
		demo.Yob = YOB(yob/4 + 1)
	}
	if len(langs) > 0 {
		demo.Language = NewWLangs(langs)
	}
	return demo
}

func demoNames() map[string]map[uint32]string {
	gender := map[uint32]string{}
	for k, v := range Gender2String {
		gender[uint32(k)] = v
	}
	out := map[string]map[uint32]string{
		"gender": gender,
	}
	lang := map[uint32]string{}
	for k, v := range LANGUAGE2Short {
		lang[uint32(k)] = Short2Full[v]
	}
	out["language"] = lang
	yob := map[uint32]string{}
	for k, v := range YOB2String {
		if k >= YOB2018 {
			continue
		}
		yob[uint32(k)] = v
	}
	out["yob"] = yob

	return out
}

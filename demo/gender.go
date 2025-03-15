// Package demo handles user demographic data
package demo

type GENDER uint8

const (
	GENDERUndefined GENDER = iota
	GENDERM
	GENDERF
	GENDERO
)

var String2Gender = map[string]GENDER{
	"UNDEFINED": GENDERUndefined,
	"M":         GENDERM,
	"F":         GENDERF,
	"O":         GENDERO,
	"m":         GENDERM,
	"f":         GENDERF,
	"o":         GENDERO,
	"Male":      GENDERM,
	"Female":    GENDERF,
	"Other":     GENDERO,
	"male":      GENDERM,
	"female":    GENDERF,
	"other":     GENDERO,
}

var Gender2String = map[GENDER]string{
	GENDERUndefined: "All",
	GENDERM:         "Male",
	GENDERF:         "Female",
	GENDERO:         "GENDEROther",
}

package match

type LANGUAGE uint8

const (
	LanguageOther LANGUAGE = iota
	LanguageEN
	LanguageES
	LanguageRU
	LanguageDE
	LanguageFR
	LanguageJA
	LanguagePT
	LanguageTR
	LanguageIT
	LanguageFA
	LanguageNL
	LanguagePL
	LanguageZH
	LanguageVI
	LanguageID
	LanguageCS
	LanguageKO
	LanguageUK
	LanguageAR
	LanguageEL
	LanguageFI
	LanguageHE
	LanguageSV
	LanguageRO
	LanguageHU
	LanguageTH
	LanguageDA
	LanguageSK
	LanguageBG
	LanguageSR
	LanguageNB
)

var LANGUAGE2Short = map[LANGUAGE]string{
	LanguageOther: "Other",
	LanguageEN:    "EN",
	LanguageES:    "ES",
	LanguageRU:    "RU",
	LanguageDE:    "DE",
	LanguageFR:    "FR",
	LanguageJA:    "JA",
	LanguagePT:    "PT",
	LanguageTR:    "TR",
	LanguageIT:    "IT",
	LanguageFA:    "FA",
	LanguageNL:    "NL",
	LanguagePL:    "PL",
	LanguageZH:    "ZH",
	LanguageVI:    "VI",
	LanguageID:    "ID",
	LanguageCS:    "CS",
	LanguageKO:    "KO",
	LanguageUK:    "UK",
	LanguageAR:    "AR",
	LanguageEL:    "EL",
	LanguageFI:    "FI",
	LanguageHE:    "HE",
	LanguageSV:    "SV",
	LanguageRO:    "RO",
	LanguageHU:    "HU",
	LanguageTH:    "TH",
	LanguageDA:    "DA",
	LanguageSK:    "SK",
	LanguageBG:    "BG",
	LanguageSR:    "SR",
	LanguageNB:    "NB",
}

var Short2LANGUAGE = map[string]LANGUAGE{
	"Other": LanguageOther,
	"EN":    LanguageEN,
	"ES":    LanguageES,
	"RU":    LanguageRU,
	"DE":    LanguageDE,
	"FR":    LanguageFR,
	"JA":    LanguageJA,
	"PT":    LanguagePT,
	"TR":    LanguageTR,
	"IT":    LanguageIT,
	"FA":    LanguageFA,
	"NL":    LanguageNL,
	"PL":    LanguagePL,
	"ZH":    LanguageZH,
	"VI":    LanguageVI,
	"ID":    LanguageID,
	"CS":    LanguageCS,
	"KO":    LanguageKO,
	"UK":    LanguageUK,
	"AR":    LanguageAR,
	"EL":    LanguageEL,
	"FI":    LanguageFI,
	"HE":    LanguageHE,
	"SV":    LanguageSV,
	"RO":    LanguageRO,
	"HU":    LanguageHU,
	"TH":    LanguageTH,
	"DA":    LanguageDA,
	"SK":    LanguageSK,
	"BG":    LanguageBG,
	"SR":    LanguageSR,
	"NB":    LanguageNB,
}

var Short2Full = map[string]string{
	"EN":    "English",
	"ES":    "Spanish",
	"RU":    "Russian",
	"DE":    "German",
	"FR":    "French",
	"JA":    "Japanese",
	"PT":    "Portuguese",
	"TR":    "Turkish",
	"IT":    "Italian",
	"FA":    "Persian",
	"NL":    "Dutch",
	"PL":    "Polish",
	"ZH":    "Chinese",
	"VI":    "Vietnamese",
	"ID":    "Indonesian",
	"CS":    "Czech",
	"KO":    "Korean",
	"UK":    "Ukrainian",
	"AR":    "Arabic",
	"EL":    "Greek",
	"HE":    "Hebrew",
	"SV":    "Swedish",
	"RO":    "Romanian",
	"HU":    "Hungarian",
	"TH":    "Thai",
	"DA":    "Danish",
	"SK":    "Slovak",
	"FI":    "Finnish",
	"BG":    "Bulgarian",
	"SR":    "Serbian",
	"NB":    "Norwegian",
	"Other": "Other",
}

type WLangs uint32

func NewWLangs(walng []string) WLangs {
	var wlangs WLangs
	for _, lang := range walng {
		if code, ok := Short2LANGUAGE[lang]; ok {
			wlangs |= 1 << code
		}
	}
	return wlangs
}

// ToLANGUEs returns a slice of LANGUAGEs from the WLangs set.
func (self WLangs) ToLANGUEs() []LANGUAGE {
	var langs []LANGUAGE
	for lang := LanguageOther; lang <= LanguageNB; lang++ {
		if self&(1<<lang) != 0 {
			langs = append(langs, lang)
		}
	}
	return langs
}

// Has returns true if one the languages is in the WLangs set.
func (self WLangs) Has(lang WLangs) bool {
	for _, lang := range lang.ToLANGUEs() {
		if self&(1<<lang) != 0 {
			return true
		}
	}
	return false
}

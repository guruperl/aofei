package holiday

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

var Languages = map[LANGUAGE]string{
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

var LanguageCodes = map[string]LANGUAGE{
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

type WLangs uint32

func NewWLangs(walng []string) WLangs {
	var wlangs WLangs
	for _, lang := range walng {
		if code, ok := LanguageCodes[lang]; ok {
			wlangs |= 1 << code
		}
	}
	return wlangs
}

func (self WLangs) Has(lang LANGUAGE) bool {
	return self&(1<<lang) != 0
}

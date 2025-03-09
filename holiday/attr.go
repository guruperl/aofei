package holiday

import (
//"log"
)

var AttrNames = map[uint32]string{
 1:"Sex",     2:"Byear",    3:"Bmonth",  4:"Horoscope", 5:"Zodiac", 6:"Bplace",
40:"Living", 74:"Brand",   75:"Screen", 76:"Time",     77:"Price", 78:"Plan",
79:"Group",  80:"Hold",    81:"Vip",    82:"Game",     83:"Shop",  84:"Health",
85:"Learn",  86:"Finance", 87:"Travel", 88:"Gps",      89:"Car",   90:"Food",
91:"Photo",  92:"Grocery", 93:"Work",   94:"Book",     95:"Report",96:"Social",
97:"Media",  98:"Browser", 99:"Other",

1001:"fullday",   1002:"fullhour",1003:"weekday",  1004:"weekhour",

1101:"continent", 1102:"country", 1103:"state",    1104:"city",    1105:"dma",
1106:"zip",       1107:"isp",     1108:"bandwidth",1111:"areacode",

1200:"pzua",      1201:"browser", 1202:"bversion", 1203:"os",
1204:"oversion",  1205:"platform",1206:"device",

1300:"demography",1301:"gender",  1302:"yob",      1303:"married",
1304:"income",    1305:"child",   1306:"household",1307:"ethnicity",
1308:"education", 1309:"occupation",

1401:"001001", 1402:"001002", 1403:"001003", 1404:"001004", 1405:"001005",
1406:"001006", 1407:"001007", 1408:"001008", 1409:"001009", 1410:"001010",
1411:"001011", 1412:"001012", 1413:"001013", 1414:"001014", 1415:"001015",
1416:"001016",
}

var AttrValues = map[string]uint32{
   "Sex":1,   "Byear":2,  "Bmonth":3, "Horoscope":4,  "Zodiac":5, "Bplace":6,
"Living":40,  "Brand":74, "Screen":75,     "Time":76,  "Price":77,  "Plan":78,
 "Group":79,   "Hold":80,    "Vip":81,     "Game":82,   "Shop":83,"Health":84,
 "Learn":85,"Finance":86, "Travel":87,      "Gps":88,    "Car":89,  "Food":90,
 "Photo":91,"Grocery":92,   "Work":93,     "Book":94, "Report":95,"Social":96,
 "Media":97,"Browser":98,  "Other":99,

"fullday":1001,  "fullhour":1002, "weekday":1003, "weekhour":1004,

"continent":1101, "country":1102,   "state":1103,     "city":1104, "dma":1105,
"zip":1106,           "isp":1107,   "bandwidth":1108, "areacode":1111,

     "pzua":1200, "browser":1201,"bversion":1202, "os":1203,
 "oversion":1204, "platform":1205, "device":1206,

"demography":1300, "gender":1301, "yob":1302,         "married":1303,
    "income":1304,  "child":1305, "household":1306, "ethnicity":1307,
 "education":1308, "occupation":1309,

"001001":1401, "001002":1402, "001003":1403, "001004":1404, "001005":1405,
"001006":1406, "001007":1407, "001008":1408, "001009":1409, "001010":1410,
"001011":1411, "001012":1412, "001013":1413, "001014":1414, "001015":1415,
"001016":1416,
}

// list of AttrIDs
type Attrs struct {
	AttrArray []uint32   `json:"attrs"`
}

func NewAttrsFromNames(names []string) *Attrs {
	arrs := make([]uint32, 0)
	for _, name := range names {
		arrs = append(arrs, AttrValues[name])
	}
	return &Attrs{arrs}
}

func (self *Attrs) GetAttrID(name string) uint32 {
   for _, attr := range self.AttrArray {
        if attr == AttrValues[name] {
            return attr
        }
    }
    return uint32(0)
}

func (self *Attrs) GetAttrIDs() []uint32 {
	ids := make([]uint32, 0)
	for _, attr := range self.AttrArray {
		ids = append(ids, attr)
	}
	return ids
}

func AllAttrIDs() []uint32 {
	ids := make([]uint32, 0)
	for id := range AttrNames {
		ids = append(ids, id)
	}
	return ids
}

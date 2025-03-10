package holiday

// insert into adv_attrname (attrname_id, attrname) values (1112, "utcoffset");
// insert into adv_attrname (attrname_id, attrname) values (1110, "lon");
// insert into adv_attrname (attrname_id, attrname) values (1109, "lat");
// update adv_attrname set attrname = 'utcday' where attrname_id=1004;
// insert into adv_attrname (attrname_id, attrname) values (1005, "utchour");
// insert into adv_attrname (attrname_id, attrname) values (1006, "utcweek");
// insert into adv_attrname (attrname_id, attrname) values (1010, "language");
var AttrNames = map[uint32]string{
	1: "Sex", 2: "Byear", 3: "Bmonth", 4: "Horoscope", 5: "Zodiac", 6: "Bplace",
	40: "Living", 74: "Brand", 75: "Screen", 76: "Time", 77: "Price", 78: "Plan",
	79: "Group", 80: "Hold", 81: "Vip", 82: "Game", 83: "Shop", 84: "Health",
	85: "Learn", 86: "Finance", 87: "Travel", 88: "Gps", 89: "Car", 90: "Food",
	91: "Photo", 92: "Grocery", 93: "Work", 94: "Book", 95: "Report", 96: "Social",
	97: "Media", 98: "Browser", 99: "Other",

	1001: "fullday", 1002: "fullhour", 1003: "weekday",
	1004: "utcday", 1005: "utchour", 1006: "utcweek",
	1010: "language",

	1101: "continent", 1102: "country", 1103: "state", 1104: "city", 1105: "dma",
	1106: "zip", 1107: "isp", 1108: "bandwidth", 1111: "areacode", 1112: "utcoffset", 1110: "lon", 1109: "lat",

	1200: "pzua", 1201: "browser", 1202: "bversion", 1203: "os",
	1204: "oversion", 1205: "platform", 1206: "device",

	1300: "demography", 1301: "gender", 1302: "yob", 1303: "married",
	1304: "income", 1305: "child", 1306: "household", 1307: "ethnicity",
	1308: "education", 1309: "occupation",

	1401: "cat1", 1402: "cat2", 1403: "cat3", 1404: "cat4", 1405: "cat5",
	1406: "cat6", 1407: "cat7", 1408: "cat8", 1409: "cat9", 1410: "cat10",
	1411: "cat11", 1412: "cat12", 1413: "cat13",
	1501: "sectionCat1", 1502: "sectionCat2", 1503: "sectionCat3", 1504: "sectionCat4", 1505: "sectionCat5",
	1506: "sectionCat6", 1507: "sectionCat7", 1508: "sectionCat8", 1509: "sectionCat9", 1510: "sectionCat10",
	1511: "sectionCat11", 1512: "sectionCat12", 1513: "sectionCat13",
	1601: "pageCat1", 1602: "pageCat2", 1603: "pageCat3", 1604: "pageCat4", 1605: "pageCat5",
	1606: "pageCat6", 1607: "pageCat7", 1608: "pageCat8", 1609: "pageCat9", 1610: "pageCat10",
	1611: "pageCat11", 1612: "pageCat12", 1613: "pageCat13",
	1701: "acat1", 1702: "acat2", 1703: "acat3", 1704: "acat4", 1705: "acat5",
	1706: "acat6", 1707: "acat7", 1708: "acat8", 1709: "acat9", 1710: "acat10",
	1711: "acat11", 1712: "acat12", 1713: "acat13",
	1801: "bcat1", 1802: "bcat2", 1803: "bcat3", 1804: "bcat4", 1805: "bcat5",
	1806: "bcat6", 1807: "bcat7", 1808: "bcat8", 1809: "bcat9", 1810: "bcat10",
	1811: "bcat11", 1812: "bcat12", 1813: "bcat13",
}

var AttrValues = map[string]uint32{
	"Sex": 1, "Byear": 2, "Bmonth": 3, "Horoscope": 4, "Zodiac": 5, "Bplace": 6,
	"Living": 40, "Brand": 74, "Screen": 75, "Time": 76, "Price": 77, "Plan": 78,
	"Group": 79, "Hold": 80, "Vip": 81, "Game": 82, "Shop": 83, "Health": 84,
	"Learn": 85, "Finance": 86, "Travel": 87, "Gps": 88, "Car": 89, "Food": 90,
	"Photo": 91, "Grocery": 92, "Work": 93, "Book": 94, "Report": 95, "Social": 96,
	"Media": 97, "Browser": 98, "Other": 99,

	"fullday": 1001, "fullhour": 1002, "weekday": 1003,
	"utcday": 1004, "utchour": 1005, "utcweek": 1006,
	"language": 1010,

	"continent": 1101, "country": 1102, "state": 1103, "city": 1104, "dma": 1105,
	"zip": 1106, "isp": 1107, "bandwidth": 1108, "areacode": 1111, "utcoffset": 1112, "lon": 1110, "lat": 1109,

	"pzua": 1200, "browser": 1201, "bversion": 1202, "os": 1203,
	"oversion": 1204, "platform": 1205, "device": 1206,

	"demography": 1300, "gender": 1301, "yob": 1302, "married": 1303,
	"income": 1304, "child": 1305, "household": 1306, "ethnicity": 1307,
	"education": 1308, "occupation": 1309,

	"cat1": 1401, "cat2": 1402, "cat3": 1403, "cat4": 1404, "cat5": 1405,
	"cat6": 1406, "cat7": 1407, "cat8": 1408, "cat9": 1409, "cat10": 1410,
	"cat11": 1411, "cat12": 1412, "cat13": 1413,
	"sectionCat1": 1501, "sectionCat2": 1502, "sectionCat3": 1503, "sectionCat4": 1504, "sectionCat5": 1505,
	"sectionCat6": 1506, "sectionCat7": 1507, "sectionCat8": 1508, "sectionCat9": 1509, "sectionCat10": 1510,
	"sectionCat11": 1511, "sectionCat12": 1512, "sectionCat13": 1513,
	"pageCat1": 1601, "pageCat2": 1602, "pageCat3": 1603, "pageCat4": 1604, "pageCat5": 1605,
	"pageCat6": 1606, "pageCat7": 1607, "pageCat8": 1608, "pageCat9": 1609, "pageCat10": 1610,
	"pageCat11": 1611, "pageCat12": 1612, "pageCat13": 1613,
	"acat1": 1701, "acat2": 1702, "acat3": 1703, "acat4": 1704, "acat5": 1705,
	"acat6": 1706, "acat7": 1707, "acat8": 1708, "acat9": 1709, "acat10": 1710,
	"acat11": 1711, "acat12": 1712, "acat13": 1713,
	"bcat1": 1801, "bcat2": 1802, "bcat3": 1803, "bcat4": 1804, "bcat5": 1805,
	"bcat6": 1806, "bcat7": 1807, "bcat8": 1808, "bcat9": 1809, "bcat10": 1810,
	"bcat11": 1811, "bcat12": 1812, "bcat13": 1813,
}

type Attrs []uint32

func NewAttrsFromNames(names []string) Attrs {
	var arrs []uint32
	for _, name := range names {
		arrs = append(arrs, AttrValues[name])
	}
	return arrs
}

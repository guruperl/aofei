package dmp

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/pzutil"
)

type DmpAudience struct {
	Sex        uint32   // 1, 性别 3, 2
	Byears     []uint32 // 2, 出生年份 70, 7=>8
	Bmonths    []uint32 // 3, 出生月份 13, 4
	Horoscopes []uint32 // 4, 生肖 13, 4
	Zodiacs    []uint32 // 5, 星座 13, 4
	Vip        uint32   // 81, 是否高消费人群 3, 2
	Browser    uint32   // 98, 浏览器 2, 1
	Gps        uint32   // 88, 交通导航 2, 1

	Bplaces []uint32 // 6, 出生-安徽省 343, 16
	Livings []uint32 // 40, 生活-安徽省 343, 16

	Brands  []uint32 // 74, 手机品牌 48, 6=>7
	Screens []uint32 // 75, 屏幕尺寸 6, 3=>4
	Times   []uint32 // 76, 手机上市时间 30, 5
	Prices  []uint32 // 77, 手机价格 9, 4
	Plans   []uint32 // 78, 消费价格 8, 3=>4
	Groups  []uint32 // 79, 人群 7, 3
	Holds   []uint32 // 80, 用户等级 5, 3

	Games    []uint32 // 82, APP游戏 18, 32
	Shops    []uint32 // 83, 购物导购 12, 32
	Finances []uint32 // 86, 金融理财 11, 32
	Grocerys []uint32 // 92, 生活服务 17, 32
	Medias   []uint32 // 97, 影音娱乐 14, 32
	Healths  []uint32 // 84, 健康养生 7, 16
	Learns   []uint32 // 85, 教育培训 7, 16
	Travels  []uint32 // 87, 旅游出行 9, 16
	Socials  []uint32 // 96, 社交聊天 8,  16
	Cars     []uint32 // 89, 养车用车 7, 16
	Foods    []uint32 // 90, 美食佳饮 3, 16
	Photos   []uint32 // 91, 拍照摄影 6, 16
	Works    []uint32 // 93, 商务效率 10, 16
	Books    []uint32 // 94, 图书阅读 5,  16
	Reports  []uint32 // 95, 新闻资讯 6,  16
	Others   []uint32 // 99, 其它 3,2
}

func (self *DmpAudience) Pack() ([]byte, error) {
	return pzutil.PackObject(self)
}

func UnpackAudience(data []byte) (*DmpAudience, error) {
	audience := new(DmpAudience)
	err := pzutil.UnpackObject(data, audience)
	return audience, err
}

func (self *DmpAudience) MatchDmp(dmmp *Dmp) bool {
	if self.Sex > 0 && (dmmp == nil || self.Sex != dmmp.Sex) {
		return false
	}
	if self.Vip > 0 && (dmmp == nil || self.Vip != dmmp.Vip) {
		return false
	}
	if self.Browser > 0 && (dmmp == nil || self.Browser != dmmp.Browser) {
		return false
	}
	if self.Gps > 0 && (dmmp == nil || self.Gps != dmmp.Gps) {
		return false
	}

	if len(self.Byears) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Byears, dmmp.Byear)) {
		return false
	}
	if len(self.Bmonths) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Bmonths, dmmp.Bmonth)) {
		return false
	}
	if len(self.Bplaces) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Bplaces, dmmp.Bplace)) {
		return false
	}
	if len(self.Livings) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Livings, dmmp.Living)) {
		return false
	}
	if len(self.Zodiacs) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Zodiacs, dmmp.Zodiac)) {
		return false
	}
	if len(self.Horoscopes) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Horoscopes, dmmp.Horoscope)) {
		return false
	}
	if len(self.Brands) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Brands, dmmp.Brand)) {
		return false
	}
	if len(self.Screens) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Screens, dmmp.Screen)) {
		return false
	}
	if len(self.Prices) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Prices, dmmp.Price)) {
		return false
	}
	if len(self.Plans) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Plans, dmmp.Plan)) {
		return false
	}
	if len(self.Groups) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Groups, dmmp.Group)) {
		return false
	}
	if len(self.Holds) > 0 && (dmmp == nil || !pzutil.GrepUint32(self.Holds, dmmp.Hold)) {
		return false
	}

	if len(self.Games) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Games, dmmp.Games)) {
		return false
	}
	if len(self.Finances) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Finances, dmmp.Finances)) {
		return false
	}
	if len(self.Grocerys) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Grocerys, dmmp.Grocerys)) {
		return false
	}
	if len(self.Medias) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Medias, dmmp.Medias)) {
		return false
	}
	if len(self.Healths) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Healths, dmmp.Healths)) {
		return false
	}
	if len(self.Learns) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Learns, dmmp.Learns)) {
		return false
	}
	if len(self.Travels) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Travels, dmmp.Travels)) {
		return false
	}
	if len(self.Socials) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Socials, dmmp.Socials)) {
		return false
	}
	if len(self.Cars) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Cars, dmmp.Cars)) {
		return false
	}
	if len(self.Foods) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Foods, dmmp.Foods)) {
		return false
	}
	if len(self.Photos) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Photos, dmmp.Photos)) {
		return false
	}
	if len(self.Works) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Works, dmmp.Works)) {
		return false
	}
	if len(self.Books) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Books, dmmp.Books)) {
		return false
	}
	if len(self.Reports) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Reports, dmmp.Reports)) {
		return false
	}
	if len(self.Others) > 0 && (dmmp == nil || !pzutil.GrepOrN(self.Others, dmmp.Others)) {
		return false
	}

	return true
}

func DmpAudienceFromArgs(ARGS url.Values) *DmpAudience {
	f := func(_ url.Values, name string, which *uint32) {
		value := ARGS.Get(name)
		if value != "" {
			v, err := strconv.ParseUint(value, 10, 32)
			if err == nil {
				*which = uint32(v)
			}
		}
	}

	g := func(_ url.Values, name string, which *[]uint32) {
		values := ARGS[name]
		if len(values) > 0 {
			for _, value := range values {
				v, err := strconv.ParseUint(value, 10, 32)
				if err == nil && v > 0 {
					*which = append(*which, uint32(v))
				}
			}
		}
	}

	aud := new(DmpAudience)

	f(ARGS, "Sex", &aud.Sex)
	f(ARGS, "Vip", &aud.Vip)
	f(ARGS, "Gps", &aud.Gps)
	f(ARGS, "Browser", &aud.Browser)

	g(ARGS, "Byear", &aud.Byears)
	g(ARGS, "Bmonth", &aud.Bmonths)
	g(ARGS, "Horoscope", &aud.Horoscopes)
	g(ARGS, "Zodiac", &aud.Zodiacs)
	g(ARGS, "Bplace", &aud.Bplaces)
	g(ARGS, "Living", &aud.Livings)
	g(ARGS, "Brand", &aud.Brands)
	g(ARGS, "Screen", &aud.Screens)
	g(ARGS, "Time", &aud.Times)
	g(ARGS, "Price", &aud.Prices)
	g(ARGS, "Plan", &aud.Plans)
	g(ARGS, "Group", &aud.Groups)
	g(ARGS, "Hold", &aud.Holds)
	g(ARGS, "Game", &aud.Games)
	g(ARGS, "Shop", &aud.Shops)
	g(ARGS, "Finance", &aud.Finances)
	g(ARGS, "Grocery", &aud.Grocerys)
	g(ARGS, "Media", &aud.Medias)
	g(ARGS, "Health", &aud.Healths)
	g(ARGS, "Learn", &aud.Learns)
	g(ARGS, "Travel", &aud.Travels)
	g(ARGS, "Social", &aud.Socials)
	g(ARGS, "Car", &aud.Cars)
	g(ARGS, "Food", &aud.Foods)
	g(ARGS, "Photo", &aud.Photos)
	g(ARGS, "Work", &aud.Works)
	g(ARGS, "Book", &aud.Books)
	g(ARGS, "Report", &aud.Reports)
	g(ARGS, "Other", &aud.Others)

	return aud
}

func (self *DmpAudience) ToArgs(ARGS url.Values) {
	f := func(args url.Values, name string, value uint32) {
		if value > 0 {
			args.Add(name, strconv.FormatUint(uint64(value), 10))
		}
	}

	g := func(args url.Values, name string, values []uint32) {
		if len(values) > 0 {
			for _, value := range values {
				args.Add(name, strconv.FormatUint(uint64(value), 10))
			}
		}
	}

	f(ARGS, "Sex", self.Sex)
	f(ARGS, "Vip", self.Vip)
	f(ARGS, "Gps", self.Gps)
	f(ARGS, "Browser", self.Browser)

	g(ARGS, "Byear", self.Byears)
	g(ARGS, "Bmonth", self.Bmonths)
	g(ARGS, "Horoscope", self.Horoscopes)
	g(ARGS, "Zodiac", self.Zodiacs)
	g(ARGS, "Bplace", self.Bplaces)
	g(ARGS, "Living", self.Livings)
	g(ARGS, "Brand", self.Brands)
	g(ARGS, "Screen", self.Screens)
	g(ARGS, "Time", self.Times)
	g(ARGS, "Price", self.Prices)
	g(ARGS, "Plan", self.Plans)
	g(ARGS, "Group", self.Groups)
	g(ARGS, "Hold", self.Holds)
	g(ARGS, "Game", self.Games)
	g(ARGS, "Shop", self.Shops)
	g(ARGS, "Finance", self.Finances)
	g(ARGS, "Grocery", self.Grocerys)
	g(ARGS, "Media", self.Medias)
	g(ARGS, "Health", self.Healths)
	g(ARGS, "Learn", self.Learns)
	g(ARGS, "Travel", self.Travels)
	g(ARGS, "Social", self.Socials)
	g(ARGS, "Car", self.Cars)
	g(ARGS, "Food", self.Foods)
	g(ARGS, "Photo", self.Photos)
	g(ARGS, "Work", self.Works)
	g(ARGS, "Book", self.Books)
	g(ARGS, "Report", self.Reports)
	g(ARGS, "Other", self.Others)

}

func (self *DmpAudience) DBFillDmpAudience(attrname string, valueID uint32) {
	switch attrname {
	case "Sex":
		self.Sex = valueID
	case "Vip":
		self.Vip = valueID
	case "Gps":
		self.Gps = valueID
	case "Browser":
		self.Browser = valueID
	case "Byear":
		self.Byears = append(self.Byears, valueID)
	case "Bmonth":
		self.Bmonths = append(self.Bmonths, valueID)
	case "Bplace":
		self.Bplaces = append(self.Bplaces, valueID)
	case "Living":
		self.Livings = append(self.Livings, valueID)
	case "Zodiac":
		self.Zodiacs = append(self.Zodiacs, valueID)
	case "Horoscope":
		self.Horoscopes = append(self.Horoscopes, valueID)
	case "Brand":
		self.Brands = append(self.Brands, valueID)
	case "Screen":
		self.Screens = append(self.Screens, valueID)
	case "Time":
		self.Times = append(self.Times, valueID)
	case "Price":
		self.Prices = append(self.Prices, valueID)
	case "Plan":
		self.Plans = append(self.Plans, valueID)
	case "Group":
		self.Groups = append(self.Groups, valueID)
	case "Hold":
		self.Holds = append(self.Holds, valueID)
	case "Game":
		self.Games = append(self.Games, valueID)
	case "Shop":
		self.Shops = append(self.Shops, valueID)
	case "Finance":
		self.Finances = append(self.Finances, valueID)
	case "Grocery":
		self.Grocerys = append(self.Grocerys, valueID)
	case "Media":
		self.Medias = append(self.Medias, valueID)
	case "Health":
		self.Healths = append(self.Healths, valueID)
	case "Learn":
		self.Learns = append(self.Learns, valueID)
	case "Travel":
		self.Travels = append(self.Travels, valueID)
	case "Social":
		self.Socials = append(self.Socials, valueID)
	case "Car":
		self.Cars = append(self.Cars, valueID)
	case "Food":
		self.Foods = append(self.Foods, valueID)
	case "Photo":
		self.Photos = append(self.Photos, valueID)
	case "Work":
		self.Works = append(self.Works, valueID)
	case "Book":
		self.Books = append(self.Books, valueID)
	case "Report":
		self.Reports = append(self.Reports, valueID)
	case "Other":
		self.Others = append(self.Others, valueID)

	default:
	}
}

func (self *DmpAudience) DBLineDmpAudience(attrname string, valueIDs string) {
	if valueIDs == "" {
		return
	}
	for _, id := range strings.Split(valueIDs, ",") {
		if valueID, err := strconv.ParseUint(id, 10, 32); err == nil {
			self.DBFillDmpAudience(attrname, uint32(valueID))
		}
	}
}

package dmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/genelet/winter/pzutil"
)

type Dmp struct {
	Sex       uint32 // 1, 性别 3, 2
	Byear     uint32 // 2, 出生年份 70, 7=>8
	Bmonth    uint32 // 3, 出生月份 13, 4
	Horoscope uint32 // 4, 生肖 13, 4
	Zodiac    uint32 // 5, 星座 13, 4
	Vip       uint32 // 81, 是否高消费人群 3, 2
	Browser   uint32 // 98, 浏览器 2, 1
	Gps       uint32 // 88, 交通导航 2, 1

	Bplace uint32 // 6, 出生-安徽省 343, 16
	Living uint32 // 40, 生活-安徽省 343, 16

	Brand  uint32 // 74, 手机品牌 48, 6=>7
	Screen uint32 // 75, 屏幕尺寸 6, 3=>4
	Time   uint32 // 76, 手机上市时间 30, 5
	Price  uint32 // 77, 手机价格 9, 4
	Plan   uint32 // 78, 消费价格 8, 3=>4
	Group  uint32 // 79, 人群 7, 3
	Hold   uint32 // 80, 用户等级 5, 3

	Games    []uint32 // 82, APP游戏 18, 32
	Shops    []uint32 // 83, 购物导购 12, 32
	Finances []uint32 // 86, 金融理财 11, 32
	Grocerys []uint32 // 92, 生活服务 17, 32
	Medias   []uint32 // 97, 影音娱乐 14, 32

	Healths []uint32 // 84, 健康养生 7, 16
	Learns  []uint32 // 85, 教育培训 7, 16

	Travels []uint32 // 87, 旅游出行 9, 16
	Socials []uint32 // 96, 社交聊天 8,  16

	Cars  []uint32 // 89, 养车用车 7, 16
	Foods []uint32 // 90, 美食佳饮 3, 16

	Photos []uint32 // 91, 拍照摄影 6, 16
	Works  []uint32 // 93, 商务效率 10, 16

	Books   []uint32 // 94, 图书阅读 5,  16
	Reports []uint32 // 95, 新闻资讯 6,  16

	Others []uint32 // 99, 其它 3,
}

func pushSingle(bs *[]uint8, name string, value uint32) {
	if value != 0 {
		*bs = append(*bs, uint8(pzutil.AttrValue[name]), uint8(value))
	}
}

func pushSlice(bs *[]uint8, name string, values []uint32) {
	if values != nil && len(values) > 0 {
		for _, value := range values {
			pushSingle(bs, name, value)
		}
	}
}

func (self *Dmp) PackSimple() ([]byte, error) {
	bs := make([]uint8, 0)
	pushSingle(&bs, "Sex", self.Sex)
	pushSingle(&bs, "Byear", self.Byear)
	pushSingle(&bs, "Bmonth", self.Bmonth)
	pushSingle(&bs, "Horoscope", self.Horoscope)
	pushSingle(&bs, "Zodiac", self.Zodiac)
	pushSingle(&bs, "Vip", self.Vip)
	pushSingle(&bs, "Browser", self.Browser)
	pushSingle(&bs, "Gps", self.Gps)

	if self.Bplace != 0 {
		item := Type06_hex2name[self.Bplace]
		i, err := strconv.ParseInt("0x"+item[0:2], 0, 32)
		if err != nil {
			return nil, err
		}
		j, err := strconv.ParseInt("0x"+item[2:4], 0, 32)
		if err != nil {
			return nil, err
		}
		bs = append(bs, uint8(i), uint8(j))
	}
	if self.Living != 0 {
		item := Type06_hex2name[self.Living]
		i, err := strconv.ParseInt("0x"+item[0:2], 0, 32)
		if err != nil {
			return nil, err
		}
		j, err := strconv.ParseInt("0x"+item[2:4], 0, 32)
		if err != nil {
			return nil, err
		}
		bs = append(bs, uint8(i+34), uint8(j))
	}

	pushSingle(&bs, "Brand", self.Brand)
	pushSingle(&bs, "Screen", self.Screen)
	pushSingle(&bs, "Time", self.Time)
	pushSingle(&bs, "Price", self.Price)
	pushSingle(&bs, "Plan", self.Plan)
	pushSingle(&bs, "Group", self.Group)
	pushSingle(&bs, "Hold", self.Hold)

	pushSlice(&bs, "Game", self.Games)
	pushSlice(&bs, "Shop", self.Shops)
	pushSlice(&bs, "Finance", self.Finances)
	pushSlice(&bs, "Grocery", self.Grocerys)
	pushSlice(&bs, "Media", self.Medias)
	pushSlice(&bs, "Health", self.Healths)
	pushSlice(&bs, "Learn", self.Learns)
	pushSlice(&bs, "Travel", self.Travels)
	pushSlice(&bs, "Social", self.Socials)
	pushSlice(&bs, "Car", self.Cars)
	pushSlice(&bs, "Food", self.Foods)
	pushSlice(&bs, "Photo", self.Photos)
	pushSlice(&bs, "Work", self.Works)
	pushSlice(&bs, "Book", self.Books)
	pushSlice(&bs, "Report", self.Reports)
	pushSlice(&bs, "Other", self.Others)

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, bs)
	return buf.Bytes(), err
}

func UnpackDmpSimple(data []byte) (*Dmp, error) {
	bs := make([]uint8, len(data))
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, bs)
	if err != nil {
		return nil, err
	}

	dmp := new(Dmp)
	for n := 0; n < len(bs)/2; n++ {
		i := uint32(bs[n*2])
		j := uint32(bs[n*2+1])
		switch i {
		case pzutil.AttrValue["Sex"]:
			dmp.Sex = j
		case pzutil.AttrValue["Byear"]:
			dmp.Byear = j
		case pzutil.AttrValue["Bmonth"]:
			dmp.Bmonth = j
		case pzutil.AttrValue["Horoscope"]:
			dmp.Horoscope = j
		case pzutil.AttrValue["Zodiac"]:
			dmp.Zodiac = j
		case pzutil.AttrValue["Vip"]:
			dmp.Vip = j
		case pzutil.AttrValue["Gps"]:
			dmp.Gps = j
		case pzutil.AttrValue["Browser"]:
			dmp.Browser = j
		case 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39:
			dmp.Bplace = Type06_hex2value[fmt.Sprintf("%02x%02x", i, j)]
		case 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73:
			dmp.Living = Type06_hex2value[fmt.Sprintf("%02x%02x", i-34, j)]
		case pzutil.AttrValue["Brand"]:
			dmp.Brand = j
		case pzutil.AttrValue["Screen"]:
			dmp.Screen = j
		case pzutil.AttrValue["Time"]:
			dmp.Time = j
		case pzutil.AttrValue["Price"]:
			dmp.Price = j
		case pzutil.AttrValue["Plan"]:
			dmp.Plan = j
		case pzutil.AttrValue["Group"]:
			dmp.Group = j
		case pzutil.AttrValue["Hold"]:
			dmp.Hold = j
		case pzutil.AttrValue["Game"]:
			dmp.Games = append(dmp.Games, j)
		case pzutil.AttrValue["Shop"]:
			dmp.Shops = append(dmp.Shops, j)
		case pzutil.AttrValue["Finance"]:
			dmp.Finances = append(dmp.Finances, j)
		case pzutil.AttrValue["Grocery"]:
			dmp.Grocerys = append(dmp.Grocerys, j)
		case pzutil.AttrValue["Media"]:
			dmp.Medias = append(dmp.Medias, j)
		case pzutil.AttrValue["Health"]:
			dmp.Healths = append(dmp.Healths, j)
		case pzutil.AttrValue["Learn"]:
			dmp.Learns = append(dmp.Learns, j)
		case pzutil.AttrValue["Travel"]:
			dmp.Travels = append(dmp.Travels, j)
		case pzutil.AttrValue["Social"]:
			dmp.Socials = append(dmp.Socials, j)
		case pzutil.AttrValue["Car"]:
			dmp.Cars = append(dmp.Cars, j)
		case pzutil.AttrValue["Food"]:
			dmp.Foods = append(dmp.Foods, j)
		case pzutil.AttrValue["Photo"]:
			dmp.Photos = append(dmp.Photos, j)
		case pzutil.AttrValue["Work"]:
			dmp.Works = append(dmp.Works, j)
		case pzutil.AttrValue["Book"]:
			dmp.Books = append(dmp.Books, j)
		case pzutil.AttrValue["Report"]:
			dmp.Reports = append(dmp.Reports, j)
		case pzutil.AttrValue["Other"]:
			dmp.Others = append(dmp.Others, j)
		default:
		}
	}

	return dmp, nil
}

func (self *Dmp) Pack() ([]byte, error) {
	if self.Sex >= 4 {
		self.Sex = 0
	}
	if self.Byear >= 256 {
		self.Byear = 0
	}
	if self.Bmonth >= 13 {
		self.Bmonth = 0
	}
	if self.Horoscope >= 13 {
		self.Horoscope = 0
	}
	if self.Zodiac >= 13 {
		self.Zodiac = 0
	}
	if self.Vip >= 4 {
		self.Vip = 0
	}
	if self.Browser >= 2 {
		self.Browser = 0
	}
	if self.Gps >= 2 {
		self.Gps = 0
	}
	d1 := uint32(self.Sex)<<0 + uint32(self.Byear)<<2 + uint32(self.Bmonth)<<10 + uint32(self.Horoscope)<<14 + uint32(self.Zodiac)<<18 + uint32(self.Vip)<<22 + uint32(self.Browser)<<24 + uint32(self.Gps)<<25

	if self.Bplace >= 65536 {
		self.Bplace = 0
	}
	if self.Living >= 65536 {
		self.Living = 0
	}
	d2 := uint32(self.Bplace) + uint32(self.Living)<<16

	if self.Brand >= 128 {
		self.Brand = 0
	}
	if self.Screen >= 16 {
		self.Screen = 0
	}
	if self.Time >= 32 {
		self.Time = 0
	}
	if self.Price >= 16 {
		self.Price = 0
	}
	if self.Plan >= 16 {
		self.Plan = 0
	}
	if self.Group >= 8 {
		self.Group = 0
	}
	if self.Hold >= 8 {
		self.Hold = 0
	}
	d3 := uint32(self.Brand) + uint32(self.Screen)<<7 + uint32(self.Time)<<11 + uint32(self.Price)<<16 + uint32(self.Plan)<<20 + uint32(self.Group)<<24 + uint32(self.Hold)<<27

	games := uint32(0)
	shops := uint32(0)
	finances := uint32(0)
	grocerys := uint32(0)
	medias := uint32(0)
	others := uint32(0)
	for _, item := range self.Games {
		games += (1 << item)
	}
	for _, item := range self.Shops {
		shops += (1 << item)
	}
	for _, item := range self.Finances {
		finances += (1 << item)
	}
	for _, item := range self.Grocerys {
		grocerys += (1 << item)
	}
	for _, item := range self.Medias {
		medias += (1 << item)
	}
	for _, item := range self.Others {
		others += (1 << item)
	}

	health_learn := uint32(0)
	for _, item := range self.Healths {
		health_learn += (1 << item)
	}
	for _, item := range self.Learns {
		health_learn += (1 << (16 + item))
	}
	travel_social := uint32(0)
	for _, item := range self.Travels {
		travel_social += (1 << item)
	}
	for _, item := range self.Socials {
		travel_social += (1 << (16 + item))
	}
	car_food := uint32(0)
	for _, item := range self.Cars {
		car_food += (1 << item)
	}
	for _, item := range self.Foods {
		car_food += (1 << (16 + item))
	}
	photo_work := uint32(0)
	for _, item := range self.Photos {
		photo_work += (1 << item)
	}
	for _, item := range self.Works {
		photo_work += (1 << (16 + item))
	}
	book_report := uint32(0)
	for _, item := range self.Books {
		book_report += (1 << item)
	}
	for _, item := range self.Reports {
		book_report += (1 << (16 + item))
	}

	data := []uint32{d1, d2, d3, games, shops, finances, grocerys, medias, health_learn, travel_social, car_food, photo_work, book_report, others}

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)
	return buf.Bytes(), err
}

func UnpackDmp(data []byte) (*Dmp, error) {
	v := make([]uint32, 14)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, v)
	if err != nil {
		return nil, err
	}

	dmp := new(Dmp)

	dmp.Sex = (v[0] >> 0) & 3
	dmp.Byear = (v[0] >> 2) & 255
	dmp.Bmonth = (v[0] >> 10) & 15
	dmp.Horoscope = (v[0] >> 14) & 15
	dmp.Zodiac = (v[0] >> 18) & 15
	dmp.Vip = (v[0] >> 22) & 3
	dmp.Browser = (v[0] >> 24) & 1
	dmp.Gps = (v[0] >> 25) & 1

	dmp.Bplace = (v[1] >> 0) & 65535
	dmp.Living = (v[1] >> 16) & 65535

	dmp.Brand = (v[2] >> 0) & 127
	dmp.Screen = (v[2] >> 7) & 15
	dmp.Time = (v[2] >> 11) & 31
	dmp.Price = (v[2] >> 16) & 15
	dmp.Plan = (v[2] >> 20) & 15
	dmp.Group = (v[2] >> 24) & 7
	dmp.Hold = (v[2] >> 27) & 7

	if v[3] > 0 {
		items := make([]uint32, 0)
		for i := uint32(0); i < 32; i++ {
			if (v[3]>>i)&1 == 1 {
				items = append(items, i)
			}
		}
		dmp.Games = items
	}
	if v[4] > 0 {
		items := make([]uint32, 0)
		for i := uint32(0); i < 32; i++ {
			if (v[4]>>i)&1 == 1 {
				items = append(items, i)
			}
		}
		dmp.Shops = items
	}
	if v[5] > 0 {
		items := make([]uint32, 0)
		for i := uint32(0); i < 32; i++ {
			if (v[5]>>i)&1 == 1 {
				items = append(items, i)
			}
		}
		dmp.Finances = items
	}
	if v[6] > 0 {
		items := make([]uint32, 0)
		for i := uint32(0); i < 32; i++ {
			if (v[6]>>i)&1 == 1 {
				items = append(items, i)
			}
		}
		dmp.Grocerys = items
	}
	if v[7] > 0 {
		items := make([]uint32, 0)
		for i := uint32(0); i < 32; i++ {
			if (v[7]>>i)&1 == 1 {
				items = append(items, i)
			}
		}
		dmp.Medias = items
	}

	if v[8] > 0 {
		items := make([]uint32, 0)
		item1 := make([]uint32, 0)
		for i := uint32(0); i < 16; i++ {
			if (v[8]>>i)&1 == 1 {
				items = append(items, i)
			}
			if (v[8]>>(16+i))&1 == 1 {
				item1 = append(item1, i)
			}
		}
		dmp.Healths = items
		dmp.Learns = item1
	}

	if v[9] > 0 {
		items := make([]uint32, 0)
		item1 := make([]uint32, 0)
		for i := uint32(0); i < 16; i++ {
			if (v[9]>>i)&1 == 1 {
				items = append(items, i)
			}
			if (v[9]>>(16+i))&1 == 1 {
				item1 = append(item1, i)
			}
		}
		dmp.Travels = items
		dmp.Socials = item1
	}

	if v[10] > 0 {
		items := make([]uint32, 0)
		item1 := make([]uint32, 0)
		for i := uint32(0); i < 16; i++ {
			if (v[10]>>i)&1 == 1 {
				items = append(items, i)
			}
			if (v[10]>>(16+i))&1 == 1 {
				item1 = append(item1, i)
			}
		}
		dmp.Cars = items
		dmp.Foods = item1
	}

	if v[11] > 0 {
		items := make([]uint32, 0)
		item1 := make([]uint32, 0)
		for i := uint32(0); i < 16; i++ {
			if (v[11]>>i)&1 == 1 {
				items = append(items, i)
			}
			if (v[11]>>(16+i))&1 == 1 {
				item1 = append(item1, i)
			}
		}
		dmp.Photos = items
		dmp.Works = item1
	}

	if v[12] > 0 {
		items := make([]uint32, 0)
		item1 := make([]uint32, 0)
		for i := uint32(0); i < 16; i++ {
			if (v[12]>>i)&1 == 1 {
				items = append(items, i)
			}
			if (v[12]>>(16+i))&1 == 1 {
				item1 = append(item1, i)
			}
		}
		dmp.Books = items
		dmp.Reports = item1
	}

	if v[13] > 0 {
		items := make([]uint32, 0)
		for i := uint32(0); i < 32; i++ {
			if (v[13]>>i)&1 == 1 {
				items = append(items, i)
			}
		}
		dmp.Others = items
	}

	return dmp, nil
}

func DmpNames() map[string]map[uint32]string {
	return map[string]map[uint32]string{
		"Sex":       Type01_name,
		"Byear":     Type02_name,
		"Bmonth":    Type03_name,
		"Horoscope": Type04_name,
		"Zodiac":    Type05_name,
		"Bplace":    Type06_name,
		"Living":    Type06_name,
		"Brand":     Type4a_name,
		"Screen":    Type4b_name,
		"Time":      Type4c_name,
		"Price":     Type4d_name,
		"Plan":      Type4e_name,
		"Group":     Type4f_name,
		"Hold":      Type50_name,
		"Vip":       Type51_name,
		"Game":      Type52_name,
		"Shop":      Type53_name,
		"Health":    Type54_name,
		"Learn":     Type55_name,
		"Finance":   Type56_name,
		"Travel":    Type57_name,
		"Gps":       Type58_name,
		"Car":       Type59_name,
		"Food":      Type5a_name,
		"Photo":     Type5b_name,
		"Grocery":   Type5c_name,
		"Works":     Type5d_name,
		"Books":     Type5e_name,
		"Reports":   Type5f_name,
		"Social":    Type60_name,
		"Media":     Type61_name,
		"Browser":   Type62_name,
		"Other":     Type63_name}
}

func GetDmpParameters(other map[string]interface{}) {
	other["Sex"] = Type01_name
	other["Byear"] = Type02_name
	other["Bmonth"] = Type03_name
	other["Horoscope"] = Type04_name
	other["Zodiac"] = Type05_name
	other["Bplace"] = Type06_name
	other["Living"] = Type06_name

	other["Brand"] = Type4a_name
	other["Screen"] = Type4b_name
	other["Time"] = Type4c_name
	other["Price"] = Type4d_name
	other["Plan"] = Type4e_name
	other["Group"] = Type4f_name
	other["Hold"] = Type50_name
	other["Vip"] = Type51_name
	other["Game"] = Type52_name
	other["Shop"] = Type53_name
	other["Health"] = Type54_name
	other["Learn"] = Type55_name
	other["Finance"] = Type56_name
	other["Travel"] = Type57_name
	other["Gps"] = Type58_name
	other["Car"] = Type59_name
	other["Food"] = Type5a_name
	other["Photo"] = Type5b_name
	other["Grocery"] = Type5c_name
	other["Works"] = Type5d_name
	other["Books "] = Type5e_name
	other["Reports"] = Type5f_name
	other["Social"] = Type60_name
	other["Media"] = Type61_name
	other["Browser"] = Type62_name
	other["Other"] = Type63_name
}

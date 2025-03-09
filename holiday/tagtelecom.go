package holiday

import (
	"os"
	"bufio"
	"strconv"
	"strings"
)

// NewTagMapTelecom creates a telecom TagMap
// Default with xx00 means differently for Bplace and Living
// from other tags. In the former 2 cases, 00 means state, i.e.
// the top of ALL CITIES in that state.
// In all other cases (such as Sex, NByear, Brand etc.,), default with Val=0
// is for all other choices, which normally shouldn't be present --
// we should not allow advertiser to choose this.
func NewTagMapTelecom(filename string) (*TagMap, error) {
	if filename=="" { return nil, nil }
	var HELPS = map[string]string{
"性别":"Sex", "出生年份":"Byear", "出生月份":"Bmonth", "生肖":"Horoscope",
"星座":"Zodiac", "出生":"Bplace", "常住":"Living", "手机品牌":"Brand",
"屏幕尺寸":"Screen", "手机上市时间":"Time", "手机价格":"Price",
"消费价格":"Plan", "人群":"Group", "用户等级":"Hold", "是否高消费人群":"Vip",
"APP游戏":"Game", "购物导购":"Shop", "健康养生":"Health",
"教育培训":"Learn", "金融理财":"Finance", "旅游出行":"Travel",
"交通导航":"Gps", "养车用车":"Car", "美食佳饮":"Food", "拍照摄影":"Photo",
"生活服务":"Grocery", "商务效率":"Work", "图书阅读":"Book",
"新闻资讯":"Report", "社交聊天":"Social", "影音娱乐":"Media",
"浏览器":"Browser", "其它":"Other",
	}

	Marker := "00"

	f, err := os.Open(filename)
	if err != nil { return nil, err }
	defer f.Close()

	attrs := []uint32{6,40}

	TagRef := map[string]*Tag{}

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
		arrs := strings.Split(scanner.Text(), ",")
		ID := arrs[4]
		Name := arrs[3]
		Parent := arrs[1]

		k, err := strconv.ParseInt(arrs[0], 10, 32)
		if err != nil { return nil, err }
		val, err := strconv.ParseInt(arrs[2], 10, 32)
		if err != nil { return nil, err }
		Val := uint32(val)

		if k>=6 && k<=39 {
			if ID[2:] == Marker {
				Name = Parent
				Parent = "Bplace"
				Val = uint32(k)
			} else {
				Parent = ID[:2] + Marker
				decoded, err := Hex2Byte(arrs[4]+"0000")
				if err != nil { return nil, err }
				Val = Byte32Uint(decoded) // LittleEndian
			}
		} else if k>=40 && k<=73 {
			if ID[2:] == Marker {
				Name = Parent
				Parent = "Living"
				Val = uint32(k)
			} else {
				Parent = ID[:2] + Marker
				decoded, err := Hex2Byte(arrs[4]+"0000")
				if err != nil { return nil, err }
				Val = Byte32Uint(decoded)
			}
		} else {
			Parent = HELPS[Parent]
			if ID[2:] == Marker {
				attrs = append(attrs, uint32(k))
			}
		}
		TagRef[ID] = &Tag{ID, Name, Parent, Val}
    }
    if err = scanner.Err(); err != nil { return nil, err }
	return &TagMap{Attrs{attrs}, TagRef}, nil
}

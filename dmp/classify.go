package dmp

var HELPS = map[string]string{
	"性别":      "Sex",
	"出生年份":    "Byear",
	"出生月份":    "Bmonth",
	"生肖":      "Horoscope",
	"星座":      "Zodiac",
	"出生-安徽省":  "Bplace",
	"手机品牌":    "Brand",
	"屏幕尺寸":    "Screen",
	"手机上市时间":  "Time",
	"手机价格":    "Price",
	"消费价格":    "Plan",
	"人群":      "Group",
	"用户等级":    "Hold",
	"是否高消费人群": "Vip",
	"APP游戏":   "Games",
	"购物导购":    "Shops",
	"健康养生":    "Healths",
	"教育培训":    "Learns",
	"金融理财":    "Finances",
	"旅游出行":    "Travels",
	"交通导航":    "Gpss",
	"养车用车":    "Cars",
	"美食佳饮":    "Foods",
	"拍照摄影":    "Photos",
	"生活服务":    "Grocerys",
	"商务效率":    "Works",
	"图书阅读":    "Books",
	"新闻资讯":    "Reports",
	"社交聊天":    "Socials",
	"影音娱乐":    "Medias",
	"浏览器":     "Browsers",
	"其它":      "Others",
}

/*
func main() {
	f, err := os.Open("dx.data")
	if err != nil { panic(err) }
	defer f.Close()

	type06 := []string{
"var Type06_hex2name  = map[uint32]string {",
}
	typez6 := []string{
"var Type06_hex2value = map[string]uint32 {",
}
	cats := make(map[int][][]string)
	names := []string{
"package dmp\n",
"type Dmp struct {",
}
	total := []int{0}

	i:=0
	j:=0
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
		arrs := strings.Split(scanner.Text(), ",")
		num, err := strconv.Atoi(arrs[0])
		if err != nil { panic(err) }
		//if num>=82 {
		//	i=82
		if num >=40 && num <= 73 {
			continue
		} else if num >=6 && num <=39 {
			i=6
		} else {
			i=num
		}
		tname := "Type"+fmt.Sprintf("%02x",i)

		_, ok := cats[i]
		if ok {
			if arrs[2]=="0" { continue }
		} else {
			if i>81 {
				names = append(names, "\t"+HELPS[arrs[1]]+"\t[]uint32\t// "+arrs[0]+", "+arrs[1])
			} else {
				names = append(names, "\t"+HELPS[arrs[1]]+"\tuint32\t// "+arrs[0]+", "+arrs[1])
			}
			total = append(total, j)
			j=0
			cats[i] = [][]string{
{"package dmp\n", "type "+tname+" uint32", "const ("},
{"var "+tname+"_name = map[uint32]string{"},
{"var "+tname+"_value = map[string]uint32{"},
}
		}

		cats[i][0] = append(cats[i][0], fmt.Sprintf("\t%s_%s = %d", tname, arrs[4][0:4], j))
		cats[i][1] = append(cats[i][1], fmt.Sprintf("\t%d:\"%s_%s\",", j, arrs[1], arrs[3]))
		cats[i][2] = append(cats[i][2], fmt.Sprintf("\t\"%s_%s\":%d,", arrs[1], arrs[3], j))
		if i==6 {
			type06 = append(type06, fmt.Sprintf("\t%d:\"%s\",",j,arrs[4][0:4]))
			typez6 = append(typez6, fmt.Sprintf("\t\"%s\":%d,",arrs[4][0:4],j))
		}

		j++
	}
	total = append(total, j)
    if err := scanner.Err(); err != nil { panic(err) }

	for num, cat := range cats {
		out, _ := os.Create(fmt.Sprintf("cat%02x%s", num, ".go"))
		out.WriteString("package dmp\n\n")
	//	for _, item := range cat[0] {
	//		out.WriteString(item+"\n")
	//	}
	//	out.WriteString(")\n\n")
		for _, item := range cat[1] {
			out.WriteString(item+"\n")
		}
		out.WriteString("}\n\n")
		for _, item := range cat[2] {
		out.WriteString(item+"\n")
		}
		out.WriteString("}\n")
		if num==6 {
			out.WriteString("\n")
			for _, item := range type06 {
				out.WriteString(item+"\n")
			}
			out.WriteString("}\n\n")
			for _, item := range typez6 {
				out.WriteString(item+"\n")
			}
			out.WriteString("}\n")
		}
		out.Close()
	}

	for i, name := range names {
		fmt.Printf("%s %d, %d\n", name, total[i], int(math.Log2(float64(total[i]-1))) + 1)
	}
	fmt.Println("}")
}
*/

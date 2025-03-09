package holiday

import (
	"testing"
)

func TestTagMap(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil { t.Fatal(err) }
	telecomMap, err := NewTagMapTelecom(c.Telecom)
	if err != nil { t.Fatal(err) }
	for i:=uint32(1); i<=99; i++ {
		if (i>6 && i<40) || (i>40 && i<74) {
			if IndexUint32(telecomMap.GetAttrIDs(), i) >= 0 {
				t.Errorf("%d", i)
			}
		} else {
			if IndexUint32(telecomMap.GetAttrIDs(), i) < 0 {
				t.Errorf("%d", i)
			}
		}
	}

	n := 0
	for k, tag := range telecomMap.TagRef {
		if tag.Parent == "Bmonth" {
			n++
			if (k=="0300" && tag.Val!=0) ||
				(k=="0301" && tag.Val!=1) ||
				(k=="0308" && tag.Val!=8) ||
				(k=="030c" && tag.Val!=12)  {
					t.Errorf("%v=>%v", k, tag)
			}
		}
	}
	if n!=13 {
		t.Errorf("13 month records expected but we got: %v", n)
	}

	n = 0
	for k, tag := range telecomMap.TagRef {
		if tag.Parent == "Bplace" {
			n++
			if (k=="0600" && tag.Val!=6) ||
				(k=="0700" && tag.Val!=7) ||
				(k=="2700" && tag.Val!=39) {
					t.Errorf("%v=>%v", k, tag)
			}
		}
	}
	if n!=34 {
		t.Errorf("34 stated but we got: %v", n)
	}

	n = 0
	for k, tag := range telecomMap.TagRef {
		if tag.Parent == "2800" {
			n++
			if (k=="2801" && tag.Val!=296) ||
				(k=="280c" && tag.Val!=3112) ||
				(k=="2811" && tag.Val!=4392) {
					t.Errorf("%v=>%v", k, tag)
			}
		}
	}
	if n!=17 {
		t.Errorf("17 cities in 常住-安徽省 but we got: %v", n)
	}
}

func TestTags(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil { t.Fatal(err) }
	tagMap, err := NewTagMapTelecom(c.Telecom)
	if err != nil { t.Fatal(err) }
	str := "xxxx,yyyy,zzzz"
	tags_obj := tagMap.GetTagsFromCodes(str)
	if tags_obj != nil {
		t.Errorf("%s should make no tags object", str)
	}

	val := tagMap.GetAttrID("Xxx")
	if val != 0 {
		t.Errorf("val: %v", val)
	}
	val = tagMap.GetAttrID("Sex")
	if val != 1 {
		t.Errorf("val: %v", val)
	}

	str = "xxxx,yyyy,0102,0101,0201"
	tags_obj = tagMap.GetTagsFromCodes(str)
	n := 0
	for _, tags := range tags_obj.TagHashArray {
		n += len(tags)
	}
	if n!= 3 {
		t.Errorf("3 expected but %d=>%v", n, tags_obj.TagHashArray)
	}

	str = "xxxx,yyyy,0102,0100,0201"
	tags_obj = tagMap.GetTagsFromCodes(str)
	n = 0
	for _, tags := range tags_obj.TagHashArray {
		n += len(tags)
	}
	if n!= 3 {
		t.Errorf("3 expected but %d=>%v", n, tags_obj.TagHashArray)
	}
	if tags_obj.TagHashArray[2][0] != 1 {
		for attrID, tags := range tags_obj.TagHashArray {
			for i, tag := range tags {
				t.Errorf("%d=>%d=>%v", attrID, i, tag)
			}
		}
	}

	str = "xxxx,yyyy,0102,0100,0201,0601,0611,0900,0901,0902"
	tags_obj = tagMap.GetTagsFromCodes(str)
	tags := tags_obj.TagHashArray[6] // Bplace
	for _, tag := range tags {
		if IndexUint32([]uint32{6,262,4358,9,265,521},tag) < 0 {
			for i, tag := range tags {
				t.Errorf("%d=>%v", i, tag)
			}
		}
	}
}

func TestAofeiTagMap(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil { t.Fatal(err) }
	aofeiMap, err := NewTagMapAofei(c.Aofei)
	if err != nil { t.Fatal(err) }
	for i:=uint32(1401); i<=1416; i++ {
		if IndexUint32(aofeiMap.GetAttrIDs(), i) < 0 {
			t.Errorf("%d", i)
		}
	}
// &{001001 时段定向 0000 1401}
// &{001002 星期定向 0000 1402}
// &{001003 省市区定向 0000 1403}
// &{001004 移动系统定向 0000 1404}
// &{001005 网络类型定向 0000 1405}
// &{001006 运营商定向 0000 1406}
// &{001007 行业分类定向 0000 1407}
// &{001008 PC系统定向 0000 1408}
// &{001009 省份定向 0000 1409}
// &{001010 城市定向 0000 1410}
// &{001011 区域定向 0000 1411}
// &{001012 媒体定向 0000 1412}
// &{001013 场景定向 0000 1413}
// &{001014 手机品牌定向 0000 1414}
// &{001015 手机价格定向 0000 1415}
// &{001016 APP分类定向 0000 1416}

	n := 0
	for k, tag := range aofeiMap.TagRef {
		if tag.Parent == "001002" {
			n++
			if (k=="001002001" && tag.Val!=1) ||
				(k=="001002002" && tag.Val!=2) ||
				(k=="001002007" && tag.Val!=7) {
					t.Errorf("%v=>%v", k, tag)
			}
		}
	}
	if n!=7 {
		t.Errorf("7 month records expected but we got: %v", n)
	}

	n = 0
	for k, tag := range aofeiMap.TagRef {
		if tag.Parent == "001013" {
			n++
			if (k=="01" && tag.Val!=24) ||
				(k=="02" && tag.Val!=38) ||
				(k=="07" && tag.Val!=1392) {
					t.Errorf("%v=>%v", k, tag)
			}
		}
	}
	if n!=7 {
		t.Errorf("7 scenes but we got: %v", n)
	}

	n = 0
	for k, tag := range aofeiMap.TagRef {
		if tag.Parent == "0702" {
			n++
			if (k=="070201" && tag.Val!=1398) ||
				(k=="070202" && tag.Val!=1399) ||
				(k=="070218" && tag.Val!=1415) {
					t.Errorf("%v=>%v", k, tag)
			}
		}
	}
	if n!=18 {
		t.Errorf("18 车场类型 but we got: %v", n)
	}
}

func TestAofeiTags(t *testing.T) {
	c, err := NewConfig("sample.json")
	if err != nil { t.Fatal(err) }
	tagMap, err := NewTagMapAofei(c.Aofei)
	if err != nil { t.Fatal(err) }

	str := "xxxx,yyyy,zzzz"
	tags_obj := tagMap.GetTagsFromCodes(str)
	if tags_obj != nil {
		t.Errorf("%s should make no tags object", str)
	}

	val := tagMap.GetAttrID("Xxx")
	if val != 0 {
		t.Errorf("val: %v", val)
	}
	val = tagMap.GetAttrID("001013")
	if val != 1413 {
		t.Errorf("val: %v", val)
	}

	str = "xxxx,yyyy,06,0701,001001001"
	tags_obj = tagMap.GetTagsFromCodes(str)
	n := 0
	for _, tags := range tags_obj.TagHashArray {
		n += len(tags)
	}
	if n!= 4 {
		t.Errorf("4 expected but %d=>%v", n, tags_obj.TagHashArray)
	}

	if IndexUint32([]uint32{293,1392,1393},tags_obj.TagHashArray[1413][0])<0 ||
		IndexUint32([]uint32{293,1392,1393},tags_obj.TagHashArray[1413][1])<0 ||
		IndexUint32([]uint32{293,1392,1393},tags_obj.TagHashArray[1413][2])<0 {
		for attrID, tags := range tags_obj.TagHashArray {
			for i, tag := range tags {
// 1413=>0=>&{06 住宿服务 001013 293}
// 1413=>1=>&{0701 车辆档次 07 1393}
// 1413=>2=>&{07 ETCP停车 001013 1392}
// 1401=>0=>&{001001001 00:00-01:00 001001 0}
				t.Errorf("%d=>%d=>%v", attrID, i, tag)
			}
		}
	}

	str = "xxxx,yyyy,06,0701,001001001,130200,130102,140105"
	tags_obj = tagMap.GetTagsFromCodes(str)
	tags := tags_obj.TagHashArray[1403] // 地理
	for _, tag := range tags {
		if IndexUint32([]uint32{130200,130000,130102,130100,140105,140100,140000},tag) < 0 {
			for i, tag := range tags {
// &{130200 唐山市 130000 130200}
// &{130000 河北省 001003 130000}
// &{130102 长安区 130100 130102}
// &{130100 石家庄市 130000 130100}
// &{140105 小店区 140100 140105}
// &{140100 太原市 140000 140100}
// &{140000 山西省 001003 140000}
				t.Errorf("%d=>%v", i, tag)
			}
		}
	}
}

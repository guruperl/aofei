package summer

import (
	"testing"
)

func TestDictionary(t *testing.T) {
	if Dictionary("Black") != "黑名单" ||
		Dictionary("White") != "白名单" ||
		Dictionary("Inherit") != "默认" {
		t.Errorf("%v", Dictionary("Black"))
	}
	if Dictionary("Bbbbb") != "Bbbbb" {
		t.Errorf("%v", Dictionary("Bbbbb"))
	}

	var arr = []string{"Black", "White", "Inherit", "Bbbbb"}
	outarr := Translate(arr)
	out := outarr.([]string)
	if out[0] != "黑名单" ||
		out[1] != "白名单" ||
		out[2] != "默认" ||
		out[3] != "Bbbbb" {
		t.Errorf("%v", out)
	}

	var hash = map[string]string{"a": "Black", "b": "White", "c": "Inherit", "d": "Bbbbb"}
	outhash := Translate(hash)
	o := outhash.(map[string]string)
	if o["a"] != "黑名单" ||
		o["b"] != "白名单" ||
		o["c"] != "默认" ||
		o["d"] != "Bbbbb" {
		t.Errorf("%v", o)
	}

	var comp = map[string]map[uint32]string{
		"a": {1: "Black", 2: "White", 3: "Inherit", 4: "Bbbbb"},
		"b": {5: "Content", 6: "Visual", 7: "Ccccc"},
	}
	comphash := Translate(comp)
	c := comphash.(map[string]map[uint32]string)
	if c["a"][1] != "黑名单" ||
		c["a"][2] != "白名单" ||
		c["a"][3] != "默认" ||
		c["a"][4] != "Bbbbb" ||
		c["b"][5] != "内容" ||
		c["b"][6] != "视觉" ||
		c["b"][7] != "Ccccc" {
		t.Errorf("%v", c)
	}

	var items = []map[string]interface{}{
		{"name": "Black", "qty": 1},
		{"name": "White", "qty": 2},
		{"name": "Inherit", "qty": 3},
		{"name": "Bbbbb", "qty": 4}}
	TranslateOne(items, "name", "chinese")
	if items[0]["chinese"].(string) != "黑名单" ||
		items[1]["chinese"].(string) != "白名单" ||
		items[2]["chinese"].(string) != "默认" ||
		items[3]["chinese"].(string) != "Bbbbb" {
		t.Errorf("%v", items)
	}

	var item = map[string]interface{}{"name": "Black", "qty": 1}
	TranslateOne(item, "name", "chinese")
	if item["chinese"].(string) != "黑名单" {
		t.Errorf("%v", item)
	}
	item = map[string]interface{}{"name": "Bbbbb", "qty": 1}
	TranslateOne(item, "name", "chinese")
	if item["chinese"].(string) != "Bbbbb" {
		t.Errorf("%v", item)
	}
}

package match

import (
	"database/sql"
	"testing"

	"github.com/genelet/winter/pzutil"
	_ "github.com/go-sql-driver/mysql"
)

func getItemSample() *Item {
	creatives := []*Creative{{1, 0.9, "js1"}, {2, 0.1, "js1"}}
	return &Item{ItemID: 222, AdvID: 333, SizeID: 444, Creatives: creatives}
}

func TestItem(t *testing.T) {
	item := getItemSample()
	n1 := 0
	n2 := 0
	for i := 0; i < 100; i++ {
		creative := item.SelectCreative()
		if creative.CreativeID == 1 {
			n1++
		} else {
			n2++
		}
	}
	if !(n1 > 80 && n2 < 20) {
		t.Errorf("%d", n1)
		t.Errorf("%d", n2)
	}

	bs, err := item.Pack()
	if err != nil {
		t.Fatal(err)
	}
	newItem, err := UnpackItem(bs)
	if err != nil {
		t.Fatal(err)
	}
	c1 := newItem.Creatives[0]
	if c1.CreativeID != 1 || c1.Weight != 0.9 || c1.Content != "js1" {
		t.Errorf("%v", c1)
	}

	c, err := pzutil.NewConfig("../conf/gotest.conf")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	item, err = DBGetItem(db, uint32(125))
	if err != nil {
		t.Fatal(err)
	}
	creative := item.Creatives[0]
	if creative.Weight != float32(0.5) {
		t.Errorf("%v", creative)
	}
}

package summer

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/genelet/winter/genelet"
	_ "github.com/go-sql-driver/mysql"
)

func TestInitLedger(t *testing.T) {

	slot_idend := 2
	item_idend := 2
	slot_idstart := 1
	item_idstart := 1
	upperImp := 200
	upperCli := 5
	upperSpe := float32(2.0)

	//day := "2018-05-06"
	dayTime := time.Now().AddDate(0, 0, -1).String()
	day := dayTime[0:10]
	//rand.Seed(time.Now().UTC().UnixNano())

	config := "../conf/gotest.json"
	c := genelet.NewConfig(config)
	db, err := sql.Open(c.Db[0], c.Db[1])
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sthSlot, err := db.Prepare(
		`SELECT s.site_id, t.pub_id 
FROM pub_slot s
INNER JOIN pub_site t USING (site_id)
WHERE s.slot_id=?`)
	if err != nil {
		panic(err)
	}
	defer sthSlot.Close()
	slots := make(map[int][]int)
	for slot_id := slot_idstart; slot_id <= slot_idend; slot_id++ {
		var site_id, pub_id int
		if err := sthSlot.QueryRow(slot_id).Scan(&site_id, &pub_id); err != nil {
			panic(err)
		}
		slots[slot_id] = []int{site_id, pub_id}
	}

	sthItem, err := db.Prepare(
		`SELECT i.campaign_id, c.adv_id 
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
WHERE i.item_id=?`)
	if err != nil {
		panic(err)
	}
	defer sthItem.Close()
	items := make(map[int][]int)
	for item_id := item_idstart; item_id <= item_idend; item_id++ {
		var campaign_id, adv_id int
		if err := sthItem.QueryRow(item_id).Scan(&campaign_id, &adv_id); err != nil {
			panic(err)
		}
		items[item_id] = []int{campaign_id, adv_id}
	}

	for hour := 0; hour < 24; hour++ {
		for minute := 0; minute < 60; minute += 5 {
			imps := make(map[int]map[int]int)
			clis := make(map[int]map[int]int)
			spes := make(map[int]map[int]float32)

			// simulations
			for slot_id := range slots {
				imps[slot_id] = make(map[int]int)
				clis[slot_id] = make(map[int]int)
				spes[slot_id] = make(map[int]float32)
				for item_id := range items {
					i := rand.Intn(upperImp)
					c := rand.Intn(upperCli)
					s := upperSpe * rand.Float32()
					imps[slot_id][item_id] = i
					clis[slot_id][item_id] = c
					spes[slot_id][item_id] = s
				}
			}

			myDay := fmt.Sprintf("%s %d:%d:0", day, hour, minute)
			err := InsertLedger(db, myDay, slots, items, imps, clis, spes)
			if err != nil {
				panic(err)
			}
		}
	}

	if err := InsertDaily(db, day); err != nil {
		panic(err)
	}
}

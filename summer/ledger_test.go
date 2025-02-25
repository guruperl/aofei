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

	slotIdend := 2
	itemIdend := 2
	slotIdstart := 1
	itemIdstart := 1
	upperImp := 200
	upperCli := 5
	upperSpe := float32(2.0)

	//day := "2018-05-06"
	dayTime := time.Now().AddDate(0, 0, -1).String()
	day := dayTime[0:10]
	//rand.Seed(time.Now().UTC().UnixNano())

	config := "../conf/gotest.json"
	c, err := genelet.NewConfig(config)
	if err != nil {
		panic(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
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
	for slotID := slotIdstart; slotID <= slotIdend; slotID++ {
		var siteID, pubID int
		if err := sthSlot.QueryRow(slotID).Scan(&siteID, &pubID); err != nil {
			panic(err)
		}
		slots[slotID] = []int{siteID, pubID}
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
	for itemID := itemIdstart; itemID <= itemIdend; itemID++ {
		var campaignID, advID int
		if err := sthItem.QueryRow(itemID).Scan(&campaignID, &advID); err != nil {
			panic(err)
		}
		items[itemID] = []int{campaignID, advID}
	}

	for hour := 0; hour < 24; hour++ {
		for minute := 0; minute < 60; minute += 5 {
			imps := make(map[int]map[int]int)
			clis := make(map[int]map[int]int)
			spes := make(map[int]map[int]float32)

			// simulations
			for slotID := range slots {
				imps[slotID] = make(map[int]int)
				clis[slotID] = make(map[int]int)
				spes[slotID] = make(map[int]float32)
				for itemID := range items {
					i := rand.Intn(upperImp)
					c := rand.Intn(upperCli)
					s := upperSpe * rand.Float32()
					imps[slotID][itemID] = i
					clis[slotID][itemID] = c
					spes[slotID][itemID] = s
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

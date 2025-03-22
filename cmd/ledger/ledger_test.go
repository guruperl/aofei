package main

import (
	"database/sql"
	"fmt"
	"genelet"
	"math/rand"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestInitLedger(t *testing.T) {

	slotID_end := 2
	itemID_end := 2
	slotID_start := 1
	itemID_start := 1
	upperImp := 200
	upperCli := 5
	upperSpe := float32(2.0)

	//day := "2018-05-06"
	day_time := time.Now().AddDate(0, 0, -1).String()
	day := day_time[0:10]
	//rand.Seed(time.Now().UTC().UnixNano())

	config := "../../conf/gotest.json"
	c := genelet.NewConfig(config)
	db, err := sql.Open(c.Db[0], c.Db[1])
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sth_slot, err := db.Prepare(
		`SELECT s.site_id, t.pub_id 
FROM pub_slot s
INNER JOIN pub_site t USING (site_id)
WHERE s.slot_id=?`)
	if err != nil {
		panic(err)
	}
	defer sth_slot.Close()
	slots := make(map[int][]int)
	for slotID := slotID_start; slotID <= slotID_end; slotID++ {
		var siteID, pubID int
		if err := sth_slot.QueryRow(slotID).Scan(&siteID, &pubID); err != nil {
			panic(err)
		}
		slots[slotID] = []int{siteID, pubID}
	}

	sth_item, err := db.Prepare(
		`SELECT i.campaign_id, c.adv_id 
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
WHERE i.item_id=?`)
	if err != nil {
		panic(err)
	}
	defer sth_item.Close()
	items := make(map[int][]int)
	for itemID := itemID_start; itemID <= itemID_end; itemID++ {
		var campaignID, advID int
		if err := sth_item.QueryRow(itemID).Scan(&campaignID, &advID); err != nil {
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
			for slotID, _ := range slots {
				imps[slotID] = make(map[int]int)
				clis[slotID] = make(map[int]int)
				spes[slotID] = make(map[int]float32)
				for itemID, _ := range items {
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

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/genelet/winter/genelet"
	_ "github.com/go-sql-driver/mysql"
)

func usage() {
	fmt.Println(`Usage: PROGRAM [-c config] [-s slotEnd] [-i itemEnd] day`)
}

func main() {
	var config = flag.String("c", "./summer.json", "Config File")
	var slotID_end = flag.Int("s", 143, "SlotID End")
	var itemID_end = flag.Int("i", 143, "ItemID End")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(0)
	}
	day := args[0]
	fmt.Printf("config is %s\n", *config)

	slotID_start := 1
	itemID_start := 1
	upperImp := 200
	upperCli := 5
	upperSpe := float32(2.0)

	rand.Seed(time.Now().UTC().UnixNano())

	c := genelet.NewConfig(*config)
	c.Db = []string{"mysql", "eightran_goto:12pass34@/gotest"}
	dbh, err := sql.Open(c.Db[0], c.Db[1])
	if err != nil {
		panic(err)
	}
	defer dbh.Close()

	sth_slot, err := dbh.Prepare(
		`SELECT s.site_id, t.pub_id 
FROM pub_slot s
INNER JOIN pub_site t USING (site_id)
WHERE s.slot_id=?`)
	if err != nil {
		panic(err)
	}
	defer sth_slot.Close()
	slots := make(map[int][]int)
	for slotID := slotID_start; slotID <= *slotID_end; slotID++ {
		var siteID, pubID int
		if err := sth_slot.QueryRow(slotID).Scan(&siteID, &pubID); err != nil {
			panic(err)
		}
		slots[slotID] = []int{siteID, pubID}
	}

	sth_item, err := dbh.Prepare(
		`SELECT i.campaign_id, c.adv_id 
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
WHERE i.item_id=?`)
	if err != nil {
		panic(err)
	}
	defer sth_item.Close()
	items := make(map[int][]int)
	for itemID := itemID_start; itemID <= *itemID_end; itemID++ {
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
			err := summer.InsertLedger(dbh, myDay, slots, items, imps, clis, spes)
			if err != nil {
				panic(err)
			}
		}
	}

	if err := summer.InsertDaily(dbh, day); err != nil {
		panic(err)
	}

	os.Exit(0)
}

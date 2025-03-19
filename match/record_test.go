package match

/*
import (
	"database/sql"
	"io"
	"math"
	"os"
	"testing"
	"time"

	"github.com/genelet/winter/pzutil"
	_ "github.com/go-sql-driver/mysql"
)

func TestRecord(t *testing.T) {
	c, err := pzutil.NewConfig("../conf/gotest.conf")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatalf("error opening Mysql handler: %v", err)
	}

	rows, err := db.Query(
		`SELECT w.item_id, i.campaign_id, IF(i.cost_type="CPC", -1.0*i.cost, i.cost) AS price, c.adv_id,
w.slot_id, l.site_id, s.pub_id
FROM pub_weight w
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN pub_slot l USING (slot_id)
INNER JOIN pub_site s USING (site_id)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	records := make([]*Record, 0)
	pass := makePassing()

	for rows.Next() {
		i := Imp{}
		w := Win{}
		err = rows.Scan(&w.ItemID, &w.CampaignID, &w.Price, &w.AdvID, &w.SlotID, &i.SiteID, &i.PubID)
		if err != nil {
			t.Fatal(err)
		}
		status := new(pzutil.Status)
		if w.Price < 0 {
			status.Request = pzutil.CLIC
		} else {
			status.Request = pzutil.IMPR
		}
		i.Status16 = status.Pack()
		r := &Record{i, []Win{w}}
		records = append(records, r)
		r.AddLedger(pass)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	db.Exec("DELETE FROM ledger_pub_adv")
	db.Exec("DELETE FROM ledger_adv")
	db.Exec("DELETE FROM ledger_pub")
	db.Exec("DELETE FROM ledger_log")
	err = SetLedger(db, WhenToString(time.Now(), 5), pass)
	if err != nil {
		t.Fatal(err)
	}

	rows, err = db.Query(
		`SELECT la.item_id, SUM(pa.spend) AS income, SUM(pa.imps) AS imp, SUM(pa.clis) AS cli
FROM ledger_pub_adv pa
INNER JOIN ledger_pub lp USING (lp_id)
INNER JOIN ledger_adv la USING (la_id)
INNER JOIN ledger_log l ON (lp.log_id=l.log_id)
WHERE lp.slot_id=125
GROUP BY la.item_id ORDER BY la.item_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var itemid, imp, cli uint32
		var income float64
		err = rows.Scan(&itemid, &income, &imp, &cli)
		if err != nil {
			t.Fatal(err)
		}
		switch itemid {
		case 5, 25, 45, 65, 85, 105, 125:
			if imp != 1 || cli != 0 || math.Abs(income-0.005) > 0.00001 {
				t.Errorf("%f %d %d", income, imp, cli)
			}
		case 15, 35, 55, 75, 95, 115:
			if imp != 0 || cli != 1 || math.Abs(income-5) > 0.00001 {
				t.Errorf("%f %d %d", income, imp, cli)
			}
		default:
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	fn := "tmp.bin"
	f, err := os.Create(fn)
	if err != nil {
		t.Fatal(err)
	}

	end := time.Now()
	n := len(records)
	for i, record := range records {
		record.Nano64 = end.Add(time.Second * time.Duration(i-n)).UnixNano()
		if err := record.PackIO(f); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	g, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	i := 0
	gecords := make([]*Record, 0)
	for {
		record, err := UnpackRecordIO(g)
		i++
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		gecords = append(gecords, record)
	}
	if len(records) != len(gecords) ||
		records[0].Nano64 != gecords[0].Nano64 ||
		records[35].Nano64 != gecords[35].Nano64 {
		t.Errorf("%v", records[0])
		t.Errorf("%v", gecords[0])
		t.Errorf("%v", records[35])
		t.Errorf("%v", gecords[35])
	}

	db.Exec("DELETE FROM ledger_pub_adv")
	db.Exec("DELETE FROM ledger_adv")
	db.Exec("DELETE FROM ledger_pub")
	db.Exec("DELETE FROM ledger_log")
	if err = SetRecordFromFile(db, 5, fn); err != nil {
		t.Fatal(err)
	}

}
*/

package ledger

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/match"
)

func TestStatisticsMissingWinLossFileReturnsMissingInput(t *testing.T) {
	ledger := &Ledger{LogWinLoss: t.TempDir(), Current: 123}

	_, _, _, _, _, err := ledger.Statistics()
	if !errors.Is(err, ErrMissingInput) {
		t.Fatalf("Statistics error = %v, want ErrMissingInput", err)
	}
}

func TestRunIntervalEmbeddedTreatsMissingInputAsSkip(t *testing.T) {
	ledger := &Ledger{LogWinLoss: t.TempDir(), Current: 123}
	err := ledger.StatisticsToLedger()
	if !errors.Is(err, ErrMissingInput) {
		t.Fatalf("StatisticsToLedger error = %v, want ErrMissingInput", err)
	}
}

func TestStatisticsScansLargeLinesAndAggregatesByCreativeID(t *testing.T) {
	dir := t.TempDir()
	ledger := &Ledger{
		LogWinLoss: dir,
		Current:    123,
		slots: map[uint32][2]uint32{
			10: {20, 30},
		},
		creatives: map[uint32][3]uint32{
			99: {77, 66, 55},
		},
	}
	writeWinLossLog(t, filepath.Join(dir, "winloss.123"),
		dsp.WinLoss{
			Status:    dsp.StatusTrackImp,
			RPub:      match.RPub{SlotID: 10},
			RAdv:      match.RAdv{Demand: match.Demand{ItemID: 77, CreativeID: 99}, Cost: 1.25},
			AuctionID: strings.Repeat("x", 128*1024),
		},
		dsp.WinLoss{
			Status: dsp.StatusTrackClk,
			RPub:   match.RPub{SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{ItemID: 77, CreativeID: 99}, Cost: 1.25},
		},
	)

	slots, creatives, imps, clis, spend, err := ledger.Statistics()
	if err != nil {
		t.Fatal(err)
	}
	if slots[10] != [2]uint32{20, 30} {
		t.Fatalf("slot metadata = %v", slots[10])
	}
	if creatives[99] != [3]uint32{77, 66, 55} {
		t.Fatalf("creative metadata = %v", creatives[99])
	}
	if imps[10][99] != 1 {
		t.Fatalf("imps by creative = %v, want 1 for creative 99", imps)
	}
	if clis[10][99] != 1 {
		t.Fatalf("clicks by creative = %v, want 1 for creative 99", clis)
	}
	if spend[10][99] != 1.25 {
		t.Fatalf("spend by creative = %v, want 1.25", spend)
	}
	if _, ok := imps[10][77]; ok {
		t.Fatalf("statistics used item id as creative id: %v", imps[10])
	}
}

func TestStatisticsAggregatesDirectSSPTrackerWinLossWithoutSchemaChange(t *testing.T) {
	dir := t.TempDir()
	ledger := &Ledger{
		LogWinLoss: dir,
		Current:    456,
		slots: map[uint32][2]uint32{
			100: {10, 1},
		},
		creatives: map[uint32][3]uint32{
			10000: {1000, 10, 1},
		},
	}
	writeWinLossLog(t, filepath.Join(dir, "winloss.456"),
		dsp.WinLoss{
			Status: dsp.StatusTrackImp,
			RPub:   match.RPub{PubID: 1, SiteID: 10, SlotID: 100},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 1, CampaignID: 10, ItemID: 1000, CreativeID: 10000}, Cost: 2.5},
		},
		dsp.WinLoss{
			Status: dsp.StatusTrackClk,
			RPub:   match.RPub{PubID: 1, SiteID: 10, SlotID: 100},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 1, CampaignID: 10, ItemID: 1000, CreativeID: 10000}, Cost: 2.5},
		},
	)

	slots, creatives, imps, clis, spend, err := ledger.Statistics()
	if err != nil {
		t.Fatal(err)
	}
	if slots[100] != [2]uint32{10, 1} {
		t.Fatalf("slot metadata = %v, want site/pub metadata", slots[100])
	}
	if creatives[10000] != [3]uint32{1000, 10, 1} {
		t.Fatalf("creative metadata = %v, want item/campaign/adv metadata", creatives[10000])
	}
	if imps[100][10000] != 1 || clis[100][10000] != 1 || spend[100][10000] != 2.5 {
		t.Fatalf("ledger aggregates imps=%v clis=%v spend=%v, want 1/1/2.5", imps, clis, spend)
	}
}

func TestStatisticsAggregatesMiddlemanMetadata(t *testing.T) {
	dir := t.TempDir()
	ledger := &Ledger{
		LogWinLoss: dir,
		Current:    124,
		slots: map[uint32][2]uint32{
			10: {20, 30},
		},
		creatives: map[uint32][3]uint32{
			99: {77, 66, 55},
		},
	}
	meta := &dsp.MiddlemanWinLossMeta{
		BidderID:          7,
		GroupID:           8,
		RouteBidderID:     9,
		TargetID:          10,
		Source:            "win",
		ChargePrice:       1.20,
		PayPrice:          0.95,
		ForwardStatus:     "ok",
		ForwardHTTPStatus: 204,
	}
	writeWinLossLog(t, filepath.Join(dir, "winloss.124"),
		dsp.WinLoss{
			Status:    dsp.StatusWin,
			RPub:      match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:      match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: meta,
		},
		dsp.WinLoss{
			Status: dsp.StatusWin,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "win",
				ForwardStatus: "duplicate",
			},
		},
		dsp.WinLoss{
			Status:    dsp.StatusTrackImp,
			RPub:      match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:      match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}, Cost: 1.20},
			Middleman: meta,
		},
		dsp.WinLoss{
			Status: dsp.StatusTrackClk,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "click",
				ForwardStatus: "none",
			},
		},
		dsp.WinLoss{
			Status: dsp.StatusLoss,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "loss",
				ForwardStatus: "http_error",
			},
		},
	)

	stats, err := ledger.StatisticsAll()
	if err != nil {
		t.Fatal(err)
	}
	key := middlemanLedgerKey{
		BidderID:      7,
		GroupID:       8,
		RouteBidderID: 9,
		TargetID:      10,
		AdvID:         55,
		CampaignID:    66,
		ItemID:        77,
		CreativeID:    99,
		PubID:         30,
		SiteID:        20,
		SlotID:        10,
	}
	got := stats.Middleman[key]
	if got == nil {
		t.Fatalf("middleman stats missing: %#v", stats.Middleman)
	}
	if got.Wins != 1 || got.Losses != 1 || got.Imps != 1 || got.Clis != 1 {
		t.Fatalf("middleman counts = %#v", got)
	}
	if !closeFloat64(got.ChargeSpend, 1.20) || !closeFloat64(got.PaySpend, 0.95) || !closeFloat64(got.MarginSpend, 0.25) {
		t.Fatalf("middleman spend = charge %.2f pay %.2f margin %.2f", got.ChargeSpend, got.PaySpend, got.MarginSpend)
	}
	if got.ForwardOK != 1 || got.ForwardDuplicate != 1 || got.ForwardHTTPError != 1 || got.ForwardNone != 0 {
		t.Fatalf("forward counters = %#v", got)
	}
}

func TestMiddlemanLedgerIntervalAndDailyTables(t *testing.T) {
	db := openLedgerTestDB(t)
	defer db.Close()

	active := time.Date(2099, 1, 2, 3, 40, 0, 0, time.UTC)
	day := "2099-01-02"
	timely := active.Format("2006-01-02 15:04:05")
	cleanupMiddlemanLedgerTestRows(t, db, day, timely)
	defer cleanupMiddlemanLedgerTestRows(t, db, day, timely)

	dir := t.TempDir()
	current := int64(777001)
	ledger := &Ledger{DB: db, LogWinLoss: dir, Current: current, Active: active}
	writeWinLossLog(t, filepath.Join(dir, "winloss.777001"),
		dsp.WinLoss{
			Status: dsp.StatusWin,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "win",
				ForwardStatus: "ok",
			},
		},
		dsp.WinLoss{
			Status: dsp.StatusTrackImp,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "bill",
				ChargePrice:   1.20,
				PayPrice:      0.95,
				ForwardStatus: "ok",
			},
		},
		dsp.WinLoss{
			Status: dsp.StatusTrackClk,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "click",
				ForwardStatus: "none",
			},
		},
		dsp.WinLoss{
			Status: dsp.StatusLoss,
			RPub:   match.RPub{PubID: 30, SiteID: 20, SlotID: 10},
			RAdv:   match.RAdv{Demand: match.Demand{AdvID: 55, CampaignID: 66, ItemID: 77, CreativeID: 99}},
			Middleman: &dsp.MiddlemanWinLossMeta{
				BidderID:      7,
				GroupID:       8,
				RouteBidderID: 9,
				TargetID:      10,
				Source:        "loss",
				ForwardStatus: "http_error",
			},
		},
	)

	if err := ledger.StatisticsToLedger(); err != nil {
		t.Fatal(err)
	}
	assertMiddlemanLedgerRow(t, db, "ledger_mid", "ledger_log", "timely", timely)

	if err := InsertDaily(db, day); err != nil {
		t.Fatal(err)
	}
	assertMiddlemanLedgerRow(t, db, "daily_mid", "daily_log", "daily", day)
}

func TestJobSchemaSmoke(t *testing.T) {
	db := openLedgerTestDB(t)
	defer db.Close()

	for _, table := range []string{"mid_callback_retry", "ledger_log", "ledger_mid", "daily_log", "daily_mid"} {
		var count int
		err := db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("schema table %s count = %d, want 1", table, count)
		}
	}
}

func openLedgerTestDB(t *testing.T) *sql.DB {
	t.Helper()

	configPath := os.Getenv("AOFEI")
	if configPath == "" {
		configPath = "../../../etc/aofei.local.json"
	}
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("AOFEI config %s is missing; run ./scripts/aofei-local.sh up", configPath)
		}
		t.Fatal(err)
	}
	c, err := dsp.NewConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("configured DB is unavailable: %v", err)
	}
	return db
}

func cleanupMiddlemanLedgerTestRows(t *testing.T, db *sql.DB, day, timely string) {
	t.Helper()

	if _, err := db.Exec(`DELETE FROM daily_mid WHERE log_id IN (SELECT log_id FROM daily_log WHERE daily=?)`, day); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM daily_pub_adv WHERE lp_id IN (SELECT lp_id FROM daily_pub p INNER JOIN daily_log l USING (log_id) WHERE l.daily=?)`, day); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM daily_pub WHERE log_id IN (SELECT log_id FROM daily_log WHERE daily=?)`, day); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM daily_adv WHERE log_id IN (SELECT log_id FROM daily_log WHERE daily=?)`, day); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM daily_log WHERE daily=?`, day); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM ledger_mid WHERE log_id IN (SELECT log_id FROM ledger_log WHERE timely=?)`, timely); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM ledger_pub_adv WHERE lp_id IN (SELECT lp_id FROM ledger_pub p INNER JOIN ledger_log l USING (log_id) WHERE l.timely=?)`, timely); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM ledger_pub WHERE log_id IN (SELECT log_id FROM ledger_log WHERE timely=?)`, timely); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM ledger_adv WHERE log_id IN (SELECT log_id FROM ledger_log WHERE timely=?)`, timely); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM ledger_log WHERE timely=?`, timely); err != nil {
		t.Fatal(err)
	}
}

func assertMiddlemanLedgerRow(t *testing.T, db *sql.DB, table, logTable, logField, logValue string) {
	t.Helper()

	var wins, losses, imps, clis, forwardOK, forwardHTTPError, forwardNone int
	var chargeSpend, paySpend, marginSpend float64
	err := db.QueryRow(`
SELECT m.wins, m.losses, m.imps, m.clis, m.charge_spend, m.pay_spend,
	m.margin_spend, m.forward_ok, m.forward_http_error, m.forward_none
FROM `+table+` m
INNER JOIN `+logTable+` l USING (log_id)
WHERE l.`+logField+`=? AND m.bidder_id=7 AND m.group_id=8 AND m.route_bidder_id=9
	AND m.target_id=10 AND m.adv_id=55 AND m.campaign_id=66 AND m.item_id=77
	AND m.creative_id=99 AND m.pub_id=30 AND m.site_id=20 AND m.slot_id=10`,
		logValue).Scan(
		&wins, &losses, &imps, &clis, &chargeSpend, &paySpend, &marginSpend,
		&forwardOK, &forwardHTTPError, &forwardNone,
	)
	if err != nil {
		t.Fatal(err)
	}
	if wins != 1 || losses != 1 || imps != 1 || clis != 1 {
		t.Fatalf("%s counts = win %d loss %d imp %d cli %d", table, wins, losses, imps, clis)
	}
	if !closeFloat64(chargeSpend, 1.20) || !closeFloat64(paySpend, 0.95) || !closeFloat64(marginSpend, 0.25) {
		t.Fatalf("%s spend = charge %.2f pay %.2f margin %.2f", table, chargeSpend, paySpend, marginSpend)
	}
	if forwardOK != 2 || forwardHTTPError != 1 || forwardNone != 0 {
		t.Fatalf("%s forwards = ok %d http_error %d none %d", table, forwardOK, forwardHTTPError, forwardNone)
	}
}

func closeFloat64(a, b float64) bool {
	return math.Abs(a-b) < 0.000001
}

func writeWinLossLog(t *testing.T, name string, records ...dsp.WinLoss) {
	t.Helper()
	fh, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	enc := json.NewEncoder(fh)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			t.Fatal(err)
		}
	}
}

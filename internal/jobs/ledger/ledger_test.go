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
	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/dsp"
	"github.com/guruperl/aofei/match"
)

func TestExactMiddlemanCPMRejectsInvalidV3RangeAndIdentity(t *testing.T) {
	for _, meta := range []*dsp.MiddlemanWinLossMeta{
		{ChargeCPM: accounting.MaxCPM + 1, PayCPM: 1, ChargePrice: 1},
		{AccountingVersion: accounting.ExactMoneyContract, ChargeCPM: 2, PayCPM: 1, MarginCPMExact: 2},
		{ChargeCPM: 1, PayCPM: 2},
		{AccountingVersion: accounting.ExactMoneyContract, ChargePrice: 2, PayPrice: 1, Currency: "USD"},
		{AccountingVersion: accounting.ExactMoneyContract, ChargeCPM: 2, PayCPM: 1, MarginCPMExact: 1, Currency: "EUR"},
	} {
		if _, _, err := exactMiddlemanCPM(meta); err == nil {
			t.Fatalf("invalid middleman exact fact was accepted: %+v", meta)
		}
	}
}

func TestNormalizeWinLossAccountingSeparatesLegacyAndExactFacts(t *testing.T) {
	legacy := dsp.WinLoss{
		Status: dsp.StatusTrackImp,
		RAdv:   match.RAdv{CostType: match.CostTypeCPM, Cost: 1.25},
	}
	if err := normalizeWinLossAccounting(&legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.AccountingVersion != accounting.LegacyMoneyContract {
		t.Fatalf("legacy version = %q", legacy.AccountingVersion)
	}

	unmarkedExact := dsp.WinLoss{
		Status: dsp.StatusTrackImp,
		RAdv:   match.RAdv{CostType: match.CostTypeCPM, CostCPM: 1_250_001},
	}
	if err := normalizeWinLossAccounting(&unmarkedExact); err == nil {
		t.Fatal("unmarked exact field was promoted to v3")
	}
}

func TestNormalizeWinLossAccountingRejectsBrokenExactMiddlemanIdentity(t *testing.T) {
	for _, wl := range []dsp.WinLoss{
		{
			Status: dsp.StatusTrackImp, AccountingVersion: accounting.ExactMoneyContract,
			RAdv: match.RAdv{CostType: match.CostTypeCPM, CostCPM: 2},
			Middleman: &dsp.MiddlemanWinLossMeta{
				AccountingVersion: accounting.ExactMoneyContract,
				ChargePrice:       2, PayPrice: 1, Currency: "USD",
			},
		},
		{
			Status: dsp.StatusTrackImp, AccountingVersion: accounting.ExactMoneyContract,
			RAdv: match.RAdv{CostType: match.CostTypeCPM, CostCPM: 3},
			Middleman: &dsp.MiddlemanWinLossMeta{
				AccountingVersion: accounting.ExactMoneyContract,
				ChargeCPM:         2, PayCPM: 1, MarginCPMExact: 1, Currency: "USD",
			},
		},
	} {
		if err := normalizeWinLossAccounting(&wl); err == nil {
			t.Fatalf("broken exact middleman fact was accepted: %+v", wl)
		}
	}
}

func TestNormalizeWinLossAccountingRejectsMixedLegacyExactFact(t *testing.T) {
	wl := dsp.WinLoss{
		Status:            dsp.StatusTrackImp,
		AccountingVersion: accounting.LegacyMoneyContract,
		RAdv:              match.RAdv{CostType: match.CostTypeCPM, Cost: 1.25, CostCPM: 1_250_000},
	}
	if err := normalizeWinLossAccounting(&wl); err == nil {
		t.Fatal("v2 win/loss fact carrying exact CPM was accepted")
	}
}

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
	if !closeFloat64(spend[10][99], 0.00125) {
		t.Fatalf("spend by creative = %v, want 0.00125", spend)
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
	if imps[100][10000] != 1 || clis[100][10000] != 1 || !closeFloat64(spend[100][10000], 0.0025) {
		t.Fatalf("ledger aggregates imps=%v clis=%v spend=%v, want 1/1/0.0025", imps, clis, spend)
	}
}

func TestStatisticsAggregatesMinimumCPMWithoutFloatRounding(t *testing.T) {
	dir := t.TempDir()
	ledger := &Ledger{
		LogWinLoss: dir, Current: 457,
		slots:     map[uint32][2]uint32{1: {2, 3}},
		creatives: map[uint32][3]uint32{4: {5, 6, 7}},
	}
	events := make([]dsp.WinLoss, 1_000)
	for index := range events {
		events[index] = dsp.WinLoss{
			Status: dsp.StatusTrackImp, AccountingVersion: accounting.ExactMoneyContract, RPub: match.RPub{SlotID: 1},
			RAdv: match.RAdv{Demand: match.Demand{ItemID: 5, CreativeID: 4}, CostType: match.CostTypeCPM, CostCPM: 1},
		}
	}
	writeWinLossLog(t, filepath.Join(dir, "winloss.457"), events...)
	stats, err := ledger.StatisticsAll()
	if err != nil {
		t.Fatal(err)
	}
	if got := stats.SpendNano[1][4]; got != 1_000 || got.String() != "0.000001000" {
		t.Fatalf("1,000 minimum CPM impressions = %s (%d nano), want 0.000001000", got, got)
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
	if !closeFloat64(got.ChargeSpend, 0.00120) || !closeFloat64(got.PaySpend, 0.00095) || !closeFloat64(got.MarginSpend, 0.00025) {
		t.Fatalf("middleman spend = charge %.2f pay %.2f margin %.2f", got.ChargeSpend, got.PaySpend, got.MarginSpend)
	}
	if got.ForwardOK != 1 || got.ForwardDuplicate != 1 || got.ForwardHTTPError != 1 || got.ForwardNone != 0 {
		t.Fatalf("forward counters = %#v", got)
	}
}

func TestAggregateReportingLedgerSeparatesLocalAndMiddlemanCommercialFacts(t *testing.T) {
	out := make(map[reportingLedgerKey]*reportingLedgerStats)
	local := dsp.WinLoss{
		Status: dsp.StatusTrackImp,
		RPub:   match.RPub{PubID: 1, SiteID: 2, SlotID: 3},
		RAdv:   match.RAdv{Demand: match.Demand{AdvID: 4, CampaignID: 5, ItemID: 6, CreativeID: 7}, Cost: 2.5},
		Reporting: &dsp.ReportingDimensions{
			CountryID: 8, StateID: 9, DeviceOS: 10, DeviceType: 11,
			Environment: "Web", IntegrationMode: "BrowserTag", MediaIntent: "Banner", Placement: "AboveFold",
			RenderContext: "WebPage", RefreshMode: "Timed", RefreshSeconds: 60, AdDensity: "Standard", TrafficQuality: "Reviewed",
			SourceQuality: "OwnedOperated", ManagementControl: "Publisher", SellerType: "Publisher", SellerID: "seller-1",
		},
	}
	aggregateReportingLedger(out, local)
	localKey := reportingLedgerKey{
		DemandSource: "Local", AdvID: 4, CampaignID: 5, ItemID: 6, CreativeID: 7,
		PubID: 1, SiteID: 2, SlotID: 3, CountryID: 8, StateID: 9, DeviceOS: 10, DeviceType: 11,
		Environment: "Web", IntegrationMode: "BrowserTag", MediaIntent: "Banner", Placement: "AboveFold",
		RenderContext: "WebPage", RefreshMode: "Timed", RefreshSeconds: 60, AdDensity: "Standard", TrafficQuality: "Reviewed",
		SourceQuality: "OwnedOperated", ManagementControl: "Publisher", SellerType: "Publisher", SellerID: "seller-1",
	}
	localStats := out[localKey]
	if localStats == nil || localStats.Imps != 1 || !closeFloat64(localStats.SpendUSD, 0.0025) || !closeFloat64(localStats.RevenueUSD, 0.0025) || !closeFloat64(localStats.CostUSD, 0.0025) || localStats.MarginUSD != 0 || localStats.ReturnedCPMSum != 2.5 {
		t.Fatalf("local report stats = %#v", localStats)
	}

	middleman := local
	middleman.Middleman = &dsp.MiddlemanWinLossMeta{
		BidderID: 12, GroupID: 13, RouteBidderID: 14, TargetID: 15,
		TriggerMode: "Always", ChargePrice: 3.0, PayPrice: 2.0,
	}
	aggregateReportingLedger(out, middleman)
	middlemanKey := localKey
	middlemanKey.DemandSource = "Always"
	middlemanKey.BidderID, middlemanKey.GroupID, middlemanKey.RouteBidderID, middlemanKey.TargetID = 12, 13, 14, 15
	middlemanStats := out[middlemanKey]
	if middlemanStats == nil || !closeFloat64(middlemanStats.SpendUSD, 0.002) || !closeFloat64(middlemanStats.RevenueUSD, 0.003) || !closeFloat64(middlemanStats.CostUSD, 0.002) || !closeFloat64(middlemanStats.MarginUSD, 0.001) || middlemanStats.DownstreamCPMSum != 2 || middlemanStats.ReturnedCPMSum != 3 {
		t.Fatalf("middleman report stats = %#v", middlemanStats)
	}
}

func TestAggregateReportingLedgerKeepsCPMSumsExact(t *testing.T) {
	out := make(map[reportingLedgerKey]*reportingLedgerStats)
	wl := dsp.WinLoss{
		Status:            dsp.StatusTrackImp,
		AccountingVersion: accounting.ExactMoneyContract,
		RPub:              match.RPub{PubID: 1, SiteID: 2, SlotID: 3},
		RAdv: match.RAdv{
			Demand:   match.Demand{AdvID: 4, CampaignID: 5, ItemID: 6, CreativeID: 7},
			CostType: match.CostTypeCPM,
			CostCPM:  100_000_002,
		},
	}
	for range 3 {
		if err := aggregateReportingLedger(out, wl); err != nil {
			t.Fatal(err)
		}
	}
	key := reportingLedgerKey{
		AccountingVersion: accounting.ExactMoneyContract,
		DemandSource:      "Local",
		AdvID:             4, CampaignID: 5, ItemID: 6, CreativeID: 7,
		PubID: 1, SiteID: 2, SlotID: 3,
		Environment: "Unknown", IntegrationMode: "Unknown", MediaIntent: "Unknown", Placement: "Unknown",
		RenderContext: "Unknown", RefreshMode: "Unknown", AdDensity: "Unknown", TrafficQuality: "Unknown",
		SourceQuality: "Unknown", ManagementControl: "Unknown", SellerType: "Unknown",
	}
	stats := out[key]
	if stats == nil || stats.ReturnedCPMTotal.String() != "300.000006" {
		t.Fatalf("exact returned CPM sum = %#v, want 300.000006", stats)
	}
	if stats.ReturnedCPMSum != 300.000006 {
		t.Fatalf("compatibility returned CPM sum = %.6f, want 300.000006", stats.ReturnedCPMSum)
	}
}

func TestReportingCPMSumRejectsOverflow(t *testing.T) {
	stats := &reportingLedgerStats{ReturnedCPMTotal: accounting.CPMTotal(math.MaxInt64)}
	if err := stats.addReturnedCPM(1); err == nil {
		t.Fatal("overflowing returned CPM aggregate was accepted")
	}
}

func TestReportingDimensionHashSeparatesSupplyTaxonomy(t *testing.T) {
	first := reportingLedgerKey{DemandSource: "Local", PubID: 1, SiteID: 2, SlotID: 3, Environment: "Web", Placement: "AboveFold"}
	second := first
	second.Placement = "InFeed"
	if string(reportingDimensionHash(first)) == string(reportingDimensionHash(second)) {
		t.Fatal("distinct supply taxonomy produced the same report dimension hash")
	}
	firstHash := string(reportingDimensionHash(first))
	if firstHash != string(reportingDimensionHash(first)) {
		t.Fatal("report dimension hash is not deterministic")
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

func TestAdvertiserBalanceReconciliation(t *testing.T) {
	db := openLedgerTestDB(t)
	defer db.Close()

	const (
		advID      = 990001
		campaignID = 990002
		itemID     = 990003
		creativeID = 990004
		ledgerID   = 990005
		dailyID    = 990006
		campTotal  = 990011
		campDaily  = 990012
		itemTotal  = 990013
		itemDaily  = 990014
	)
	cleanup := func() {
		for _, statement := range []string{
			"DELETE FROM daily_adv WHERE log_id=990006",
			"DELETE FROM daily_log WHERE log_id=990006",
			"DELETE FROM ledger_adv WHERE log_id=990005",
			"DELETE FROM ledger_log WHERE log_id=990005",
			"DELETE FROM adv_item WHERE item_id=990003",
			"DELETE FROM adv_campaign WHERE campaign_id=990002",
			"DELETE FROM adv_balance WHERE balance_id BETWEEN 990011 AND 990014",
			"DELETE FROM adv WHERE adv_id=990001",
		} {
			if _, err := db.Exec(statement); err != nil {
				t.Errorf("cleanup %q: %v", statement, err)
			}
		}
	}
	cleanup()
	defer cleanup()

	for _, balanceID := range []int{campTotal, campDaily, itemTotal, itemDaily} {
		if _, err := db.Exec(`INSERT INTO adv_balance (balance_id, limit_spend, limit_imp, limit_cli, current_spend, current_imp, current_cli, created) VALUES (?, 100, 100, 100, 0, 0, 0, NOW())`, balanceID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO adv (adv_id, email, passwd, active, created) VALUES (?, 'd01-ledger@example.test', 'not-a-login-hash', 'Yes', NOW())`, advID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO adv_campaign (campaign_id, adv_id, campaign_name, total_balance_id, daily_balance_id, active, created) VALUES (?, ?, 'D01 ledger test', ?, ?, 'Yes', NOW())`, campaignID, advID, campTotal, campDaily); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO adv_item (item_id, campaign_id, item_name, item_click, total_balance_id, daily_balance_id, active, created) VALUES (?, ?, 'D01 ledger item', '', ?, ?, 'Yes', NOW())`, itemID, campaignID, itemTotal, itemDaily); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ledger_log (log_id, timely, created) VALUES (?, '2099-02-03 04:00:00', NOW())`, ledgerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ledger_adv (log_id, creative_id, item_id, campaign_id, adv_id, spend, imps, clis) VALUES (?, ?, ?, ?, ?, 3.5, 7, 2)`, ledgerID, creativeID, itemID, campaignID, advID); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileAdvertiserTotalBalances(tx); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := reconcileAdvertiserDailyBalancesFromIntervals(tx, "2099-02-03"); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, balanceID := range []int{campTotal, itemTotal} {
		assertAdvertiserBalance(t, db, balanceID, 3.5, 7, 2, "")
	}
	for _, balanceID := range []int{campDaily, itemDaily} {
		assertAdvertiserBalance(t, db, balanceID, 3.5, 7, 2, "2099-02-03")
	}

	if _, err := db.Exec(`INSERT INTO daily_log (log_id, daily, created) VALUES (?, '2099-02-03', NOW())`, dailyID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO daily_adv (log_id, creative_id, item_id, campaign_id, adv_id, spend, imps, clis) VALUES (?, ?, ?, ?, ?, 4.5, 8, 3)`, dailyID, creativeID, itemID, campaignID, advID); err != nil {
		t.Fatal(err)
	}
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileAdvertiserDailyBalancesFromDaily(tx, "2099-02-03"); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, balanceID := range []int{campDaily, itemDaily} {
		assertAdvertiserBalance(t, db, balanceID, 4.5, 8, 3, "2099-02-03")
	}
}

func assertAdvertiserBalance(t *testing.T, db *sql.DB, balanceID int, spend float64, imps, clicks int, day string) {
	t.Helper()
	var gotSpend float64
	var gotImps, gotClicks int
	var gotDay sql.NullTime
	if err := db.QueryRow(`SELECT current_spend, current_imp, current_cli, current_day FROM adv_balance WHERE balance_id=?`, balanceID).Scan(&gotSpend, &gotImps, &gotClicks, &gotDay); err != nil {
		t.Fatal(err)
	}
	gotDayText := ""
	if gotDay.Valid {
		gotDayText = gotDay.Time.Format("2006-01-02")
	}
	if !closeFloat64(gotSpend, spend) || gotImps != imps || gotClicks != clicks || gotDayText != day || gotDay.Valid != (day != "") {
		t.Fatalf("balance %d = spend %.2f imps %d clicks %d day %#v, want %.2f/%d/%d/%q", balanceID, gotSpend, gotImps, gotClicks, gotDay, spend, imps, clicks, day)
	}
}

func openLedgerTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("AOFEI_LEDGER_TEST_DSN")
	if dsn == "" {
		t.Skip("AOFEI_LEDGER_TEST_DSN is unset; integration tests require an explicit disposable database")
	}
	db, err := sql.Open("mysql", dsn)
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
	if !closeFloat64(chargeSpend, 0.00120) || !closeFloat64(paySpend, 0.00095) || !closeFloat64(marginSpend, 0.00025) {
		t.Fatalf("%s spend = charge %.6f pay %.6f margin %.6f", table, chargeSpend, paySpend, marginSpend)
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

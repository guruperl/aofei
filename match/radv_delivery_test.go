package match

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestHydrateRAdvDeliveriesCarriesAuthoritativePolicy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	columns := make([]string, 39)
	for i := range columns {
		columns[i] = fmt.Sprintf("c%d", i)
	}
	campaignSchedule := "1" + strings.Repeat("0", DeliveryHoursPerWeek-1)
	itemSchedule := strings.Repeat("0", DeliveryHoursPerWeek-1) + "1"
	rows := sqlmock.NewRows(columns).AddRow(
		11, 7,
		100, 200, 110, 190,
		"Asia/Shanghai", campaignSchedule, "Even", itemSchedule, "Fast",
		101, "100.500000000", 1000, 50, "10.250000000", 100, 5,
		102, "20.500000000", 200, 10, "2.250000000", 20, 1,
		103, "80.500000000", 800, 40, "8.250000000", 80, 4,
		104, "10.500000000", 100, 5, "1.250000000", 10, 1,
	)
	mock.ExpectQuery(`(?s)SELECT i\.item_id, c\.campaign_id,.*WHERE i\.item_id IN \(\?\)`).
		WithArgs(uint32(11)).WillReturnRows(rows)

	before := time.Now().Add(-time.Second).Unix()
	blocks, err := hydrateRAdvDeliveries(context.Background(), db, RAdvs{
		{Demand: Demand{CampaignID: 7, ItemID: 11, CreativeID: 21}},
		{Demand: Demand{CampaignID: 7, ItemID: 11, CreativeID: 22}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 || blocks[0].Delivery != blocks[1].Delivery {
		t.Fatalf("delivery policy was not applied to every creative: %#v", blocks)
	}
	delivery := blocks[0].Delivery
	if delivery.GeneratedAtUnix < before || delivery.TimezoneName() != "Asia/Shanghai" {
		t.Fatalf("delivery metadata = %#v", delivery)
	}
	if delivery.Campaign.Pacing != DeliveryPacingEven || delivery.Item.Pacing != DeliveryPacingFast {
		t.Fatalf("pacing = campaign %d item %d", delivery.Campaign.Pacing, delivery.Item.Pacing)
	}
	if delivery.Campaign.WeeklySchedule() != campaignSchedule || delivery.Item.WeeklySchedule() != itemSchedule {
		t.Fatal("weekly schedules were not hydrated")
	}
	if delivery.CampaignTotal.ID != 101 || delivery.CampaignTotal.LimitSpendNano != 100_500_000_000 || delivery.ItemDaily.CurrentClick != 1 {
		t.Fatalf("balance facts = %#v", delivery)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHydrateRAdvDeliveriesRejectsMissingPolicy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	columns := make([]string, 39)
	for i := range columns {
		columns[i] = fmt.Sprintf("c%d", i)
	}
	mock.ExpectQuery(`(?s)WHERE i\.item_id IN \(\?\)`).WithArgs(uint32(11)).WillReturnRows(sqlmock.NewRows(columns))
	_, err = hydrateRAdvDeliveries(context.Background(), db, RAdvs{{Demand: Demand{CampaignID: 7, ItemID: 11}}})
	if err == nil || !strings.Contains(err.Error(), "delivery policy missing for item 11") {
		t.Fatalf("missing policy error = %v", err)
	}
}

func TestParseDeliveryPacingRejectsUnknownValue(t *testing.T) {
	if got, err := parseDeliveryPacing("Even"); err != nil || got != DeliveryPacingEven {
		t.Fatalf("Even pacing = %d, %v", got, err)
	}
	if _, err := parseDeliveryPacing(""); err == nil {
		t.Fatal("empty pacing should fail instead of silently becoming Fast")
	}
}

func TestDBRAdvsRejectsLegacyCommercialTypeBeforeCachePublication(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	columns := []string{
		"adv_id", "campaign_id", "item_id", "creative_id", "weight",
		"cost_type", "cost", "cap_number", "cap_period", "cap_throttle",
		"click_number", "click_period", "item_start", "item_end",
	}
	rows := sqlmock.NewRows(columns).AddRow(
		1, 2, 3, 4, 1,
		"CPC", 0.25, nil, nil, nil,
		nil, nil, nil, nil,
	)
	mock.ExpectQuery(`(?s)CALL proc_slotall\(\?, \?\)`).
		WithArgs(uint32(100), uint32(300250)).
		WillReturnRows(rows)

	_, err = dbRAdvsBySizeIDSlotID(context.Background(), db, 300250, 100)
	if err == nil || !strings.Contains(err.Error(), "migrate it to a reviewed positive USD CPM price") {
		t.Fatalf("legacy CPC compile error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

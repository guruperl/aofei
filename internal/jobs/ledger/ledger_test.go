package ledger

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/match"
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

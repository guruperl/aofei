package reporting

import (
	"math"
	"testing"
)

func TestMetricContractsAreCompleteAndIndependentOfCPMConversion(t *testing.T) {
	contracts := MetricContracts()
	if err := ValidateMetricContracts(contracts); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"impressions": false, "clicks": false, "actions": false, "spend": false, "revenue": false, "cost": false, "margin": false, "roi": false, "roas": false}
	for _, contract := range contracts {
		if _, exists := want[contract.Name]; exists {
			want[contract.Name] = true
		}
		if contract.Name == "spend" && contract.Formula == "SUM(spend_usd) / 1000" {
			t.Fatal("stored USD spend was converted from CPM a second time")
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("metric contract %q is missing", name)
		}
	}
}

func TestDeriveRatiosZeroDenominators(t *testing.T) {
	if got, err := DeriveRatios(0, 0, 0, 0, 12); err != nil || got != (Ratios{}) {
		t.Fatalf("zero denominator ratios = %#v", got)
	}
	got, err := DeriveRatios(100, 10, 2, 5, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.CTR != 0.1 || got.CVR != 0.2 || got.ROI != 3 || got.ROAS != 4 {
		t.Fatalf("ratios = %#v", got)
	}
}

func TestDeriveRatiosRejectsInvalidSourcesAndResults(t *testing.T) {
	for _, input := range []struct {
		impressions, clicks, actions uint64
		spend, purchase              float64
	}{
		{10, 11, 0, 1, 1},
		{10, 5, 6, 1, 1},
		{10, 5, 1, math.NaN(), 1},
		{10, 5, 1, math.Inf(1), 1},
		{10, 5, 1, -1, 1},
		{10, 5, 1, 1, math.NaN()},
		{10, 5, 1, 1, math.Inf(1)},
		{10, 5, 1, 1, -1},
		{10, 5, 1, math.SmallestNonzeroFloat64, math.MaxFloat64},
	} {
		if _, err := DeriveRatios(input.impressions, input.clicks, input.actions, input.spend, input.purchase); err == nil {
			t.Fatalf("invalid ratio sources were accepted: %#v", input)
		}
	}
}

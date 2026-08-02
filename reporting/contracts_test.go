package reporting

import "testing"

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
	if got := DeriveRatios(0, 0, 3, 0, 12); got != (Ratios{}) {
		t.Fatalf("zero denominator ratios = %#v", got)
	}
	got := DeriveRatios(100, 10, 2, 5, 20)
	if got.CTR != 0.1 || got.CVR != 0.2 || got.ROI != 3 || got.ROAS != 4 {
		t.Fatalf("ratios = %#v", got)
	}
}

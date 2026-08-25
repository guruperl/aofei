package accounting

import (
	"math"
	"testing"
)

func TestCPMToImpressionNanoIsExact(t *testing.T) {
	for _, raw := range []string{"0", "0.000001", "2.5", "999999.999999"} {
		cpm, err := ParseCPM(raw)
		if err != nil {
			t.Fatalf("ParseCPM(%q): %v", raw, err)
		}
		got, err := cpm.ImpressionNano()
		if err != nil {
			t.Fatalf("ImpressionNano(%q): %v", raw, err)
		}
		if int64(got) != int64(cpm) {
			t.Fatalf("CPM units %d became nano units %d", cpm, got)
		}
	}
	if _, err := ParseCPM("1000000"); err == nil {
		t.Fatal("accepted CPM outside DECIMAL(12,6)")
	}
	if _, err := ParseCPM("0.0000001"); err == nil {
		t.Fatal("accepted sub-minimum CPM precision")
	}
}

func TestNanoAggregateOverflowAndStatementRounding(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want string
	}{
		{"0.000000499", "0.000000"},
		{"0.000000500", "0.000001"},
		{"-0.000000499", "0.000000"},
		{"-0.000000500", "-0.000001"},
		{"1.234567500", "1.234568"},
	} {
		nano, err := ParseNano(test.raw)
		if err != nil {
			t.Fatalf("ParseNano(%q): %v", test.raw, err)
		}
		if got := nano.StatementMoney().String(); got != test.want {
			t.Fatalf("StatementMoney(%s) = %s, want %s", test.raw, got, test.want)
		}
	}
	if _, err := Nano(math.MaxInt64).Add(1); err == nil {
		t.Fatal("nano addition overflow succeeded")
	}
	if _, err := Nano(math.MinInt64).Sub(1); err == nil {
		t.Fatal("nano subtraction overflow succeeded")
	}
}

func TestCPMTotalAddsPricesWithoutSinglePriceCap(t *testing.T) {
	total, err := CPMTotal(MaxCPM).Add(MaxCPM)
	if err != nil {
		t.Fatal(err)
	}
	if got := total.String(); got != "1999999.999998" {
		t.Fatalf("CPM total = %s, want 1999999.999998", got)
	}
	if _, err := CPMTotal(math.MaxInt64).Add(1); err == nil {
		t.Fatal("CPM total overflow succeeded")
	}
	if _, err := CPMTotal(0).Add(-1); err == nil {
		t.Fatal("negative CPM was added to report total")
	}
}

func TestExactDatabaseScanRejectsBinaryFloat(t *testing.T) {
	var cpm CPM
	if err := cpm.Scan([]byte("2.500001")); err != nil || cpm.String() != "2.500001" {
		t.Fatalf("CPM database scan = %s, %v", cpm, err)
	}
	if err := cpm.Scan(float64(2.5)); err == nil {
		t.Fatal("CPM accepted binary floating-point database source")
	}
	var nano Nano
	if err := nano.Scan("0.002500001"); err != nil || nano.String() != "0.002500001" {
		t.Fatalf("Nano database scan = %s, %v", nano, err)
	}
}

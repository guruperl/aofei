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

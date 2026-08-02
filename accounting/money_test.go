package accounting

import (
	"math"
	"testing"
)

func TestMoneyRoundTripAndPrecision(t *testing.T) {
	for _, raw := range []string{"0", "1", "1.2", "0.001250", "-4.000001"} {
		money, err := ParseMoney(raw)
		if err != nil {
			t.Fatalf("ParseMoney(%q): %v", raw, err)
		}
		reparsed, err := ParseMoney(money.String())
		if err != nil || reparsed != money {
			t.Fatalf("money round trip %q = %q, %v", raw, money.String(), err)
		}
	}
	if _, err := ParseMoney("1.0000001"); err == nil {
		t.Fatal("money accepted more than six decimal places")
	}
}

func TestMoneyBoundsAndCheckedAddition(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want Money
	}{
		{"9223372036854.775807", Money(math.MaxInt64)},
		{"-9223372036854.775808", Money(math.MinInt64)},
	} {
		got, err := ParseMoney(test.raw)
		if err != nil || got != test.want || got.String() != test.raw {
			t.Fatalf("ParseMoney(%q) = %s, %v", test.raw, got, err)
		}
	}
	for _, raw := range []string{"9223372036854.775808", "-9223372036854.775809", "18446744073709551615"} {
		if _, err := ParseMoney(raw); err == nil {
			t.Fatalf("ParseMoney(%q) accepted an out-of-range value", raw)
		}
	}
	if _, err := Money(math.MaxInt64).Add(1); err == nil {
		t.Fatal("positive money overflow succeeded")
	}
	if _, err := Money(math.MinInt64).Add(-1); err == nil {
		t.Fatal("negative money overflow succeeded")
	}
	if _, err := Money(math.MinInt64).Sub(1); err == nil {
		t.Fatal("negative money subtraction overflow succeeded")
	}
	if _, err := Money(math.MaxInt64).Sub(-1); err == nil {
		t.Fatal("positive money subtraction overflow succeeded")
	}
}

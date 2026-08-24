// Package accounting implements the A01 manual statement and settlement
// contract. Monetary values use integer USD micro-units at Go boundaries and
// DECIMAL(20,6) in MySQL; binary floating point is never used for mutations.
package accounting

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const moneyScale int64 = 1_000_000

// Money is a USD amount in micro-dollars.
type Money int64

func ParseMoney(raw string) (Money, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("money is empty")
	}
	negative := strings.HasPrefix(raw, "-")
	if negative || strings.HasPrefix(raw, "+") {
		raw = raw[1:]
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid money value")
	}
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid money value")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 6 {
		return 0, fmt.Errorf("money supports at most six decimal places")
	}
	for len(fraction) < 6 {
		fraction += "0"
	}
	frac := uint64(0)
	if fraction != "" {
		frac, err = strconv.ParseUint(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid money value")
		}
	}
	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	scale := uint64(moneyScale)
	if whole > limit/scale || whole*scale > limit-frac {
		return 0, fmt.Errorf("money value is out of range")
	}
	magnitude := whole*scale + frac
	if negative {
		if magnitude == uint64(math.MaxInt64)+1 {
			return Money(math.MinInt64), nil
		}
		return -Money(magnitude), nil
	}
	return Money(magnitude), nil
}

func (m Money) String() string {
	value := int64(m)
	negative := value < 0
	var magnitude uint64
	if negative {
		// -(MinInt64) overflows int64, so form the magnitude without first
		// negating the minimum value.
		magnitude = uint64(-(value + 1)) + 1
	} else {
		magnitude = uint64(value)
	}
	formatted := fmt.Sprintf("%d.%06d", magnitude/uint64(moneyScale), magnitude%uint64(moneyScale))
	if negative {
		return "-" + formatted
	}
	return formatted
}

func (m Money) Add(other Money) (Money, error) {
	if (other > 0 && m > Money(math.MaxInt64)-other) ||
		(other < 0 && m < Money(math.MinInt64)-other) {
		return 0, fmt.Errorf("money value is out of range")
	}
	return m + other, nil
}

func (m Money) Sub(other Money) (Money, error) {
	if (other > 0 && m < Money(math.MinInt64)+other) ||
		(other < 0 && m > Money(math.MaxInt64)+other) {
		return 0, fmt.Errorf("money value is out of range")
	}
	return m - other, nil
}

func splitFixed(raw string, places int) (whole uint64, fraction uint64, negative bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, false, fmt.Errorf("value is empty")
	}
	negative = strings.HasPrefix(raw, "-")
	if negative || strings.HasPrefix(raw, "+") {
		raw = raw[1:]
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, 0, false, fmt.Errorf("invalid decimal value")
	}
	whole, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid decimal value")
	}
	fractionRaw := ""
	if len(parts) == 2 {
		fractionRaw = parts[1]
	}
	if len(fractionRaw) > places {
		return 0, 0, false, fmt.Errorf("value supports at most %d decimal places", places)
	}
	for len(fractionRaw) < places {
		fractionRaw += "0"
	}
	if fractionRaw != "" {
		fraction, err = strconv.ParseUint(fractionRaw, 10, 64)
		if err != nil {
			return 0, 0, false, fmt.Errorf("invalid decimal value")
		}
	}
	return whole, fraction, negative, nil
}

func formatFixed(value int64, places int) string {
	negative := value < 0
	var magnitude uint64
	if negative {
		magnitude = uint64(-(value + 1)) + 1
	} else {
		magnitude = uint64(value)
	}
	scale := uint64(1)
	for range places {
		scale *= 10
	}
	formatted := fmt.Sprintf("%d.%0*d", magnitude/scale, places, magnitude%scale)
	if negative {
		return "-" + formatted
	}
	return formatted
}

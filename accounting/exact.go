package accounting

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	// ExactMoneyContract is the version carried by authoritative monetary
	// caches and operational evidence introduced by A03.
	ExactMoneyContract = "usd-cpm-impression-v3"

	// CPMScale is the number of micro-dollars in one USD CPM unit.
	CPMScale int64 = 1_000_000
	// NanoScale is the number of nano-dollars in one USD unit.
	NanoScale int64 = 1_000_000_000

	// MaxCPM is the largest value represented by DECIMAL(12,6).
	MaxCPM CPM = 999_999_999_999
)

// CPM is a USD CPM price in micro-dollars per thousand impressions. One CPM
// unit is also exactly one nano-dollar per impression, so conversion never
// rounds.
type CPM int64

// Nano is an exact USD amount in nano-dollars. Auction reservations and
// ledger aggregation use Nano; statements round to Money exactly once.
type Nano int64

// ParseCPM parses the public six-decimal USD CPM representation.
func ParseCPM(raw string) (CPM, error) {
	value, err := parseFixed(raw, 6)
	if err != nil {
		return 0, fmt.Errorf("invalid USD CPM: %w", err)
	}
	if value < 0 || CPM(value) > MaxCPM {
		return 0, fmt.Errorf("USD CPM is outside 0.000000..%s", MaxCPM.String())
	}
	if value == 0 && strings.HasPrefix(strings.TrimSpace(raw), "-") {
		return 0, fmt.Errorf("USD CPM rejects negative zero")
	}
	return CPM(value), nil
}

func (c CPM) String() string { return formatFixed(int64(c), 6) }

func (c CPM) Float64() float64 { return float64(c) / float64(CPMScale) }

func (c CPM) Float32() float32 { return float32(c.Float64()) }

func (c *CPM) Scan(source any) error {
	raw, err := exactDatabaseText(source)
	if err != nil {
		return err
	}
	value, err := ParseCPM(raw)
	if err != nil {
		return err
	}
	*c = value
	return nil
}

func (c CPM) Value() (driver.Value, error) { return c.String(), nil }

// ImpressionNano converts CPM to one impression's exact USD charge. The
// integer value is unchanged: USD*1e-6/1000 equals USD*1e-9.
func (c CPM) ImpressionNano() (Nano, error) {
	if c < 0 || c > MaxCPM {
		return 0, fmt.Errorf("USD CPM is outside the supported range")
	}
	return Nano(c), nil
}

// ParseNano parses an internal nine-decimal USD aggregation amount.
func ParseNano(raw string) (Nano, error) {
	value, err := parseFixed(raw, 9)
	if err != nil {
		return 0, fmt.Errorf("invalid nano-USD amount: %w", err)
	}
	if value == 0 && strings.HasPrefix(strings.TrimSpace(raw), "-") {
		return 0, fmt.Errorf("nano-USD rejects negative zero")
	}
	return Nano(value), nil
}

func (n Nano) String() string { return formatFixed(int64(n), 9) }

func (n *Nano) Scan(source any) error {
	raw, err := exactDatabaseText(source)
	if err != nil {
		return err
	}
	value, err := ParseNano(raw)
	if err != nil {
		return err
	}
	*n = value
	return nil
}

func (n Nano) Value() (driver.Value, error) { return n.String(), nil }

func (n Nano) Add(other Nano) (Nano, error) {
	if (other > 0 && n > Nano(math.MaxInt64)-other) ||
		(other < 0 && n < Nano(math.MinInt64)-other) {
		return 0, fmt.Errorf("nano-USD amount is out of range")
	}
	return n + other, nil
}

func (n Nano) Sub(other Nano) (Nano, error) {
	if (other > 0 && n < Nano(math.MinInt64)+other) ||
		(other < 0 && n > Nano(math.MaxInt64)+other) {
		return 0, fmt.Errorf("nano-USD amount is out of range")
	}
	return n - other, nil
}

// StatementMoney rounds a nano-dollar aggregate to micro-dollars using
// round-half-away-from-zero. Callers aggregate first and round once.
func (n Nano) StatementMoney() Money {
	quotient := int64(n) / 1_000
	remainder := int64(n) % 1_000
	if remainder >= 500 {
		quotient++
	} else if remainder <= -500 {
		quotient--
	}
	return Money(quotient)
}

// Nano converts statement micro-dollars without loss.
func (m Money) Nano() (Nano, error) {
	if m > Money(math.MaxInt64/1_000) || m < Money(math.MinInt64/1_000) {
		return 0, fmt.Errorf("money value is out of nano-USD range")
	}
	return Nano(int64(m) * 1_000), nil
}

func parseFixed(raw string, places int) (int64, error) {
	// ParseMoney already implements the reviewed sign, scale, and overflow
	// rules. Reuse it directly for six places.
	if places == 6 {
		value, err := ParseMoney(raw)
		return int64(value), err
	}
	if places != 9 {
		return 0, fmt.Errorf("unsupported decimal scale")
	}
	whole, fraction, negative, err := splitFixed(raw, places)
	if err != nil {
		return 0, err
	}
	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	scale := uint64(NanoScale)
	if whole > limit/scale || whole*scale > limit-fraction {
		return 0, fmt.Errorf("value is out of range")
	}
	magnitude := whole*scale + fraction
	if negative {
		if magnitude == uint64(math.MaxInt64)+1 {
			return math.MinInt64, nil
		}
		return -int64(magnitude), nil
	}
	return int64(magnitude), nil
}

func exactDatabaseText(source any) (string, error) {
	switch value := source.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case nil:
		return "", fmt.Errorf("exact monetary value is NULL")
	default:
		return "", fmt.Errorf("exact monetary scan rejects %T", source)
	}
}

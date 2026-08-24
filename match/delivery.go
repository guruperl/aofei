package match

import (
	"fmt"
	"strings"
	"time"

	"github.com/guruperl/aofei/accounting"
)

const (
	DeliveryHoursPerWeek  = 7 * 24
	deliveryScheduleBytes = DeliveryHoursPerWeek / 8
	deliveryTimezoneBytes = 64
	maxDeliveryClockSkew  = 5 * time.Minute
)

const (
	DeliveryPacingFast uint8 = iota
	DeliveryPacingEven
)

// DeliveryBalance is the immutable source snapshot for one campaign or item
// budget. Redis owns request-time reservations; Current* seeds that mutable
// state from the reconciled MySQL ledger.
type DeliveryBalance struct {
	ID               uint32
	LimitSpendNano   accounting.Nano
	LimitImp         uint64
	LimitClick       uint64
	CurrentSpendNano accounting.Nano
	CurrentImp       uint64
	CurrentClick     uint64
}

func (b DeliveryBalance) Limited() bool {
	return b.ID != 0 && (b.LimitSpendNano > 0 || b.LimitImp > 0 || b.LimitClick > 0)
}

func (b DeliveryBalance) Exhausted() bool {
	return b.Limited() && ((b.LimitSpendNano > 0 && b.CurrentSpendNano >= b.LimitSpendNano) ||
		(b.LimitImp > 0 && b.CurrentImp >= b.LimitImp) ||
		(b.LimitClick > 0 && b.CurrentClick >= b.LimitClick))
}

func (b DeliveryBalance) Validate() error {
	if b.LimitSpendNano < 0 {
		return fmt.Errorf("invalid spend limit nano-USD %s", b.LimitSpendNano)
	}
	if b.CurrentSpendNano < 0 {
		return fmt.Errorf("invalid current spend nano-USD %s", b.CurrentSpendNano)
	}
	if b.ID == 0 && (b.LimitSpendNano != 0 || b.LimitImp != 0 || b.LimitClick != 0 || b.CurrentSpendNano != 0 || b.CurrentImp != 0 || b.CurrentClick != 0) {
		return fmt.Errorf("balance values exist without a balance id")
	}
	return nil
}

// DeliveryWindow is an inclusive start/end interval plus an optional 168-hour
// Monday-first weekly calendar. A zero Configured value means every hour is
// eligible, which keeps old cache payloads compatible.
type DeliveryWindow struct {
	StartUnix  int64
	EndUnix    int64
	Schedule   [deliveryScheduleBytes]byte
	Configured uint8
	Pacing     uint8
}

func (w DeliveryWindow) Allows(when time.Time, location *time.Location) bool {
	unix := when.Unix()
	if w.StartUnix != 0 && unix < w.StartUnix {
		return false
	}
	if w.EndUnix != 0 && unix > w.EndUnix {
		return false
	}
	if w.Configured == 0 {
		return true
	}
	local := when.In(location)
	day := (int(local.Weekday()) + 6) % 7
	hour := day*24 + local.Hour()
	return w.Schedule[hour/8]&(1<<uint(hour%8)) != 0
}

func (w *DeliveryWindow) SetWeeklySchedule(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		w.Configured = 0
		w.Schedule = [deliveryScheduleBytes]byte{}
		return nil
	}
	if len(value) != DeliveryHoursPerWeek {
		return fmt.Errorf("weekly schedule has %d hours, want %d", len(value), DeliveryHoursPerWeek)
	}
	w.Configured = 1
	w.Schedule = [deliveryScheduleBytes]byte{}
	for hour, bit := range []byte(value) {
		switch bit {
		case '0':
		case '1':
			w.Schedule[hour/8] |= 1 << uint(hour%8)
		default:
			return fmt.Errorf("weekly schedule hour %d has invalid value %q", hour, bit)
		}
	}
	return nil
}

func (w DeliveryWindow) WeeklySchedule() string {
	if w.Configured == 0 {
		return ""
	}
	value := make([]byte, DeliveryHoursPerWeek)
	for hour := range value {
		value[hour] = '0'
		if w.Schedule[hour/8]&(1<<uint(hour%8)) != 0 {
			value[hour] = '1'
		}
	}
	return string(value)
}

// Delivery is embedded in RAdv cache records but deliberately omitted from
// audit JSON. It is an auction eligibility contract, not a measurement payload.
type Delivery struct {
	GeneratedAtUnix int64
	Timezone        [deliveryTimezoneBytes]byte
	Campaign        DeliveryWindow
	Item            DeliveryWindow
	CampaignTotal   DeliveryBalance
	CampaignDaily   DeliveryBalance
	ItemTotal       DeliveryBalance
	ItemDaily       DeliveryBalance
}

func (d *Delivery) SetTimezone(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "UTC"
	}
	if len(name) >= len(d.Timezone) {
		return fmt.Errorf("delivery timezone %q is too long", name)
	}
	if _, err := time.LoadLocation(name); err != nil {
		return fmt.Errorf("invalid delivery timezone %q: %w", name, err)
	}
	d.Timezone = [deliveryTimezoneBytes]byte{}
	copy(d.Timezone[:], name)
	return nil
}

func (d Delivery) TimezoneName() string {
	name := string(d.Timezone[:])
	if end := strings.IndexByte(name, 0); end >= 0 {
		name = name[:end]
	}
	if name == "" {
		return "UTC"
	}
	return name
}

func (d Delivery) Location() (*time.Location, error) {
	location, err := time.LoadLocation(d.TimezoneName())
	if err != nil {
		return nil, fmt.Errorf("invalid cached delivery timezone %q: %w", d.TimezoneName(), err)
	}
	return location, nil
}

func (d Delivery) Limited() bool {
	return d.CampaignTotal.Limited() || d.CampaignDaily.Limited() ||
		d.ItemTotal.Limited() || d.ItemDaily.Limited()
}

func (d Delivery) HasPolicy() bool {
	return d.GeneratedAtUnix != 0 || d.Limited() ||
		d.Campaign.StartUnix != 0 || d.Campaign.EndUnix != 0 || d.Campaign.Configured != 0 ||
		d.Item.StartUnix != 0 || d.Item.EndUnix != 0 || d.Item.Configured != 0
}

func (d Delivery) EligibleAt(when time.Time, maxAge time.Duration) (bool, string) {
	if !d.HasPolicy() {
		return true, ""
	}
	if d.Campaign.Pacing > DeliveryPacingEven || d.Item.Pacing > DeliveryPacingEven {
		return false, "invalid delivery pacing"
	}
	for _, balance := range []DeliveryBalance{d.CampaignTotal, d.CampaignDaily, d.ItemTotal, d.ItemDaily} {
		if err := balance.Validate(); err != nil {
			return false, err.Error()
		}
	}
	if maxAge > 0 && d.GeneratedAtUnix != 0 && when.Unix()-d.GeneratedAtUnix > int64(maxAge/time.Second) {
		return false, "stale delivery cache"
	}
	if d.GeneratedAtUnix > when.Add(maxDeliveryClockSkew).Unix() {
		return false, "delivery cache timestamp is in the future"
	}
	location, err := d.Location()
	if err != nil {
		return false, err.Error()
	}
	if !d.Campaign.Allows(when, location) {
		return false, "campaign schedule"
	}
	if !d.Item.Allows(when, location) {
		return false, "item schedule"
	}
	for _, balance := range []DeliveryBalance{d.CampaignTotal, d.CampaignDaily, d.ItemTotal, d.ItemDaily} {
		if balance.Exhausted() {
			return false, "cached budget exhausted"
		}
	}
	return true, ""
}

// DailyPacingFraction returns the deterministic fraction of the current UTC
// day that has elapsed. Daily budget state and ledger baselines reset at UTC
// midnight, so pacing deliberately uses the same boundary.
func (d Delivery) DailyPacingFraction(when time.Time) (float64, error) {
	utc := when.UTC()
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	return elapsedFraction(when, start, end), nil
}

// TotalPacingFraction returns elapsed time across the effective campaign/item
// interval. Open-ended delivery cannot be evenly allocated and returns 1.
func (d Delivery) TotalPacingFraction(when time.Time) float64 {
	startUnix := d.Campaign.StartUnix
	if d.Item.StartUnix > startUnix {
		startUnix = d.Item.StartUnix
	}
	endUnix := d.Campaign.EndUnix
	if endUnix == 0 || (d.Item.EndUnix != 0 && d.Item.EndUnix < endUnix) {
		endUnix = d.Item.EndUnix
	}
	if startUnix == 0 || endUnix == 0 || endUnix <= startUnix {
		return 1
	}
	return elapsedFraction(when, time.Unix(startUnix, 0), time.Unix(endUnix, 0))
}

func elapsedFraction(when, start, end time.Time) float64 {
	if !end.After(start) || !when.After(start) {
		return 0
	}
	if !when.Before(end) {
		return 1
	}
	return float64(when.Sub(start)) / float64(end.Sub(start))
}

func (self RAdvs) FilterByDelivery(when time.Time, maxAge time.Duration) RAdvs {
	blocks := make(RAdvs, 0, len(self))
	for _, block := range self {
		if eligible, _ := block.Delivery.EligibleAt(when, maxAge); eligible {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

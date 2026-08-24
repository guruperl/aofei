// Package ledger aggregates win/loss logs into interval and daily ledger tables.
package ledger

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/guruperl/aofei/dsp"
)

const maxLedgerLogLineBytes = 8 * 1024 * 1024

var ErrMissingInput = errors.New("missing input")

type MissingInputError struct {
	Path string
}

func (self *MissingInputError) Error() string {
	return fmt.Sprintf("%v: %s", ErrMissingInput, self.Path)
}

func (self *MissingInputError) Unwrap() error {
	return ErrMissingInput
}

type Ledger struct {
	DB         *sql.DB
	LogWinLoss string
	Interval   int
	Active     time.Time
	Current    int64
	ctx        context.Context
	slots      map[uint32][2]uint32
	creatives  map[uint32][3]uint32
}

type IntervalResult struct {
	Current int64
	Skipped bool
}

type StatisticsResult struct {
	Slots     map[uint32][2]uint32
	Creatives map[uint32][3]uint32
	Imps      map[uint32]map[uint32]int
	Clis      map[uint32]map[uint32]int
	Spend     map[uint32]map[uint32]float64
	Middleman map[middlemanLedgerKey]*middlemanLedgerStats
	Reporting map[reportingLedgerKey]*reportingLedgerStats
}

type middlemanLedgerKey struct {
	BidderID      uint32
	GroupID       uint32
	RouteBidderID uint32
	TargetID      uint32
	AdvID         uint32
	CampaignID    uint32
	ItemID        uint32
	CreativeID    uint32
	PubID         uint32
	SiteID        uint32
	SlotID        uint32
}

type middlemanLedgerStats struct {
	Wins             int
	Losses           int
	Imps             int
	Clis             int
	ChargeSpend      float64
	PaySpend         float64
	MarginSpend      float64
	ForwardOK        int
	ForwardDuplicate int
	ForwardMissing   int
	ForwardError     int
	ForwardHTTPError int
	ForwardInvalid   int
	ForwardNone      int
}

type reportingLedgerKey struct {
	DemandSource      string
	AdvID             uint32
	CampaignID        uint32
	ItemID            uint32
	CreativeID        uint32
	BidderID          uint32
	GroupID           uint32
	RouteBidderID     uint32
	TargetID          uint32
	PubID             uint32
	SiteID            uint32
	SlotID            uint32
	CountryID         uint32
	StateID           uint32
	DeviceOS          uint8
	DeviceType        uint8
	Environment       string
	IntegrationMode   string
	MediaIntent       string
	Placement         string
	RenderContext     string
	RefreshMode       string
	RefreshSeconds    uint16
	AdDensity         string
	TrafficQuality    string
	SourceQuality     string
	ManagementControl string
	SellerType        string
	SellerID          string
}

type reportingLedgerStats struct {
	Wins             int
	Losses           int
	Imps             int
	Clis             int
	SpendUSD         float64
	RevenueUSD       float64
	CostUSD          float64
	MarginUSD        float64
	DownstreamCPMSum float64
	ReturnedCPMSum   float64
	CallbackErrors   int
}

func New(db *sql.DB, dir string, interval int, stamp ...int) (*Ledger, error) {
	return NewContext(context.Background(), db, dir, interval, stamp...)
}

func NewContext(ctx context.Context, db *sql.DB, dir string, interval int, stamp ...int) (*Ledger, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		return nil, fmt.Errorf("ledger interval must be positive")
	}
	var current int64
	if len(stamp) > 0 {
		current = int64(stamp[0])
	} else {
		current = time.Now().Unix()/int64(interval*60) - 1
	}
	active := time.Unix(current*int64(interval*60), 0).UTC()
	myDay := active.Format("2006-01-02 15:04:05")
	err := db.QueryRowContext(ctx, `SELECT 1 FROM ledger_log WHERE timely=?`, myDay).Scan(new(int))
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `
SELECT slot_id, s.site_id, s.pub_id
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slots := make(map[uint32][2]uint32)
	for rows.Next() {
		var slotID, siteID, pubID uint32
		if err := rows.Scan(&slotID, &siteID, &pubID); err != nil {
			return nil, err
		}
		slots[slotID] = [2]uint32{siteID, pubID}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.QueryContext(ctx, `
SELECT creative_id, i.item_id, i.campaign_id, c.adv_id
FROM adv_creative t
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	creatives := make(map[uint32][3]uint32)
	for rows.Next() {
		var creativeID, itemID, campaignID, advID uint32
		if err := rows.Scan(&creativeID, &itemID, &campaignID, &advID); err != nil {
			return nil, err
		}
		creatives[creativeID] = [3]uint32{itemID, campaignID, advID}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &Ledger{
		DB:         db,
		LogWinLoss: dir,
		Interval:   interval,
		Active:     active,
		Current:    current,
		ctx:        ctx,
		slots:      slots,
		creatives:  creatives,
	}, nil
}

func RunInterval(db *sql.DB, dir string, interval int, stamp ...int) (IntervalResult, error) {
	return RunIntervalContext(context.Background(), db, dir, interval, stamp...)
}

func RunIntervalContext(ctx context.Context, db *sql.DB, dir string, interval int, stamp ...int) (IntervalResult, error) {
	l, err := NewContext(ctx, db, dir, interval, stamp...)
	if err != nil {
		return IntervalResult{}, err
	}
	if l == nil {
		return IntervalResult{Skipped: true}, nil
	}
	if err := l.StatisticsToLedger(); err != nil {
		return IntervalResult{Current: l.Current}, err
	}
	return IntervalResult{Current: l.Current}, nil
}

func RunIntervalEmbedded(db *sql.DB, dir string, interval int, stamp ...int) (IntervalResult, error) {
	result, err := RunInterval(db, dir, interval, stamp...)
	if errors.Is(err, ErrMissingInput) {
		result.Skipped = true
		return result, nil
	}
	return result, err
}

func Every(ctx context.Context, every time.Duration, run func(context.Context)) {
	if every <= 0 {
		return
	}
	run(ctx)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run(ctx)
		}
	}
}

func (self *Ledger) Statistics() (map[uint32][2]uint32, map[uint32][3]uint32, map[uint32]map[uint32]int, map[uint32]map[uint32]int, map[uint32]map[uint32]float64, error) {
	stats, err := self.StatisticsAll()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return stats.Slots, stats.Creatives, stats.Imps, stats.Clis, stats.Spend, nil
}

func (self *Ledger) StatisticsAll() (StatisticsResult, error) {
	ctx := self.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	imps := make(map[uint32]map[uint32]int)
	clis := make(map[uint32]map[uint32]int)
	spes := make(map[uint32]map[uint32]float64)
	middleman := make(map[middlemanLedgerKey]*middlemanLedgerStats)
	reporting := make(map[reportingLedgerKey]*reportingLedgerStats)

	name := fmt.Sprintf("%s/winloss.%d", self.LogWinLoss, self.Current)
	fh, err := os.Open(name)
	if err != nil && os.IsNotExist(err) {
		return StatisticsResult{}, &MissingInputError{Path: name}
	} else if err != nil {
		return StatisticsResult{}, err
	}
	defer fh.Close()

	slots := make(map[uint32][2]uint32)
	creatives := make(map[uint32][3]uint32)

	scanner := bufio.NewScanner(fh)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLedgerLogLineBytes)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return StatisticsResult{}, err
		}
		var wl dsp.WinLoss
		if err := json.Unmarshal(scanner.Bytes(), &wl); err != nil {
			return StatisticsResult{}, err
		}
		aggregateMiddlemanLedger(middleman, wl)
		aggregateReportingLedger(reporting, wl)
		slotID := wl.RPub.SlotID
		creativeID := wl.RAdv.CreativeID
		var found bool
		switch wl.Status {
		case dsp.StatusTrackImp:
			if imps[slotID] == nil {
				imps[slotID] = make(map[uint32]int)
			}
			imps[slotID][creativeID] += 1
			if spes[slotID] == nil {
				spes[slotID] = make(map[uint32]float64)
			}
			spes[slotID][creativeID] += CPMToImpressionSpend(float64(wl.RAdv.Cost))
			found = true
		case dsp.StatusTrackClk:
			if clis[slotID] == nil {
				clis[slotID] = make(map[uint32]int)
			}
			clis[slotID][creativeID] += 1
			found = true
		default:
		}
		if found {
			slots[slotID] = self.slots[slotID]
			creatives[creativeID] = self.creatives[creativeID]
		}
	}
	if err := scanner.Err(); err != nil {
		return StatisticsResult{}, err
	}

	return StatisticsResult{
		Slots:     slots,
		Creatives: creatives,
		Imps:      imps,
		Clis:      clis,
		Spend:     spes,
		Middleman: middleman,
		Reporting: reporting,
	}, nil
}

func (self *Ledger) StatisticsToLedger() error {
	stats, err := self.StatisticsAll()
	if err != nil {
		return err
	}
	myDay := self.Active.Format("2006-01-02 15:04:05")
	ctx := self.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return insertLedgerContext(ctx, self.DB, myDay, stats)
}

func aggregateMiddlemanLedger(out map[middlemanLedgerKey]*middlemanLedgerStats, wl dsp.WinLoss) {
	if wl.Middleman == nil {
		return
	}
	meta := wl.Middleman
	key := middlemanLedgerKey{
		BidderID:      meta.BidderID,
		GroupID:       meta.GroupID,
		RouteBidderID: meta.RouteBidderID,
		TargetID:      meta.TargetID,
		AdvID:         wl.RAdv.AdvID,
		CampaignID:    wl.RAdv.CampaignID,
		ItemID:        wl.RAdv.ItemID,
		CreativeID:    wl.RAdv.CreativeID,
		PubID:         wl.RPub.PubID,
		SiteID:        wl.RPub.SiteID,
		SlotID:        wl.RPub.SlotID,
	}
	stats := out[key]
	if stats == nil {
		stats = &middlemanLedgerStats{}
		out[key] = stats
	}
	if middlemanForwardStatusCounts(wl.Status, meta.Source) {
		stats.addForwardStatus(meta.ForwardStatus)
	}
	switch wl.Status {
	case dsp.StatusWin:
		if meta.ForwardStatus != "duplicate" {
			stats.Wins++
		}
	case dsp.StatusLoss:
		if meta.ForwardStatus != "duplicate" {
			stats.Losses++
		}
	case dsp.StatusTrackImp:
		stats.Imps++
		charge := CPMToImpressionSpend(meta.ChargePrice)
		pay := CPMToImpressionSpend(meta.PayPrice)
		stats.ChargeSpend += charge
		stats.PaySpend += pay
		margin := charge - pay
		if margin < 0 {
			margin = 0
		}
		stats.MarginSpend += margin
	case dsp.StatusTrackClk:
		stats.Clis++
	}
}

func aggregateReportingLedger(out map[reportingLedgerKey]*reportingLedgerStats, wl dsp.WinLoss) {
	if out == nil {
		return
	}
	key := reportingLedgerKey{
		DemandSource: "Local",
		AdvID:        wl.RAdv.AdvID, CampaignID: wl.RAdv.CampaignID,
		ItemID: wl.RAdv.ItemID, CreativeID: wl.RAdv.CreativeID,
		PubID: wl.RPub.PubID, SiteID: wl.RPub.SiteID, SlotID: wl.RPub.SlotID,
		Environment: "Unknown", IntegrationMode: "Unknown", MediaIntent: "Unknown", Placement: "Unknown",
		RenderContext: "Unknown", RefreshMode: "Unknown", AdDensity: "Unknown", TrafficQuality: "Unknown",
		SourceQuality: "Unknown", ManagementControl: "Unknown", SellerType: "Unknown",
	}
	if wl.Reporting != nil {
		key.CountryID = wl.Reporting.CountryID
		key.StateID = wl.Reporting.StateID
		key.DeviceOS = wl.Reporting.DeviceOS
		key.DeviceType = wl.Reporting.DeviceType
		key.Environment = wl.Reporting.Environment
		key.IntegrationMode = wl.Reporting.IntegrationMode
		key.MediaIntent = wl.Reporting.MediaIntent
		key.Placement = wl.Reporting.Placement
		key.RenderContext = wl.Reporting.RenderContext
		key.RefreshMode = wl.Reporting.RefreshMode
		key.RefreshSeconds = wl.Reporting.RefreshSeconds
		key.AdDensity = wl.Reporting.AdDensity
		key.TrafficQuality = wl.Reporting.TrafficQuality
		key.SourceQuality = wl.Reporting.SourceQuality
		key.ManagementControl = wl.Reporting.ManagementControl
		key.SellerType = wl.Reporting.SellerType
		key.SellerID = wl.Reporting.SellerID
	}
	if wl.Middleman != nil {
		key.DemandSource = wl.Middleman.TriggerMode
		switch key.DemandSource {
		case "Fallback", "Always":
		default:
			key.DemandSource = "MiddlemanUnknown"
		}
		key.BidderID = wl.Middleman.BidderID
		key.GroupID = wl.Middleman.GroupID
		key.RouteBidderID = wl.Middleman.RouteBidderID
		key.TargetID = wl.Middleman.TargetID
	}
	stats := out[key]
	if stats == nil {
		stats = &reportingLedgerStats{}
		out[key] = stats
	}
	if wl.Middleman != nil && middlemanForwardStatusCounts(wl.Status, wl.Middleman.Source) {
		switch wl.Middleman.ForwardStatus {
		case "", "error", "request_error", "http_error", "invalid_url":
			stats.CallbackErrors++
		}
	}
	switch wl.Status {
	case dsp.StatusWin:
		if wl.Middleman == nil || wl.Middleman.ForwardStatus != "duplicate" {
			stats.Wins++
		}
	case dsp.StatusLoss:
		if wl.Middleman == nil || wl.Middleman.ForwardStatus != "duplicate" {
			stats.Losses++
		}
	case dsp.StatusTrackImp:
		stats.Imps++
		if wl.Middleman == nil {
			amount := CPMToImpressionSpend(float64(wl.RAdv.Cost))
			stats.SpendUSD += amount
			stats.RevenueUSD += amount
			stats.CostUSD += amount
			stats.ReturnedCPMSum += float64(wl.RAdv.Cost)
			return
		}
		charge := CPMToImpressionSpend(wl.Middleman.ChargePrice)
		pay := CPMToImpressionSpend(wl.Middleman.PayPrice)
		stats.SpendUSD += pay
		stats.RevenueUSD += charge
		stats.CostUSD += pay
		margin := charge - pay
		if margin < 0 {
			margin = 0
		}
		stats.MarginUSD += margin
		stats.DownstreamCPMSum += wl.Middleman.PayPrice
		stats.ReturnedCPMSum += wl.Middleman.ChargePrice
	case dsp.StatusTrackClk:
		stats.Clis++
	}
}

func reportingDimensionHash(key reportingLedgerKey) []byte {
	raw, _ := json.Marshal(key)
	digest := sha256.Sum256(raw)
	return digest[:]
}

// CPMToImpressionSpend converts an OpenRTB USD CPM price into the USD amount
// of one billable impression. It is the A01 accounting-v2 unit boundary.
func CPMToImpressionSpend(cpm float64) float64 {
	return cpm / 1000
}

func middlemanForwardStatusCounts(status dsp.Status, source string) bool {
	switch status {
	case dsp.StatusWin, dsp.StatusLoss:
		return true
	case dsp.StatusTrackImp:
		return source == "bill"
	default:
		return false
	}
}

func (s *middlemanLedgerStats) addForwardStatus(status string) {
	switch status {
	case "ok":
		s.ForwardOK++
	case "duplicate":
		s.ForwardDuplicate++
	case "missing":
		s.ForwardMissing++
	case "http_error":
		s.ForwardHTTPError++
	case "invalid_url":
		s.ForwardInvalid++
	case "none":
		s.ForwardNone++
	case "", "error", "request_error":
		s.ForwardError++
	default:
		s.ForwardError++
	}
}

func insertLedger(db *sql.DB, myDay string, stats StatisticsResult) error {
	return insertLedgerContext(context.Background(), db, myDay, stats)
}

func insertLedgerContext(ctx context.Context, db *sql.DB, myDay string, stats StatisticsResult) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insLog := `INSERT INTO ledger_log (timely, created) VALUES (?, NOW())`
	insPub := `INSERT INTO ledger_pub (log_id, slot_id, site_id, pub_id) VALUES (?,?,?,?)`
	insAdv := `INSERT INTO ledger_adv (log_id, creative_id, item_id, campaign_id, adv_id) VALUES (?,?,?,?,?)`
	insPubAdv := `INSERT INTO ledger_pub_adv (lp_id, la_id, imps, clis, spend) VALUES (?,?,?,?,?)`
	insMid := `
INSERT INTO ledger_mid (
	log_id, bidder_id, group_id, route_bidder_id, target_id,
	adv_id, campaign_id, item_id, creative_id, pub_id, site_id, slot_id,
	wins, losses, imps, clis, charge_spend, pay_spend, margin_spend,
	forward_ok, forward_duplicate, forward_missing, forward_error,
	forward_http_error, forward_invalid, forward_none
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	insReport := `
INSERT INTO report_delivery (
	timely, demand_source, adv_id, campaign_id, item_id, creative_id,
	bidder_id, group_id, route_bidder_id, target_id, pub_id, site_id, slot_id,
	country_id, state_id, device_os, device_type,
	inventory_environment, integration_mode, media_intent, placement,
	render_context, refresh_mode, refresh_seconds, ad_density, traffic_quality, source_quality,
	management_control, seller_type, seller_id, dimension_hash,
	wins, losses, imps, clis, spend_usd, revenue_usd, cost_usd, margin_usd,
	downstream_cpm_sum, returned_cpm_sum, callback_errors
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	updPub := `UPDATE ledger_pub lp
INNER JOIN (
	SELECT pa.lp_id AS lp_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_pub p USING (lp_id)
	WHERE p.log_id=?
	GROUP BY pa.lp_id
) tmp ON (lp.lp_id=tmp.lp_id)
SET lp.imps=tmp.imps, lp.clis=tmp.clis, lp.spend=tmp.spend`
	updAdv := `UPDATE ledger_adv la
INNER JOIN (
	SELECT pa.la_id AS la_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_adv a USING (la_id)
	WHERE a.log_id=?
	GROUP BY pa.la_id
) tmp ON (la.la_id=tmp.la_id)
SET la.imps=tmp.imps, la.clis=tmp.clis, la.spend=tmp.spend`
	updLog := `UPDATE ledger_log SET imps=?, clis=?, spend=? WHERE log_id=?`

	var logID int64
	lpIds := make(map[uint32]int64)
	laIds := make(map[uint32]int64)

	var res sql.Result

	if res, err = tx.ExecContext(ctx, insLog, myDay); err != nil {
		return err
	}
	if logID, err = res.LastInsertId(); err != nil {
		return err
	}

	for slotID, ids := range stats.Slots {
		var lpID int64
		if res, err = tx.ExecContext(ctx, insPub, logID, slotID, ids[0], ids[1]); err != nil {
			return err
		}
		if lpID, err = res.LastInsertId(); err != nil {
			return err
		}
		lpIds[slotID] = lpID
	}

	for creativeID, ids := range stats.Creatives {
		var laID int64
		if res, err = tx.ExecContext(ctx, insAdv, logID, creativeID, ids[0], ids[1], ids[2]); err != nil {
			return err
		}
		if laID, err = res.LastInsertId(); err != nil {
			return err
		}
		laIds[creativeID] = laID
	}

	is := 0
	cs := 0
	ss := float64(0)
	for slotID, lpID := range lpIds {
		for creativeID, laID := range laIds {
			i, ok1 := stats.Imps[slotID][creativeID]
			c, ok2 := stats.Clis[slotID][creativeID]
			s, ok3 := stats.Spend[slotID][creativeID]
			if !ok1 && !ok2 && !ok3 {
				continue
			}
			if _, err = tx.ExecContext(ctx, insPubAdv, lpID, laID, i, c, s); err != nil {
				return err
			}
			is += i
			cs += c
			ss += s
		}
	}

	for key, value := range stats.Middleman {
		if _, err = tx.ExecContext(ctx, insMid,
			logID, key.BidderID, key.GroupID, key.RouteBidderID, key.TargetID,
			key.AdvID, key.CampaignID, key.ItemID, key.CreativeID, key.PubID, key.SiteID, key.SlotID,
			value.Wins, value.Losses, value.Imps, value.Clis,
			value.ChargeSpend, value.PaySpend, value.MarginSpend,
			value.ForwardOK, value.ForwardDuplicate, value.ForwardMissing, value.ForwardError,
			value.ForwardHTTPError, value.ForwardInvalid, value.ForwardNone,
		); err != nil {
			return err
		}
	}

	for key, value := range stats.Reporting {
		if _, err = tx.ExecContext(ctx, insReport,
			myDay, key.DemandSource, key.AdvID, key.CampaignID, key.ItemID, key.CreativeID,
			key.BidderID, key.GroupID, key.RouteBidderID, key.TargetID, key.PubID, key.SiteID, key.SlotID,
			key.CountryID, key.StateID, key.DeviceOS, key.DeviceType,
			key.Environment, key.IntegrationMode, key.MediaIntent, key.Placement,
			key.RenderContext, key.RefreshMode, key.RefreshSeconds, key.AdDensity, key.TrafficQuality, key.SourceQuality,
			key.ManagementControl, key.SellerType, key.SellerID, reportingDimensionHash(key),
			value.Wins, value.Losses, value.Imps, value.Clis,
			value.SpendUSD, value.RevenueUSD, value.CostUSD, value.MarginUSD,
			value.DownstreamCPMSum, value.ReturnedCPMSum, value.CallbackErrors,
		); err != nil {
			return err
		}
	}

	if _, err = tx.ExecContext(ctx, updPub, logID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, updAdv, logID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, updLog, is, cs, ss, logID); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `
UPDATE adv_balance b
INNER JOIN (
	SELECT total_balance_id, imps, clis, spend
	FROM ledger_pub l
	INNER JOIN pub p USING (pub_id)
	WHERE l.log_id=?
) tmp ON (b.balance_id=tmp.total_balance_id)
SET b.current_imp=tmp.imps, b.current_cli=tmp.clis, b.current_spend=tmp.spend`, logID); err != nil {
		return err
	}
	if err = reconcileAdvertiserTotalBalancesContext(ctx, tx); err != nil {
		return err
	}
	if err = reconcileAdvertiserDailyBalancesFromIntervalsContext(ctx, tx, myDay); err != nil {
		return err
	}
	return tx.Commit()
}

func reconcileAdvertiserTotalBalances(tx *sql.Tx) error {
	return reconcileAdvertiserTotalBalancesContext(context.Background(), tx)
}

func reconcileAdvertiserTotalBalancesContext(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
UPDATE adv_balance b
INNER JOIN (
	SELECT balance_id, SUM(imps) AS imps, SUM(clis) AS clis, SUM(spend) AS spend
	FROM (
		SELECT c.total_balance_id AS balance_id, la.imps, la.clis, la.spend
		FROM ledger_adv la
		INNER JOIN adv_campaign c USING (campaign_id)
		WHERE c.total_balance_id IS NOT NULL
		UNION ALL
		SELECT i.total_balance_id AS balance_id, la.imps, la.clis, la.spend
		FROM ledger_adv la
		INNER JOIN adv_item i USING (item_id)
		WHERE i.total_balance_id IS NOT NULL
	) facts
	GROUP BY balance_id
) totals ON totals.balance_id=b.balance_id
SET b.current_imp=totals.imps, b.current_cli=totals.clis, b.current_spend=totals.spend, b.current_day=NULL`)
	return err
}

func reconcileAdvertiserDailyBalancesFromIntervals(tx *sql.Tx, myDay string) error {
	return reconcileAdvertiserDailyBalancesFromIntervalsContext(context.Background(), tx, myDay)
}

func reconcileAdvertiserDailyBalancesFromIntervalsContext(ctx context.Context, tx *sql.Tx, myDay string) error {
	_, err := tx.ExecContext(ctx, `
UPDATE adv_balance b
INNER JOIN (
	SELECT balance_id, SUM(imps) AS imps, SUM(clis) AS clis, SUM(spend) AS spend
	FROM (
		SELECT c.daily_balance_id AS balance_id, la.imps, la.clis, la.spend
		FROM ledger_adv la
		INNER JOIN ledger_log ll USING (log_id)
		INNER JOIN adv_campaign c USING (campaign_id)
		WHERE c.daily_balance_id IS NOT NULL AND DATE(ll.timely)=DATE(?)
		UNION ALL
		SELECT i.daily_balance_id AS balance_id, la.imps, la.clis, la.spend
		FROM ledger_adv la
		INNER JOIN ledger_log ll USING (log_id)
		INNER JOIN adv_item i USING (item_id)
		WHERE i.daily_balance_id IS NOT NULL AND DATE(ll.timely)=DATE(?)
	) facts
	GROUP BY balance_id
) totals ON totals.balance_id=b.balance_id
SET b.current_imp=totals.imps, b.current_cli=totals.clis, b.current_spend=totals.spend, b.current_day=DATE(?)`, myDay, myDay, myDay)
	return err
}

func InsertDaily(db *sql.DB, myDays ...string) error {
	return InsertDailyContext(context.Background(), db, myDays...)
}

func InsertDailyContext(ctx context.Context, db *sql.DB, myDays ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var myDay string
	if len(myDays) == 0 {
		myDay = time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	} else {
		myDay = myDays[0]
	}

	err := db.QueryRowContext(ctx, `SELECT 1 FROM daily_log WHERE daily=?`, myDay).Scan(new(int))
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else {
		return nil
	}

	insLog := `
INSERT INTO daily_log (daily, imps, clis, spend, created)
SELECT DATE(timely) AS daily, SUM(imps), SUM(clis), SUM(spend), NOW()
FROM ledger_log
WHERE DATE(timely)=? GROUP BY daily`
	insPub := `
INSERT INTO daily_pub (log_id, slot_id, site_id, pub_id, imps, clis, spend)
SELECT dl.log_id, tmp.slot_id, tmp.site_id, tmp.pub_id, tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT ANY_VALUE(DATE(timely)) AS daily, slot_id, ANY_VALUE(site_id) AS site_id, ANY_VALUE(pub_id) AS pub_id, SUM(p.imps) AS imps, SUM(p.clis) AS clis, SUM(p.spend) AS spend
	FROM ledger_pub p
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(timely)=? GROUP BY slot_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`
	insAdv := `
INSERT INTO daily_adv (log_id, creative_id, item_id, campaign_id, adv_id, imps, clis, spend)
SELECT dl.log_id, tmp.creative_id, tmp.item_id, tmp.campaign_id, tmp.adv_id, tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT ANY_VALUE(DATE(timely)) AS daily, creative_id, ANY_VALUE(item_id) AS item_id, ANY_VALUE(campaign_id) AS campaign_id, ANY_VALUE(adv_id) AS adv_id, SUM(a.imps) AS imps, SUM(a.clis) AS clis, SUM(a.spend) AS spend
	FROM ledger_adv a
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(timely)=? GROUP BY creative_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`
	insPubAdv := `
INSERT INTO daily_pub_adv (lp_id, la_id, imps, clis, spend)
SELECT dp.lp_id, da.la_id, tmp.imps, tmp.clis, tmp.spend
FROM (
	SELECT DATE(timely) AS daily, slot_id, creative_id, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, SUM(pa.spend) AS spend
	FROM ledger_pub_adv pa
	INNER JOIN ledger_pub p USING (lp_id)
	INNER JOIN ledger_adv a USING (la_id)
	INNER JOIN ledger_log l ON (p.log_id=l.log_id)
	WHERE DATE(timely)=? GROUP BY daily, p.slot_id, a.creative_id
) tmp
INNER JOIN daily_pub dp ON (tmp.slot_id=dp.slot_id)
INNER JOIN daily_log ldp ON (dp.log_id=ldp.log_id)
INNER JOIN daily_adv da ON (tmp.creative_id=da.creative_id)
INNER JOIN daily_log lda ON (da.log_id=lda.log_id)
WHERE ldp.daily=? AND lda.daily=?`
	insMid := `
INSERT INTO daily_mid (
	log_id, bidder_id, group_id, route_bidder_id, target_id,
	adv_id, campaign_id, item_id, creative_id, pub_id, site_id, slot_id,
	wins, losses, imps, clis, charge_spend, pay_spend, margin_spend,
	forward_ok, forward_duplicate, forward_missing, forward_error,
	forward_http_error, forward_invalid, forward_none
)
SELECT dl.log_id, tmp.bidder_id, tmp.group_id, tmp.route_bidder_id, tmp.target_id,
	tmp.adv_id, tmp.campaign_id, tmp.item_id, tmp.creative_id, tmp.pub_id, tmp.site_id, tmp.slot_id,
	tmp.wins, tmp.losses, tmp.imps, tmp.clis, tmp.charge_spend, tmp.pay_spend, tmp.margin_spend,
	tmp.forward_ok, tmp.forward_duplicate, tmp.forward_missing, tmp.forward_error,
	tmp.forward_http_error, tmp.forward_invalid, tmp.forward_none
FROM (
	SELECT DATE(l.timely) AS daily,
		m.bidder_id, m.group_id, m.route_bidder_id, m.target_id,
		m.adv_id, m.campaign_id, m.item_id, m.creative_id, m.pub_id, m.site_id, m.slot_id,
		SUM(m.wins) AS wins, SUM(m.losses) AS losses, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
		SUM(m.charge_spend) AS charge_spend, SUM(m.pay_spend) AS pay_spend, SUM(m.margin_spend) AS margin_spend,
		SUM(m.forward_ok) AS forward_ok, SUM(m.forward_duplicate) AS forward_duplicate,
		SUM(m.forward_missing) AS forward_missing, SUM(m.forward_error) AS forward_error,
		SUM(m.forward_http_error) AS forward_http_error, SUM(m.forward_invalid) AS forward_invalid,
		SUM(m.forward_none) AS forward_none
	FROM ledger_mid m
	INNER JOIN ledger_log l USING (log_id)
	WHERE DATE(l.timely)=?
	GROUP BY daily, m.bidder_id, m.group_id, m.route_bidder_id, m.target_id,
		m.adv_id, m.campaign_id, m.item_id, m.creative_id, m.pub_id, m.site_id, m.slot_id
) tmp
INNER JOIN daily_log dl ON (dl.daily=tmp.daily)`

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, insLog, myDay); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, insPub, myDay); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, insAdv, myDay); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, insPubAdv, myDay, myDay, myDay); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, insMid, myDay); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `
UPDATE adv_balance b
INNER JOIN (
	SELECT daily_balance_id, l.imps, l.clis, l.spend
	FROM daily_pub l
	INNER JOIN pub p USING (pub_id)
	INNER JOIN daily_log dl USING (log_id)
	WHERE dl.daily=?
) tmp ON (b.balance_id=tmp.daily_balance_id)
SET b.current_imp=tmp.imps, b.current_cli=tmp.clis, b.current_spend=tmp.spend`, myDay); err != nil {
		return err
	}
	if err = reconcileAdvertiserDailyBalancesFromDailyContext(ctx, tx, myDay); err != nil {
		return err
	}
	return tx.Commit()
}

func reconcileAdvertiserDailyBalancesFromDaily(tx *sql.Tx, myDay string) error {
	return reconcileAdvertiserDailyBalancesFromDailyContext(context.Background(), tx, myDay)
}

func reconcileAdvertiserDailyBalancesFromDailyContext(ctx context.Context, tx *sql.Tx, myDay string) error {
	_, err := tx.ExecContext(ctx, `
UPDATE adv_balance b
INNER JOIN (
	SELECT balance_id, SUM(imps) AS imps, SUM(clis) AS clis, SUM(spend) AS spend
	FROM (
		SELECT c.daily_balance_id AS balance_id, da.imps, da.clis, da.spend
		FROM daily_adv da
		INNER JOIN daily_log dl USING (log_id)
		INNER JOIN adv_campaign c USING (campaign_id)
		WHERE c.daily_balance_id IS NOT NULL AND dl.daily=?
		UNION ALL
		SELECT i.daily_balance_id AS balance_id, da.imps, da.clis, da.spend
		FROM daily_adv da
		INNER JOIN daily_log dl USING (log_id)
		INNER JOIN adv_item i USING (item_id)
		WHERE i.daily_balance_id IS NOT NULL AND dl.daily=?
	) facts
	GROUP BY balance_id
) totals ON totals.balance_id=b.balance_id
SET b.current_imp=totals.imps, b.current_cli=totals.clis, b.current_spend=totals.spend, b.current_day=?`, myDay, myDay, myDay)
	return err
}

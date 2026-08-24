package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type Demand struct {
	AdvID      uint32 `json:"adv_id,omitempty"`
	CampaignID uint32 `json:"campaign_id,omitempty"`
	ItemID     uint32 `json:"item_id,omitempty"`
	CreativeID uint32 `json:"creative_id,omitempty"`
}

// PackString serializes the audience into a RawURL string
func (self Demand) PackString() (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// PackIO packs the Demand object into an IO writer.
func (self Demand) PackIO(w *bytes.Buffer) error {
	return binary.Write(w, binary.LittleEndian, self)
}

// UnpackDemandString deserializes the audience from a RawURL string
func UnpackDemandString(text string) (Demand, error) {
	var demand Demand
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return demand, err
	}
	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, &demand)
	return demand, err
}

// UnpackDemandIO decodes a byte slice from an IO reader into a Demand object.
func UnpackDemandIO(r *bytes.Reader) (Demand, error) {
	var demand Demand
	err := binary.Read(r, binary.LittleEndian, &demand)
	return demand, err
}

// RAdv is the compiled demand record for one slot candidate.
type RAdv struct {
	Demand
	Weight   float32        `json:"weight,omitempty"`
	CostType uint8          `json:"cost_type,omitempty"`
	Cost     float32        `json:"cost,omitempty"` // OpenRTB compatibility projection only.
	CostCPM  accounting.CPM `json:"cost_cpm_micros,omitempty"`
	Cap
	Delivery Delivery `json:"-"`
}

const (
	CostTypeROI uint8 = iota + 1
	CostTypeCPM
	CostTypeCPC
	CostTypeCPA
)

type legacyRAdv struct {
	Demand
	Weight   float32
	CostType uint8
	Cost     float32
	Cap
}

type legacyDeliveryBalanceV2 struct {
	ID           uint32
	LimitSpend   float64
	LimitImp     uint64
	LimitClick   uint64
	CurrentSpend float64
	CurrentImp   uint64
	CurrentClick uint64
}

type legacyDeliveryV2 struct {
	GeneratedAtUnix int64
	Timezone        [deliveryTimezoneBytes]byte
	Campaign        DeliveryWindow
	Item            DeliveryWindow
	CampaignTotal   legacyDeliveryBalanceV2
	CampaignDaily   legacyDeliveryBalanceV2
	ItemTotal       legacyDeliveryBalanceV2
	ItemDaily       legacyDeliveryBalanceV2
}

type legacyRAdvV2 struct {
	Demand
	Weight   float32
	CostType uint8
	Cost     float32
	Cap
	Delivery legacyDeliveryV2
}

type radvWireV3 struct {
	Demand
	Weight   float32
	CostType uint8
	CostCPM  accounting.CPM
	Cap
	Delivery Delivery
}

func (self RAdv) exactCPM() (accounting.CPM, bool) {
	if self.CostType != CostTypeCPM && self.CostType != 0 {
		return 0, false
	}
	if self.CostCPM != 0 {
		if self.CostCPM > 0 && self.CostCPM <= accounting.MaxCPM {
			return self.CostCPM, true
		}
		// A present v3 value is authoritative. It must never be hidden by a
		// plausible legacy compatibility projection.
		return 0, false
	}
	// Headerless/v1/v2 payloads and old in-process callers carry only the
	// protocol float. This bounded adapter is read compatibility; new database
	// and cache writes always carry CostCPM.
	if !finitePositiveFloat32(self.Cost) {
		return 0, false
	}
	cpm, err := accounting.ParseCPM(strconv.FormatFloat(float64(self.Cost), 'f', 6, 32))
	return cpm, err == nil && cpm > 0
}

// ExactCPM returns the authoritative CPM used for billing and signed tracking.
// The float projection is consulted only when no v3 value is present.
func (self RAdv) ExactCPM() (accounting.CPM, bool) { return self.exactCPM() }

func legacySpendNano(value float64) (accounting.Nano, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, fmt.Errorf("invalid legacy spend %v", value)
	}
	return accounting.ParseNano(strconv.FormatFloat(value, 'f', 9, 64))
}

func convertLegacyBalanceV2(value legacyDeliveryBalanceV2) (DeliveryBalance, error) {
	limit, err := legacySpendNano(value.LimitSpend)
	if err != nil {
		return DeliveryBalance{}, err
	}
	current, err := legacySpendNano(value.CurrentSpend)
	if err != nil {
		return DeliveryBalance{}, err
	}
	return DeliveryBalance{
		ID: value.ID, LimitSpendNano: limit, LimitImp: value.LimitImp, LimitClick: value.LimitClick,
		CurrentSpendNano: current, CurrentImp: value.CurrentImp, CurrentClick: value.CurrentClick,
	}, nil
}

func convertLegacyRAdvV2(value legacyRAdvV2) (RAdv, error) {
	cpm, err := accounting.ParseCPM(strconv.FormatFloat(float64(value.Cost), 'f', 6, 32))
	if err != nil {
		return RAdv{}, err
	}
	balances := []legacyDeliveryBalanceV2{
		value.Delivery.CampaignTotal, value.Delivery.CampaignDaily,
		value.Delivery.ItemTotal, value.Delivery.ItemDaily,
	}
	converted := make([]DeliveryBalance, len(balances))
	for index, balance := range balances {
		converted[index], err = convertLegacyBalanceV2(balance)
		if err != nil {
			return RAdv{}, err
		}
	}
	return RAdv{
		Demand: value.Demand, Weight: value.Weight, CostType: value.CostType,
		Cost: value.Cost, CostCPM: cpm, Cap: value.Cap,
		Delivery: Delivery{
			GeneratedAtUnix: value.Delivery.GeneratedAtUnix, Timezone: value.Delivery.Timezone,
			Campaign: value.Delivery.Campaign, Item: value.Delivery.Item,
			CampaignTotal: converted[0], CampaignDaily: converted[1], ItemTotal: converted[2], ItemDaily: converted[3],
		},
	}, nil
}

// UpdatePerRow
func (self RAdv) updateRow(
	cost any,
	capNumber, clickNumber, capPeriod, clickPeriod, capThrottle sql.NullInt64,
	costType sql.NullString) (RAdv, error) {
	switch value := cost.(type) {
	case sql.NullString:
		if value.Valid {
			parsed, err := accounting.ParseCPM(value.String)
			if err != nil {
				return self, err
			}
			self.CostCPM = parsed
			self.Cost = parsed.Float32()
		}
	case sql.NullFloat64:
		if value.Valid {
			if math.IsNaN(value.Float64) || math.IsInf(value.Float64, 0) || value.Float64 < 0 {
				return self, fmt.Errorf("invalid legacy CPM")
			}
			parsed, err := accounting.ParseCPM(strconv.FormatFloat(value.Float64, 'f', 6, 64))
			if err != nil {
				return self, err
			}
			self.CostCPM = parsed
			self.Cost = parsed.Float32()
		}
	default:
		return self, fmt.Errorf("unsupported CPM source %T", cost)
	}
	if capNumber.Valid {
		if capNumber.Int64 < 0 || capNumber.Int64 > int64(^uint8(0)) {
			return self, fmt.Errorf("impression cap number %d is outside uint8 range", capNumber.Int64)
		}
		self.CapNumber = uint8(capNumber.Int64)
	}
	if clickNumber.Valid {
		if clickNumber.Int64 < 0 || clickNumber.Int64 > int64(^uint8(0)) {
			return self, fmt.Errorf("click cap number %d is outside uint8 range", clickNumber.Int64)
		}
		self.ClickNumber = uint8(clickNumber.Int64)
	}
	if capPeriod.Valid {
		if capPeriod.Int64 < 0 || capPeriod.Int64 > int64(^uint16(0)) {
			return self, fmt.Errorf("impression cap period %d is outside uint16 range", capPeriod.Int64)
		}
		self.CapPeriod = uint16(capPeriod.Int64)
	}
	if clickPeriod.Valid {
		if clickPeriod.Int64 < 0 || clickPeriod.Int64 > int64(^uint16(0)) {
			return self, fmt.Errorf("click cap period %d is outside uint16 range", clickPeriod.Int64)
		}
		self.ClickPeriod = uint16(clickPeriod.Int64)
	}
	if capThrottle.Valid {
		if capThrottle.Int64 < 0 || capThrottle.Int64 > int64(^uint16(0)) {
			return self, fmt.Errorf("impression cap throttle %d is outside uint16 range", capThrottle.Int64)
		}
		self.CapThrottle = uint16(capThrottle.Int64)
	}
	if costType.Valid {
		switch costType.String {
		case "ROI":
			self.CostType = CostTypeROI
		case "CPM":
			self.CostType = CostTypeCPM
		case "CPC":
			self.CostType = CostTypeCPC
		case "CPA":
			self.CostType = CostTypeCPA
		default:
		}
	}
	if err := self.Cap.Validate(); err != nil {
		return self, err
	}
	return self, nil
}

type RAdvs []RAdv

// DBGetActiveCreativeSizeIDs returns the active creative sizes that can produce
// slot RAdv cache entries under the same demand-side filters used by proc_slot.
func DBGetActiveCreativeSizeIDs(ctx context.Context, db *sql.DB) ([]uint32, error) {
	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT v.size_id
FROM adv_creative v
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv a USING (adv_id)
WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
ORDER BY v.size_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sizeIDs []uint32
	for rows.Next() {
		var sizeID uint32
		if err := rows.Scan(&sizeID); err != nil {
			return nil, err
		}
		sizeIDs = append(sizeIDs, sizeID)
	}
	return sizeIDs, rows.Err()
}

// Pack packs the creatives to binary.
func (self RAdvs) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := writeCachePayloadHeader(buf, cachePayloadKindRAdvs, cachePayloadVersionRAdvs); err != nil {
		return nil, err
	}
	err := self.packCurrent(buf)
	return buf.Bytes(), err
}

// PackIO packs the RAdvs to an IO writer.
func (self RAdvs) PackIO(w *bytes.Buffer) error {
	if err := writeCachePayloadHeader(w, cachePayloadKindRAdvs, cachePayloadVersionRAdvs); err != nil {
		return err
	}
	return self.packCurrent(w)
}

func (self RAdvs) packCurrent(w io.Writer) error {
	wire := make([]radvWireV3, len(self))
	for index, block := range self {
		cpm, ok := block.exactCPM()
		if !ok {
			return fmt.Errorf("item %d has no exact USD CPM", block.ItemID)
		}
		wire[index] = radvWireV3{
			Demand: block.Demand, Weight: block.Weight, CostType: block.CostType,
			CostCPM: cpm, Cap: block.Cap, Delivery: block.Delivery,
		}
	}
	return binary.Write(w, binary.LittleEndian, wire)
}

// Update updates the current RAdvs with the new RAdv blocks.
// This is used to merge the new results with the existing RAdvs.
func (self RAdvs) Update(newBlocks map[uint32]RAdv) RAdvs {
	var newRAdvs []RAdv
	for _, block := range self {
		key := block.CreativeID
		if newBlock, ok := newBlocks[key]; ok {
			newRAdvs = append(newRAdvs, newBlock)
			delete(newBlocks, key) // remove from newBlocks to avoid duplicates
		} else {
			newRAdvs = append(newRAdvs, block)
		}
	}
	for _, v := range newBlocks {
		newRAdvs = append(newRAdvs, v) // add any new blocks that were not in the original
	}
	return newRAdvs
}

// Delete removes the RAdv with the given creativeID map.
// This is used to remove the RAdvs that are no longer valid.
func (self RAdvs) Delete(invalids map[uint32]RAdv) RAdvs {
	var newRAdvs []RAdv
	for _, block := range self {
		key := block.CreativeID
		if _, ok := invalids[key]; ok {
			continue
		} else {
			newRAdvs = append(newRAdvs, block)
		}
	}
	return newRAdvs
}

// UnpackRAdvs unpacks the weights from binary.
func UnpackRAdvs(data []byte) (RAdvs, error) {
	body, version, err := unpackRAdvsPayload(data)
	if err != nil {
		return nil, err
	}
	return unpackRAdvsBody(body, version)
}

// UnpackRAdvsIO unpacks the RAdvs from an IO reader.
func UnpackRAdvsIO(r io.Reader) (RAdvs, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return UnpackRAdvs(data)
}

func unpackRAdvsLegacy(data []byte) (RAdvs, error) {
	return unpackRAdvsBody(data, 0)
}

func unpackRAdvsPayload(data []byte) ([]byte, uint8, error) {
	headerSize := len(cachePayloadMagic) + 2
	if len(data) < headerSize || !bytes.Equal(data[:len(cachePayloadMagic)], cachePayloadMagic) {
		return data, 0, nil
	}
	kind := data[len(cachePayloadMagic)]
	version := data[len(cachePayloadMagic)+1]
	if kind != cachePayloadKindRAdvs {
		return nil, 0, fmt.Errorf("cache payload kind %d does not match expected kind %d", kind, cachePayloadKindRAdvs)
	}
	if version != 1 && version != 2 && version != cachePayloadVersionRAdvs {
		return nil, 0, fmt.Errorf("unsupported cache payload version %d for kind %d", version, kind)
	}
	return data[headerSize:], version, nil
}

func unpackRAdvsBody(data []byte, version uint8) (RAdvs, error) {
	currentSize := binary.Size(radvWireV3{})
	legacySize := binary.Size(legacyRAdv{})
	if version == cachePayloadVersionRAdvs {
		if currentSize <= 0 || len(data)%currentSize != 0 {
			return nil, fmt.Errorf("invalid RAdvs payload length %d", len(data))
		}
		wire := make([]radvWireV3, len(data)/currentSize)
		if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &wire); err != nil {
			return nil, err
		}
		blocks := make(RAdvs, len(wire))
		for index, block := range wire {
			blocks[index] = RAdv{Demand: block.Demand, Weight: block.Weight, CostType: block.CostType, Cost: block.CostCPM.Float32(), CostCPM: block.CostCPM, Cap: block.Cap, Delivery: block.Delivery}
		}
		return blocks, nil
	}
	if version == 2 {
		legacyV2Size := binary.Size(legacyRAdvV2{})
		if legacyV2Size <= 0 || len(data)%legacyV2Size != 0 {
			return nil, fmt.Errorf("invalid version-2 RAdvs payload length %d", len(data))
		}
		legacy := make([]legacyRAdvV2, len(data)/legacyV2Size)
		if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &legacy); err != nil {
			return nil, err
		}
		blocks := make(RAdvs, len(legacy))
		for index, block := range legacy {
			converted, err := convertLegacyRAdvV2(block)
			if err != nil {
				return nil, err
			}
			blocks[index] = converted
		}
		return blocks, nil
	}
	if legacySize <= 0 || len(data)%legacySize != 0 {
		return nil, fmt.Errorf("invalid RAdvs payload length %d", len(data))
	}
	legacy := make([]legacyRAdv, len(data)/legacySize)
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &legacy); err != nil {
		return nil, err
	}
	blocks := make(RAdvs, len(legacy))
	for i, block := range legacy {
		blocks[i] = RAdv{Demand: block.Demand, Weight: block.Weight, CostType: block.CostType, Cost: block.Cost, Cap: block.Cap}
	}
	return blocks, nil
}

// DBGetRAdvsToRedisSpreadByItemID retrieves RAdvs from the database with give item and inserts them into Redis.
func DBGetRAdvsToRedisSpreadByItemID(ctx context.Context, conn any, db *sql.DB, itemID uint32, top ...string) error {
	return dbRAdvsToRedisSpreadByItemID(ctx, "Get", conn, db, itemID, top...)
}

// DBDeleteRAdvsToRedisSpreadByItemID retrieves RAdvs from the database with give item and inserts them into Redis.
func DBDeleteRAdvsToRedisSpreadByItemID(ctx context.Context, conn any, db *sql.DB, itemID uint32, top ...string) error {
	// This function is used to delete the RAdvs from Redis by itemID
	return dbRAdvsToRedisSpreadByItemID(ctx, "Delete", conn, db, itemID, top...)
}

func dbRAdvsToRedisSpreadByItemID(ctx context.Context, how string, conn any, db *sql.DB, itemID uint32, top ...string) error {
	if how != "Get" && how != "Delete" {
		return nil
	}
	sink, err := CacheSinkFor(conn)
	if err != nil {
		return err
	}

	rows, err := db.QueryContext(ctx, `
	SELECT DISTINCT size_id
	FROM adv_creative
	WHERE item_id=?`, itemID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var sizeIDs []uint32
	for rows.Next() {
		var sizeID uint32
		if err = rows.Scan(&sizeID); err != nil {
			return err
		}
		sizeIDs = append(sizeIDs, sizeID)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	for _, sizeID := range sizeIDs {
		hash, err := dbRAdvsBySizeID(ctx, db, sizeID)
		if err != nil {
			return err
		}
		if err = radvHashToCacheSinkBySizeID(ctx, sink, hash, sizeID); err != nil {
			return err
		}
	}

	return nil
}

// DBGetRAdvsToRedisSpreadBySizeID retrieves RAdvs from the database with given size and inserts them into Redis.
func DBGetRAdvsToRedisSpreadBySizeID(ctx context.Context, conn interface{}, db *sql.DB, sizeID uint32) error {
	hash, err := dbRAdvsBySizeID(ctx, db, sizeID)
	if err != nil {
		return err
	}
	sink, err := CacheSinkFor(conn)
	if err != nil {
		return err
	}
	return radvHashToCacheSinkBySizeID(ctx, sink, hash, sizeID)
}

func dbRAdvsBySizeID(ctx context.Context, db *sql.DB, sizeID uint32) (map[uint32]RAdvs, error) {
	rows, err := db.QueryContext(ctx, `
	SELECT t.slot_id
	FROM pub_slot t
	INNER JOIN pub_site s USING (site_id)
	INNER JOIN pub      p USING (pub_id)
	WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hash := make(map[uint32]RAdvs)
	for rows.Next() {
		var slotID uint32
		err = rows.Scan(&slotID)
		if err != nil {
			return nil, err
		}
		hash[slotID], err = dbRAdvsBySizeIDSlotID(ctx, db, sizeID, slotID)
		if err != nil {
			return nil, err
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return hash, nil
}

func dbRAdvsBySizeIDSlotID(ctx context.Context, db *sql.DB, sizeID, slotID uint32) (RAdvs, error) {
	rows, err := db.QueryContext(ctx, `
	CALL proc_slotall(?, ?)`, slotID, sizeID)
	if err != nil {
		return nil, err
	}

	var blocks RAdvs
	for rows.Next() {
		w := RAdv{}
		var costType sql.NullString
		var capNumber, capPeriod, capThrottle, clickNumber, clickPeriod sql.NullInt64
		var cost sql.NullString
		var itemStart, itemEnd any
		err = rows.Scan(&w.AdvID, &w.CampaignID, &w.ItemID, &w.CreativeID, &w.Weight, &costType, &cost, &capNumber, &capPeriod, &capThrottle, &clickNumber, &clickPeriod, &itemStart, &itemEnd)
		if err != nil {
			rows.Close()
			return nil, err
		}
		w, err = w.updateRow(cost, capNumber, clickNumber, capPeriod, clickPeriod, capThrottle, costType)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("item %d has invalid frequency-cap configuration: %w", w.ItemID, err)
		}
		if _, ok := w.ECPM(); !ok {
			rows.Close()
			return nil, fmt.Errorf("item %d uses unsupported commercial cost type %q or invalid price %v; migrate it to a reviewed positive USD CPM price", w.ItemID, costType.String, cost)
		}
		if !finitePositiveFloat32(w.Weight) {
			rows.Close()
			return nil, fmt.Errorf("creative %d has invalid rotation weight %v", w.CreativeID, w.Weight)
		}
		blocks = append(blocks, w)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	for rows.NextResultSet() {
		for rows.Next() {
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return hydrateRAdvDeliveries(ctx, db, blocks)
}

type deliveryBalanceRow struct {
	id               uint32
	limitSpendNano   accounting.Nano
	limitImp         uint64
	limitClick       uint64
	currentSpendNano accounting.Nano
	currentImp       uint64
	currentClick     uint64
}

func (r *deliveryBalanceRow) scanArgs() []any {
	return []any{&r.id, &r.limitSpendNano, &r.limitImp, &r.limitClick, &r.currentSpendNano, &r.currentImp, &r.currentClick}
}

func (r deliveryBalanceRow) value() DeliveryBalance {
	return DeliveryBalance{
		ID:               r.id,
		LimitSpendNano:   r.limitSpendNano,
		LimitImp:         r.limitImp,
		LimitClick:       r.limitClick,
		CurrentSpendNano: r.currentSpendNano,
		CurrentImp:       r.currentImp,
		CurrentClick:     r.currentClick,
	}
}

func hydrateRAdvDeliveries(ctx context.Context, db *sql.DB, blocks RAdvs) (RAdvs, error) {
	if len(blocks) == 0 {
		return blocks, nil
	}
	itemIDs := make([]uint32, 0, len(blocks))
	seen := make(map[uint32]struct{}, len(blocks))
	for _, block := range blocks {
		if _, ok := seen[block.ItemID]; ok {
			continue
		}
		seen[block.ItemID] = struct{}{}
		itemIDs = append(itemIDs, block.ItemID)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(itemIDs)), ",")
	query := `
SELECT i.item_id, c.campaign_id,
  COALESCE(TIMESTAMPDIFF(SECOND, '1970-01-01 00:00:00', c.startx), 0), COALESCE(TIMESTAMPDIFF(SECOND, '1970-01-01 00:00:00', c.endx), 0),
  COALESCE(TIMESTAMPDIFF(SECOND, '1970-01-01 00:00:00', i.startx), 0), COALESCE(TIMESTAMPDIFF(SECOND, '1970-01-01 00:00:00', i.endx), 0),
  COALESCE(c.delivery_timezone, 'UTC'), COALESCE(c.weekly_schedule, ''), COALESCE(c.pacing_mode, 'Fast'),
  COALESCE(i.weekly_schedule, ''), COALESCE(i.pacing_mode, 'Fast'),
  COALESCE(ct.balance_id, 0), COALESCE(ct.limit_spend, 0), COALESCE(ct.limit_imp, 0), COALESCE(ct.limit_cli, 0), COALESCE(ct.current_spend, 0), COALESCE(ct.current_imp, 0), COALESCE(ct.current_cli, 0),
  COALESCE(cd.balance_id, 0), COALESCE(cd.limit_spend, 0), COALESCE(cd.limit_imp, 0), COALESCE(cd.limit_cli, 0), IF(cd.current_day=UTC_DATE(), COALESCE(cd.current_spend, 0), 0), IF(cd.current_day=UTC_DATE(), COALESCE(cd.current_imp, 0), 0), IF(cd.current_day=UTC_DATE(), COALESCE(cd.current_cli, 0), 0),
  COALESCE(it.balance_id, 0), COALESCE(it.limit_spend, 0), COALESCE(it.limit_imp, 0), COALESCE(it.limit_cli, 0), COALESCE(it.current_spend, 0), COALESCE(it.current_imp, 0), COALESCE(it.current_cli, 0),
  COALESCE(id.balance_id, 0), COALESCE(id.limit_spend, 0), COALESCE(id.limit_imp, 0), COALESCE(id.limit_cli, 0), IF(id.current_day=UTC_DATE(), COALESCE(id.current_spend, 0), 0), IF(id.current_day=UTC_DATE(), COALESCE(id.current_imp, 0), 0), IF(id.current_day=UTC_DATE(), COALESCE(id.current_cli, 0), 0)
FROM adv_item i
INNER JOIN adv_campaign c USING (campaign_id)
LEFT JOIN adv_balance ct ON ct.balance_id=c.total_balance_id
LEFT JOIN adv_balance cd ON cd.balance_id=c.daily_balance_id
LEFT JOIN adv_balance it ON it.balance_id=i.total_balance_id
LEFT JOIN adv_balance id ON id.balance_id=i.daily_balance_id
WHERE i.item_id IN (` + placeholders + `)`
	args := make([]any, len(itemIDs))
	for i, itemID := range itemIDs {
		args[i] = itemID
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deliveries := make(map[uint32]Delivery, len(itemIDs))
	generatedAt := time.Now().Unix()
	for rows.Next() {
		var itemID, campaignID uint32
		var campaignStart, campaignEnd, itemStart, itemEnd int64
		var timezone, campaignSchedule, campaignPacing, itemSchedule, itemPacing string
		var campaignTotal, campaignDaily, itemTotal, itemDaily deliveryBalanceRow
		scan := []any{&itemID, &campaignID, &campaignStart, &campaignEnd, &itemStart, &itemEnd, &timezone, &campaignSchedule, &campaignPacing, &itemSchedule, &itemPacing}
		scan = append(scan, campaignTotal.scanArgs()...)
		scan = append(scan, campaignDaily.scanArgs()...)
		scan = append(scan, itemTotal.scanArgs()...)
		scan = append(scan, itemDaily.scanArgs()...)
		if err := rows.Scan(scan...); err != nil {
			return nil, err
		}
		campaignPacingValue, err := parseDeliveryPacing(campaignPacing)
		if err != nil {
			return nil, fmt.Errorf("campaign %d: %w", campaignID, err)
		}
		itemPacingValue, err := parseDeliveryPacing(itemPacing)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", itemID, err)
		}
		delivery := Delivery{
			GeneratedAtUnix: generatedAt,
			Campaign:        DeliveryWindow{StartUnix: campaignStart, EndUnix: campaignEnd, Pacing: campaignPacingValue},
			Item:            DeliveryWindow{StartUnix: itemStart, EndUnix: itemEnd, Pacing: itemPacingValue},
			CampaignTotal:   campaignTotal.value(),
			CampaignDaily:   campaignDaily.value(),
			ItemTotal:       itemTotal.value(),
			ItemDaily:       itemDaily.value(),
		}
		for _, named := range []struct {
			name    string
			balance DeliveryBalance
		}{
			{name: "campaign total", balance: delivery.CampaignTotal},
			{name: "campaign daily", balance: delivery.CampaignDaily},
			{name: "item total", balance: delivery.ItemTotal},
			{name: "item daily", balance: delivery.ItemDaily},
		} {
			if err := named.balance.Validate(); err != nil {
				return nil, fmt.Errorf("item %d %s balance: %w", itemID, named.name, err)
			}
		}
		if err := delivery.SetTimezone(timezone); err != nil {
			return nil, fmt.Errorf("item %d: %w", itemID, err)
		}
		if err := delivery.Campaign.SetWeeklySchedule(campaignSchedule); err != nil {
			return nil, fmt.Errorf("campaign %d: %w", campaignID, err)
		}
		if err := delivery.Item.SetWeeklySchedule(itemSchedule); err != nil {
			return nil, fmt.Errorf("item %d: %w", itemID, err)
		}
		deliveries[itemID] = delivery
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range blocks {
		delivery, ok := deliveries[blocks[i].ItemID]
		if !ok {
			return nil, fmt.Errorf("delivery policy missing for item %d", blocks[i].ItemID)
		}
		blocks[i].Delivery = delivery
	}
	return blocks, nil
}

func parseDeliveryPacing(value string) (uint8, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fast":
		return DeliveryPacingFast, nil
	case "even":
		return DeliveryPacingEven, nil
	default:
		return 0, fmt.Errorf("invalid delivery pacing %q", value)
	}
}

// radvHashToRedisSpread the size slot cache, used for the 10 minute refresh.
func radvHashToRedisSpreadBySizeID(ctx context.Context, conn interface{}, hash map[uint32]RAdvs, sizeID uint32) error {
	sink, err := CacheSinkFor(conn)
	if err != nil {
		return err
	}
	return radvHashToCacheSinkBySizeID(ctx, sink, hash, sizeID)
}

func radvHashToCacheSinkBySizeID(ctx context.Context, sink CacheSink, hash map[uint32]RAdvs, sizeID uint32) error {
	if err := sink.ResetRAdvs(ctx, sizeID); err != nil {
		return err
	}
	i := 0
	for slotID, radvs := range hash {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(radvs) == 0 {
			continue
		}
		data, err := radvs.Pack()
		if err != nil {
			return err
		}
		cleanup := i == 0
		if err := sink.PutRAdvs(ctx, sizeID, slotID, data, cleanup); err != nil {
			return err
		}
		i++
	}
	if i == 0 {
		return sink.CleanupRAdvs(ctx, sizeID)
	}
	return nil
}

func publishRAdvsSpreadCleanup(conn *nats.Conn, sizeID uint32) error {
	return conn.Publish(fmt.Sprintf("%s:%d:cleanup", HashNameSlot, sizeID), nil)
}

var HashNameSlot = "slot"

func HashNameRAdvs(sizeID uint32) string {
	return fmt.Sprintf("%s:%d", HashNameSlot, sizeID)
}

func HashIONameRAdvs(slotID uint32) string {
	return fmt.Sprintf("%s/%d", HashNameSlot, slotID)
}

// ToRedis inserts RAdvs into Redis.
func (self RAdvs) ToRedis(ctx context.Context, conn radix.Client, slotID, sizeID uint32) error {
	data, err := self.Pack()
	if err == nil {
		err = newRedisCacheSink(conn).PutRAdvs(ctx, sizeID, slotID, data, false)
	}
	return err
}

// ToSpread publishes the RAdvs to nats
func (self RAdvs) ToSpread(conn *nats.Conn, slotID, sizeID uint32, cleanup ...bool) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	return SpreadCacheSink{Conn: conn}.PutRAdvs(context.Background(), sizeID, slotID, data, len(cleanup) > 0 && cleanup[0])
}

// RAdvsFromRedisBySizeIDSlotID builds RAdvs from redis.
func RAdvsFromRedisBySizeIDSlotID(ctx context.Context, conn radix.Client, sizeID, slotID uint32) (RAdvs, error) {
	key := strconv.FormatUint(uint64(slotID), 10)
	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "HGET", HashNameRAdvs(sizeID), key))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	return UnpackRAdvs(data)
}

// RAdvsFromIOBySizeIDSlotID builds RAdvs from redis.
func RAdvsFromIOBySizeIDSlotID(top string, sizeID, slotID uint32) (RAdvs, error) {
	r, err := os.OpenFile(fmt.Sprintf("%s/%s/%d", top, HashIONameRAdvs(sizeID), slotID), os.O_RDONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No file found
		}
		return nil, err // Error opening file
	}
	defer r.Close()
	return UnpackRAdvsIO(r)
}

// RAdvsFromRedisBySizeID builds RAdvs from redis by sizeID.
func RAdvsFromRedisBySizeID(ctx context.Context, conn radix.Client, sizeID uint32) (map[uint32]RAdvs, error) {
	key := HashNameRAdvs(sizeID)
	var arr []string
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", key))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil // No RAdvs found in Redis
	}
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows // Invalid format
	}
	// Decode each key-value pair into the RAdvs map
	hash := make(map[uint32]RAdvs)
	for i := 0; i < len(arr); i += 2 {
		slotID, err := strconv.ParseUint(arr[i], 10, 32)
		if err != nil {
			return nil, err
		}
		data := []byte(arr[i+1])
		radvs, err := UnpackRAdvs(data)
		if err != nil {
			return nil, err
		}
		hash[uint32(slotID)] = radvs
	}
	return hash, nil
}

// RAdvsFromIOBySizeID builds RAdvs from redis by sizeID.
func RAdvsFromIOBySizeID(top string, sizeID uint32) (map[uint32]RAdvs, error) {
	key := HashIONameRAdvs(sizeID)
	// Open the directory containing the files
	dir := fmt.Sprintf("%s/%s", top, key)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No files found
		}
		return nil, err // Error reading directory
	}
	// Initialize a map to store the RAdvs
	hash := make(map[uint32]RAdvs)
	// Iterate over the files in the directory
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}
		name := file.Name()
		// Parse the slotID from the file name
		slotID, err := strconv.ParseUint(name, 10, 32)
		if err != nil {
			continue // Skip files with invalid names
		}
		// Open the file for reading
		radvs, err := RAdvsFromIOBySizeIDSlotID(top, sizeID, uint32(slotID))
		if err != nil {
			return nil, err // Error unpacking RAdvs
		}
		if radvs == nil {
			continue // Skip files that returned nil RAdvs
		}
		// Store the RAdvs in the hash map using the slotID as the key
		hash[uint32(slotID)] = radvs
	}
	if len(hash) == 0 {
		return nil, nil // No RAdvs found in the directory
	}
	return hash, nil // Return the populated hash map
}

func (self RAdvs) capItemIDs() []string {
	var ids []string
	for _, block := range self {
		if block.Cap.CapNumber == 0 && block.Cap.CapThrottle == 0 && block.Cap.ClickNumber == 0 {
			continue
		}
		ids = append(ids, fmt.Sprintf("%d", block.ItemID))
	}
	return ids
}

func (self RAdvs) FilterByCaps(ctx context.Context, conn radix.Client, when time.Time, pid string) (RAdvs, map[uint32]BothCap, error) {
	// Partition candidates up front: an invalid cap configuration is excluded
	// (and counted) without failing the valid candidates in the same slot or
	// reading their cap state from Redis.
	valid := make(RAdvs, 0, len(self))
	for _, block := range self {
		if err := block.Cap.Validate(); err != nil {
			metricInvalidCapCandidates.Add(1)
			continue
		}
		valid = append(valid, block)
	}
	if len(valid) == 0 {
		return nil, nil, nil
	}
	itemIDs := valid.capItemIDs()
	bothcaps, err := BothCapsFromRedis(ctx, conn, pid, itemIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(bothcaps) == 0 {
		return valid, nil, nil
	}

	var blocks []RAdv
	for _, block := range valid {
		bothcap, ok := bothcaps[block.ItemID]
		if !ok {
			blocks = append(blocks, block)
			continue
		}
		if block.Cap.CanServe(when, bothcap) { //do we need denied list?
			blocks = append(blocks, block)
		}
	}

	if len(blocks) == 0 {
		return blocks, nil, nil
	}
	return blocks, bothcaps, nil
}

// FilterByAudiences filters RAdvs by audiences from Redis.
func (self RAdvs) FilterByAudiences(ctx context.Context, conn radix.Client, bid *openrtb2.BidRequest, audiences Audiences, attr *Attribute) (RAdvs, Audiences, error) {
	bools := audiences.Match(attr)

	for i, candidate := range self {
		if !bools[i] {
			continue
		}
		aud := audiences[i]
		if aud == nil || aud.UploadAudience == nil || aud.UploadAudience.Uploads == 0 {
			continue
		}
		ok, err := aud.UploadAudience.Has(ctx, conn, bid, candidate.AdvID)
		if err != nil {
			return nil, nil, err
		}
		bools[i] = ok
	}

	var blocks RAdvs
	var trueAudiences []*Audience
	for i, block := range self {
		if bools[i] {
			blocks = append(blocks, block)
			trueAudiences = append(trueAudiences, audiences[i])
			continue
		}
	}
	return blocks, trueAudiences, nil
}

func (self RAdv) ECPM() (float32, bool) {
	cpm, ok := self.exactCPM()
	if !ok {
		return 0, false
	}
	return cpm.Float32(), true
}

// ImpressionSpendNano returns the authoritative per-impression charge. The
// OpenRTB float projection is consulted only for bounded legacy payloads.
func (self RAdv) ImpressionSpendNano() (accounting.Nano, bool) {
	cpm, ok := self.exactCPM()
	if !ok {
		return 0, false
	}
	spend, err := cpm.ImpressionNano()
	return spend, err == nil
}

func (self RAdv) GetItemWeight(bidFloor float64, bidFoorCur string) (float32, bool) {
	if !supportedBidFloorCurrency(bidFoorCur) || math.IsNaN(bidFloor) || math.IsInf(bidFloor, 0) || bidFloor < 0 {
		return 0.0, false
	}
	cpm, ok := self.ECPM()
	if ok && cpm >= float32(bidFloor) {
		return cpm, true
	}
	return 0.0, false
}

func (self RAdvs) PickIndex(bidFloor float64, bidFoorCur string) int {
	index, _ := self.PickIndexPrice(bidFloor, bidFoorCur)
	return index
}

func (self RAdvs) PickIndexPrice(bidFloor float64, bidFoorCur string) (int, float32) {
	return self.pickIndexPriceAt(bidFloor, bidFoorCur, rand.Float32())
}

// pickIndexPriceAt applies the commercial auction contract with a caller-
// supplied point in [0,1). Price selects the winning demand unit first;
// creative weight is used only to rotate creatives inside that unit.
func (self RAdvs) pickIndexPriceAt(bidFloor float64, bidFloorCur string, point float32) (int, float32) {
	bestPrice := float32(0)
	best := RAdv{}
	found := false
	for _, candidate := range self {
		price, ok := candidate.GetItemWeight(bidFloor, bidFloorCur)
		if !ok || !finitePositiveFloat32(candidate.Weight) {
			continue
		}
		if !found || price > bestPrice || (price == bestPrice && demandUnitLess(candidate, best)) {
			bestPrice = price
			best = candidate
			found = true
		}
	}
	if !found {
		return -1, 0
	}

	weights := make([]float32, len(self))
	total := float32(0)
	lastPositive := -1
	for i, candidate := range self {
		price, ok := candidate.GetItemWeight(bidFloor, bidFloorCur)
		if !ok || price != bestPrice || !sameDemandUnit(candidate, best) || !finitePositiveFloat32(candidate.Weight) {
			continue
		}
		weights[i] = candidate.Weight
		total += candidate.Weight
		lastPositive = i
	}
	if total <= 0 || math.IsNaN(float64(total)) || math.IsInf(float64(total), 0) {
		return -1, 0
	}
	if point < 0 {
		point = 0
	}
	if point >= 1 {
		point = math.Nextafter32(1, 0)
	}
	return selectOneAt(weights, point*total, lastPositive), bestPrice
}

func sameDemandUnit(a, b RAdv) bool {
	return a.AdvID == b.AdvID && a.CampaignID == b.CampaignID && a.ItemID == b.ItemID
}

func demandUnitLess(a, b RAdv) bool {
	if a.CampaignID != b.CampaignID {
		return a.CampaignID < b.CampaignID
	}
	if a.ItemID != b.ItemID {
		return a.ItemID < b.ItemID
	}
	return a.AdvID < b.AdvID
}

func finitePositiveFloat32(value float32) bool {
	return value > 0 && !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func selectOne(weights []float32) int {
	total := float32(0.0)
	lastPositive := -1
	for i, weight := range weights {
		if weight > 0 {
			total += weight
			lastPositive = i
		}
	}
	if total <= 0 {
		return -1
	}
	return selectOneAt(weights, rand.Float32()*total, lastPositive)
}

func selectOneAt(weights []float32, point float32, lastPositive int) int {
	sump := float32(0.0)
	for i, weight := range weights {
		if weight <= 0 {
			continue
		}
		sump += weight
		if sump > point {
			return i
		}
	}
	return lastPositive
}

func supportedBidFloorCurrency(cur string) bool {
	cur = strings.TrimSpace(strings.ToUpper(cur))
	return cur == "" || cur == "USD"
}

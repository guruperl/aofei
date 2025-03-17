package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/mediocregopher/radix/v4"
)

// RAdv is the block of the slot.
type RAdv struct {
	AdvID      uint32
	CampaignID uint32
	ItemID     uint32
	CreativeID uint32
	Weight     float32
	CostType   uint8
	Cost       float32
	Cap
}

type RAdvs []RAdv

// Pack packs the creatives to binary.
func (self RAdvs) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// UnpackRAdvs unpacks the weights from binary.
func UnpackRAdvs(data []byte) (RAdvs, error) {
	n := len(data) / 32
	blocks := make([]RAdv, n)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &blocks)
	return blocks, err
}

// DBGetRAdvs builds slots' block map, according to what = 'App' or 'Web'
func DBGetRAdvs(ctx context.Context, db *sql.DB, conn radix.Client, what string) (map[uint32]RAdvs, error) {
	hash := make(map[uint32]RAdvs)
	rows, err := db.QueryContext(ctx, `
SELECT slot_id, creative_id, weight, item_id, campaign_id, adv_id, cost_type, cost, cpm_fc, cpm_length, cpm_throttle, cpc_fc, cpc_length
FROM ViewRedis`+what)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		w := RAdv{}
		var slotID uint32
		var costType sql.NullString
		var capNumber, capPeriod, capThrottle, clickNumber, clickPeriod sql.NullInt64
		var cost sql.NullFloat64
		err = rows.Scan(&slotID, &w.CreativeID, &w.Weight, &w.ItemID, &w.CampaignID, &w.AdvID, &costType, &cost, &capNumber, &capPeriod, &capThrottle, &clickNumber, &clickPeriod)
		if err != nil {
			return nil, err
		}
		if cost.Valid {
			w.Cost = float32(cost.Float64)
		}
		if capNumber.Valid {
			w.CapNumber = uint8(capNumber.Int64)
		}
		if clickNumber.Valid {
			w.ClickNumber = uint8(clickNumber.Int64)
		}
		if capPeriod.Valid {
			w.CapPeriod = uint16(capPeriod.Int64)
		}
		if clickPeriod.Valid {
			w.ClickPeriod = uint16(clickPeriod.Int64)
		}
		if capThrottle.Valid {
			w.CapThrottle = uint16(capThrottle.Int64)
		}
		if costType.Valid {
			switch costType.String {
			case "ROI":
				w.CostType = 1
			case "CPM":
				w.CostType = 2
			case "CPC":
				w.CostType = 3
			case "CPA":
				w.CostType = 4
			default:
			}
		}
		hash[slotID] = append(hash[slotID], w)
	}

	return hash, rows.Err()
}

func HashNameRAdvs(what string) string {
	return "slot" + what
}

// ToRedis inserts RAdvs into Redis.
func (self RAdvs) ToRedis(ctx context.Context, conn radix.Client, what string, slotID uint32) error {
	key := strconv.FormatUint(uint64(slotID), 10)
	data, err := self.Pack()
	if err == nil {
		err = conn.Do(ctx, radix.Cmd(nil, "HSET", HashNameRAdvs(what), key, string(data)))
	}
	return err
}

// RAdvsFromRedis builds RAdvs from redis.
func RAdvsFromRedis(ctx context.Context, conn radix.Client, what string, slotID uint32) (RAdvs, error) {
	key := strconv.FormatUint(uint64(slotID), 10)
	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "HGET", HashNameRAdvs(what), key))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	return UnpackRAdvs(data)
}

func (self RAdvs) GetItemIDs() []string {
	var ids []string
	for _, block := range self {
		if block.Cap.CapNumber == 0 && block.Cap.ClickNumber == 0 {
			continue
		}
		ids = append(ids, strconv.FormatUint(uint64(block.ItemID), 10))
	}
	return ids
}

func (self RAdvs) FilterByCaps(ctx context.Context, conn radix.Client, when time.Time, pid string) (RAdvs, map[uint32]BothCap, []string, []uint32, error) {
	slotIDs := self.GetItemIDs()
	bothcaps, err := BothCapsFromRedis(ctx, conn, pid, slotIDs)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if len(bothcaps) == 0 {
		return self, nil, nil, nil, nil
	}

	var blocks []RAdv
	var expired []string
	var denied []uint32
	for _, block := range self {
		bothcap, ok := bothcaps[block.ItemID]
		if !ok {
			blocks = append(blocks, block)
			continue
		}
		if block.Cap.CanServe(when, bothcap) {
			blocks = append(blocks, block)
		} else {
			denied = append(denied, block.ItemID)
		}
		if !block.Cap.ValidPeriodImp(when, bothcap.Imp) {
			expired = append(expired, fmt.Sprintf("%d", block.ItemID))
		}
	}
	return blocks, bothcaps, expired, denied, nil
}

func (self RAdvs) FilterByAudiences(ctx context.Context, conn radix.Client, attr *Attribute) (RAdvs, Audiences, error) {
	audiences, err := AudiencesFromRedis(ctx, conn, self.GetItemIDs())
	if err != nil {
		return nil, nil, err
	}
	bools := audiences.Match(attr)

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

func (self RAdv) GetItemWeight(bidFloor float64, bidFoorCur string) (float32, bool) {
	var cpm float32
	switch self.CostType {
	case 1:
		cpm = 100.0 * self.Cost
	case 2:
		cpm = self.Weight
	case 3:
		cpm = 100.0 * self.Cost
	}
	if cpm >= float32(bidFloor) {
		return cpm, true
	}
	return 0.0, false
}

func (self RAdvs) PickIndex(bidFloor float64, bidFoorCur string) int {
	var weights []float32
	for _, block := range self {
		weight, engage := block.GetItemWeight(bidFloor, bidFoorCur)
		if engage {
			weights = append(weights, weight*block.Weight)
		} else {
			weights = append(weights, 0.0)
		}
	}
	return selectOne(weights)
}

func selectOne(weights []float32) int {
	total := float32(0.0)
	n := len(weights)
	for i := 0; i < n; i++ {
		total += weights[i]
	}
	for i := 0; i < n; i++ {
		weights[i] /= total
	}
	randp := rand.Float32()
	sump := float32(0.0)
	for i := 0; i < n; i++ {
		sump += weights[i]
		if sump > randp {
			return i
		}
	}
	return -1
}

package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"strconv"

	"github.com/mediocregopher/radix/v4"
)

// Block is the block of the slot.
type Block struct {
	CreativeID uint32
	Weight     float32
	ItemID     uint32
	CostType   uint8
	Cost       float32
	Cap
}

// PackBlocks packs the creatives to binary.
func PackBlocks(blocks []Block) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, blocks)
	return buf.Bytes(), err
}

// UnpackBlocks unpacks the weights from binary.
func UnpackBlocks(data []byte) ([]Block, error) {
	n := len(data) / 25
	blocks := make([]Block, n)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, blocks)
	return blocks, err
}

// BuildSlotDB2Redis builds the slot map and write to redis.
func BuildSlotDB2Redis(ctx context.Context, db *sql.DB, conn radix.Client, what string) error {
	hash := make(map[uint32][]Block)
	rows, err := db.QueryContext(ctx, `
SELECT slot_id, creative_id, weight, item_id, cost_type, cost, cpm_fc, cpm_length, cpm_throttle, cpc_fc, cpc_length
FROM ViewRedis`+what)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		w := Block{}
		var slotID uint32
		var costType sql.NullString
		var capNumber, capPeriod, capThrottle, clickNumber, clickPeriod sql.NullInt64
		var cost sql.NullFloat64
		err = rows.Scan(&slotID, &w.CreativeID, &w.Weight, &w.ItemID, &costType, &cost, &capNumber, &capPeriod, &capThrottle, &clickNumber, &clickPeriod)
		if err != nil {
			return err
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
	if err := rows.Err(); err != nil {
		return err
	}

	hashName := "slot" + what
	for slotID, blocks := range hash {
		key := strconv.FormatUint(uint64(slotID), 10)
		data, err := PackBlocks(blocks)
		if err != nil {
			return err
		}
		if err := conn.Do(ctx, radix.Cmd(nil, "HSET", hashName, key, string(data))); err != nil {
			return err
		}
	}
	return nil
}

// BuildSlotRedis builds the slot map from redis.
func BuildSlotRedis(ctx context.Context, conn radix.Client, what string, slotID uint32) ([]Block, error) {
	hashName := "slot" + what
	key := strconv.FormatUint(uint64(slotID), 10)
	var data []byte
	err := conn.Do(ctx, radix.Cmd(&data, "HGET", hashName, key))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	return UnpackBlocks(data)
}

/*
type Slot struct {
	SlotID  uint32
	SizeID  uint32
	Weights []Weight
}

func (self *Slot) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self.SlotID)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, self.SizeID)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, self.Weights)
	return buf.Bytes(), err
}

func UnpackSlot(data []byte) (*Slot, error) {
	var slotid, sizeid uint32
	n := (len(data) - 4) / 28
	weights := make([]Weight, n)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &slotid)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.LittleEndian, &sizeid)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.LittleEndian, weights)
	if err != nil {
		return nil, err
	}
	return &Slot{slotid, sizeid, weights}, nil
}

func DBMakeNWeights(db *sql.DB, slotID uint32) (*Slot, error) {
	slot, err := DBGetNWeights(db, slotID)
	if err != nil || slot != nil {
		return slot, err
	}

	model := new(weight.Model)
	model.DB = db
	model.CurrentTable = "pub_weight"
	model.CurrentKey = "weight_id"
	model.CurrentIDAuto = "weight_id"
	storage := map[string]interface{}{}
	args := url.Values{}
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	extra := []url.Values{{}}
	model.SetDefaults(args, &lists, &other, storage)

	args.Set("slot_id", fmt.Sprintf("%d", slotID))
	err = model.Insupd(extra...)
	if err != nil {
		return nil, err
	}

	slot, err = DBGetNWeights(db, slotID)
	if err != nil {
		return slot, err
	}
	if slot == nil {
		return nil, errors.New("Slot has no weights")
	}
	return slot, err
}

func DBGetNWeights(db *sql.DB, slotID uint32) (*Slot, error) {
	rows, err := db.Query(`
SELECT w.weight_id, w.item_id, w.weight, i.size_id, IF(i.cost_type="CPC", -1.0*i.cost, i.cost) AS cost, UNIX_TIMESTAMP(i.endx), i.qa_mime+0, c.campaign_id,
	cpm_fc, cpm_length, cpm_throttle, cpc_fc, cpc_length
FROM pub_weight w
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE w.slot_id=?`, slotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sizeID uint32
	weights := make([]Weight, 0)
	for rows.Next() {
		w := Weight{}
		var Endx, CapNumber, CapPeriod, CapThrottle, ClickNumber, ClickPeriod sql.NullInt64
		var Price sql.NullFloat64
		err = rows.Scan(&w.WeightID, &w.ItemID, &w.Weight, &sizeID, &Price, &Endx, &w.Mime8, &w.CampaignID, &CapNumber, &CapPeriod, &CapThrottle, &ClickNumber, &ClickPeriod)
		if err != nil {
			return nil, err
		}
		if Price.Valid {
			w.Price = float32(Price.Float64)
		}
		if Endx.Valid {
			w.Endx = uint32(Endx.Int64)
		}
		if CapNumber.Valid {
			w.CapNumber = uint8(CapNumber.Int64)
		}
		if ClickNumber.Valid {
			w.ClickNumber = uint8(ClickNumber.Int64)
		}
		if CapPeriod.Valid {
			w.CapPeriod = uint16(CapPeriod.Int64)
		}
		if ClickPeriod.Valid {
			w.ClickPeriod = uint16(ClickPeriod.Int64)
		}
		if CapThrottle.Valid {
			w.CapThrottle = uint16(CapThrottle.Int64)
		}
		weights = append(weights, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(weights) == 0 {
		return nil, nil
	}

	return &Slot{slotID, sizeID, weights}, nil
}
*/

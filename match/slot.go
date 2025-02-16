package match

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"

	"github.com/genelet/winter/summer/weight"
)

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
	model.Db = db
	model.Current_table = "pub_weight"
	model.Current_key = "weight_id"
	model.Current_id_auto = "weight_id"
	storage := map[string]interface{}{}
	args := url.Values{}
	lists := make([]map[string]interface{}, 0)
	other := make(map[string]interface{})
	extra := []url.Values{{}}
	model.Set_defaults(args, &lists, &other, storage)

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

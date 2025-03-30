package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type Demand struct {
	AdvID      uint32 `json:"adv_id,omitempty"`
	CampaignID uint32 `json:"campaign_id,omitempty"`
	ItemID     uint32 `json:"item_id,omitempty"`
	CreativeID uint32 `json:"creative_id,omitempty"`
}

// RAdv is the block of the slot. it is 33 bytes long.
type RAdv struct {
	Demand
	Weight   float32 `json:"weight,omitempty"`
	CostType uint8   `json:"cost_type,omitempty"`
	Cost     float32 `json:"cost,omitempty"`
	Cap
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

type RAdvs []RAdv

// Pack packs the creatives to binary.
func (self RAdvs) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// PackIO packs the RAdvs to an IO writer.
func (self RAdvs) PackIO(w *bytes.Buffer) error {
	return binary.Write(w, binary.LittleEndian, self)
}

// UnpackRAdvs unpacks the weights from binary.
func UnpackRAdvs(data []byte) (RAdvs, error) {
	n := len(data) / 33
	blocks := make([]RAdv, n)
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &blocks)
	return blocks, err
}

// UnpackRAdvsIO unpacks the RAdvs from an IO reader.
func UnpackRAdvsIO(r io.Reader) (RAdvs, error) {
	// Create a bytes.Reader to read from the io.Reader
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, err // Error reading from the io.Reader
	}
	return UnpackRAdvs(buf.Bytes())
}

// DBGetRAdvsToRedis retrieves RAdvs from the database and inserts them into Redis.
func DBGetRAdvsToRedis(ctx context.Context, conn radix.Client, db *sql.DB, sizeID uint32) error {
	hash, err := dbGetRAdvs(ctx, db, sizeID)
	if err != nil {
		return err
	}
	for slotID, radvs := range hash {
		if err = radvs.ToRedis(ctx, conn, slotID, sizeID); err != nil {
			return err
		}
	}
	return nil
}

// DBGetRAdvsToSpread retrieves RAdvs from the database and publishes them to nats.
func DBGetRAdvsToSpread(ctx context.Context, conn *nats.Conn, db *sql.DB, sizeID uint32) error {
	hash, err := dbGetRAdvs(ctx, db, sizeID)
	if err != nil {
		return err
	}
	for slotID, radvs := range hash {
		if err = radvs.ToSpread(conn, slotID, sizeID); err != nil {
			return err
		}
	}
	return nil
}

// dbGetRAdvs builds slots' block map, according to size
func dbGetRAdvs(ctx context.Context, db *sql.DB, siteID uint32) (map[uint32]RAdvs, error) {
	hash := make(map[uint32]RAdvs)
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

	for rows.Next() {
		var slotID uint32
		err = rows.Scan(&slotID)
		if err != nil {
			return nil, err
		}
		hash[slotID], err = RAdvsFromDatabase(ctx, db, slotID, siteID)
		if err != nil {
			return nil, err
		}
	}
	return hash, rows.Err()
}

func HashNameRAdvs(sizeID uint32) string {
	return fmt.Sprintf("slot:%d", sizeID)
}

// ToRedis inserts RAdvs into Redis.
func (self RAdvs) ToRedis(ctx context.Context, conn radix.Client, slotID, sizeID uint32) error {
	key := strconv.FormatUint(uint64(slotID), 10)
	data, err := self.Pack()
	if err == nil {
		err = conn.Do(ctx, radix.Cmd(nil, "HSET", HashNameRAdvs(sizeID), key, string(data)))
	}
	return err
}

// ToSpread publishes the RAdvs to nats
func (self RAdvs) ToSpread(conn *nats.Conn, slotID, sizeID uint32) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("%s:%d", HashNameRAdvs(sizeID), slotID)
	return conn.Publish(subject, data)
}

// RAdvsFromDatabase builds RAdvs from the database.
func RAdvsFromDatabase(ctx context.Context, db *sql.DB, slotID, sizeID uint32) (RAdvs, error) {
	rows, err := db.QueryContext(ctx, `
CALL proc_slot(?, ?)`, slotID, sizeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks RAdvs
	for rows.Next() {
		w := RAdv{}
		var costType sql.NullString
		var capNumber, capPeriod, capThrottle, clickNumber, clickPeriod sql.NullInt64
		var cost sql.NullFloat64
		err = rows.Scan(&w.AdvID, &w.CampaignID, &w.ItemID, &w.CreativeID, &w.Weight, &costType, &cost, &capNumber, &capPeriod, &capThrottle, &clickNumber, &clickPeriod)
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
		blocks = append(blocks, w)
	}
	return blocks, rows.Err()
}

// RAdvsFromRedisBySizeIDSlotID builds RAdvs from redis.
func RAdvsFromRedisBySizeIDSlotID(ctx context.Context, conn radix.Client, slotID, sizeID uint32) (RAdvs, error) {
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
func RAdvsFromIOBySizeIDSlotID(top string, slotID, sizeID uint32) (RAdvs, error) {
	r, err := os.OpenFile(fmt.Sprintf("%s/%s/%d", top, HashNameRAdvs(sizeID), slotID), os.O_RDONLY, 0644)
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
	key := HashNameRAdvs(sizeID)
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
		radvs, err := RAdvsFromIOBySizeIDSlotID(top, uint32(slotID), sizeID)
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
		if block.Cap.CapNumber == 0 && block.Cap.ClickNumber == 0 {
			continue
		}
		ids = append(ids, fmt.Sprintf("%d", block.ItemID))
	}
	return ids
}

func (self RAdvs) FilterByCaps(ctx context.Context, conn radix.Client, when time.Time, pid string) (RAdvs, map[uint32]BothCap, error) {
	itemIDs := self.capItemIDs()
	bothcaps, err := BothCapsFromRedis(ctx, conn, pid, itemIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(bothcaps) == 0 {
		return self, nil, nil
	}

	var blocks []RAdv
	var expired []string
	//var denied []uint32
	for _, block := range self {
		bothcap, ok := bothcaps[block.ItemID]
		if !ok {
			blocks = append(blocks, block)
			continue
		}
		if !block.Cap.ValidPeriodImp(when, bothcap.Imp) { // cap expired so start over again
			expired = append(expired, fmt.Sprintf("%d", block.ItemID))
			delete(bothcaps, block.ItemID)
			blocks = append(blocks, block)
			continue
		}
		if block.Cap.CanServe(when, bothcap) { //do we need denied list?
			blocks = append(blocks, block)
		}
	}
	if len(expired) > 0 {
		err = BothCapsCleanupExpired(ctx, conn, pid, expired)
	}

	if len(blocks) == 0 {
		return blocks, nil, nil
	}
	return blocks, bothcaps, err
}

func (self RAdvs) FilterByAudiences(ctx context.Context, conn radix.Client, attr *Attribute) (RAdvs, Audiences, error) {
	audiences, err := self.AudiencesFromRedis(ctx, conn)
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

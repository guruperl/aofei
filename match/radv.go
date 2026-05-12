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

// RAdv is the block of the slot. it is 33 bytes long.
type RAdv struct {
	Demand
	Weight   float32 `json:"weight,omitempty"`
	CostType uint8   `json:"cost_type,omitempty"`
	Cost     float32 `json:"cost,omitempty"`
	Cap
}

// UpdatePerRow
func (self RAdv) updateRow(
	cost sql.NullFloat64,
	capNumber, clickNumber, capPeriod, clickPeriod, capThrottle sql.NullInt64,
	costType sql.NullString) RAdv {
	if cost.Valid {
		self.Cost = float32(cost.Float64)
	}
	if capNumber.Valid {
		self.CapNumber = uint8(capNumber.Int64)
	}
	if clickNumber.Valid {
		self.ClickNumber = uint8(clickNumber.Int64)
	}
	if capPeriod.Valid {
		self.CapPeriod = uint16(capPeriod.Int64)
	}
	if clickPeriod.Valid {
		self.ClickPeriod = uint16(clickPeriod.Int64)
	}
	if capThrottle.Valid {
		self.CapThrottle = uint16(capThrottle.Int64)
	}
	if costType.Valid {
		switch costType.String {
		case "ROI":
			self.CostType = 1
		case "CPM":
			self.CostType = 2
		case "CPC":
			self.CostType = 3
		case "CPA":
			self.CostType = 4
		default:
		}
	}
	return self
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
AND (i.startx <= NOW() OR i.startx IS NULL)
AND (i.endx >= NOW() OR i.endx IS NULL)
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
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// PackIO packs the RAdvs to an IO writer.
func (self RAdvs) PackIO(w *bytes.Buffer) error {
	return binary.Write(w, binary.LittleEndian, self)
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
	slotHash := func(hash map[uint32]map[uint32]RAdv, creativeID uint32) error {
		rows, err := db.QueryContext(ctx, `
CALL proc_creative(?)`, creativeID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			w := RAdv{}
			var pubID, siteID, slotID, sizeID uint32
			var costType sql.NullString
			var capNumber, capPeriod, capThrottle, clickNumber, clickPeriod sql.NullInt64
			var cost sql.NullFloat64
			err = rows.Scan(&pubID, &siteID, &slotID, &sizeID, &w.AdvID, &w.CampaignID, &w.ItemID, &w.CreativeID, &w.Weight, &costType, &cost, &capNumber, &capPeriod, &capThrottle, &clickNumber, &clickPeriod)
			if err != nil {
				return err
			}
			if hash[slotID] == nil {
				hash[slotID] = make(map[uint32]RAdv)
			}
			hash[slotID][creativeID] = w.updateRow(cost, capNumber, clickNumber, capPeriod, clickPeriod, capThrottle, costType) // update the row with the latest values
		}
		return rows.Err()
	}

	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT creative_id, size_id
FROM adv_creative v
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv a USING (adv_id)
WHERE item_id=?
AND c.active="Yes" AND a.active="Yes"`, itemID)
	if err != nil {
		return err
	}
	defer rows.Close()

	ref := make(map[uint32]map[uint32]map[uint32]RAdv)
	for rows.Next() {
		var creativeID, sizeID uint32
		if err = rows.Scan(&creativeID, &sizeID); err != nil {
			return err
		}
		if _, ok := ref[sizeID]; !ok {
			ref[sizeID] = make(map[uint32]map[uint32]RAdv)
		}
		if err = slotHash(ref[sizeID], creativeID); err != nil {
			return err
		}
	}

	for sizeID, block := range ref {
		output := make(map[uint32]RAdvs)
		for slotID, hash := range block {
			var radvs RAdvs
			var err error
			switch t := conn.(type) {
			case radix.Client:
				radvs, err = RAdvsFromRedisBySizeIDSlotID(ctx, t, sizeID, slotID)
			case *nats.Conn:
				radvs, err = RAdvsFromIOBySizeIDSlotID(top[0], sizeID, slotID)
			}
			if err != nil {
				return err
			}
			switch how {
			case "Get":
				output[slotID] = radvs.Update(hash)
			case "Delete":
				output[slotID] = radvs.Delete(hash)
			default:
			}
		}
		if len(output) == 0 {
			continue
		}
		if err = radvHashToRedisSpreadBySizeID(ctx, conn, output, sizeID); err != nil {
			return err
		}
	}

	return nil
}

// DBGetRAdvsToRedisSpreadBySizeID retrieves RAdvs from the database with given size and inserts them into Redis.
func DBGetRAdvsToRedisSpreadBySizeID(ctx context.Context, conn interface{}, db *sql.DB, sizeID uint32) error {
	slotSlice := func(sizeID, slotID uint32) (RAdvs, error) {
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
			blocks = append(blocks, w.updateRow(cost, capNumber, clickNumber, capPeriod, clickPeriod, capThrottle, costType))
		}
		return blocks, rows.Err()
	}

	rows, err := db.QueryContext(ctx, `
SELECT t.slot_id
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hash := make(map[uint32]RAdvs)
	for rows.Next() {
		var slotID uint32
		err = rows.Scan(&slotID)
		if err != nil {
			return err
		}
		hash[slotID], err = slotSlice(sizeID, slotID)
		if err != nil {
			return err
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}

	return radvHashToRedisSpreadBySizeID(ctx, conn, hash, sizeID)
}

// radvHashToRedisSpread the size slot cache, used for the 10 minute refresh.
func radvHashToRedisSpreadBySizeID(ctx context.Context, conn interface{}, hash map[uint32]RAdvs, sizeID uint32) error {
	switch t := conn.(type) {
	case radix.Client:
		if err := t.Do(ctx, radix.Cmd(nil, "DEL", HashNameRAdvs(sizeID))); err != nil {
			return err
		}
	case *nats.Conn:
	default:
	}
	i := 0
	for slotID, radvs := range hash {
		if len(radvs) == 0 {
			continue
		}
		var err error
		switch t := conn.(type) {
		case radix.Client:
			err = radvs.ToRedis(ctx, t, slotID, sizeID)
		case *nats.Conn:
			if i == 0 {
				err = radvs.ToSpread(t, slotID, sizeID, true)
			}
			i++
		default:
			err = fmt.Errorf("unsupported connection type: %T", conn)
		}
		if err != nil {
			return err
		}
	}
	return nil
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
	key := strconv.FormatUint(uint64(slotID), 10)
	data, err := self.Pack()
	if err == nil {
		err = conn.Do(ctx, radix.Cmd(nil, "HSET", HashNameRAdvs(sizeID), key, string(data)))
	}
	return err
}

// ToSpread publishes the RAdvs to nats
func (self RAdvs) ToSpread(conn *nats.Conn, slotID, sizeID uint32, cleanup ...bool) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("%s:%d", HashNameRAdvs(sizeID), slotID)
	// cleanup is attached to the subject, to clean up the size directory.
	// this should be assigned only for the 10 minute refresh in radvHashToRedisSpreadBySizeID
	if len(cleanup) > 0 && cleanup[0] {
		subject += "cleanup"
	}
	return conn.Publish(subject, data)
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

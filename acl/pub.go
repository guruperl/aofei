package acl

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/internal/atomicfile"
	"github.com/guruperl/aofei/internal/spreadcache"
	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type Pub struct {
	AccountingVersion string
	PubID             uint32
	Active            bool
	LimitImps         uint32
	CurrentImps       uint32
	DefaultWebSiteID  uint32
	DefaultAppSiteID  uint32
	DefaultWebSlotID  uint32
	DefaultAppSlotID  uint32
	Sites             map[string]uint32
	SiteTypes         map[uint32]SiteType
	Slots             map[uint32]map[string]uint32
	SlotSizes         map[uint32]map[uint32]uint32
	SlotFloors        map[uint32]map[uint32]float64
	SlotFloorCPMs     map[uint32]map[uint32]accounting.CPM
	Seller            SellerMetadata
	SiteSupply        map[uint32]SiteSupplyMetadata
	SlotSupply        map[uint32]map[uint32]SlotSupplyMetadata
}

type publisherMutationDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Pack encodes a Pub object into a byte slice.
func (self *Pub) Pack() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(self)
	return buf.Bytes(), err
}

// PackIO packs a Pub object into a byte slice in IO writer.
func (self *Pub) PackIO(w io.Writer) error {
	enc := gob.NewEncoder(w)
	return enc.Encode(self)
}

// UnpackPub decodes a byte slice into a Pub object.
func UnpackPub(data []byte) (*Pub, error) {
	var p Pub
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&p)
	return &p, err
}

// UnpackPubIO decodes a byte slice from an IO reader into a Pub object.
func UnpackPubIO(r io.Reader) (*Pub, error) {
	var p Pub
	dec := gob.NewDecoder(r)
	err := dec.Decode(&p)
	return &p, err
}

// GetRPub returns three IDs from the bid request.
// This is the main purpose of the Pub object.
func (self *Pub) GetRPub(siteStr, slotStr string, isApp bool) (uint32, uint32, uint32) {
	defaultRPub := func() (uint32, uint32, uint32) {
		if isApp {
			return self.PubID, self.DefaultAppSiteID, self.DefaultAppSlotID
		} else {
			return self.PubID, self.DefaultWebSiteID, self.DefaultWebSlotID
		}
	}

	siteID, ok := self.Sites[siteStr]
	if !ok {
		return defaultRPub()
	}
	slots, ok := self.Slots[siteID]
	if !ok {
		return defaultRPub()
	}
	slotID, ok := slots[slotStr]
	if !ok {
		return defaultRPub()
	}

	return self.PubID, siteID, slotID
}

// SupplyFor returns the public, cache-approved taxonomy for an exact inventory
// tuple. Missing fields from an older gob generation become Unknown values.
func (self *Pub) SupplyFor(siteID, slotID uint32) SupplyMetadata {
	if self == nil {
		return SupplyMetadata{
			Site: SiteSupplyMetadata{}.Normalize(),
			Slot: SlotSupplyMetadata{}.Normalize(),
		}
	}
	metadata := SupplyMetadata{Seller: self.Seller}
	metadata.Site = self.SiteSupply[siteID].Normalize()
	metadata.Slot = self.SlotSupply[siteID][slotID].Normalize()
	if metadata.Seller.Type == "" {
		metadata.Seller.Type = "Publisher"
	}
	if !metadata.Seller.Authorized || metadata.Seller.Validate() != nil {
		metadata.Seller = SellerMetadata{}
	}
	return metadata
}

// ToRedis writes the Pub object to Redis.
func (self *Pub) ToRedis(ctx context.Context, conn radix.Client, domain string) error {
	if self.Active && (self.LimitImps == 0 || self.CurrentImps < self.LimitImps) {
		bs, err := self.Pack()
		if err != nil {
			return err
		}
		if err := conn.Do(ctx, radix.Cmd(nil, "HSET", HashNamePubmap, domain, string(bs))); err != nil {
			return err
		}
		direct, err := NewDirectPub(domain, self).Pack()
		if err != nil {
			return err
		}
		return conn.Do(ctx, radix.Cmd(nil, "HSET", HashNamePubByID, strconv.FormatUint(uint64(self.PubID), 10), string(direct)))
	}
	if err := conn.Do(ctx, radix.Cmd(nil, "HDEL", HashNamePubmap, domain)); err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "HDEL", HashNamePubByID, strconv.FormatUint(uint64(self.PubID), 10)))
}

// ToSpread put Pub to spread
func (self *Pub) ToSpread(conn *nats.Conn, domain string) error {
	var bs []byte
	var err error
	if self.Active && (self.LimitImps == 0 || self.CurrentImps < self.LimitImps) {
		bs, err = self.Pack()
		if err != nil {
			return err
		}
	} else {
		bs = []byte("DELETE")
	}
	return conn.Publish(HashNamePubmap+":"+domain, bs)
}

// SpreadGetPub retrieves Pub from nats
func SpreadGetPub(m *nats.Msg, top string) error {
	if m == nil {
		return errors.New("spread publisher message is nil")
	}
	relative, ok := spreadcache.RelativePath(m.Subject)
	if !ok || !strings.HasPrefix(relative, HashNamePubmap+string(filepath.Separator)) {
		return fmt.Errorf("invalid spread publisher subject %q", m.Subject)
	}
	filename := filepath.Join(top, relative)
	if err := atomicfile.EnsureDir(filepath.Dir(filename), 0750); err != nil {
		return err
	}
	return atomicfile.Write(filename, 0640, func(out io.Writer) error {
		_, err := out.Write(m.Data)
		return err
	})
}

// DBGetPubByID retrieves the Pub from the database using pubID
func DBGetPubByID(db *sql.DB, pubID string) (*Pub, string, error) {
	var domain string
	err := db.QueryRow(`SELECT domain FROM pub WHERE pub_id = ?`, pubID).Scan(&domain)
	if err != nil {
		return nil, "", err
	}
	pub, err := DBGetPub(db, domain)
	return pub, domain, err
}

// DBGetPub retrieves the Pub from the database using domain
func DBGetPub(db *sql.DB, domain string) (*Pub, error) {
	rows, err := db.Query(`
	SELECT p.pub_id, p.active, foreign_id, s.site_id, s.site_type, t.slot_name,
	       t.slot_id, t.size_id, t.bidfloor, b.limit_imp, b.current_imp,
	       p.seller_id, p.seller_type, p.seller_asi, p.seller_name,
	       p.seller_domain, p.seller_authorized,
	       s.inventory_environment, s.canonical_identity, s.store_url, s.integration_mode,
	       t.media_intent, t.placement, t.render_context, t.refresh_mode,
	       t.refresh_seconds, t.ad_density, t.traffic_quality, t.source_quality,
	       t.management_control
	FROM pub p
	INNER JOIN pub_site s USING (pub_id)
	INNER JOIN pub_slot t USING (site_id)
	LEFT JOIN adv_balance b ON (p.total_balance_id=b.balance_id)
	WHERE domain = ? AND s.active='Yes' AND t.active='Yes'`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	p := &Pub{
		AccountingVersion: accounting.ExactMoneyContract,
		Sites:             make(map[string]uint32),
		SiteTypes:         make(map[uint32]SiteType),
		Slots:             make(map[uint32]map[string]uint32),
		SlotSizes:         make(map[uint32]map[uint32]uint32),
		SlotFloors:        make(map[uint32]map[uint32]float64),
		SlotFloorCPMs:     make(map[uint32]map[uint32]accounting.CPM),
		SiteSupply:        make(map[uint32]SiteSupplyMetadata),
		SlotSupply:        make(map[uint32]map[uint32]SlotSupplyMetadata),
	}
	for rows.Next() {
		var pubID, siteID, slotID, sizeID uint32
		var limitImps, currentImps sql.NullInt64
		var bidFloor sql.NullString
		var slotName, foreignID, siteType string
		var sellerAuthorized string
		var seller SellerMetadata
		var siteSupply SiteSupplyMetadata
		var slotSupply SlotSupplyMetadata
		var active sql.NullString
		err = rows.Scan(
			&pubID, &active, &foreignID, &siteID, &siteType, &slotName, &slotID, &sizeID, &bidFloor, &limitImps, &currentImps,
			&seller.ID, &seller.Type, &seller.ASI, &seller.Name, &seller.Domain, &sellerAuthorized,
			&siteSupply.Environment, &siteSupply.CanonicalIdentity, &siteSupply.StoreURL, &siteSupply.IntegrationMode,
			&slotSupply.MediaIntent, &slotSupply.Placement, &slotSupply.RenderContext, &slotSupply.RefreshMode,
			&slotSupply.RefreshSeconds, &slotSupply.AdDensity, &slotSupply.TrafficQuality, &slotSupply.SourceQuality,
			&slotSupply.ManagementControl,
		)
		if err != nil {
			return nil, err
		}
		if p.PubID != 0 && p.PubID != pubID {
			return nil, fmt.Errorf("publisher domain %q maps to ids %d and %d", domain, p.PubID, pubID)
		}
		p.PubID = pubID
		seller.Authorized = sellerAuthorized == "Yes"
		p.Seller = seller
		if active.Valid && active.String == "Yes" {
			p.Active = true
		}
		if limitImps.Valid {
			p.LimitImps = uint32(limitImps.Int64)
		}
		if currentImps.Valid {
			p.CurrentImps = uint32(currentImps.Int64)
		}
		if existing, ok := p.Sites[foreignID]; ok && existing != siteID {
			return nil, fmt.Errorf("publisher site identity %q maps to ids %d and %d", foreignID, existing, siteID)
		}
		if _, ok := p.Sites[foreignID]; !ok {
			p.Sites[foreignID] = siteID
		}
		parsedSiteType := ParseSiteType(siteType)
		if existing, ok := p.SiteTypes[siteID]; ok && existing != parsedSiteType {
			return nil, fmt.Errorf("publisher site %d has conflicting types", siteID)
		}
		p.SiteTypes[siteID] = parsedSiteType
		p.SiteSupply[siteID] = siteSupply.Normalize()
		if _, ok := p.Slots[siteID]; !ok {
			p.Slots[siteID] = make(map[string]uint32)
		}
		if _, ok := p.SlotSizes[siteID]; !ok {
			p.SlotSizes[siteID] = make(map[uint32]uint32)
		}
		if _, ok := p.SlotFloors[siteID]; !ok {
			p.SlotFloors[siteID] = make(map[uint32]float64)
		}
		if _, ok := p.SlotFloorCPMs[siteID]; !ok {
			p.SlotFloorCPMs[siteID] = make(map[uint32]accounting.CPM)
		}
		if _, ok := p.SlotSupply[siteID]; !ok {
			p.SlotSupply[siteID] = make(map[uint32]SlotSupplyMetadata)
		}
		p.SlotSupply[siteID][slotID] = slotSupply.Normalize()
		if existing, ok := p.Slots[siteID][slotName]; ok && existing != slotID {
			return nil, fmt.Errorf("publisher site %d slot name %q maps to ids %d and %d", siteID, slotName, existing, slotID)
		}
		if _, ok := p.Slots[siteID][slotName]; !ok {
			p.Slots[siteID][slotName] = slotID
		}
		if _, ok := p.SlotSizes[siteID][slotID]; !ok {
			p.SlotSizes[siteID][slotID] = sizeID
		}
		floor := accounting.CPM(0)
		if bidFloor.Valid {
			floor, err = accounting.ParseCPM(bidFloor.String)
			if err != nil {
				return nil, fmt.Errorf("publisher site %d slot %d bid floor: %w", siteID, slotID, err)
			}
		}
		p.SlotFloorCPMs[siteID][slotID] = floor
		p.SlotFloors[siteID][slotID] = floor.Float64()
		if foreignID == SITEDefaultApp && slotName == SLOTDefault {
			p.DefaultAppSiteID = siteID
			p.DefaultAppSlotID = slotID
		}
		if foreignID == SITEDefaultWeb && slotName == SLOTDefault {
			p.DefaultWebSiteID = siteID
			p.DefaultWebSlotID = slotID
		}
	}
	return p, rows.Err()
}

func (self *Pub) addSlot(db *sql.DB, siteID uint32, slotStr string) (uint32, error) {
	return self.addSlotContext(context.Background(), db, siteID, slotStr)
}

func (self *Pub) addSlotContext(ctx context.Context, db publisherMutationDB, siteID uint32, slotStr string) (uint32, error) {
	slotID, err := insertSlotContext(ctx, db, siteID, slotStr)
	if err != nil {
		return 0, err
	}
	self.rememberSlot(siteID, slotID, slotStr)
	return slotID, nil
}

func insertSlotContext(ctx context.Context, db publisherMutationDB, siteID uint32, slotStr string) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	row, err := db.ExecContext(ctx, `
INSERT INTO pub_slot (site_id, slot_name, active, created)
VALUES (?, ?, 'Yes', NOW())`, siteID, slotStr)
	if err != nil {
		return 0, err
	}
	id, err := row.LastInsertId()
	if err != nil {
		return 0, err
	}
	slotID := uint32(id)
	return slotID, nil
}

func (self *Pub) rememberSlot(siteID, slotID uint32, slotStr string) {
	if self.Slots == nil {
		self.Slots = make(map[uint32]map[string]uint32)
	}
	if self.Slots[siteID] == nil {
		self.Slots[siteID] = make(map[string]uint32)
	}
	self.Slots[siteID][slotStr] = slotID
	if self.SlotSizes == nil {
		self.SlotSizes = make(map[uint32]map[uint32]uint32)
	}
	if self.SlotSizes[siteID] == nil {
		self.SlotSizes[siteID] = make(map[uint32]uint32)
	}
	self.SlotSizes[siteID][slotID] = defaultPubSlotSizeID
	if self.SlotFloors == nil {
		self.SlotFloors = make(map[uint32]map[uint32]float64)
	}
	if self.SlotFloors[siteID] == nil {
		self.SlotFloors[siteID] = make(map[uint32]float64)
	}
	self.SlotFloors[siteID][slotID] = 0
	if self.SlotFloorCPMs == nil {
		self.SlotFloorCPMs = make(map[uint32]map[uint32]accounting.CPM)
	}
	if self.SlotFloorCPMs[siteID] == nil {
		self.SlotFloorCPMs[siteID] = make(map[uint32]accounting.CPM)
	}
	self.SlotFloorCPMs[siteID][slotID] = 0
}

// ExactSlotFloor returns the authoritative configured USD CPM floor. Current
// cache generations require the fixed-point field; legacy gob generations may
// use their bounded six-place float projection during a rolling drain.
func (self *Pub) ExactSlotFloor(siteID, slotID uint32) (accounting.CPM, bool) {
	if self == nil {
		return 0, false
	}
	if floors := self.SlotFloorCPMs[siteID]; floors != nil {
		if floor, ok := floors[slotID]; ok {
			return floor, floor >= 0 && floor <= accounting.MaxCPM
		}
	}
	if self.AccountingVersion == accounting.ExactMoneyContract {
		return 0, false
	}
	floors := self.SlotFloors[siteID]
	floor, ok := floors[slotID]
	if !ok || floor < 0 || math.IsNaN(floor) || math.IsInf(floor, 0) {
		return 0, false
	}
	parsed, err := accounting.ParseCPM(strconv.FormatFloat(floor, 'f', 6, 64))
	return parsed, err == nil
}

func (self *Pub) addSite(db *sql.DB, siteStr string, siteType string) (uint32, error) {
	return self.addSiteContext(context.Background(), db, siteStr, siteType)
}

func (self *Pub) addSiteContext(ctx context.Context, db publisherMutationDB, siteStr string, siteType string) (uint32, error) {
	siteID, err := insertSiteContext(ctx, db, self.PubID, siteStr, siteType)
	if err != nil {
		return 0, err
	}
	self.rememberSite(siteID, siteStr, siteType)
	return siteID, nil
}

func insertSiteContext(ctx context.Context, db publisherMutationDB, pubID uint32, siteStr, siteType string) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	row, err := db.ExecContext(ctx, `
INSERT INTO pub_site (pub_id, site_name, foreign_id, site_url, site_type, active, created)
VALUES (?, ?, ?, ?, ?, 'Yes', NOW())`, pubID, siteStr, siteStr, siteStr, siteType)
	if err != nil {
		return 0, err
	}
	id, err := row.LastInsertId()
	if err != nil {
		return 0, err
	}
	siteID := uint32(id)
	return siteID, nil
}

func (self *Pub) rememberSite(siteID uint32, siteStr, siteType string) {
	if self.Sites == nil {
		self.Sites = make(map[string]uint32)
	}
	self.Sites[siteStr] = siteID
	if self.SiteTypes == nil {
		self.SiteTypes = make(map[uint32]SiteType)
	}
	self.SiteTypes[siteID] = ParseSiteType(siteType)
}

func (self *Pub) addSiteAndSlotContext(ctx context.Context, db *sql.DB, siteStr, siteType, slotStr string) (uint32, uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	siteID, err := insertSiteContext(ctx, tx, self.PubID, siteStr, siteType)
	if err != nil {
		return 0, 0, err
	}
	slotID, err := insertSlotContext(ctx, tx, siteID, slotStr)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	self.rememberSite(siteID, siteStr, siteType)
	self.rememberSlot(siteID, slotID, slotStr)
	return siteID, slotID, nil
}

const publisherIDAllocationAttempts = 16

func randomPublisherID() (uint32, error) {
	for {
		var data [4]byte
		if _, err := io.ReadFull(cryptorand.Reader, data[:]); err != nil {
			return 0, fmt.Errorf("generate publisher id: %w", err)
		}
		if id := binary.LittleEndian.Uint32(data[:]); id != 0 {
			return id, nil
		}
	}
}

func insertPublisher(db *sql.DB, pubStr string, generateID func() (uint32, error)) (uint32, error) {
	return insertPublisherContext(context.Background(), db, pubStr, generateID)
}

func insertPublisherContext(ctx context.Context, db publisherMutationDB, pubStr string, generateID func() (uint32, error)) (uint32, error) {
	for attempt := 0; attempt < publisherIDAllocationAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		pubID, err := generateID()
		if err != nil {
			return 0, err
		}
		_, err = db.ExecContext(ctx, `
INSERT INTO pub (pub_id, domain, email, passwd, address_id, active, created)
VALUES (?, ?, ?, '123456789', 1, 'Yes', NOW())`, pubID, pubStr, pubStr)
		if err == nil {
			return pubID, nil
		}
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 {
			return 0, err
		}
		var existingEmail string
		lookupErr := db.QueryRowContext(ctx, `SELECT email FROM pub WHERE pub_id=? LIMIT 1`, pubID).Scan(&existingEmail)
		if lookupErr == nil {
			continue
		}
		if !errors.Is(lookupErr, sql.ErrNoRows) {
			return 0, lookupErr
		}
		return 0, err
	}
	return 0, fmt.Errorf("could not allocate a unique publisher id after %d attempts", publisherIDAllocationAttempts)
}

func AddPub(db *sql.DB, pubStr string) (*Pub, error) {
	return AddPubContext(context.Background(), db, pubStr)
}

func AddPubContext(ctx context.Context, db *sql.DB, pubStr string) (*Pub, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if db == nil {
		return nil, errors.New("publisher database is nil")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	pubID, err := insertPublisherContext(ctx, tx, pubStr, randomPublisherID)
	if err != nil {
		return nil, err
	}
	pub := &Pub{
		AccountingVersion: accounting.ExactMoneyContract,
		PubID:             pubID,
		Sites:             make(map[string]uint32),
		Slots:             make(map[uint32]map[string]uint32),
		SlotSizes:         make(map[uint32]map[uint32]uint32),
		SlotFloors:        make(map[uint32]map[uint32]float64),
		SlotFloorCPMs:     make(map[uint32]map[uint32]accounting.CPM),
	}

	if pub.DefaultAppSiteID, err = pub.addSiteContext(ctx, tx, SITEDefaultApp, "App"); err == nil {
		if pub.DefaultWebSiteID, err = pub.addSiteContext(ctx, tx, SITEDefaultWeb, "Web"); err == nil {
			if pub.DefaultAppSlotID, err = pub.addSlotContext(ctx, tx, pub.DefaultAppSiteID, SLOTDefault); err == nil {
				pub.DefaultWebSlotID, err = pub.addSlotContext(ctx, tx, pub.DefaultWebSiteID, SLOTDefault)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return pub, nil
}

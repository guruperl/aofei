package acl

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type PubMap map[string]*Pub

type DirectPub struct {
	Domain     string
	Pub        *Pub
	Sites      map[uint32]string
	SiteTypes  map[uint32]SiteType
	Slots      map[uint32]map[uint32]string
	SlotSizes  map[uint32]map[uint32]uint32
	SlotFloors map[uint32]map[uint32]float64
}

type DirectPubMap map[uint32]*DirectPub

// ToRedis encodes the PubMap to a byte slice and stores it in Redis
func (self PubMap) ToRedis(ctx context.Context, conn radix.Client) error {
	return self.ToRedisKeys(ctx, conn, HashNamePubmap, HashNamePubByID)
}

// ToRedisKeys writes publisher caches to explicit Redis hash keys.
func (self PubMap) ToRedisKeys(ctx context.Context, conn radix.Client, pubmapKey, byIDKey string) error {
	arr := []string{pubmapKey}
	byID := []string{byIDKey}
	for k, v := range self {
		var bs []byte
		var err error
		if v != nil && v.Active && (v.LimitImps == 0 || v.CurrentImps < v.LimitImps) {
			if bs, err = v.Pack(); err == nil {
				arr = append(arr, k, string(bs))
			}
			if err == nil {
				var direct []byte
				direct, err = NewDirectPub(k, v).Pack()
				if err == nil {
					byID = append(byID, strconv.FormatUint(uint64(v.PubID), 10), string(direct))
				}
			}
		} else {
			// delete old pubmap
			err = conn.Do(ctx, radix.Cmd(nil, "HDEL", pubmapKey, k))
			if err == nil && v != nil {
				err = conn.Do(ctx, radix.Cmd(nil, "HDEL", byIDKey, strconv.FormatUint(uint64(v.PubID), 10)))
			}
		}
		if err != nil {
			return err
		}
	}
	if len(arr) > 1 {
		if err := conn.Do(ctx, radix.Cmd(nil, "HMSET", arr...)); err != nil {
			return err
		}
	}
	if len(byID) > 1 {
		return conn.Do(ctx, radix.Cmd(nil, "HMSET", byID...))
	}
	return nil
}

// ToSpread encodes the PubMap to a byte slice and publish it to nats
func (self PubMap) ToSpread(conn *nats.Conn) error {
	for k, v := range self {
		err := v.ToSpread(conn, k)
		if err != nil {
			return err
		}
	}
	return nil
}

// PubMapFromRedis retrieves the PubMap from Redis and decodes it from a byte slice
func PubMapFromRedis(ctx context.Context, conn radix.Client) (PubMap, error) {
	name := HashNamePubmap
	arr := make([]string, 0)
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", name))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil // No PubMap found in Redis
	}
	pubmap := make(PubMap)
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows // Invalid format
	}
	// Decode each key-value pair into the PubMap
	for i := 0; i < len(arr); i += 2 {
		pubStr := arr[i]
		data := []byte(arr[i+1])
		pub, err := UnpackPub(data)
		if err != nil {
			return nil, err
		}
		pubmap[pubStr] = pub
	}
	return pubmap, nil
}

var HashNamePubmap = "pubmap"

var HashNamePubByID = "pubmap:by-id"

func NewDirectPub(domain string, pub *Pub) *DirectPub {
	direct := &DirectPub{
		Domain:     domain,
		Pub:        pub,
		Sites:      make(map[uint32]string),
		SiteTypes:  make(map[uint32]SiteType),
		Slots:      make(map[uint32]map[uint32]string),
		SlotSizes:  make(map[uint32]map[uint32]uint32),
		SlotFloors: make(map[uint32]map[uint32]float64),
	}
	if pub == nil {
		return direct
	}
	for siteStr, siteID := range pub.Sites {
		if _, ok := direct.Sites[siteID]; !ok {
			direct.Sites[siteID] = siteStr
		}
	}
	for siteID, siteType := range pub.SiteTypes {
		direct.SiteTypes[siteID] = siteType
	}
	for siteID, slots := range pub.Slots {
		if direct.Slots[siteID] == nil {
			direct.Slots[siteID] = make(map[uint32]string)
		}
		for slotStr, slotID := range slots {
			if _, ok := direct.Slots[siteID][slotID]; !ok {
				direct.Slots[siteID][slotID] = slotStr
			}
		}
	}
	for siteID, slots := range pub.SlotSizes {
		if direct.SlotSizes[siteID] == nil {
			direct.SlotSizes[siteID] = make(map[uint32]uint32)
		}
		for slotID, sizeID := range slots {
			if _, ok := direct.SlotSizes[siteID][slotID]; !ok {
				direct.SlotSizes[siteID][slotID] = sizeID
			}
		}
	}
	for siteID, slots := range pub.SlotFloors {
		if direct.SlotFloors[siteID] == nil {
			direct.SlotFloors[siteID] = make(map[uint32]float64)
		}
		for slotID, floor := range slots {
			direct.SlotFloors[siteID][slotID] = floor
		}
	}
	return direct
}

func DirectPubMapFromPubMap(pubmap PubMap) DirectPubMap {
	if pubmap == nil {
		return nil
	}
	byID := make(DirectPubMap)
	for domain, pub := range pubmap {
		if pub == nil || !pub.Active || (pub.LimitImps != 0 && pub.CurrentImps >= pub.LimitImps) {
			continue
		}
		byID[pub.PubID] = NewDirectPub(domain, pub)
	}
	return byID
}

// ValidateCommercialPubMap verifies the cache-owned P01 activation metadata
// before a generation can be published. Inactive publishers are intentionally
// omitted by writers and do not block an unrelated active generation.
func ValidateCommercialPubMap(pubmap PubMap) error {
	seenPubIDs := make(map[uint32]string)
	for domain, pub := range pubmap {
		if pub == nil || !pub.Active || (pub.LimitImps != 0 && pub.CurrentImps >= pub.LimitImps) {
			continue
		}
		if strings.TrimSpace(domain) == "" || pub.PubID == 0 {
			return fmt.Errorf("active publisher has empty domain or id")
		}
		if prior, ok := seenPubIDs[pub.PubID]; ok && prior != domain {
			return fmt.Errorf("publisher id %d is mapped to both %q and %q", pub.PubID, prior, domain)
		}
		seenPubIDs[pub.PubID] = domain
		if err := pub.Seller.Validate(); err != nil {
			return fmt.Errorf("publisher %d: %w", pub.PubID, err)
		}
		if len(pub.Sites) == 0 {
			return fmt.Errorf("active publisher %d has no approved site/app", pub.PubID)
		}
		seenSiteIDs := make(map[uint32]string)
		for siteIdentity, siteID := range pub.Sites {
			siteType := pub.SiteTypes[siteID]
			if siteID == 0 || (siteType != SiteTypeWeb && siteType != SiteTypeAPP) || !validCommercialSiteIdentity(siteIdentity, siteType) {
				return fmt.Errorf("publisher %d site %d has incomplete identity or type", pub.PubID, siteID)
			}
			if prior, ok := seenSiteIDs[siteID]; ok && prior != siteIdentity {
				return fmt.Errorf("publisher %d site id %d maps to both %q and %q", pub.PubID, siteID, prior, siteIdentity)
			}
			seenSiteIDs[siteID] = siteIdentity
			siteSupply := pub.SiteSupply[siteID].Normalize()
			if err := siteSupply.Validate(); err != nil {
				return fmt.Errorf("publisher %d site %d: %w", pub.PubID, siteID, err)
			}
			if (siteSupply.Environment == "Web" && siteType != SiteTypeWeb) ||
				(siteSupply.Environment == "App" && siteType != SiteTypeAPP) ||
				(siteSupply.IntegrationMode == "BrowserTag" && siteType != SiteTypeWeb) ||
				(siteSupply.IntegrationMode == "SDK" && siteType != SiteTypeAPP) {
				return fmt.Errorf("publisher %d site %d taxonomy conflicts with legacy site type", pub.PubID, siteID)
			}
			slots := pub.Slots[siteID]
			if len(slots) == 0 {
				return fmt.Errorf("publisher %d site %d has no approved slot", pub.PubID, siteID)
			}
			seenSlotIDs := make(map[uint32]string)
			for slotName, slotID := range slots {
				if strings.TrimSpace(slotName) == "" || slotID == 0 {
					return fmt.Errorf("publisher %d site %d has incomplete slot identity", pub.PubID, siteID)
				}
				if prior, ok := seenSlotIDs[slotID]; ok && prior != slotName {
					return fmt.Errorf("publisher %d site %d slot id %d maps to both %q and %q", pub.PubID, siteID, slotID, prior, slotName)
				}
				seenSlotIDs[slotID] = slotName
				if err := pub.SlotSupply[siteID][slotID].Validate(); err != nil {
					return fmt.Errorf("publisher %d site %d slot %d: %w", pub.PubID, siteID, slotID, err)
				}
				sizeID, ok := pub.SlotSizes[siteID][slotID]
				w, h := commercialSize(sizeID)
				if !ok || w == 0 || h == 0 {
					return fmt.Errorf("publisher %d site %d slot %d has invalid size", pub.PubID, siteID, slotID)
				}
				floor, ok := pub.SlotFloors[siteID][slotID]
				if !ok || floor < 0 || math.IsNaN(floor) || math.IsInf(floor, 0) {
					return fmt.Errorf("publisher %d site %d slot %d has invalid USD CPM floor", pub.PubID, siteID, slotID)
				}
			}
		}
	}
	return nil
}

func validCommercialSiteIdentity(identity string, siteType SiteType) bool {
	if identity == "" || identity != strings.TrimSpace(identity) || len(identity) > 255 {
		return false
	}
	if siteType == SiteTypeAPP {
		for _, character := range identity {
			if character <= ' ' || character == 0x7f {
				return false
			}
		}
		return true
	}
	if siteType != SiteTypeWeb {
		return false
	}
	if net.ParseIP(identity) != nil {
		return true
	}
	if len(identity) > 253 || strings.HasSuffix(identity, ".") {
		return false
	}
	for _, label := range strings.Split(identity, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') &&
				(character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}

func commercialSize(sizeID uint32) (uint16, uint16) {
	return uint16(sizeID >> 16), uint16(sizeID & 0xffff)
}

func (self *DirectPub) Pack() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(self)
	return buf.Bytes(), err
}

func UnpackDirectPub(data []byte) (*DirectPub, error) {
	var pub DirectPub
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&pub)
	return &pub, err
}

func (self *DirectPub) Validate(siteID, slotID, sizeID uint32) (string, string, bool) {
	if self == nil || self.Pub == nil || !self.Pub.Active ||
		(self.Pub.LimitImps != 0 && self.Pub.CurrentImps >= self.Pub.LimitImps) {
		return "", "", false
	}
	siteStr, ok := self.Sites[siteID]
	if !ok {
		return "", "", false
	}
	slots := self.Slots[siteID]
	if slots == nil {
		return "", "", false
	}
	slotStr, ok := slots[slotID]
	if !ok {
		return "", "", false
	}
	sizes := self.SlotSizes[siteID]
	if sizes == nil {
		return "", "", false
	}
	configuredSizeID, ok := sizes[slotID]
	if !ok || configuredSizeID != sizeID {
		return "", "", false
	}
	return siteStr, slotStr, true
}

// CommercialSlot returns the cache-owned direct-supply policy for one exact
// packed site/slot/size tuple. Missing type/floor metadata fails closed so a
// P01 binary cannot serve from a pre-P01 publisher cache generation.
func (self *DirectPub) CommercialSlot(siteID, slotID, sizeID uint32) (string, string, SiteType, float64, bool) {
	siteStr, slotStr, ok := self.Validate(siteID, slotID, sizeID)
	if !ok {
		return "", "", SiteTypeUnknown, 0, false
	}
	siteType, ok := self.SiteTypes[siteID]
	if !ok || siteType == SiteTypeUnknown {
		return "", "", SiteTypeUnknown, 0, false
	}
	floors := self.SlotFloors[siteID]
	floor, ok := floors[slotID]
	if !ok || floor < 0 || math.IsNaN(floor) || math.IsInf(floor, 0) {
		return "", "", SiteTypeUnknown, 0, false
	}
	return siteStr, slotStr, siteType, floor, true
}

func (self DirectPubMap) PubByID(pubID uint32) *DirectPub {
	if self == nil {
		return nil
	}
	return self[pubID]
}

func PubByIDFromRedis(ctx context.Context, conn radix.Client, pubID uint32) (*DirectPub, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNamePubByID, strconv.FormatUint(uint64(pubID), 10)))
	if err == nil && len(bs) > 2 {
		return UnpackDirectPub(bs)
	}
	return nil, err
}

func DirectPubMapFromRedis(ctx context.Context, conn radix.Client) (DirectPubMap, error) {
	arr := make([]string, 0)
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", HashNamePubByID))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil
	}
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows
	}
	byID := make(DirectPubMap)
	for i := 0; i < len(arr); i += 2 {
		pubID64, err := strconv.ParseUint(arr[i], 10, 32)
		if err != nil {
			return nil, err
		}
		pub, err := UnpackDirectPub([]byte(arr[i+1]))
		if err != nil {
			return nil, err
		}
		byID[uint32(pubID64)] = pub
	}
	return byID, nil
}

// PubMapFromIO retrieves the PubMap from IO
func PubMapFromIO(top string) (PubMap, error) {
	files, err := os.ReadDir(top + "/" + HashNamePubmap)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No files found
		}
		return nil, err // Error reading directory
	}

	pubmap := make(PubMap)
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}
		name := file.Name()
		pub, err := PubFromIO(top, name)
		if err != nil {
			return nil, err // Error unpacking Pub
		}
		if pub == nil {
			continue // Skip nil Pub objects
		}
		pubmap[name] = pub
	}

	return pubmap, nil
}

// DBGetPubMap retrieves the PubMap from the database
func DBGetPubMap(db *sql.DB) (PubMap, error) {
	return DBGetPubMapContext(context.Background(), db)
}

func DBGetPubMapContext(ctx context.Context, db *sql.DB) (PubMap, error) {
	rows, err := db.QueryContext(ctx, `
	SELECT domain, p.pub_id, p.active, foreign_id, s.site_id, s.site_type,
	       t.slot_name, t.slot_id, t.size_id, t.bidfloor, b.limit_imp, b.current_imp,
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
	WHERE s.active='Yes' AND t.active='Yes'
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pubMap := make(map[string]*Pub)
	for rows.Next() {
		var pubID, siteID, slotID, sizeID uint32
		var limitImps, currentImps sql.NullInt64
		var bidFloor sql.NullFloat64
		var active sql.NullString
		var domain, slotName, foreignID, siteType, sellerAuthorized string
		var seller SellerMetadata
		var siteSupply SiteSupplyMetadata
		var slotSupply SlotSupplyMetadata
		err = rows.Scan(
			&domain, &pubID, &active, &foreignID, &siteID, &siteType, &slotName, &slotID, &sizeID, &bidFloor, &limitImps, &currentImps,
			&seller.ID, &seller.Type, &seller.ASI, &seller.Name, &seller.Domain, &sellerAuthorized,
			&siteSupply.Environment, &siteSupply.CanonicalIdentity, &siteSupply.StoreURL, &siteSupply.IntegrationMode,
			&slotSupply.MediaIntent, &slotSupply.Placement, &slotSupply.RenderContext, &slotSupply.RefreshMode,
			&slotSupply.RefreshSeconds, &slotSupply.AdDensity, &slotSupply.TrafficQuality, &slotSupply.SourceQuality,
			&slotSupply.ManagementControl,
		)
		if err != nil {
			return nil, err
		}
		if existing, ok := pubMap[domain]; ok && existing.PubID != pubID {
			return nil, fmt.Errorf("publisher domain %q maps to ids %d and %d", domain, existing.PubID, pubID)
		}
		if _, ok := pubMap[domain]; !ok {
			pubMap[domain] = &Pub{
				PubID:      pubID,
				Sites:      make(map[string]uint32),
				SiteTypes:  make(map[uint32]SiteType),
				Slots:      make(map[uint32]map[string]uint32),
				SlotSizes:  make(map[uint32]map[uint32]uint32),
				SlotFloors: make(map[uint32]map[uint32]float64),
				SiteSupply: make(map[uint32]SiteSupplyMetadata),
				SlotSupply: make(map[uint32]map[uint32]SlotSupplyMetadata),
			}
		}
		seller.Authorized = sellerAuthorized == "Yes"
		pubMap[domain].Seller = seller
		if active.Valid && active.String == "Yes" {
			pubMap[domain].Active = true
		}
		if limitImps.Valid {
			pubMap[domain].LimitImps = uint32(limitImps.Int64)
		}
		if currentImps.Valid {
			pubMap[domain].CurrentImps = uint32(currentImps.Int64)
		}
		if existing, ok := pubMap[domain].Sites[foreignID]; ok && existing != siteID {
			return nil, fmt.Errorf("publisher %d site identity %q maps to ids %d and %d", pubID, foreignID, existing, siteID)
		}
		if _, ok := pubMap[domain].Sites[foreignID]; !ok {
			pubMap[domain].Sites[foreignID] = siteID
		}
		parsedSiteType := ParseSiteType(siteType)
		if existing, ok := pubMap[domain].SiteTypes[siteID]; ok && existing != parsedSiteType {
			return nil, fmt.Errorf("publisher %d site %d has conflicting types", pubID, siteID)
		}
		pubMap[domain].SiteTypes[siteID] = parsedSiteType
		pubMap[domain].SiteSupply[siteID] = siteSupply.Normalize()
		if _, ok := pubMap[domain].Slots[siteID]; !ok {
			pubMap[domain].Slots[siteID] = make(map[string]uint32)
		}
		if _, ok := pubMap[domain].SlotSizes[siteID]; !ok {
			pubMap[domain].SlotSizes[siteID] = make(map[uint32]uint32)
		}
		if _, ok := pubMap[domain].SlotFloors[siteID]; !ok {
			pubMap[domain].SlotFloors[siteID] = make(map[uint32]float64)
		}
		if _, ok := pubMap[domain].SlotSupply[siteID]; !ok {
			pubMap[domain].SlotSupply[siteID] = make(map[uint32]SlotSupplyMetadata)
		}
		pubMap[domain].SlotSupply[siteID][slotID] = slotSupply.Normalize()
		if existing, ok := pubMap[domain].Slots[siteID][slotName]; ok && existing != slotID {
			return nil, fmt.Errorf("publisher %d site %d slot name %q maps to ids %d and %d", pubID, siteID, slotName, existing, slotID)
		}
		if _, ok := pubMap[domain].Slots[siteID][slotName]; !ok {
			pubMap[domain].Slots[siteID][slotName] = slotID
		}
		if _, ok := pubMap[domain].SlotSizes[siteID][slotID]; !ok {
			pubMap[domain].SlotSizes[siteID][slotID] = sizeID
		}
		if bidFloor.Valid {
			pubMap[domain].SlotFloors[siteID][slotID] = bidFloor.Float64
		} else {
			pubMap[domain].SlotFloors[siteID][slotID] = 0
		}
		if foreignID == SITEDefaultApp && slotName == SLOTDefault {
			pubMap[domain].DefaultAppSiteID = siteID
			pubMap[domain].DefaultAppSlotID = slotID
		}
		if foreignID == SITEDefaultWeb && slotName == SLOTDefault {
			pubMap[domain].DefaultWebSiteID = siteID
			pubMap[domain].DefaultWebSlotID = slotID
		}
	}
	return pubMap, rows.Err()
}

// DBAddNew adds a new RPub object to the database.
func (self PubMap) DBAddNew(db *sql.DB, pubStr, siteStr, siteType, slotStr string) (*Pub, error) {
	return self.DBAddNewContext(context.Background(), db, pubStr, siteStr, siteType, slotStr)
}

// DBAddNewContext adds discovered publisher inventory while honoring the
// owning cache job's cancellation boundary.
func (self PubMap) DBAddNewContext(ctx context.Context, db *sql.DB, pubStr, siteStr, siteType, slotStr string) (*Pub, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var pub *Pub
	var ok1 bool
	var err error
	pub, ok1 = self[pubStr]
	if ok1 {
		siteID, siteExists := pub.Sites[siteStr]
		if siteExists {
			if _, slotExists := pub.Slots[siteID][slotStr]; !slotExists {
				_, err = pub.addSlotContext(ctx, db, siteID, slotStr)
			}
		} else {
			_, _, err = pub.addSiteAndSlotContext(ctx, db, siteStr, siteType, slotStr)
		}
	} else {
		pub, err = AddPubContext(ctx, db, pubStr)
	}

	return pub, err
}

// PubFromRedis returns the Pub object by pub string.
// This is the main purpose of PubMap, used in Controller.
func PubFromRedis(ctx context.Context, conn radix.Client, pubStr string) (*Pub, error) {
	var bs []byte
	name := HashNamePubmap
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", name, pubStr))
	if err == nil && len(bs) > 2 {
		return UnpackPub(bs)
	}

	return nil, err
}

// PubFromIO returns the Pub object by pub string.
// This is the main purpose of PubMap, used in Controller.
func PubFromIO(top, pubStr string) (*Pub, error) {
	fullname := top + "/" + HashNamePubmap + "/" + pubStr
	f, err := os.Open(fullname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No file found
		}
		return nil, err // Error opening file
	}
	defer f.Close()
	return UnpackPubIO(f)
}

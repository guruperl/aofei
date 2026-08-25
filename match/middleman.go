package match

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/acl"
	"github.com/mediocregopher/radix/v4"
)

const (
	HashNameMiddlemanRoutes   = "middleman:routes"
	HashNameMiddlemanRoutesV2 = "middleman:routes:v2"

	MiddlemanRouteCacheLegacyVersion = 1
	MiddlemanRouteCacheVersion       = 2
	MiddlemanRouteAccountingVersion  = accounting.ExactMoneyContract
)

type MiddlemanRouteCache struct {
	Version  int                          `json:"version"`
	Metadata *MiddlemanRouteCacheMetadata `json:"metadata,omitempty"`
	Entries  []MiddlemanRouteEntry        `json:"entries"`
}

type MiddlemanRouteCacheMetadata struct {
	GeneratedAt      string `json:"generated_at"`
	EntryCount       int    `json:"entry_count"`
	Source           string `json:"source"`
	RouteDBHighWater string `json:"route_db_high_water,omitempty"`
	Checksum         string `json:"checksum"`
}

type MiddlemanRouteEntry struct {
	AccountingVersion   string           `json:"accounting_version,omitempty"`
	TargetID            uint32           `json:"target_id"`
	GroupID             uint32           `json:"group_id"`
	TriggerMode         string           `json:"trigger_mode,omitempty"`
	RouteBidderID       uint32           `json:"route_bidder_id"`
	BidderID            uint32           `json:"bidder_id"`
	AdvID               uint32           `json:"adv_id"`
	EntityTypeID        *uint8           `json:"entitytype_id,omitempty"`
	EntityID            *uint32          `json:"entity_id,omitempty"`
	SizeID              *uint32          `json:"size_id,omitempty"`
	TargetPriority      uint16           `json:"target_priority"`
	RouteBidderPriority uint16           `json:"route_bidder_priority"`
	GroupTimeoutMS      uint16           `json:"group_timeout_ms"`
	BidderTimeoutMS     uint16           `json:"bidder_timeout_ms"`
	RouteTimeoutMS      *uint16          `json:"route_timeout_ms,omitempty"`
	GroupMarginPct      float64          `json:"group_margin_pct"`
	GroupMinMarginCPM   float64          `json:"group_min_margin_cpm"`
	RouteMarginPct      *float64         `json:"route_margin_pct,omitempty"`
	RouteMinMarginCPM   *float64         `json:"route_min_margin_cpm,omitempty"`
	GroupMarginUnits    uint32           `json:"group_margin_fraction_1e4,omitempty"`
	GroupMinMarginExact accounting.CPM   `json:"group_min_margin_cpm_micros,omitempty"`
	RouteMarginUnits    *uint32          `json:"route_margin_fraction_1e4,omitempty"`
	RouteMinMarginExact *accounting.CPM  `json:"route_min_margin_cpm_micros,omitempty"`
	EndpointURL         string           `json:"endpoint_url"`
	OpenRTBVersion      string           `json:"openrtb_version"`
	Seat                string           `json:"seat,omitempty"`
	CredentialRef       string           `json:"credential_ref"`
	SyntheticCampaignID uint32           `json:"synthetic_campaign_id"`
	SyntheticItemID     uint32           `json:"synthetic_item_id"`
	SyntheticCreativeID uint32           `json:"synthetic_creative_id"`
	Audience            *acl.ACLAudience `json:"audience,omitempty"`
}

func (e MiddlemanRouteEntry) Matches(attr *Attribute) bool {
	if attr == nil {
		return false
	}
	if e.SizeID != nil && *e.SizeID != attr.SizeID {
		return false
	}
	if e.EntityTypeID == nil && e.EntityID == nil {
		return true
	}
	if e.EntityTypeID == nil || e.EntityID == nil {
		return false
	}
	switch *e.EntityTypeID {
	case 3:
		return *e.EntityID == attr.PubID
	case 31:
		return *e.EntityID == attr.SiteID
	case 32:
		return *e.EntityID == attr.SlotID
	default:
		return false
	}
}

func (e MiddlemanRouteEntry) Eligible(attr *Attribute) bool {
	if !e.Matches(attr) {
		return false
	}
	return e.Audience == nil || e.Audience.Has(attr.ACL)
}

func (e MiddlemanRouteEntry) Specificity() int {
	score := 0
	if e.SizeID != nil {
		score += 10
	}
	if e.EntityTypeID == nil {
		return score
	}
	switch *e.EntityTypeID {
	case 32:
		score += 3
	case 31:
		score += 2
	case 3:
		score++
	}
	return score
}

func (e MiddlemanRouteEntry) EffectiveTimeoutMS() uint16 {
	if e.RouteTimeoutMS != nil && *e.RouteTimeoutMS > 0 {
		return *e.RouteTimeoutMS
	}
	return e.BidderTimeoutMS
}

func (e MiddlemanRouteEntry) EffectiveMarginPct() float64 {
	if e.RouteMarginPct != nil {
		return *e.RouteMarginPct
	}
	return e.GroupMarginPct
}

func (e MiddlemanRouteEntry) EffectiveMinMarginCPM() float64 {
	if e.RouteMinMarginCPM != nil {
		return *e.RouteMinMarginCPM
	}
	return e.GroupMinMarginCPM
}

// ExactMarginTerms returns a four-place nonnegative margin fraction and an
// exact six-place minimum CPM. New cache generations carry these values
// directly; old generations use a bounded compatibility conversion.
func (e MiddlemanRouteEntry) ExactMarginTerms() (uint32, accounting.CPM, error) {
	switch e.AccountingVersion {
	case MiddlemanRouteAccountingVersion:
		units := e.GroupMarginUnits
		minimum := e.GroupMinMarginExact
		if e.RouteMarginUnits != nil {
			units = *e.RouteMarginUnits
		}
		if e.RouteMinMarginExact != nil {
			minimum = *e.RouteMinMarginExact
		}
		if units > 10_000 || minimum < 0 || minimum > accounting.MaxCPM {
			return 0, 0, fmt.Errorf("exact partner margin is outside its supported range")
		}
		return units, minimum, nil
	case "":
		// Pre-A03 route generations carry only the bounded float terms below.
	default:
		return 0, 0, fmt.Errorf("unsupported middleman accounting version %q", e.AccountingVersion)
	}
	units, err := parseMarginFraction4(strconv.FormatFloat(e.EffectiveMarginPct(), 'f', 4, 64))
	if err != nil {
		return 0, 0, err
	}
	minimum, err := accounting.ParseCPM(strconv.FormatFloat(e.EffectiveMinMarginCPM(), 'f', 6, 64))
	if err != nil {
		return 0, 0, err
	}
	return units, minimum, nil
}

// validateExactMarginProjections guarantees that additive float terms remain
// compatibility projections of the v3 authority. It validates group and route
// values independently so an override cannot hide malformed base terms from an
// older binary that still reads the projections.
func (e MiddlemanRouteEntry) validateExactMarginProjections() error {
	if e.AccountingVersion != MiddlemanRouteAccountingVersion {
		return fmt.Errorf("exact middleman margin projections require accounting version %q", MiddlemanRouteAccountingVersion)
	}
	if e.GroupMarginUnits > 10_000 {
		return fmt.Errorf("group margin fraction is outside zero through one")
	}
	if err := validateMarginProjection("group margin fraction", e.GroupMarginPct, float64(e.GroupMarginUnits)/10_000); err != nil {
		return err
	}
	if e.GroupMinMarginExact < 0 || e.GroupMinMarginExact > accounting.MaxCPM {
		return fmt.Errorf("group minimum margin CPM is outside the supported range")
	}
	if err := validateMarginProjection("group minimum margin CPM", e.GroupMinMarginCPM, e.GroupMinMarginExact.Float64()); err != nil {
		return err
	}
	if (e.RouteMarginUnits == nil) != (e.RouteMarginPct == nil) {
		return fmt.Errorf("route margin fraction exact and compatibility fields must be present together")
	}
	if e.RouteMarginUnits != nil {
		if *e.RouteMarginUnits > 10_000 {
			return fmt.Errorf("route margin fraction is outside zero through one")
		}
		if err := validateMarginProjection("route margin fraction", *e.RouteMarginPct, float64(*e.RouteMarginUnits)/10_000); err != nil {
			return err
		}
	}
	if (e.RouteMinMarginExact == nil) != (e.RouteMinMarginCPM == nil) {
		return fmt.Errorf("route minimum margin CPM exact and compatibility fields must be present together")
	}
	if e.RouteMinMarginExact != nil {
		if *e.RouteMinMarginExact < 0 || *e.RouteMinMarginExact > accounting.MaxCPM {
			return fmt.Errorf("route minimum margin CPM is outside the supported range")
		}
		if err := validateMarginProjection("route minimum margin CPM", *e.RouteMinMarginCPM, e.RouteMinMarginExact.Float64()); err != nil {
			return err
		}
	}
	return nil
}

func validateMarginProjection(name string, value, exact float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value == 0 && math.Signbit(value) {
		return fmt.Errorf("%s compatibility projection is invalid", name)
	}
	if value != exact {
		return fmt.Errorf("%s compatibility projection does not match its exact value", name)
	}
	return nil
}

func parseMarginFraction4(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts) == 2 && len(parts[1]) > 4 {
		return 0, fmt.Errorf("margin fraction must have at most four decimal places")
	}
	whole, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid margin fraction")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	for len(fraction) < 4 {
		fraction += "0"
	}
	frac := uint64(0)
	if fraction != "" {
		frac, err = strconv.ParseUint(fraction, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid margin fraction")
		}
	}
	if whole > 1 || whole == 1 && frac != 0 {
		return 0, fmt.Errorf("margin fraction must be between zero and one")
	}
	return uint32(whole*10_000 + frac), nil
}

// ValidatePartnerProfile verifies the bounded OpenRTB contract carried by an
// active route. Network-address policy remains a runtime safehttp check so DNS
// rebinding and private destinations cannot be approved from cached metadata.
func (e MiddlemanRouteEntry) ValidatePartnerProfile() error {
	if e.BidderID == 0 || e.AdvID == 0 {
		return fmt.Errorf("bidder and advertiser ids are required")
	}
	if e.SyntheticCampaignID == 0 || e.SyntheticItemID == 0 || e.SyntheticCreativeID == 0 {
		return fmt.Errorf("active synthetic reporting ids are required")
	}
	if e.OpenRTBVersion != "2.5" {
		return fmt.Errorf("openrtb_version must be exactly 2.5")
	}
	parsed, err := url.Parse(e.EndpointURL)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("endpoint_url must be an absolute http or https URL")
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("endpoint_url must not contain user info or a fragment")
	}
	if !ValidMiddlemanCredentialRefName(e.CredentialRef) {
		return fmt.Errorf("credential_ref must be a bounded environment variable name")
	}
	if strings.TrimSpace(e.Seat) != e.Seat || len(e.Seat) > 128 || containsControl(e.Seat) {
		return fmt.Errorf("seat must be at most 128 bytes without surrounding whitespace or control characters")
	}
	if e.GroupTimeoutMS == 0 || e.GroupTimeoutMS > 5000 || e.BidderTimeoutMS == 0 || e.BidderTimeoutMS > 5000 {
		return fmt.Errorf("partner timeouts must be between 1 and 5000 milliseconds")
	}
	if e.RouteTimeoutMS != nil && (*e.RouteTimeoutMS == 0 || *e.RouteTimeoutMS > 5000) {
		return fmt.Errorf("route timeout must be between 1 and 5000 milliseconds")
	}
	for _, value := range []float64{e.GroupMarginPct, e.GroupMinMarginCPM, e.EffectiveMarginPct(), e.EffectiveMinMarginCPM()} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("partner margins must be finite and nonnegative")
		}
	}
	if e.AccountingVersion == MiddlemanRouteAccountingVersion {
		if err := e.validateExactMarginProjections(); err != nil {
			return fmt.Errorf("partner margins: %w", err)
		}
	}
	if _, _, err := e.ExactMarginTerms(); err != nil {
		return fmt.Errorf("partner margins: %w", err)
	}
	return nil
}

// ValidMiddlemanCredentialRefName accepts portable environment-variable names
// only, so an approved reference can be injected consistently by shells and
// service managers.
func ValidMiddlemanCredentialRefName(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (index > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func containsControl(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0
}

func DBGetMiddlemanRouteCache(ctx context.Context, db *sql.DB) (*MiddlemanRouteCache, error) {
	rows, err := db.QueryContext(ctx, `
SELECT
	t.target_id, t.entitytype_id, t.entity_id, t.size_id, t.priority,
	g.group_id, g.trigger_mode, g.total_timeout_ms, g.margin_pct, g.min_margin_cpm,
	rb.route_bidder_id, rb.priority, rb.timeout_ms, rb.margin_pct, rb.min_margin_cpm,
	b.bidder_id, b.adv_id, b.synthetic_campaign_id, b.synthetic_item_id,
	b.synthetic_creative_id, b.endpoint_url, b.openrtb_version, b.seat,
	b.credential_ref, b.timeout_ms
FROM mid_route_target t
INNER JOIN mid_route_group g USING (group_id)
INNER JOIN mid_route_bidder rb USING (group_id)
INNER JOIN adv_bidder b USING (bidder_id)
INNER JOIN adv a USING (adv_id)
INNER JOIN adv_campaign c ON (c.campaign_id=b.synthetic_campaign_id AND c.adv_id=b.adv_id)
INNER JOIN adv_item i ON (i.item_id=b.synthetic_item_id AND i.campaign_id=c.campaign_id)
INNER JOIN adv_creative v ON (v.creative_id=b.synthetic_creative_id AND v.item_id=i.item_id)
WHERE t.active='Yes'
  AND g.active='Yes'
  AND rb.active='Yes'
  AND b.active='Yes'
  AND b.credential_status='Active'
  AND b.credential_ref IS NOT NULL
  AND b.credential_ref <> ''
  AND a.active='Yes'
ORDER BY t.priority, rb.priority, t.target_id, rb.route_bidder_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cache := &MiddlemanRouteCache{Version: MiddlemanRouteCacheVersion, Entries: make([]MiddlemanRouteEntry, 0)}
	audiences := make(map[uint32]*acl.ACLAudience)
	for rows.Next() {
		var e MiddlemanRouteEntry
		var entityType, entityID, sizeID, routeTimeout sql.NullInt64
		var seat sql.NullString
		var groupMargin, groupMinMargin sql.NullString
		var routeMargin, routeMinMargin sql.NullString
		if err := rows.Scan(
			&e.TargetID, &entityType, &entityID, &sizeID, &e.TargetPriority,
			&e.GroupID, &e.TriggerMode, &e.GroupTimeoutMS, &groupMargin, &groupMinMargin,
			&e.RouteBidderID, &e.RouteBidderPriority, &routeTimeout, &routeMargin, &routeMinMargin,
			&e.BidderID, &e.AdvID, &e.SyntheticCampaignID, &e.SyntheticItemID,
			&e.SyntheticCreativeID, &e.EndpointURL, &e.OpenRTBVersion, &seat,
			&e.CredentialRef, &e.BidderTimeoutMS,
		); err != nil {
			return nil, err
		}
		if !groupMargin.Valid || !groupMinMargin.Valid {
			return nil, fmt.Errorf("middleman bidder %d has NULL group margin", e.BidderID)
		}
		e.GroupMarginUnits, err = parseMarginFraction4(groupMargin.String)
		if err != nil {
			return nil, fmt.Errorf("middleman bidder %d group margin: %w", e.BidderID, err)
		}
		e.GroupMinMarginExact, err = accounting.ParseCPM(groupMinMargin.String)
		if err != nil {
			return nil, fmt.Errorf("middleman bidder %d group minimum margin: %w", e.BidderID, err)
		}
		e.AccountingVersion = MiddlemanRouteAccountingVersion
		e.GroupMarginPct = float64(e.GroupMarginUnits) / 10_000
		e.GroupMinMarginCPM = e.GroupMinMarginExact.Float64()
		if entityType.Valid {
			v := uint8(entityType.Int64)
			e.EntityTypeID = &v
		}
		if entityID.Valid {
			v := uint32(entityID.Int64)
			e.EntityID = &v
		}
		if sizeID.Valid {
			v := uint32(sizeID.Int64)
			e.SizeID = &v
		}
		if routeTimeout.Valid {
			v := uint16(routeTimeout.Int64)
			e.RouteTimeoutMS = &v
		}
		if routeMargin.Valid {
			v, err := parseMarginFraction4(routeMargin.String)
			if err != nil {
				return nil, fmt.Errorf("middleman bidder %d route margin: %w", e.BidderID, err)
			}
			e.RouteMarginUnits = &v
			projection := float64(v) / 10_000
			e.RouteMarginPct = &projection
		}
		if routeMinMargin.Valid {
			v, err := accounting.ParseCPM(routeMinMargin.String)
			if err != nil {
				return nil, fmt.Errorf("middleman bidder %d route minimum margin: %w", e.BidderID, err)
			}
			e.RouteMinMarginExact = &v
			projection := v.Float64()
			e.RouteMinMarginCPM = &projection
		}
		if seat.Valid {
			e.Seat = seat.String
		}
		if err := e.ValidatePartnerProfile(); err != nil {
			return nil, fmt.Errorf("middleman bidder %d profile: %w", e.BidderID, err)
		}
		aud, ok := audiences[e.SyntheticItemID]
		if !ok {
			aud, err = acl.DBGetACLAudienceContext(ctx, db, e.SyntheticItemID)
			if err != nil {
				return nil, fmt.Errorf("middleman bidder %d synthetic item %d audience: %w", e.BidderID, e.SyntheticItemID, err)
			}
			audiences[e.SyntheticItemID] = aud
		}
		e.Audience = aud
		cache.Entries = append(cache.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	highWater, err := DBGetMiddlemanRouteHighWater(ctx, db)
	if err != nil {
		return nil, err
	}
	cache.Metadata = &MiddlemanRouteCacheMetadata{
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		EntryCount:       len(cache.Entries),
		Source:           "mysql",
		RouteDBHighWater: highWater,
		Checksum:         cache.RouteChecksum(),
	}
	return cache, nil
}

// DBValidateMiddlemanActivation rejects active route topology that the cache
// compiler would otherwise omit, plus synthetic reporting rows that could bid
// as ordinary local demand. It returns counts and internal ids only, never
// endpoints, advertiser identities, or credential values.
func DBValidateMiddlemanActivation(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("middleman activation database is nil")
	}
	checks := []struct {
		label string
		query string
	}{
		{
			label: "active route groups without active targets",
			query: `SELECT COUNT(*) FROM mid_route_group g
WHERE g.active='Yes' AND NOT EXISTS (
 SELECT 1 FROM mid_route_target t WHERE t.group_id=g.group_id AND t.active='Yes')`,
		},
		{
			label: "active route groups without active bidders",
			query: `SELECT COUNT(*) FROM mid_route_group g
WHERE g.active='Yes' AND NOT EXISTS (
 SELECT 1 FROM mid_route_bidder rb WHERE rb.group_id=g.group_id AND rb.active='Yes')`,
		},
		{
			label: "inactive, unapproved, or invalid routed bidders",
			query: `SELECT COUNT(*)
FROM mid_route_group g
INNER JOIN mid_route_bidder rb USING (group_id)
INNER JOIN adv_bidder b USING (bidder_id)
LEFT JOIN adv_campaign c ON (c.campaign_id=b.synthetic_campaign_id AND c.adv_id=b.adv_id)
LEFT JOIN adv_item i ON (i.item_id=b.synthetic_item_id AND i.campaign_id=c.campaign_id)
LEFT JOIN adv_creative v ON (v.creative_id=b.synthetic_creative_id AND v.item_id=i.item_id)
WHERE g.active='Yes' AND rb.active='Yes'
  AND (b.active<>'Yes' OR b.credential_status<>'Active'
    OR b.credential_ref IS NULL OR b.credential_ref=''
    OR c.campaign_id IS NULL OR i.item_id IS NULL OR v.creative_id IS NULL)`,
		},
		{
			label: "routed synthetic reporting rows enabled as local demand",
			query: `SELECT COUNT(*)
FROM mid_route_group g
INNER JOIN mid_route_bidder rb USING (group_id)
INNER JOIN adv_bidder b USING (bidder_id)
INNER JOIN adv_campaign c ON (c.campaign_id=b.synthetic_campaign_id AND c.adv_id=b.adv_id)
INNER JOIN adv_item i ON (i.item_id=b.synthetic_item_id AND i.campaign_id=c.campaign_id)
INNER JOIN adv_creative v ON (v.creative_id=b.synthetic_creative_id AND v.item_id=i.item_id)
WHERE g.active='Yes' AND rb.active='Yes'
  AND (c.active='Yes' OR i.active='Yes' OR v.active='Yes')`,
		},
		{
			label: "active route targets with invalid inventory or size",
			query: `SELECT COUNT(*)
FROM mid_route_group g
INNER JOIN mid_route_target t USING (group_id)
WHERE g.active='Yes' AND t.active='Yes' AND (
  (t.entitytype_id IS NULL AND t.entity_id IS NOT NULL)
  OR (t.entitytype_id IS NOT NULL AND t.entity_id IS NULL)
  OR (t.entitytype_id IS NOT NULL AND t.entitytype_id NOT IN (3,31,32))
  OR (t.entitytype_id=3 AND NOT EXISTS (SELECT 1 FROM pub p WHERE p.pub_id=t.entity_id AND p.active='Yes'))
  OR (t.entitytype_id=31 AND NOT EXISTS (SELECT 1 FROM pub_site s WHERE s.site_id=t.entity_id AND s.active='Yes'))
  OR (t.entitytype_id=32 AND NOT EXISTS (SELECT 1 FROM pub_slot sl WHERE sl.slot_id=t.entity_id AND sl.active='Yes'))
  OR (t.size_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM def_size ds WHERE ds.size_id=t.size_id)))`,
		},
	}
	for _, check := range checks {
		var count int
		if err := db.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return fmt.Errorf("middleman activation %s: %w", check.label, err)
		}
		if count != 0 {
			return fmt.Errorf("middleman activation has %d %s", count, check.label)
		}
	}
	return nil
}

func (c *MiddlemanRouteCache) RouteChecksum() string {
	if c == nil {
		return ""
	}
	data, err := json.Marshal(c.Entries)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func DBGetMiddlemanRouteHighWater(ctx context.Context, db *sql.DB) (string, error) {
	var highWater sql.NullString
	err := db.QueryRowContext(ctx, `
SELECT DATE_FORMAT(MAX(updated), '%Y-%m-%dT%H:%i:%sZ') FROM (
	SELECT updated FROM adv_bidder
	UNION ALL SELECT updated FROM mid_route_group
	UNION ALL SELECT updated FROM mid_route_bidder
	UNION ALL SELECT updated FROM mid_route_target
) route_updates`).Scan(&highWater)
	if err != nil {
		return "", err
	}
	if !highWater.Valid {
		return "", nil
	}
	return highWater.String, nil
}

func (c *MiddlemanRouteCache) ToRedis(ctx context.Context, conn radix.Client) error {
	return c.ToRedisKey(ctx, conn, HashNameMiddlemanRoutesV2)
}

func (c *MiddlemanRouteCache) ToRedisKey(ctx context.Context, conn radix.Client, key string) error {
	if c == nil || c.Version != MiddlemanRouteCacheVersion {
		return fmt.Errorf("new middleman route writes require cache version %d", MiddlemanRouteCacheVersion)
	}
	if err := c.validateAccountingProvenance(); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "SET", key, string(data)))
}

func MiddlemanRouteCacheFromRedis(ctx context.Context, conn radix.Client) (*MiddlemanRouteCache, error) {
	cache, err := middlemanRouteCacheFromRedisKey(ctx, conn, HashNameMiddlemanRoutesV2)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		return cache, nil
	}
	cache, err = middlemanRouteCacheFromRedisKey(ctx, conn, HashNameMiddlemanRoutes)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		return cache, nil
	}
	return &MiddlemanRouteCache{Version: MiddlemanRouteCacheVersion}, nil
}

func middlemanRouteCacheFromRedisKey(ctx context.Context, conn radix.Client, key string) (*MiddlemanRouteCache, error) {
	var data []byte
	if err := conn.Do(ctx, radix.Cmd(&data, "GET", key)); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var cache MiddlemanRouteCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	switch cache.Version {
	case MiddlemanRouteCacheVersion, MiddlemanRouteCacheLegacyVersion:
		if err := cache.validateAccountingProvenance(); err != nil {
			return nil, err
		}
		return &cache, nil
	default:
		return nil, fmt.Errorf("middleman route cache version %d, want %d or %d", cache.Version, MiddlemanRouteCacheVersion, MiddlemanRouteCacheLegacyVersion)
	}
}

func DBGetMiddlemanRoutesToRedis(ctx context.Context, conn radix.Client, db *sql.DB) error {
	return DBGetMiddlemanRoutesToRedisKeys(ctx, conn, db, HashNameMiddlemanRoutes, HashNameMiddlemanRoutesV2)
}

// DBGetMiddlemanRoutesToRedisKeys writes both route cache versions to explicit keys.
func DBGetMiddlemanRoutesToRedisKeys(ctx context.Context, conn radix.Client, db *sql.DB, legacyKey, currentKey string) error {
	cache, err := DBGetMiddlemanRouteCache(ctx, db)
	if err != nil {
		return err
	}
	return writeMiddlemanRouteCacheKeys(ctx, conn, cache, legacyKey, currentKey)
}

func writeMiddlemanRouteCacheKeys(ctx context.Context, conn radix.Client, cache *MiddlemanRouteCache, legacyKey, currentKey string) error {
	if conn == nil {
		return errors.New("middleman route Redis client is nil")
	}
	if legacyKey == "" || currentKey == "" || legacyKey == currentKey {
		return errors.New("middleman route Redis keys must be nonempty and distinct")
	}
	if cache == nil || cache.Version != MiddlemanRouteCacheVersion {
		return fmt.Errorf("current middleman route cache version must be %d", MiddlemanRouteCacheVersion)
	}
	if err := cache.validateAccountingProvenance(); err != nil {
		return err
	}
	legacy := cache.legacyFallbackCache()
	if err := legacy.validateAccountingProvenance(); err != nil {
		return err
	}
	legacyData, err := json.Marshal(legacy)
	if err != nil {
		return err
	}
	currentData, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "MSET",
		legacyKey, string(legacyData),
		currentKey, string(currentData),
	))
}

func (c *MiddlemanRouteCache) validateAccountingProvenance() error {
	if c == nil {
		return fmt.Errorf("middleman route cache is nil")
	}
	for index, entry := range c.Entries {
		switch c.Version {
		case MiddlemanRouteCacheVersion:
			if entry.AccountingVersion != MiddlemanRouteAccountingVersion {
				return fmt.Errorf("current middleman route entry %d has accounting version %q, want %q", index, entry.AccountingVersion, MiddlemanRouteAccountingVersion)
			}
		case MiddlemanRouteCacheLegacyVersion:
			if entry.AccountingVersion != "" && entry.AccountingVersion != MiddlemanRouteAccountingVersion {
				return fmt.Errorf("legacy middleman route entry %d has unsupported accounting version %q", index, entry.AccountingVersion)
			}
		default:
			return fmt.Errorf("unsupported middleman route cache version %d", c.Version)
		}
		if entry.AccountingVersion == MiddlemanRouteAccountingVersion {
			if err := entry.validateExactMarginProjections(); err != nil {
				return fmt.Errorf("middleman route entry %d: %w", index, err)
			}
		}
	}
	return nil
}

func (c *MiddlemanRouteCache) legacyFallbackCache() *MiddlemanRouteCache {
	legacy := &MiddlemanRouteCache{Version: MiddlemanRouteCacheLegacyVersion, Entries: make([]MiddlemanRouteEntry, 0)}
	if c == nil {
		return legacy
	}
	for _, entry := range c.Entries {
		if entry.TriggerMode != "" && entry.TriggerMode != "Fallback" {
			continue
		}
		entry.TriggerMode = ""
		legacy.Entries = append(legacy.Entries, entry)
	}
	legacy.Metadata = &MiddlemanRouteCacheMetadata{
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		EntryCount:       len(legacy.Entries),
		Source:           "mysql-legacy",
		RouteDBHighWater: "",
		Checksum:         legacy.RouteChecksum(),
	}
	if c.Metadata != nil {
		legacy.Metadata.GeneratedAt = c.Metadata.GeneratedAt
		legacy.Metadata.RouteDBHighWater = c.Metadata.RouteDBHighWater
	}
	return legacy
}

func EntityPointer(entityType, entityID uint32) (*uint8, *uint32) {
	t := uint8(entityType)
	id := entityID
	return &t, &id
}

func Uint32Pointer(v uint32) *uint32 {
	return &v
}

func Uint16Pointer(v uint16) *uint16 {
	return &v
}

func Float64Pointer(v float64) *float64 {
	return &v
}

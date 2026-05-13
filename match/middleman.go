package match

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/genelet/winter/acl"
	"github.com/mediocregopher/radix/v4"
)

const (
	HashNameMiddlemanRoutes = "middleman:routes"

	MiddlemanRouteCacheVersion = 1
)

type MiddlemanRouteCache struct {
	Version int                   `json:"version"`
	Entries []MiddlemanRouteEntry `json:"entries"`
}

type MiddlemanRouteEntry struct {
	TargetID            uint32           `json:"target_id"`
	GroupID             uint32           `json:"group_id"`
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

func DBGetMiddlemanRouteCache(ctx context.Context, db *sql.DB) (*MiddlemanRouteCache, error) {
	rows, err := db.QueryContext(ctx, `
SELECT
	t.target_id, t.entitytype_id, t.entity_id, t.size_id, t.priority,
	g.group_id, g.total_timeout_ms, g.margin_pct, g.min_margin_cpm,
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
  AND g.trigger_mode='Fallback'
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

	cache := &MiddlemanRouteCache{Version: MiddlemanRouteCacheVersion}
	audiences := make(map[uint32]*acl.ACLAudience)
	for rows.Next() {
		var e MiddlemanRouteEntry
		var entityType, entityID, sizeID, routeTimeout sql.NullInt64
		var seat sql.NullString
		var routeMargin, routeMinMargin sql.NullFloat64
		if err := rows.Scan(
			&e.TargetID, &entityType, &entityID, &sizeID, &e.TargetPriority,
			&e.GroupID, &e.GroupTimeoutMS, &e.GroupMarginPct, &e.GroupMinMarginCPM,
			&e.RouteBidderID, &e.RouteBidderPriority, &routeTimeout, &routeMargin, &routeMinMargin,
			&e.BidderID, &e.AdvID, &e.SyntheticCampaignID, &e.SyntheticItemID,
			&e.SyntheticCreativeID, &e.EndpointURL, &e.OpenRTBVersion, &seat,
			&e.CredentialRef, &e.BidderTimeoutMS,
		); err != nil {
			return nil, err
		}
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
			v := routeMargin.Float64
			e.RouteMarginPct = &v
		}
		if routeMinMargin.Valid {
			v := routeMinMargin.Float64
			e.RouteMinMarginCPM = &v
		}
		if seat.Valid {
			e.Seat = seat.String
		}
		aud, ok := audiences[e.SyntheticItemID]
		if !ok {
			aud, err = acl.DBGetACLAudience(db, e.SyntheticItemID)
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
	return cache, nil
}

func (c *MiddlemanRouteCache) ToRedis(ctx context.Context, conn radix.Client) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "SET", HashNameMiddlemanRoutes, string(data)))
}

func MiddlemanRouteCacheFromRedis(ctx context.Context, conn radix.Client) (*MiddlemanRouteCache, error) {
	var data []byte
	if err := conn.Do(ctx, radix.Cmd(&data, "GET", HashNameMiddlemanRoutes)); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return &MiddlemanRouteCache{Version: MiddlemanRouteCacheVersion}, nil
	}
	var cache MiddlemanRouteCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache.Version != MiddlemanRouteCacheVersion {
		return nil, fmt.Errorf("middleman route cache version %d, want %d", cache.Version, MiddlemanRouteCacheVersion)
	}
	return &cache, nil
}

func DBGetMiddlemanRoutesToRedis(ctx context.Context, conn radix.Client, db *sql.DB) error {
	cache, err := DBGetMiddlemanRouteCache(ctx, db)
	if err != nil {
		return err
	}
	return cache.ToRedis(ctx, conn)
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

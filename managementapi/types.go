package managementapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	ScopeCampaignRead   = "api.campaign.read"
	ScopeCampaignWrite  = "api.campaign.write"
	ScopeCreativeRead   = "api.creative.read"
	ScopeCreativeWrite  = "api.creative.write"
	ScopeTargetingRead  = "api.targeting.read"
	ScopeTargetingWrite = "api.targeting.write"
	ScopeReportRead     = "api.report.read"
)

var allowedScopes = map[string]struct{}{
	ScopeCampaignRead: {}, ScopeCampaignWrite: {},
	ScopeCreativeRead: {}, ScopeCreativeWrite: {},
	ScopeTargetingRead: {}, ScopeTargetingWrite: {}, ScopeReportRead: {},
}

var (
	ErrUnauthorized        = errors.New("invalid service credential")
	ErrForbidden           = errors.New("credential scope does not permit this operation")
	ErrNotFound            = errors.New("resource not found")
	ErrConflict            = errors.New("resource version conflict")
	ErrIdempotencyConflict = errors.New("idempotency key was already used for a different request")
	ErrIdempotencyPending  = errors.New("idempotent request is still being processed")
)

type Actor struct {
	Role string
	ID   uint64
}

type Principal struct {
	CredentialID uint64
	AdvertiserID uint64
	Name         string
	Scopes       map[string]struct{}
	ExpiresAt    time.Time
}

func (p Principal) Has(scope string) bool {
	_, ok := p.Scopes[scope]
	return ok
}

type Credential struct {
	ID           uint64     `json:"id"`
	AdvertiserID uint64     `json:"advertiser_id"`
	Name         string     `json:"name"`
	PublicID     string     `json:"public_id"`
	Scopes       []string   `json:"scopes"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}

type Limits struct {
	SpendUSD *float64 `json:"spend_usd,omitempty"`
	Imps     *uint64  `json:"impressions,omitempty"`
	Clicks   *uint64  `json:"clicks,omitempty"`
}

type DeliveryPolicy struct {
	StartUTC       *time.Time `json:"start_utc,omitempty"`
	EndUTC         *time.Time `json:"end_utc,omitempty"`
	Timezone       string     `json:"timezone,omitempty"`
	WeeklySchedule *string    `json:"weekly_schedule,omitempty"`
	Pacing         string     `json:"pacing"`
	TotalLimits    Limits     `json:"total_limits"`
	DailyLimits    Limits     `json:"daily_limits"`
}

type Campaign struct {
	ID           uint64         `json:"id"`
	AdvertiserID uint64         `json:"advertiser_id"`
	Name         string         `json:"name"`
	ExternalID   string         `json:"external_id,omitempty"`
	TargetType   string         `json:"target_type,omitempty"`
	Description  string         `json:"description,omitempty"`
	Status       string         `json:"status"`
	Version      uint64         `json:"version"`
	Delivery     DeliveryPolicy `json:"delivery"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Item struct {
	ID             uint64         `json:"id"`
	CampaignID     uint64         `json:"campaign_id"`
	Name           string         `json:"name"`
	LandingURL     string         `json:"landing_url"`
	ImpressionURLs []string       `json:"impression_urls,omitempty"`
	ClickURLs      []string       `json:"click_urls,omitempty"`
	PriceCPMUSD    float64        `json:"price_cpm_usd"`
	Status         string         `json:"status"`
	Version        uint64         `json:"version"`
	Delivery       DeliveryPolicy `json:"delivery"`
	CreatedAt      time.Time      `json:"created_at"`
}

type NativeCreative struct {
	Version      string `json:"version"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	CTA          string `json:"cta,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	MainImageURL string `json:"main_image_url"`
}

type Creative struct {
	ID        uint64          `json:"id"`
	ItemID    uint64          `json:"item_id"`
	Name      string          `json:"name"`
	SizeID    uint64          `json:"-"`
	Width     uint16          `json:"width"`
	Height    uint16          `json:"height"`
	MediaType string          `json:"media_type"`
	SourceURL string          `json:"source_url,omitempty"`
	Native    *NativeCreative `json:"native,omitempty"`
	Weight    float64         `json:"weight"`
	Status    string          `json:"status"`
	Version   uint64          `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
}

type Targeting struct {
	ItemID       uint64   `json:"item_id"`
	Version      uint64   `json:"version"`
	SiteTypes    []string `json:"site_types"`
	Languages    []string `json:"languages"`
	DeviceTypes  []string `json:"device_types"`
	Positions    []string `json:"positions"`
	AccessOrder  string   `json:"access_order"`
	ChannelOrder string   `json:"channel_order"`
}

type Operation struct {
	ID                 string     `json:"id"`
	ResourceType       string     `json:"resource_type"`
	ResourceID         uint64     `json:"resource_id"`
	AcceptedVersion    uint64     `json:"accepted_version"`
	ConfigurationState string     `json:"configuration_state"`
	ActivationState    string     `json:"activation_state"`
	AcceptedAt         time.Time  `json:"accepted_at"`
	ActivationDeadline time.Time  `json:"activation_deadline"`
	ActivatedAt        *time.Time `json:"activated_at,omitempty"`
	PublicationMode    string     `json:"publication_mode,omitempty"`
}

type DeliveryReport struct {
	ID                   uint64    `json:"id"`
	IntervalUTC          time.Time `json:"interval_utc"`
	CampaignID           uint64    `json:"campaign_id"`
	ItemID               uint64    `json:"item_id"`
	CreativeID           uint64    `json:"creative_id"`
	DemandSource         string    `json:"demand_source"`
	InventoryEnvironment string    `json:"inventory_environment"`
	IntegrationMode      string    `json:"integration_mode"`
	MediaIntent          string    `json:"media_intent"`
	SellerType           string    `json:"seller_type"`
	SellerID             string    `json:"seller_id,omitempty"`
	Wins                 uint64    `json:"wins"`
	Impressions          uint64    `json:"impressions"`
	Clicks               uint64    `json:"clicks"`
	SpendUSD             string    `json:"spend_usd"`
	AccountingVersion    string    `json:"accounting_version"`
}

type MutationResult struct {
	Status int
	Body   []byte
	Replay bool
}

type pageMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	Limit      int    `json:"limit"`
	Order      string `json:"order"`
	Timezone   string `json:"timezone"`
	Currency   string `json:"currency,omitempty"`
	Freshness  string `json:"freshness,omitempty"`
	Source     string `json:"source,omitempty"`
	Partial    bool   `json:"partial,omitempty"`
}

type envelope struct {
	Data any       `json:"data,omitempty"`
	Meta *pageMeta `json:"meta,omitempty"`
}

type errorDetail struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

func parseUint(raw, name string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return id, nil
}

func validateScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 || len(scopes) > len(allowedScopes) {
		return nil, fmt.Errorf("at least one supported API scope is required")
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if _, ok := allowedScopes[scope]; !ok {
			return nil, fmt.Errorf("unsupported API scope %q", scope)
		}
		seen[scope] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for scope := range seen {
		out = append(out, scope)
	}
	sortStrings(out)
	return out, nil
}

func scopesMap(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, scope := range strings.Split(raw, ",") {
		if _, ok := allowedScopes[scope]; ok {
			out[scope] = struct{}{}
		}
	}
	return out
}

func marshalEnvelope(data any) ([]byte, error) {
	return json.Marshal(envelope{Data: data})
}

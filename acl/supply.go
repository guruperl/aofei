package acl

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SellerMetadata is public commercial transparency data owned by the existing
// publisher account. Authorization is an operator approval; it is not payment
// authority and does not change the publisher settlement owner.
type SellerMetadata struct {
	ID         string `json:"id,omitempty"`
	Type       string `json:"type,omitempty"`
	ASI        string `json:"asi,omitempty"`
	Name       string `json:"name,omitempty"`
	Domain     string `json:"domain,omitempty"`
	Authorized bool   `json:"authorized,omitempty"`
}

// SiteSupplyMetadata describes one approved inventory source. Empty values
// decode as explicit Unknown categories for pre-P02 cache generations.
type SiteSupplyMetadata struct {
	Environment       string `json:"environment,omitempty"`
	CanonicalIdentity string `json:"canonical_identity,omitempty"`
	StoreURL          string `json:"store_url,omitempty"`
	IntegrationMode   string `json:"integration_mode,omitempty"`
}

// SlotSupplyMetadata describes durable placement intent and quality. Request
// mediaTypes remain separately validated runtime input.
type SlotSupplyMetadata struct {
	MediaIntent       string `json:"media_intent,omitempty"`
	Placement         string `json:"placement,omitempty"`
	RenderContext     string `json:"render_context,omitempty"`
	RefreshMode       string `json:"refresh_mode,omitempty"`
	RefreshSeconds    uint16 `json:"refresh_seconds,omitempty"`
	AdDensity         string `json:"ad_density,omitempty"`
	TrafficQuality    string `json:"traffic_quality,omitempty"`
	SourceQuality     string `json:"source_quality,omitempty"`
	ManagementControl string `json:"management_control,omitempty"`
}

// SupplyMetadata is the privacy-safe, public subset allowed in request audits,
// reporting facts, and partner requests.
type SupplyMetadata struct {
	Seller SellerMetadata     `json:"seller"`
	Site   SiteSupplyMetadata `json:"site"`
	Slot   SlotSupplyMetadata `json:"slot"`
}

var (
	SellerTypes           = []string{"Publisher", "Intermediary"}
	InventoryEnvironments = []string{"Unknown", "Web", "App", "CTV", "DOOH", "Other"}
	IntegrationModes      = []string{"Unknown", "ADX", "BrowserTag", "SDK", "ServerAPI"}
	MediaIntents          = []string{"Unknown", "Banner", "Video", "Native", "Audio", "Multi"}
	Placements            = []string{"Unknown", "AboveFold", "InFeed", "Interstitial", "Rewarded", "Sticky", "Popup", "Other"}
	RenderContexts        = []string{"Unknown", "WebPage", "InApp", "Player", "Fullscreen", "Other"}
	RefreshModes          = []string{"Unknown", "None", "Timed", "Event"}
	AdDensities           = []string{"Unknown", "Low", "Standard", "High"}
	TrafficQualities      = []string{"Unknown", "Reviewed", "Sampled", "Suspicious", "Blocked"}
	SourceQualities       = []string{"Unknown", "OwnedOperated", "Partner", "Network", "Resale"}
	ManagementControls    = []string{"Unknown", "Publisher", "Operator", "Partner"}
)

func normalizeControlled(value string, allowed []string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return value
}

func validControlled(value string, allowed []string) bool {
	value = normalizeControlled(value, allowed)
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func (m SellerMetadata) Validate() error {
	if m.Type == "" {
		m.Type = "Publisher"
	}
	if !validControlled(m.Type, SellerTypes) {
		return fmt.Errorf("invalid seller type %q", m.Type)
	}
	if len(m.ID) > 64 || !validSellerID(m.ID) {
		return fmt.Errorf("invalid seller id")
	}
	for label, value := range map[string]string{"asi": m.ASI, "domain": m.Domain} {
		if value != "" && !validSellerDomain(value) {
			return fmt.Errorf("invalid seller %s", label)
		}
	}
	if len(m.Name) > 255 || m.Name != strings.TrimSpace(m.Name) || hasControlCharacter(m.Name) {
		return fmt.Errorf("invalid seller name")
	}
	if m.Authorized && (m.ID == "" || m.ASI == "") {
		return fmt.Errorf("authorized seller requires id and asi")
	}
	return nil
}

func validSellerDomain(value string) bool {
	return strings.Contains(value, ".") && net.ParseIP(value) == nil &&
		validCommercialSiteIdentity(value, SiteTypeWeb)
}

func hasControlCharacter(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func validSellerID(value string) bool {
	if value == "" {
		return true
	}
	if value != strings.TrimSpace(value) {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || strings.ContainsRune("-_.:", character) {
			continue
		}
		return false
	}
	return true
}

func (m SiteSupplyMetadata) Normalize() SiteSupplyMetadata {
	m.Environment = normalizeControlled(m.Environment, InventoryEnvironments)
	m.IntegrationMode = normalizeControlled(m.IntegrationMode, IntegrationModes)
	return m
}

func (m SiteSupplyMetadata) Validate() error {
	m = m.Normalize()
	if !validControlled(m.Environment, InventoryEnvironments) || !validControlled(m.IntegrationMode, IntegrationModes) {
		return fmt.Errorf("invalid site supply taxonomy")
	}
	if len(m.CanonicalIdentity) > 255 || (m.CanonicalIdentity != "" && m.CanonicalIdentity != strings.TrimSpace(m.CanonicalIdentity)) {
		return fmt.Errorf("invalid canonical inventory identity")
	}
	if m.CanonicalIdentity != "" && !ValidCanonicalIdentity(m.CanonicalIdentity, m.Environment) {
		return fmt.Errorf("canonical inventory identity does not match its environment")
	}
	if m.StoreURL != "" {
		parsed, err := url.Parse(m.StoreURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || len(m.StoreURL) > 1024 {
			return fmt.Errorf("invalid public inventory URL")
		}
	}
	return nil
}

// ValidCanonicalIdentity applies the public identity rules used by both the
// publisher UI and cache publication.
func ValidCanonicalIdentity(identity, environment string) bool {
	identity = strings.TrimSpace(identity)
	if identity == "" || len(identity) > 255 {
		return false
	}
	if environment == "Web" {
		return validCommercialSiteIdentity(identity, SiteTypeWeb)
	}
	for _, character := range identity {
		if character <= ' ' || character == 0x7f {
			return false
		}
	}
	return true
}

func (m SlotSupplyMetadata) Normalize() SlotSupplyMetadata {
	m.MediaIntent = normalizeControlled(m.MediaIntent, MediaIntents)
	m.Placement = normalizeControlled(m.Placement, Placements)
	m.RenderContext = normalizeControlled(m.RenderContext, RenderContexts)
	m.RefreshMode = normalizeControlled(m.RefreshMode, RefreshModes)
	m.AdDensity = normalizeControlled(m.AdDensity, AdDensities)
	m.TrafficQuality = normalizeControlled(m.TrafficQuality, TrafficQualities)
	m.SourceQuality = normalizeControlled(m.SourceQuality, SourceQualities)
	m.ManagementControl = normalizeControlled(m.ManagementControl, ManagementControls)
	return m
}

func (m SlotSupplyMetadata) Validate() error {
	m = m.Normalize()
	checks := []struct {
		value   string
		allowed []string
	}{
		{m.MediaIntent, MediaIntents}, {m.Placement, Placements}, {m.RenderContext, RenderContexts},
		{m.RefreshMode, RefreshModes}, {m.AdDensity, AdDensities}, {m.TrafficQuality, TrafficQualities},
		{m.SourceQuality, SourceQualities}, {m.ManagementControl, ManagementControls},
	}
	for _, check := range checks {
		if !validControlled(check.value, check.allowed) {
			return fmt.Errorf("invalid slot supply taxonomy")
		}
	}
	if m.RefreshMode != "Timed" && m.RefreshSeconds != 0 {
		return fmt.Errorf("refresh seconds require Timed refresh mode")
	}
	if m.RefreshMode == "Timed" && (m.RefreshSeconds < 15 || m.RefreshSeconds > 3600) {
		return fmt.Errorf("timed refresh must be between 15 and 3600 seconds")
	}
	return nil
}

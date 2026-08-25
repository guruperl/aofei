package managementapi

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/guruperl/aofei/accounting"
	"github.com/guruperl/aofei/match"
)

type campaignWrite struct {
	Name        string         `json:"name"`
	ExternalID  string         `json:"external_id,omitempty"`
	TargetType  string         `json:"target_type,omitempty"`
	Description string         `json:"description,omitempty"`
	Delivery    DeliveryPolicy `json:"delivery"`
}

type itemWrite struct {
	Name           string         `json:"name"`
	LandingURL     string         `json:"landing_url"`
	ImpressionURLs []string       `json:"impression_urls,omitempty"`
	ClickURLs      []string       `json:"click_urls,omitempty"`
	PriceCPMUSD    ExactDecimal   `json:"price_cpm_usd"`
	Delivery       DeliveryPolicy `json:"delivery"`
}

type creativeWrite struct {
	Name      string          `json:"name"`
	Width     uint16          `json:"width"`
	Height    uint16          `json:"height"`
	MediaType string          `json:"media_type"`
	SourceURL string          `json:"source_url,omitempty"`
	Native    *NativeCreative `json:"native,omitempty"`
	Weight    float64         `json:"weight"`
	Status    string          `json:"status,omitempty"`
}

type targetingWrite struct {
	SiteTypes    []string `json:"site_types"`
	Languages    []string `json:"languages"`
	DeviceTypes  []string `json:"device_types"`
	Positions    []string `json:"positions"`
	AccessOrder  string   `json:"access_order"`
	ChannelOrder string   `json:"channel_order"`
}

// Public request shapes used by generated clients. They intentionally omit
// advertiser ids, roles, permissions, audit actors, and activation claims.
type CampaignInput = campaignWrite
type ItemInput = itemWrite
type CreativeInput = creativeWrite
type TargetingInput = targetingWrite

func validateCampaignWrite(input *campaignWrite) error {
	if input == nil {
		return fmt.Errorf("campaign body is required")
	}
	if err := validateName("name", &input.Name, 255); err != nil {
		return err
	}
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	if len(input.ExternalID) > 255 || hasControl(input.ExternalID) {
		return fmt.Errorf("external_id must be at most 255 characters without control characters")
	}
	if len(input.Description) > 4096 || hasControl(input.Description) {
		return fmt.Errorf("description must be at most 4096 characters without control characters")
	}
	if input.TargetType != "" && input.TargetType != "App" && input.TargetType != "Web" {
		return fmt.Errorf("target_type must be App or Web")
	}
	return validateDelivery(&input.Delivery, true)
}

func validateItemWrite(input *itemWrite) error {
	if input == nil {
		return fmt.Errorf("item body is required")
	}
	if err := validateName("name", &input.Name, 255); err != nil {
		return err
	}
	input.LandingURL = strings.TrimSpace(input.LandingURL)
	if err := validateHTTPURL("landing_url", input.LandingURL, false, nil); err != nil {
		return err
	}
	if len(input.ImpressionURLs) > 10 || len(input.ClickURLs) > 10 {
		return fmt.Errorf("no more than ten impression_urls or click_urls are allowed")
	}
	for _, set := range []struct {
		name string
		urls []string
	}{{"impression_urls", input.ImpressionURLs}, {"click_urls", input.ClickURLs}} {
		for index, raw := range set.urls {
			raw = strings.TrimSpace(raw)
			if strings.Contains(raw, ",") {
				return fmt.Errorf("%s cannot contain a comma", set.name)
			}
			if err := validateHTTPURL(set.name, raw, false, nil); err != nil {
				return err
			}
			set.urls[index] = raw
		}
	}
	cpm, err := accounting.ParseCPM(input.PriceCPMUSD.String())
	if err != nil || cpm <= 0 || input.PriceCPMUSD.String() != cpm.String() {
		return fmt.Errorf("price_cpm_usd must be a canonical six-place exact decimal string from 0.000001 through %s", accounting.MaxCPM)
	}
	return validateDelivery(&input.Delivery, false)
}

func validateCreativeWrite(input *creativeWrite) error {
	if input == nil {
		return fmt.Errorf("creative body is required")
	}
	if err := validateName("name", &input.Name, 255); err != nil {
		return err
	}
	if input.Width == 0 || input.Height == 0 {
		return fmt.Errorf("positive width and height are required")
	}
	if input.Weight <= 0 || math.IsNaN(input.Weight) || math.IsInf(input.Weight, 0) || input.Weight > 1_000_000 {
		return fmt.Errorf("weight must be a finite positive value no greater than 1000000")
	}
	if input.Status == "" {
		input.Status = "Yes"
	}
	if input.Status != "Yes" && input.Status != "No" {
		return fmt.Errorf("status must be Yes or No")
	}
	switch input.MediaType {
	case match.CreativeMediaBanner:
		if input.Native != nil {
			return fmt.Errorf("native must be omitted for Banner creative")
		}
		return validateHTTPURL("source_url", input.SourceURL, true, map[string]struct{}{".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}})
	case match.CreativeMediaVideo:
		if input.Native != nil {
			return fmt.Errorf("native must be omitted for Video creative")
		}
		return validateHTTPURL("source_url", input.SourceURL, true, map[string]struct{}{".mp4": {}, ".webm": {}, ".m3u8": {}})
	case match.CreativeMediaNative:
		if strings.TrimSpace(input.SourceURL) != "" || input.Native == nil {
			return fmt.Errorf("Native creative requires native and omits source_url")
		}
		native := match.NativeCreativeV1{
			Version: input.Native.Version, Title: strings.TrimSpace(input.Native.Title),
			Description: strings.TrimSpace(input.Native.Description), CTA: strings.TrimSpace(input.Native.CTA),
			IconURL: strings.TrimSpace(input.Native.IconURL), MainImageURL: strings.TrimSpace(input.Native.MainImageURL),
		}
		if err := native.Validate(); err != nil {
			return err
		}
		if native.IconURL != "" {
			if err := validateHTTPURL("native.icon_url", native.IconURL, true, map[string]struct{}{".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}}); err != nil {
				return err
			}
		}
		return validateHTTPURL("native.main_image_url", native.MainImageURL, true, map[string]struct{}{".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}})
	default:
		return fmt.Errorf("media_type must be Banner, Video, or Native")
	}
}

func validateTargetingWrite(input *targetingWrite) error {
	if input == nil {
		return fmt.Errorf("targeting body is required")
	}
	sets := []struct {
		name    string
		values  *[]string
		allowed map[string]struct{}
	}{
		{"site_types", &input.SiteTypes, stringSet("App", "Web")},
		{"languages", &input.Languages, stringSet("EN", "ES", "RU", "DE", "FR", "JA", "PT", "TR", "IT", "FA", "NL", "PL", "ZH", "VI", "ID", "CS", "KO", "UK", "AR", "EL", "FI", "HE", "SV", "RO", "HU", "TH", "DA", "SK", "BG", "SR", "NB", "Other")},
		{"device_types", &input.DeviceTypes, stringSet("0", "1", "2", "3", "4", "5", "6", "7")},
		{"positions", &input.Positions, stringSet("0", "1", "2", "3", "4", "5", "6", "7")},
	}
	for _, set := range sets {
		if len(*set.values) == 0 {
			return fmt.Errorf("%s must contain at least one value", set.name)
		}
		seen := make(map[string]struct{}, len(*set.values))
		for _, value := range *set.values {
			if _, ok := set.allowed[value]; !ok {
				return fmt.Errorf("%s contains unsupported value %q", set.name, value)
			}
			seen[value] = struct{}{}
		}
		normalized := make([]string, 0, len(seen))
		for value := range seen {
			normalized = append(normalized, value)
		}
		sortStrings(normalized)
		*set.values = normalized
	}
	if input.AccessOrder == "" {
		input.AccessOrder = "Inherit"
	}
	if input.AccessOrder != "White" && input.AccessOrder != "Black" && input.AccessOrder != "Inherit" {
		return fmt.Errorf("access_order must be White, Black, or Inherit")
	}
	if input.ChannelOrder == "" {
		input.ChannelOrder = "Black"
	}
	if input.ChannelOrder != "White" && input.ChannelOrder != "Black" {
		return fmt.Errorf("channel_order must be White or Black")
	}
	return nil
}

func validateDelivery(policy *DeliveryPolicy, campaign bool) error {
	if policy.Pacing == "" {
		policy.Pacing = "Fast"
	}
	if policy.Pacing != "Fast" && policy.Pacing != "Even" {
		return fmt.Errorf("delivery.pacing must be Fast or Even")
	}
	if campaign {
		if policy.Timezone == "" {
			policy.Timezone = "UTC"
		}
		if len(policy.Timezone) >= 64 {
			return fmt.Errorf("delivery.timezone is too long")
		}
		if _, err := time.LoadLocation(policy.Timezone); err != nil {
			return fmt.Errorf("delivery.timezone is invalid")
		}
	} else if policy.Timezone != "" {
		return fmt.Errorf("item delivery inherits the campaign timezone")
	}
	for _, field := range []struct {
		name  string
		value **time.Time
	}{{"start_utc", &policy.StartUTC}, {"end_utc", &policy.EndUTC}} {
		if *field.value == nil {
			continue
		}
		value := (*field.value).UTC()
		if value.Year() < 1000 || value.Year() > 9999 {
			return fmt.Errorf("delivery.%s is outside the database time range", field.name)
		}
		*field.value = &value
	}
	if policy.StartUTC != nil && policy.EndUTC != nil && policy.EndUTC.Before(*policy.StartUTC) {
		return fmt.Errorf("delivery.end_utc cannot be before start_utc")
	}
	if policy.WeeklySchedule != nil {
		value := *policy.WeeklySchedule
		if len(value) != 168 || strings.Trim(value, "01") != "" || !strings.Contains(value, "1") {
			return fmt.Errorf("delivery.weekly_schedule must contain 168 zero/one hours and at least one enabled hour")
		}
	}
	for _, field := range []struct {
		name   string
		limits Limits
	}{{"total_limits", policy.TotalLimits}, {"daily_limits", policy.DailyLimits}} {
		if field.limits.SpendUSD != nil {
			amount, err := accounting.ParseNano(field.limits.SpendUSD.String())
			if err != nil || amount < 0 || field.limits.SpendUSD.String() != amount.String() {
				return fmt.Errorf("delivery.%s.spend_usd must be a canonical nine-place non-negative exact decimal string", field.name)
			}
		}
		if field.limits.Imps != nil && *field.limits.Imps > math.MaxUint32 {
			return fmt.Errorf("delivery.%s.impressions must fit an unsigned 32-bit limit", field.name)
		}
		if field.limits.Clicks != nil && *field.limits.Clicks > math.MaxUint32 {
			return fmt.Errorf("delivery.%s.clicks must fit an unsigned 32-bit limit", field.name)
		}
	}
	return nil
}

func validateHTTPURL(name, raw string, https bool, extensions map[string]struct{}) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL without credentials or fragment", name)
	}
	if https && u.Scheme != "https" {
		return fmt.Errorf("%s must use HTTPS", name)
	}
	if len(raw) > 2048 || hasControl(raw) {
		return fmt.Errorf("%s is too long or contains a control character", name)
	}
	if len(extensions) != 0 {
		ext := strings.ToLower(path.Ext(u.Path))
		if _, ok := extensions[ext]; !ok {
			return fmt.Errorf("%s has an unsupported file type %q", name, ext)
		}
	}
	return nil
}

func validateName(name string, value *string, max int) error {
	*value = strings.TrimSpace(*value)
	if *value == "" || utf8.RuneCountInString(*value) > max || hasControl(*value) {
		return fmt.Errorf("%s is required and must be at most %d characters without control characters", name, max)
	}
	return nil
}

func hasControl(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0
}

func stringSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func packedSize(width, height uint16) uint64 {
	return uint64(uint32(width)<<16 | uint32(height))
}

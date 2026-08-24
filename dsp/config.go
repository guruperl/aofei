package dsp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/guruperl/aofei/hostedpayment"
	"github.com/guruperl/aofei/managementapi"
	"github.com/guruperl/aofei/publisherauth"
	"github.com/guruperl/aofei/trafficquality"
	"github.com/mediocregopher/radix/v4"
)

type Red struct {
	Network string
	Addr    string
	User    string
	Pass    string
	Size    int
}

type Config struct {
	DocumentRoot                string                   `json:"document_root"`
	ServerURL                   string                   `json:"server_url"`
	ServerPort                  string                   `json:"server_port"`
	Ips                         string                   `json:"ips,omitempty"`
	Redis                       *Red                     `json:"redis,omitempty"`
	NatsURL                     string                   `json:"nats_url,omitempty"`
	TrackingSecret              string                   `json:"tracking_secret,omitempty"`
	TrackingSignatureTTLSeconds int                      `json:"tracking_signature_ttl_seconds,omitempty"`
	DirectSSPTokens             DirectSSPTokenConfig     `json:"direct_ssp_tokens,omitempty"`
	DirectSSPAuth               publisherauth.Config     `json:"direct_ssp_auth,omitempty"`
	CapStateTTLSeconds          int                      `json:"cap_state_ttl_seconds,omitempty"`
	ConnectArray                []string                 `json:"connect_array,omitempty"`
	Spread                      string                   `json:"spread,omitempty"`
	IsLocal                     bool                     `json:"is_local,omitempty"`
	LocalCacheMaxAgeSeconds     int                      `json:"local_cache_max_age_seconds,omitempty"`
	DeliveryCacheMaxAgeSeconds  int                      `json:"delivery_cache_max_age_seconds,omitempty"`
	DeliveryReservationSeconds  int                      `json:"delivery_reservation_ttl_seconds,omitempty"`
	DeliveryStateTTLSeconds     int                      `json:"delivery_state_ttl_seconds,omitempty"`
	MiddlemanEnabled            bool                     `json:"middleman_enabled,omitempty"`
	MiddlemanAlwaysEnabled      bool                     `json:"middleman_always_enabled,omitempty"`
	MiddlemanTimeoutMS          int                      `json:"middleman_timeout_ms,omitempty"`
	MiddlemanMaxBiddersPerImp   int                      `json:"middleman_max_bidders_per_imp,omitempty"`
	MiddlemanRouteCacheTTLMS    int                      `json:"middleman_route_cache_ttl_ms,omitempty"`
	MiddlemanExchangeDomain     string                   `json:"middleman_exchange_domain,omitempty"`
	MiddlemanCallbackTTLSeconds int                      `json:"middleman_callback_ttl_seconds,omitempty"`
	MiddlemanCallbackTimeoutMS  int                      `json:"middleman_callback_timeout_ms,omitempty"`
	MiddlemanCallbackBaseURL    string                   `json:"middleman_callback_base_url,omitempty"`
	OpenRTBDebugEnabled         bool                     `json:"openrtb_debug_enabled,omitempty"`
	OpenRTBDebugSampleRate      float64                  `json:"openrtb_debug_sample_rate,omitempty"`
	ActionTokenTTLSeconds       int                      `json:"action_token_ttl_seconds,omitempty"`
	ActionClickWindowHours      int                      `json:"action_click_window_hours,omitempty"`
	ActionViewWindowHours       int                      `json:"action_view_window_hours,omitempty"`
	ActionMaxAgeHours           int                      `json:"action_max_age_hours,omitempty"`
	ActionRequestSkewSeconds    int                      `json:"action_request_skew_seconds,omitempty"`
	ActionRetentionHours        int                      `json:"action_retention_hours,omitempty"`
	PrivacyTCFVendorID          int                      `json:"privacy_tcf_vendor_id,omitempty"`
	PrivacyTCFMinPolicyVersion  int                      `json:"privacy_tcf_min_policy_version,omitempty"`
	PrivacyTCFPurposeIDs        []int                    `json:"privacy_tcf_purpose_ids,omitempty"`
	PrivacyContextualMiddleman  bool                     `json:"privacy_contextual_middleman_enabled,omitempty"`
	PrivacyBrowserIDTTLSeconds  int                      `json:"privacy_browser_id_ttl_seconds,omitempty"`
	PrivacyLogRetentionHours    int                      `json:"privacy_log_retention_hours,omitempty"`
	PrivacyAudienceTTLSeconds   int                      `json:"privacy_audience_ttl_seconds,omitempty"`
	TrustedProxyCIDRs           []string                 `json:"trusted_proxy_cidrs,omitempty"`
	MetricsAllowedCIDRs         []string                 `json:"metrics_allowed_cidrs,omitempty"`
	TrafficDefault              TrafficPolicy            `json:"traffic_default,omitempty"`
	TrafficPartners             map[string]TrafficPolicy `json:"traffic_partners,omitempty"`
	ManagementAPI               managementapi.Config     `json:"management_api,omitempty"`
	TrafficQuality              trafficquality.Config    `json:"traffic_quality,omitempty"`
	HostedPayments              hostedpayment.Config     `json:"hosted_payments,omitempty"`
	LogRequest                  string                   `json:"log_request,omitempty"`
	LogResponse                 string                   `json:"log_response,omitempty"`
	LogAttribute                string                   `json:"log_attribute,omitempty"`
	LogWinLoss                  string                   `json:"log_winloss,omitempty"`
	DBMaxOpenConns              int                      `json:"db_max_open_conns,omitempty"`
	DBMaxIdleConns              int                      `json:"db_max_idle_conns,omitempty"`
	DBConnMaxLifetimeSeconds    int                      `json:"db_conn_max_lifetime_seconds,omitempty"`
	DBConnMaxIdleTimeSeconds    int                      `json:"db_conn_max_idle_time_seconds,omitempty"`
}

type ConfigMode string

const (
	ConfigModeBid      ConfigMode = "bid"
	ConfigModeCache    ConfigMode = "cache"
	ConfigModeLedger   ConfigMode = "ledger"
	ConfigModeRetry    ConfigMode = "retry"
	ConfigModeSpread   ConfigMode = "spread"
	ConfigModeMaxMind  ConfigMode = "maxmind"
	ConfigModeNATS     ConfigMode = "nats"
	ConfigModeDatabase ConfigMode = "database"
	ConfigModeRedis    ConfigMode = "redis"
)

const minimumProductionTrackingSecretBytes = 32

var publicExampleTrackingSecrets = map[string]struct{}{
	"local-dev-tracking-secret":   {},
	"o02-disposable-drill-secret": {},
}

func NewConfig(filename string) (*Config, error) {
	parsed := new(Config)
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, parsed)
	if err != nil {
		return nil, err
	}

	if parsed.Ips == "" {
		parsed.Ips = "qq-pz.dat"
	}
	if parsed.ConnectArray == nil {
		if os.Getenv("DBUSER") != "" && os.Getenv("DBPASS") != "" && os.Getenv("DBNAME") != "" {
			host := "localhost:3306"
			if x := os.Getenv("DBHOST"); x != "" {
				host = x
				if !strings.Contains(host, ":") {
					host += ":3306"
				}
			}
			parsed.ConnectArray = []string{"mysql", os.Getenv("DBUSER") + ":" + os.Getenv("DBPASS") + "@tcp(" + host + ")/" + os.Getenv("DBNAME")}
		} else {
			return nil, fmt.Errorf("ConnectArray is not set")
		}

	}

	if parsed.Redis == nil {
		parsed.Redis = &Red{}
	}
	if parsed.Redis.Network == "" {
		parsed.Redis.Network = "tcp"
	}
	if parsed.Redis.User == "" && os.Getenv("REDISUSER") != "" {
		parsed.Redis.User = os.Getenv("REDISUSER")
	}
	if parsed.Redis.Pass == "" && os.Getenv("REDISPASS") != "" {
		parsed.Redis.Pass = os.Getenv("REDISPASS")
	}
	if parsed.Redis.Addr == "" && os.Getenv("REDISHOST") != "" {
		parsed.Redis.Addr = os.Getenv("REDISHOST")
	}
	if parsed.Redis.Addr == "" {
		parsed.Redis.Addr = "localhost"
	}
	if !strings.Contains(parsed.Redis.Addr, ":") {
		parsed.Redis.Addr += ":6379"
	}

	if parsed.ServerPort == "" {
		parsed.ServerPort = "80"
	}

	if parsed.NatsURL == "" {
		parsed.NatsURL = "nats://localhost:4222"
	}
	if parsed.TrackingSecret == "" {
		parsed.TrackingSecret = os.Getenv("TRACKING_SECRET")
	}
	if parsed.TrackingSignatureTTLSeconds <= 0 {
		parsed.TrackingSignatureTTLSeconds = int(defaultTrackingSignatureTTL.Seconds())
	}
	if parsed.CapStateTTLSeconds <= 0 {
		parsed.CapStateTTLSeconds = int(defaultCapStateTTL.Seconds())
	}
	if parsed.DeliveryCacheMaxAgeSeconds <= 0 {
		parsed.DeliveryCacheMaxAgeSeconds = 15 * 60
	}
	if parsed.DeliveryReservationSeconds <= 0 {
		parsed.DeliveryReservationSeconds = parsed.TrackingSignatureTTLSeconds + int(maxTrackingSignatureFutureSkew/time.Second)
	}
	if parsed.DeliveryStateTTLSeconds <= 0 {
		parsed.DeliveryStateTTLSeconds = 2 * 24 * 60 * 60
		minimumStateTTL := parsed.DeliveryReservationSeconds + parsed.DeliveryCacheMaxAgeSeconds
		if parsed.DeliveryStateTTLSeconds < minimumStateTTL {
			parsed.DeliveryStateTTLSeconds = minimumStateTTL
		}
	}
	if parsed.MiddlemanTimeoutMS <= 0 {
		parsed.MiddlemanTimeoutMS = 100
	}
	if parsed.MiddlemanMaxBiddersPerImp <= 0 {
		parsed.MiddlemanMaxBiddersPerImp = 5
	}
	if parsed.MiddlemanRouteCacheTTLMS <= 0 {
		parsed.MiddlemanRouteCacheTTLMS = 5000
	}
	if parsed.MiddlemanExchangeDomain == "" {
		parsed.MiddlemanExchangeDomain = serverURLHost(parsed.ServerURL)
	}
	if parsed.MiddlemanCallbackTTLSeconds <= 0 {
		parsed.MiddlemanCallbackTTLSeconds = parsed.TrackingSignatureTTLSeconds + int(maxTrackingSignatureFutureSkew/time.Second)
	}
	if parsed.MiddlemanCallbackTimeoutMS <= 0 {
		parsed.MiddlemanCallbackTimeoutMS = 1000
	}
	if parsed.MiddlemanCallbackBaseURL == "" {
		parsed.MiddlemanCallbackBaseURL = parsed.ServerURL
	}
	if parsed.OpenRTBDebugEnabled && parsed.OpenRTBDebugSampleRate == 0 {
		parsed.OpenRTBDebugSampleRate = 0.01
	}
	if parsed.ActionClickWindowHours == 0 {
		parsed.ActionClickWindowHours = 30 * 24
	}
	if parsed.ActionViewWindowHours == 0 {
		parsed.ActionViewWindowHours = 7 * 24
	}
	if parsed.ActionMaxAgeHours == 0 {
		parsed.ActionMaxAgeHours = 90 * 24
	}
	if parsed.ActionRequestSkewSeconds == 0 {
		parsed.ActionRequestSkewSeconds = 5 * 60
	}
	if parsed.ActionTokenTTLSeconds == 0 {
		parsed.ActionTokenTTLSeconds = parsed.ActionClickWindowHours * 60 * 60
	}
	if parsed.ActionRetentionHours == 0 {
		parsed.ActionRetentionHours = parsed.ActionMaxAgeHours
	}
	if parsed.PrivacyTCFMinPolicyVersion == 0 {
		parsed.PrivacyTCFMinPolicyVersion = defaultPrivacyTCFMinPolicyVersion
	}
	if len(parsed.PrivacyTCFPurposeIDs) == 0 {
		parsed.PrivacyTCFPurposeIDs = append([]int(nil), defaultPrivacyTCFPurposeIDs...)
	}
	if parsed.PrivacyBrowserIDTTLSeconds == 0 {
		parsed.PrivacyBrowserIDTTLSeconds = int(defaultPrivacyBrowserIDTTL.Seconds())
	}
	if parsed.PrivacyLogRetentionHours == 0 {
		parsed.PrivacyLogRetentionHours = int(defaultPrivacyLogRetention / time.Hour)
	}
	if parsed.PrivacyAudienceTTLSeconds == 0 {
		parsed.PrivacyAudienceTTLSeconds = int(defaultPrivacyAudienceTTL.Seconds())
	}
	if len(parsed.MetricsAllowedCIDRs) == 0 {
		parsed.MetricsAllowedCIDRs = append([]string(nil), defaultMetricsAllowedCIDRs...)
	}
	parsed.TrafficDefault = parsed.TrafficDefault.withDefaults(defaultTrafficPolicy())
	for partner, policy := range parsed.TrafficPartners {
		parsed.TrafficPartners[partner] = policy.withDefaults(parsed.TrafficDefault)
	}
	parsed.ManagementAPI = parsed.ManagementAPI.WithDefaults(parsed.DeliveryCacheMaxAgeSeconds)
	parsed.TrafficQuality = parsed.TrafficQuality.WithDefaults()
	parsed.HostedPayments = parsed.HostedPayments.WithDefaults()
	parsed.DirectSSPTokens = parsed.DirectSSPTokens.withDefaults()
	parsed.DirectSSPAuth = parsed.DirectSSPAuth.WithDefaults()

	if err := parsed.Validate(); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (self *Config) Validate(modes ...ConfigMode) error {
	if self == nil {
		return fmt.Errorf("config is nil")
	}
	if len(self.ConnectArray) < 2 {
		return fmt.Errorf("connect_array must contain driver and DSN")
	}
	if strings.TrimSpace(self.ConnectArray[0]) == "" {
		return fmt.Errorf("connect_array driver is empty")
	}
	if strings.TrimSpace(self.ConnectArray[1]) == "" {
		return fmt.Errorf("connect_array DSN is empty")
	}
	if self.Redis == nil {
		return fmt.Errorf("redis config is missing")
	}
	if self.Redis.Network == "" {
		return fmt.Errorf("redis network is empty")
	}
	if self.Redis.Network != "tcp" && self.Redis.Network != "unix" {
		return fmt.Errorf("redis network %q is invalid", self.Redis.Network)
	}
	if self.Redis.Addr == "" {
		return fmt.Errorf("redis addr is empty")
	}
	if self.Redis.Network == "tcp" {
		if _, _, err := net.SplitHostPort(self.Redis.Addr); err != nil {
			return fmt.Errorf("redis addr %q must be host:port: %w", self.Redis.Addr, err)
		}
	}
	if self.TrackingSignatureTTLSeconds <= 0 {
		return fmt.Errorf("tracking_signature_ttl_seconds must be positive")
	}
	if self.CapStateTTLSeconds <= 0 {
		return fmt.Errorf("cap_state_ttl_seconds must be positive")
	}
	if self.MiddlemanCallbackTTLSeconds <= 0 {
		return fmt.Errorf("middleman_callback_ttl_seconds must be positive")
	}
	if self.MiddlemanCallbackTimeoutMS <= 0 || self.MiddlemanCallbackTimeoutMS > 60_000 {
		return fmt.Errorf("middleman_callback_timeout_ms must be between 1 and 60000")
	}
	minimumMiddlemanCallbackTTL := self.TrackingSignatureTTLSeconds + int(maxTrackingSignatureFutureSkew/time.Second)
	processingTTLSeconds := (self.MiddlemanCallbackTimeoutMS+999)/1000 + 5
	if processingTTLSeconds < int(defaultTrackingProcessingTTL/time.Second) {
		processingTTLSeconds = int(defaultTrackingProcessingTTL / time.Second)
	}
	if processingTTLSeconds > minimumMiddlemanCallbackTTL {
		minimumMiddlemanCallbackTTL = processingTTLSeconds
	}
	if self.MiddlemanCallbackTTLSeconds < minimumMiddlemanCallbackTTL {
		return fmt.Errorf("middleman_callback_ttl_seconds must cover the accepted tracking signature lifetime and callback processing lease (%d seconds)", minimumMiddlemanCallbackTTL)
	}
	if self.MiddlemanRouteCacheTTLMS <= 0 {
		return fmt.Errorf("middleman_route_cache_ttl_ms must be positive")
	}
	if math.IsNaN(self.OpenRTBDebugSampleRate) || math.IsInf(self.OpenRTBDebugSampleRate, 0) || self.OpenRTBDebugSampleRate < 0 || self.OpenRTBDebugSampleRate > 1 {
		return fmt.Errorf("openrtb_debug_sample_rate must be between 0 and 1")
	}
	if self.OpenRTBDebugEnabled && self.OpenRTBDebugSampleRate == 0 {
		return fmt.Errorf("openrtb_debug_sample_rate must be positive when OpenRTB debug diagnostics are enabled")
	}
	clickWindow, viewWindow, maxAge, tokenTTL, requestSkew := self.actionPolicyValues()
	if clickWindow <= 0 || clickWindow > 90*24 {
		return fmt.Errorf("action_click_window_hours must be between 1 and 2160")
	}
	if viewWindow <= 0 || viewWindow > clickWindow {
		return fmt.Errorf("action_view_window_hours must be positive and no greater than action_click_window_hours")
	}
	if maxAge < clickWindow || maxAge > 365*24 {
		return fmt.Errorf("action_max_age_hours must cover action_click_window_hours and be at most 8760")
	}
	if tokenTTL < clickWindow*60*60 || tokenTTL > 365*24*60*60 {
		return fmt.Errorf("action_token_ttl_seconds must cover action_click_window_hours and be at most 31536000")
	}
	if requestSkew <= 0 || requestSkew > 60*60 {
		return fmt.Errorf("action_request_skew_seconds must be between 1 and 3600")
	}
	retentionHours := self.ActionRetentionHours
	if retentionHours == 0 {
		retentionHours = maxAge
	}
	if retentionHours < maxAge || retentionHours > 365*24 {
		return fmt.Errorf("action_retention_hours must cover action_max_age_hours and be at most 8760")
	}
	if self.PrivacyTCFVendorID < 0 || self.PrivacyTCFVendorID > 65535 {
		return fmt.Errorf("privacy_tcf_vendor_id must be between 0 and 65535")
	}
	if self.PrivacyTCFMinPolicyVersion < 0 || self.PrivacyTCFMinPolicyVersion > 63 {
		return fmt.Errorf("privacy_tcf_min_policy_version must be between 0 and 63")
	}
	seenPrivacyPurposes := make(map[int]struct{}, len(self.PrivacyTCFPurposeIDs))
	for _, purposeID := range self.PrivacyTCFPurposeIDs {
		if purposeID < 1 || purposeID > 24 {
			return fmt.Errorf("privacy_tcf_purpose_ids entry %d must be between 1 and 24", purposeID)
		}
		if _, exists := seenPrivacyPurposes[purposeID]; exists {
			return fmt.Errorf("privacy_tcf_purpose_ids contains duplicate %d", purposeID)
		}
		seenPrivacyPurposes[purposeID] = struct{}{}
	}
	if self.PrivacyTCFVendorID != 0 && len(self.PrivacyTCFPurposeIDs) == 0 {
		return fmt.Errorf("privacy_tcf_purpose_ids is required when privacy_tcf_vendor_id is configured")
	}
	if self.PrivacyBrowserIDTTLSeconds < 0 || self.PrivacyBrowserIDTTLSeconds > 365*24*60*60 {
		return fmt.Errorf("privacy_browser_id_ttl_seconds must be between 0 and 31536000")
	}
	if self.PrivacyLogRetentionHours < 0 || self.PrivacyLogRetentionHours > 365*24 {
		return fmt.Errorf("privacy_log_retention_hours must be between 0 and 8760")
	}
	if self.PrivacyAudienceTTLSeconds < 0 || self.PrivacyAudienceTTLSeconds > 365*24*60*60 {
		return fmt.Errorf("privacy_audience_ttl_seconds must be between 0 and 31536000")
	}
	if self.LocalCacheMaxAgeSeconds < 0 {
		return fmt.Errorf("local_cache_max_age_seconds must be non-negative")
	}
	if self.DeliveryCacheMaxAgeSeconds <= 0 {
		return fmt.Errorf("delivery_cache_max_age_seconds must be positive")
	}
	if self.DeliveryReservationSeconds <= 0 {
		return fmt.Errorf("delivery_reservation_ttl_seconds must be positive")
	}
	minimumReservationTTL := self.TrackingSignatureTTLSeconds + int(maxTrackingSignatureFutureSkew/time.Second)
	if self.DeliveryReservationSeconds < minimumReservationTTL {
		return fmt.Errorf("delivery_reservation_ttl_seconds must cover tracking_signature_ttl_seconds plus accepted future clock skew (%d seconds)", minimumReservationTTL)
	}
	if self.DeliveryStateTTLSeconds < self.DeliveryReservationSeconds+self.DeliveryCacheMaxAgeSeconds {
		return fmt.Errorf("delivery_state_ttl_seconds must be at least delivery_reservation_ttl_seconds plus delivery_cache_max_age_seconds")
	}
	if _, err := parseTrustedProxyCIDRs(self.TrustedProxyCIDRs); err != nil {
		return err
	}
	if _, err := parseMetricsAllowedCIDRs(self.MetricsAllowedCIDRs); err != nil {
		return err
	}
	if err := self.validateTrafficPolicies(); err != nil {
		return err
	}
	if err := self.ManagementAPI.Validate(); err != nil {
		return err
	}
	if err := self.TrafficQuality.Validate(); err != nil {
		return err
	}
	if err := self.HostedPayments.Validate(); err != nil {
		return err
	}
	if err := self.DirectSSPTokens.withDefaults().validate(); err != nil {
		return err
	}
	if err := self.DirectSSPAuth.WithDefaults().Validate(); err != nil {
		return err
	}

	needNATS := false
	needTracking := false
	needSpread := false
	for _, mode := range modes {
		switch mode {
		case ConfigModeBid:
			needNATS = true
			needTracking = true
		case ConfigModeSpread, ConfigModeNATS:
			needNATS = true
		case ConfigModeCache, ConfigModeLedger, ConfigModeRetry, ConfigModeMaxMind, ConfigModeDatabase, ConfigModeRedis:
		default:
			return fmt.Errorf("unknown config validation mode %q", mode)
		}
		if mode == ConfigModeSpread {
			needSpread = true
		}
	}
	if needTracking && strings.TrimSpace(self.TrackingSecret) == "" {
		return fmt.Errorf("tracking_secret is required")
	}
	if needNATS {
		u, err := url.Parse(self.NatsURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("nats_url is invalid")
		}
	}
	if needSpread && strings.TrimSpace(self.Spread) == "" {
		return fmt.Errorf("spread path is required")
	}
	if self.MiddlemanEnabled {
		base := self.MiddlemanCallbackBaseURL
		if base == "" {
			base = self.ServerURL
		}
		u, err := url.Parse(base)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("middleman_callback_base_url is invalid")
		}
		if strings.TrimSpace(self.TrackingSecret) == "" {
			return fmt.Errorf("tracking_secret is required when middleman is enabled")
		}
	}
	return nil
}

// ValidateProduction applies ordinary mode validation plus the deployment
// preflight required before public bid serving. It performs no network or
// dependency access and never includes secret material in an error.
func (self *Config) ValidateProduction(modes ...ConfigMode) error {
	if err := self.Validate(modes...); err != nil {
		return err
	}
	requiresTracking := self.MiddlemanEnabled
	for _, mode := range modes {
		if mode == ConfigModeBid {
			requiresTracking = true
		}
	}
	if !requiresTracking {
		return nil
	}
	return validateProductionTrackingSecret(self.TrackingSecret)
}

func validateProductionTrackingSecret(secret string) error {
	if secret == "" || secret != strings.TrimSpace(secret) {
		return fmt.Errorf("production tracking_secret must be a non-empty value without surrounding whitespace")
	}
	if _, public := publicExampleTrackingSecrets[secret]; public {
		return fmt.Errorf("tracking_secret is a checked-in public example and cannot be used in production")
	}
	if len(secret) < minimumProductionTrackingSecretBytes {
		return fmt.Errorf("production tracking_secret must contain at least %d bytes", minimumProductionTrackingSecretBytes)
	}
	return nil
}

func (self *Config) actionPolicyValues() (clickWindowHours, viewWindowHours, maxAgeHours, tokenTTLSeconds, requestSkewSeconds int) {
	clickWindowHours, viewWindowHours, maxAgeHours = 30*24, 7*24, 90*24
	requestSkewSeconds = 5 * 60
	if self != nil {
		if self.ActionClickWindowHours != 0 {
			clickWindowHours = self.ActionClickWindowHours
		}
		if self.ActionViewWindowHours != 0 {
			viewWindowHours = self.ActionViewWindowHours
		}
		if self.ActionMaxAgeHours != 0 {
			maxAgeHours = self.ActionMaxAgeHours
		}
		if self.ActionRequestSkewSeconds != 0 {
			requestSkewSeconds = self.ActionRequestSkewSeconds
		}
		if self.ActionTokenTTLSeconds != 0 {
			tokenTTLSeconds = self.ActionTokenTTLSeconds
		}
	}
	if tokenTTLSeconds == 0 {
		tokenTTLSeconds = clickWindowHours * 60 * 60
	}
	return
}

func serverURLHost(raw string) string {
	if raw == "" {
		return ""
	}
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") {
		parts := strings.SplitN(raw, "://", 2)
		raw = parts[1]
	}
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.IndexByte(raw, ':'); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

// PrivacyLogRetention returns the configured lifetime of privacy-scrubbed
// request, response, attribute, and measurement log files.
func (self *Config) PrivacyLogRetention() time.Duration {
	if self != nil && self.PrivacyLogRetentionHours > 0 {
		return time.Duration(self.PrivacyLogRetentionHours) * time.Hour
	}
	return defaultPrivacyLogRetention
}

// GetRedisDB returns the Redis conn and database handler
func (self *Config) GetRedisDB(ctx context.Context, modes ...ConfigMode) (radix.Client, *sql.DB, error) {
	if err := self.Validate(ConfigModeDatabase, ConfigModeRedis); err != nil {
		return nil, nil, err
	}
	db, err := self.OpenDB(ctx, modes...)
	if err != nil {
		return nil, nil, err
	}
	red := self.Redis
	cfg := radix.PoolConfig{
		Dialer: radix.Dialer{
			AuthUser: red.User,
			AuthPass: red.Pass,
		},
	}
	if red.Size != 0 {
		cfg.Size = red.Size
	}
	redis, err := cfg.New(ctx, red.Network, red.Addr)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return redis, db, nil
}

func (self *Config) OpenDB(ctx context.Context, modes ...ConfigMode) (*sql.DB, error) {
	if err := self.Validate(ConfigModeDatabase); err != nil {
		return nil, err
	}
	db, err := sql.Open(self.ConnectArray[0], self.ConnectArray[1])
	if err != nil {
		return nil, err
	}
	self.ApplyDBPool(db, modes...)
	if ctx != nil {
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func (self *Config) ApplyDBPool(db *sql.DB, modes ...ConfigMode) {
	if self == nil || db == nil {
		return
	}
	defaults := dbPoolDefaults(modes...)
	maxOpen := choosePositive(self.DBMaxOpenConns, defaults.maxOpen)
	maxIdle := choosePositive(self.DBMaxIdleConns, defaults.maxIdle)
	if maxIdle > maxOpen && maxOpen > 0 {
		maxIdle = maxOpen
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(choosePositive(self.DBConnMaxLifetimeSeconds, defaults.maxLifetimeSeconds)) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(choosePositive(self.DBConnMaxIdleTimeSeconds, defaults.maxIdleTimeSeconds)) * time.Second)
}

type dbPoolProfile struct {
	maxOpen            int
	maxIdle            int
	maxLifetimeSeconds int
	maxIdleTimeSeconds int
}

func dbPoolDefaults(modes ...ConfigMode) dbPoolProfile {
	for _, mode := range modes {
		switch mode {
		case ConfigModeLedger, ConfigModeRetry, ConfigModeCache, ConfigModeMaxMind:
			return dbPoolProfile{maxOpen: 4, maxIdle: 2, maxLifetimeSeconds: 600, maxIdleTimeSeconds: 120}
		}
	}
	return dbPoolProfile{maxOpen: 32, maxIdle: 8, maxLifetimeSeconds: 1800, maxIdleTimeSeconds: 300}
}

func choosePositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

type Stamp struct {
	CurrentMinute int64
	LastMinute    int64
	CurrentTimely time.Time
	LastTimely    time.Time
	Interval      int
	CurrentDay    string
	LastDay       string
}

// NewStamp creates a new Stamp with the given interval in minutes.
func NewStamp(interval int, stamp ...int) *Stamp {
	when := time.Now()
	current := when.Unix() / int64(interval*60)
	lastMinute := current - 1
	if len(stamp) > 0 {
		lastMinute = int64(stamp[0])
	}
	currentTimely := time.Unix(current*int64(interval*60), 0)
	lastTimely := time.Unix(lastMinute*int64(interval*60), 0)
	return &Stamp{current, lastMinute, currentTimely, lastTimely, interval, when.Format("2006-01-02"), when.AddDate(0, 0, -1).Format("2006-01-02")}
}

// NewLogfileName creates a new file name with the given logname and timestamp.
func (self *Config) NewLogfileName(name string, stamp *Stamp, uptonow ...bool) string {
	if len(uptonow) > 0 && uptonow[0] {
		switch name {
		case SUBJECTWinLoss:
			return fmt.Sprintf(self.LogWinLoss+"/winloss.%d", stamp.CurrentMinute)
		case SUBJECTAttribute:
			return fmt.Sprintf(self.LogAttribute+"/attribute.%d", stamp.CurrentMinute)
		case SUBJECTRequest:
			return fmt.Sprintf(self.LogRequest+"/request.%d", stamp.CurrentMinute)
		case SUBJECTResponse:
			return fmt.Sprintf(self.LogResponse+"/response.%d", stamp.CurrentMinute)
		}
	} else {
		switch name {
		case SUBJECTWinLoss:
			return fmt.Sprintf(self.LogWinLoss+"/winloss.%d", stamp.LastMinute)
		case SUBJECTAttribute:
			return fmt.Sprintf(self.LogAttribute+"/attribute.%d", stamp.LastMinute)
		case SUBJECTRequest:
			return fmt.Sprintf(self.LogRequest+"/request.%d", stamp.LastMinute)
		case SUBJECTResponse:
			return fmt.Sprintf(self.LogResponse+"/response.%d", stamp.LastMinute)
		}
	}
	return ""
}

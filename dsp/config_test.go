package dsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestConfig(t *testing.T) {
	t.Setenv("TRACKING_SECRET", "env-secret")
	configFile := filepath.Join(t.TempDir(), "sample.json")
	content := []byte(`{
		"document_root": "/tmp/www",
		"server_url": "http://localhost",
		"connect_array": ["taosSql", "root:taosdata@/tcp(127.0.0.1:0)/holiday?parseTime=false"]
	}`)
	if err := os.WriteFile(configFile, content, 0600); err != nil {
		t.Fatal(err)
	}

	c, err := NewConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if c.DocumentRoot != "/tmp/www" ||
		c.ServerURL != "http://localhost" ||
		c.Ips != "qq-pz.dat" ||
		c.ConnectArray[0] != "taosSql" ||
		c.ConnectArray[1] != "root:taosdata@/tcp(127.0.0.1:0)/holiday?parseTime=false" {
		t.Errorf("%s", c.DocumentRoot)
		t.Errorf("%s", c.ServerURL)
		t.Errorf("%s", c.Ips)
		t.Errorf("%s", c.Ips)
		t.Errorf("%v", c.ConnectArray)
	}
	if c.Redis == nil {
		t.Fatal("Redis config should be defaulted when omitted")
	}
	if c.Redis.Network != "tcp" {
		t.Errorf("Redis.Network = %q, want tcp", c.Redis.Network)
	}
	if c.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want localhost:6379", c.Redis.Addr)
	}
	if c.TrackingSecret != "env-secret" {
		t.Errorf("TrackingSecret = %q, want env fallback", c.TrackingSecret)
	}
	if c.CapStateTTLSeconds != 90*24*60*60 {
		t.Errorf("CapStateTTLSeconds = %d, want 90 days", c.CapStateTTLSeconds)
	}
	if c.DeliveryCacheMaxAgeSeconds != 900 || c.DeliveryReservationSeconds != 86700 || c.DeliveryStateTTLSeconds != 172800 {
		t.Errorf("delivery defaults = %d/%d/%d, want 900/86700/172800", c.DeliveryCacheMaxAgeSeconds, c.DeliveryReservationSeconds, c.DeliveryStateTTLSeconds)
	}
	if c.MiddlemanEnabled {
		t.Errorf("MiddlemanEnabled = true, want default false")
	}
	if c.MiddlemanTimeoutMS != 100 {
		t.Errorf("MiddlemanTimeoutMS = %d, want 100", c.MiddlemanTimeoutMS)
	}
	if c.MiddlemanMaxBiddersPerImp != 5 {
		t.Errorf("MiddlemanMaxBiddersPerImp = %d, want 5", c.MiddlemanMaxBiddersPerImp)
	}
	if c.MiddlemanRouteCacheTTLMS != 5000 {
		t.Errorf("MiddlemanRouteCacheTTLMS = %d, want 5000", c.MiddlemanRouteCacheTTLMS)
	}
	if c.MiddlemanExchangeDomain != "localhost" {
		t.Errorf("MiddlemanExchangeDomain = %q, want localhost", c.MiddlemanExchangeDomain)
	}
	if c.MiddlemanCallbackTTLSeconds != 86400 {
		t.Errorf("MiddlemanCallbackTTLSeconds = %d, want 86400", c.MiddlemanCallbackTTLSeconds)
	}
	if c.MiddlemanCallbackTimeoutMS != 1000 {
		t.Errorf("MiddlemanCallbackTimeoutMS = %d, want 1000", c.MiddlemanCallbackTimeoutMS)
	}
	if c.MiddlemanCallbackBaseURL != "http://localhost" {
		t.Errorf("MiddlemanCallbackBaseURL = %q, want server_url default", c.MiddlemanCallbackBaseURL)
	}
	if c.PrivacyTCFVendorID != 0 || c.PrivacyTCFMinPolicyVersion != 5 {
		t.Errorf("privacy TCF defaults = vendor %d / policy %d, want 0 / 5", c.PrivacyTCFVendorID, c.PrivacyTCFMinPolicyVersion)
	}
	if got := fmt.Sprint(c.PrivacyTCFPurposeIDs); got != "[1 3 4]" {
		t.Errorf("privacy purpose defaults = %s, want [1 3 4]", got)
	}
	if c.PrivacyContextualMiddleman {
		t.Error("PrivacyContextualMiddleman = true, want explicit opt-in")
	}
	if c.PrivacyBrowserIDTTLSeconds != 30*24*60*60 || c.PrivacyLogRetentionHours != 168 || c.PrivacyAudienceTTLSeconds != 30*24*60*60 {
		t.Errorf("privacy retention defaults = %d/%d/%d", c.PrivacyBrowserIDTTLSeconds, c.PrivacyLogRetentionHours, c.PrivacyAudienceTTLSeconds)
	}
	if len(c.TrustedProxyCIDRs) != 0 {
		t.Errorf("TrustedProxyCIDRs = %#v, want default empty", c.TrustedProxyCIDRs)
	}
	if got := fmt.Sprint(c.MetricsAllowedCIDRs); got != "[127.0.0.1/32 ::1/128]" {
		t.Errorf("MetricsAllowedCIDRs = %s, want loopback defaults", got)
	}
	if c.TrafficDefault != defaultTrafficPolicy() {
		t.Errorf("TrafficDefault = %#v, want %#v", c.TrafficDefault, defaultTrafficPolicy())
	}
	if c.OpenRTBDebugEnabled || c.OpenRTBDebugSampleRate != 0 {
		t.Errorf("OpenRTB debug defaults = %t/%f, want disabled/zero", c.OpenRTBDebugEnabled, c.OpenRTBDebugSampleRate)
	}
	if c.ActionTokenTTLSeconds != 30*24*60*60 || c.ActionClickWindowHours != 30*24 || c.ActionViewWindowHours != 7*24 || c.ActionMaxAgeHours != 90*24 || c.ActionRequestSkewSeconds != 300 || c.ActionRetentionHours != 90*24 {
		t.Errorf("action defaults = token:%d click:%d view:%d max:%d skew:%d retention:%d", c.ActionTokenTTLSeconds, c.ActionClickWindowHours, c.ActionViewWindowHours, c.ActionMaxAgeHours, c.ActionRequestSkewSeconds, c.ActionRetentionHours)
	}
}

func TestActionConfigRequiresRetentionForFullLifecycle(t *testing.T) {
	c := &Config{
		ConnectArray: []string{"mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"}, Redis: &Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		TrackingSignatureTTLSeconds: 86400, CapStateTTLSeconds: 7776000, DeliveryCacheMaxAgeSeconds: 900, DeliveryReservationSeconds: 86700, DeliveryStateTTLSeconds: 172800,
		MiddlemanCallbackTTLSeconds: 86400, MiddlemanCallbackTimeoutMS: 1000, MiddlemanRouteCacheTTLMS: 5000,
		ActionClickWindowHours: 720, ActionViewWindowHours: 168, ActionMaxAgeHours: 2160, ActionTokenTTLSeconds: 2592000, ActionRequestSkewSeconds: 300, ActionRetentionHours: 2160,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid action policy: %v", err)
	}
	c.ActionRetentionHours = 168
	if err := c.Validate(); err == nil {
		t.Fatal("action retention shorter than max action age was accepted")
	}
}

func TestOpenRTBDebugConfigDefaultsAndBoundsSampling(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "openrtb-debug.json")
	content := []byte(`{
		"server_url": "http://localhost",
		"connect_array": ["mysql", "user:pass@tcp(localhost:3306)/aofei"],
		"openrtb_debug_enabled": true
	}`)
	if err := os.WriteFile(configFile, content, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !c.OpenRTBDebugEnabled || c.OpenRTBDebugSampleRate != 0.01 {
		t.Fatalf("debug config = %t/%f, want enabled/0.01", c.OpenRTBDebugEnabled, c.OpenRTBDebugSampleRate)
	}
	c.OpenRTBDebugSampleRate = 1.01
	if err := c.Validate(); err == nil {
		t.Fatal("out-of-range debug sample rate was accepted")
	}
	c.OpenRTBDebugSampleRate = 0
	if err := c.Validate(); err == nil {
		t.Fatal("enabled debug diagnostics with zero sampling was accepted")
	}
}

func TestTrafficPolicyValidationAndInheritance(t *testing.T) {
	c := &Config{
		TrafficDefault: TrafficPolicy{QPS: 100, Burst: 20, MaxConcurrency: 8, TimeoutMS: 250, MaxBodyBytes: 4096},
		TrafficPartners: map[string]TrafficPolicy{
			"adx:exchange.example": {QPS: 5, Burst: 2},
			"ssp":                  {MaxConcurrency: 4},
		},
	}
	if err := c.validateTrafficPolicies(); err != nil {
		t.Fatal(err)
	}
	gate := NewTrafficGate(c)
	if got := gate.partners["adx:exchange.example"].policy; got.QPS != 5 || got.Burst != 2 || got.MaxConcurrency != 8 || got.TimeoutMS != 250 || got.MaxBodyBytes != 4096 {
		t.Fatalf("inherited partner policy = %#v", got)
	}

	for name, mutate := range map[string]func(*Config){
		"unbounded qps": func(c *Config) { c.TrafficDefault.QPS = -1 },
		"invalid key":   func(c *Config) { c.TrafficPartners = map[string]TrafficPolicy{"partner@example": {}} },
		"oversize body": func(c *Config) { c.TrafficDefault.MaxBodyBytes = maxBidRequestBodyBytes + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := *c
			bad.TrafficPartners = map[string]TrafficPolicy{"ssp": {}}
			mutate(&bad)
			if err := bad.validateTrafficPolicies(); err == nil {
				t.Fatal("expected traffic policy validation error")
			}
		})
	}
}

func TestMetricsAllowedCIDRsRejectInvalidOrEmptyValues(t *testing.T) {
	if _, err := parseMetricsAllowedCIDRs([]string{"not-a-cidr"}); err == nil {
		t.Fatal("expected invalid metrics CIDR to fail")
	}
	if _, err := parseMetricsAllowedCIDRs([]string{""}); err == nil {
		t.Fatal("expected empty metrics CIDR list to fail")
	}
}

func TestApplyDBPoolDefaultsAndOverrides(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	c := &Config{}
	c.ApplyDBPool(db, ConfigModeRetry)
	stats := db.Stats()
	if stats.MaxOpenConnections != 4 {
		t.Fatalf("retry max open = %d, want 4", stats.MaxOpenConnections)
	}

	c.DBMaxOpenConns = 12
	c.DBMaxIdleConns = 6
	c.ApplyDBPool(db)
	stats = db.Stats()
	if stats.MaxOpenConnections != 12 {
		t.Fatalf("override max open = %d, want 12", stats.MaxOpenConnections)
	}
}

func TestGetRedisDBClosesDBWhenRedisPoolCreationFails(t *testing.T) {
	dsn := "get_redis_db_close_on_redis_error"
	db, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectPing()
	mock.ExpectClose()

	c := &Config{
		ConnectArray:                []string{"sqlmock", dsn},
		Redis:                       &Red{Network: "unix", Addr: filepath.Join(t.TempDir(), "missing.sock")},
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		DeliveryCacheMaxAgeSeconds:  900,
		DeliveryReservationSeconds:  86700,
		DeliveryStateTTLSeconds:     172800,
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
		MiddlemanCallbackBaseURL:    "http://localhost",
	}
	redis, openedDB, err := c.GetRedisDB(context.Background())
	if err == nil {
		if redis != nil {
			_ = redis.Close()
		}
		if openedDB != nil {
			_ = openedDB.Close()
		}
		t.Fatal("expected redis pool creation to fail")
	}
	if redis != nil || openedDB != nil {
		t.Fatalf("redis/db = %#v/%#v, want nil/nil on redis failure", redis, openedDB)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	mock.ExpectClose()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigRejectsTruncatedConnectArray(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "bad.json")
	content := []byte(`{
		"server_url": "http://localhost",
		"connect_array": ["mysql"]
	}`)
	if err := os.WriteFile(configFile, content, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewConfig(configFile); err == nil {
		t.Fatal("expected truncated connect_array to fail")
	}
}

func TestConfigValidateBidRequiresTrackingSecret(t *testing.T) {
	t.Setenv("TRACKING_SECRET", "")
	configFile := filepath.Join(t.TempDir(), "sample.json")
	content := []byte(`{
		"server_url": "http://localhost",
		"connect_array": ["mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"]
	}`)
	if err := os.WriteFile(configFile, content, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfig(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(ConfigModeBid); err == nil {
		t.Fatal("expected bid validation to require tracking_secret")
	}
}

func TestConfigRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	c := &Config{
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		DeliveryCacheMaxAgeSeconds:  900,
		DeliveryReservationSeconds:  86700,
		DeliveryStateTTLSeconds:     172800,
		ConnectArray:                []string{"mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"},
		Redis:                       &Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
		TrustedProxyCIDRs:           []string{"not-a-cidr"},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected invalid trusted_proxy_cidrs entry to fail")
	}
}

func TestConfigValidateModes(t *testing.T) {
	c := &Config{
		ServerURL:                   "http://localhost",
		TrackingSecret:              "secret",
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		DeliveryCacheMaxAgeSeconds:  900,
		DeliveryReservationSeconds:  86700,
		DeliveryStateTTLSeconds:     172800,
		ConnectArray:                []string{"mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"},
		Redis:                       &Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		NatsURL:                     "nats://127.0.0.1:4222",
		Spread:                      "/tmp/spread",
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
		MiddlemanCallbackBaseURL:    "http://localhost",
	}
	for _, mode := range []ConfigMode{
		ConfigModeBid,
		ConfigModeCache,
		ConfigModeLedger,
		ConfigModeRetry,
		ConfigModeSpread,
		ConfigModeMaxMind,
		ConfigModeNATS,
	} {
		if err := c.Validate(mode); err != nil {
			t.Fatalf("Validate(%s) = %v", mode, err)
		}
	}

	badNATS := *c
	badNATS.NatsURL = "://bad"
	if err := badNATS.Validate(ConfigModeNATS); err == nil {
		t.Fatal("expected invalid nats_url to fail NATS mode")
	}

	noSpread := *c
	noSpread.Spread = ""
	if err := noSpread.Validate(ConfigModeSpread); err == nil {
		t.Fatal("expected empty spread path to fail spread mode")
	}

	badLocalCacheAge := *c
	badLocalCacheAge.LocalCacheMaxAgeSeconds = -1
	if err := badLocalCacheAge.Validate(); err == nil {
		t.Fatal("expected negative local_cache_max_age_seconds to fail")
	}

	shortReservation := *c
	shortReservation.DeliveryReservationSeconds = shortReservation.TrackingSignatureTTLSeconds
	if err := shortReservation.Validate(); err == nil {
		t.Fatal("expected reservation TTL shorter than accepted callback lifetime to fail")
	}
}

func TestConfigRejectsInvalidPrivacyPolicy(t *testing.T) {
	base := Config{
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		DeliveryCacheMaxAgeSeconds:  900,
		DeliveryReservationSeconds:  86700,
		DeliveryStateTTLSeconds:     172800,
		ConnectArray:                []string{"mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"},
		Redis:                       &Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
	}
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "vendor too large", mutate: func(c *Config) { c.PrivacyTCFVendorID = 65536 }},
		{name: "policy too large", mutate: func(c *Config) { c.PrivacyTCFMinPolicyVersion = 64 }},
		{name: "purpose out of range", mutate: func(c *Config) { c.PrivacyTCFPurposeIDs = []int{0} }},
		{name: "duplicate purpose", mutate: func(c *Config) { c.PrivacyTCFPurposeIDs = []int{1, 1} }},
		{name: "configured vendor without purpose", mutate: func(c *Config) { c.PrivacyTCFVendorID = 7 }},
		{name: "negative browser ttl", mutate: func(c *Config) { c.PrivacyBrowserIDTTLSeconds = -1 }},
		{name: "excess log retention", mutate: func(c *Config) { c.PrivacyLogRetentionHours = 365*24 + 1 }},
		{name: "excess audience ttl", mutate: func(c *Config) { c.PrivacyAudienceTTLSeconds = 365*24*60*60 + 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := base
			tt.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("expected invalid privacy configuration to fail")
			}
		})
	}
}

func TestConfigRejectsInvalidEnabledTrafficQuality(t *testing.T) {
	config := Config{
		TrackingSignatureTTLSeconds: 86400,
		CapStateTTLSeconds:          90 * 24 * 60 * 60,
		DeliveryCacheMaxAgeSeconds:  900,
		DeliveryReservationSeconds:  86700,
		DeliveryStateTTLSeconds:     172800,
		ConnectArray:                []string{"mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"},
		Redis:                       &Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
		MiddlemanRouteCacheTTLMS:    5000,
	}
	config.TrafficQuality.Enabled = true
	config.TrafficQuality.DigestKeyEnv = "bad-name"
	config.TrafficQuality.EnforcementRefreshSeconds = 30
	config.TrafficQuality.EnforcementMaxAgeSeconds = 120
	if err := config.Validate(); err == nil {
		t.Fatal("expected invalid traffic-quality digest environment name to fail")
	}
}

func TestLocalConfigSmoke(t *testing.T) {
	configPath, ok := os.LookupEnv("AOFEI")
	if !ok || configPath == "" {
		t.Skip("AOFEI is unset; run ./scripts/aofei-local.sh up")
	}
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("AOFEI config %s is missing; run ./scripts/aofei-local.sh up", configPath)
		}
		t.Fatal(err)
	}

	c, err := NewConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.ConnectArray) < 2 || c.ConnectArray[0] == "" || c.ConnectArray[1] == "" {
		t.Fatalf("ConnectArray is incomplete: %#v", c.ConnectArray)
	}
	if c.Redis == nil || c.Redis.Addr == "" {
		t.Fatalf("Redis config is incomplete: %#v", c.Redis)
	}
	if c.NatsURL == "" {
		t.Fatal("NatsURL is empty")
	}
	if c.Spread == "" {
		t.Fatal("Spread is empty")
	}
	for name, path := range map[string]string{
		"log_request":   c.LogRequest,
		"log_response":  c.LogResponse,
		"log_attribute": c.LogAttribute,
		"log_winloss":   c.LogWinLoss,
	} {
		if path == "" {
			t.Fatalf("%s is empty", name)
		}
	}
}

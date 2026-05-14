package dsp

import (
	"context"
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
	if c.MiddlemanEnabled {
		t.Errorf("MiddlemanEnabled = true, want default false")
	}
	if c.MiddlemanTimeoutMS != 100 {
		t.Errorf("MiddlemanTimeoutMS = %d, want 100", c.MiddlemanTimeoutMS)
	}
	if c.MiddlemanMaxBiddersPerImp != 5 {
		t.Errorf("MiddlemanMaxBiddersPerImp = %d, want 5", c.MiddlemanMaxBiddersPerImp)
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
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
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

func TestConfigValidateModes(t *testing.T) {
	c := &Config{
		ServerURL:                   "http://localhost",
		TrackingSecret:              "secret",
		TrackingSignatureTTLSeconds: 86400,
		ConnectArray:                []string{"mysql", "user:pass@tcp(127.0.0.1:3306)/aofei"},
		Redis:                       &Red{Network: "tcp", Addr: "127.0.0.1:6379"},
		NatsURL:                     "nats://127.0.0.1:4222",
		Spread:                      "/tmp/spread",
		MiddlemanCallbackTTLSeconds: 86400,
		MiddlemanCallbackTimeoutMS:  1000,
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

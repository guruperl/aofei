package dsp

import (
	"os"
	"path/filepath"
	"testing"
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

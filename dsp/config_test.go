package dsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
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

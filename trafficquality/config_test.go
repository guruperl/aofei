package trafficquality

import (
	"encoding/base64"
	"os"
	"testing"
)

func TestConfigDefaultsAndDigestKeyValidation(t *testing.T) {
	config := (Config{Enabled: true, DigestKeyEnv: "QUALITY_TEST_KEY"}).WithDefaults()
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	if config.EnforcementRefreshSeconds != 30 || config.EnforcementMaxAgeSeconds != 120 {
		t.Fatalf("defaults=%#v", config)
	}
	config.DigestKeyEnv = "BAD-NAME"
	if err := config.Validate(); err == nil {
		t.Fatal("invalid environment name accepted")
	}
	if _, err := decodeDigestKey("short"); err == nil {
		t.Fatal("short digest key accepted")
	}
	key := make([]byte, 32)
	if _, err := decodeDigestKey(base64.StdEncoding.EncodeToString(key)); err != nil {
		t.Fatal(err)
	}
}

func TestDisabledServiceDoesNotReadEnvironment(t *testing.T) {
	os.Unsetenv("QUALITY_TEST_MISSING")
	service, err := NewService(Config{Enabled: false, DigestKeyEnv: "QUALITY_TEST_MISSING"}, nil)
	if err != nil || service != nil {
		t.Fatalf("service=%#v err=%v", service, err)
	}
}

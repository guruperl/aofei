package trafficquality

import (
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	Enabled                   bool   `json:"enabled,omitempty"`
	DigestKeyEnv              string `json:"digest_key_env,omitempty"`
	EnforcementRefreshSeconds int    `json:"enforcement_refresh_seconds,omitempty"`
	EnforcementMaxAgeSeconds  int    `json:"enforcement_max_age_seconds,omitempty"`
}

var safeEnvironmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if !safeEnvironmentName.MatchString(c.DigestKeyEnv) {
		return fmt.Errorf("traffic_quality.digest_key_env must name an environment variable")
	}
	if c.EnforcementRefreshSeconds < 1 || c.EnforcementRefreshSeconds > 300 {
		return fmt.Errorf("traffic_quality.enforcement_refresh_seconds must be between 1 and 300")
	}
	if c.EnforcementMaxAgeSeconds < c.EnforcementRefreshSeconds || c.EnforcementMaxAgeSeconds > 3600 {
		return fmt.Errorf("traffic_quality.enforcement_max_age_seconds must cover the refresh interval and be at most 3600")
	}
	return nil
}

func (c Config) WithDefaults() Config {
	if c.EnforcementRefreshSeconds == 0 {
		c.EnforcementRefreshSeconds = 30
	}
	if c.EnforcementMaxAgeSeconds == 0 {
		c.EnforcementMaxAgeSeconds = 120
	}
	return c
}

func NewService(config Config, db *sql.DB) (*Service, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("traffic-quality database is nil")
	}
	key, err := decodeDigestKey(os.Getenv(config.DigestKeyEnv))
	if err != nil {
		return nil, fmt.Errorf("traffic-quality digest key: %w", err)
	}
	return NewServiceWithKey(db, key)
}

func decodeDigestKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("configured environment value is empty")
	}
	if key, err := base64.StdEncoding.DecodeString(raw); err == nil && len(key) >= 32 {
		return key, nil
	}
	if key, err := hex.DecodeString(raw); err == nil && len(key) >= 32 {
		return key, nil
	}
	return nil, fmt.Errorf("value must be base64 or hexadecimal and decode to at least 32 bytes")
}

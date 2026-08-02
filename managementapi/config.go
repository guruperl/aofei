// Package managementapi implements the versioned external advertiser API.
// It deliberately does not reuse Summer/Genelet routes or browser sessions.
package managementapi

import (
	"fmt"
	"regexp"
	"time"
)

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	defaultRequestTimeout = 5 * time.Second
	defaultMaxBodyBytes   = int64(256 * 1024)
	defaultCredentialRPM  = 120
	defaultAccountRPM     = 600
	defaultPageSize       = 50
	maxPageSize           = 100
)

// Config is the deployment-gated management API policy. KeyEnv names a
// deployment secret; the key itself must never appear in JSON.
type Config struct {
	Enabled                  bool   `json:"enabled,omitempty"`
	KeyEnv                   string `json:"key_env,omitempty"`
	RequestTimeoutMS         int    `json:"request_timeout_ms,omitempty"`
	MaxBodyBytes             int64  `json:"max_body_bytes,omitempty"`
	CredentialRequestsMinute int    `json:"credential_requests_per_minute,omitempty"`
	AccountRequestsMinute    int    `json:"account_requests_per_minute,omitempty"`
	CacheActivationSeconds   int    `json:"cache_activation_seconds,omitempty"`
}

// WithDefaults returns a validated-policy candidate without enabling it.
func (c Config) WithDefaults(cacheActivationSeconds int) Config {
	if c.RequestTimeoutMS == 0 {
		c.RequestTimeoutMS = int(defaultRequestTimeout / time.Millisecond)
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = defaultMaxBodyBytes
	}
	if c.CredentialRequestsMinute == 0 {
		c.CredentialRequestsMinute = defaultCredentialRPM
	}
	if c.AccountRequestsMinute == 0 {
		c.AccountRequestsMinute = defaultAccountRPM
	}
	if c.CacheActivationSeconds == 0 {
		c.CacheActivationSeconds = cacheActivationSeconds
		if c.CacheActivationSeconds == 0 {
			c.CacheActivationSeconds = 300
		}
	}
	return c
}

// Validate rejects unsafe enabled policies. A disabled block is inert.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if !environmentNamePattern.MatchString(c.KeyEnv) || len(c.KeyEnv) > 128 {
		return fmt.Errorf("management_api.key_env must be a valid environment variable name of at most 128 bytes")
	}
	if c.RequestTimeoutMS < 100 || c.RequestTimeoutMS > 30_000 {
		return fmt.Errorf("management_api.request_timeout_ms must be between 100 and 30000")
	}
	if c.MaxBodyBytes < 1024 || c.MaxBodyBytes > 1<<20 {
		return fmt.Errorf("management_api.max_body_bytes must be between 1024 and 1048576")
	}
	for _, field := range []struct {
		name  string
		value int
	}{
		{"credential_requests_per_minute", c.CredentialRequestsMinute},
		{"account_requests_per_minute", c.AccountRequestsMinute},
	} {
		if field.value < 1 || field.value > 100_000 {
			return fmt.Errorf("management_api.%s must be between 1 and 100000", field.name)
		}
	}
	if c.AccountRequestsMinute < c.CredentialRequestsMinute {
		return fmt.Errorf("management_api.account_requests_per_minute must be at least credential_requests_per_minute")
	}
	if c.CacheActivationSeconds < 30 || c.CacheActivationSeconds > 3600 {
		return fmt.Errorf("management_api.cache_activation_seconds must be between 30 and 3600")
	}
	return nil
}

func (c Config) requestTimeout() time.Duration {
	return time.Duration(c.RequestTimeoutMS) * time.Millisecond
}

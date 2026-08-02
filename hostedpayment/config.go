// Package hostedpayment implements the A02 boundary between Aofei accounting
// statements and hosted/tokenized external funding and payout providers.
package hostedpayment

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const ProviderStripe = "stripe"

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

// Config contains references and policy only. API and webhook secrets are read
// from the named environment variables when the disabled-by-default service is
// constructed; they must never be written into JSON configuration.
type Config struct {
	Enabled                  bool   `json:"enabled,omitempty"`
	Provider                 string `json:"provider,omitempty"`
	LiveMode                 bool   `json:"live_mode,omitempty"`
	APIBaseURL               string `json:"api_base_url,omitempty"`
	APIKeyEnv                string `json:"api_key_env,omitempty"`
	WebhookSecretEnv         string `json:"webhook_secret_env,omitempty"`
	WebhookPreviousSecretEnv string `json:"webhook_previous_secret_env,omitempty"`
	PublicBaseURL            string `json:"public_base_url,omitempty"`
	RequestTimeoutMS         int    `json:"request_timeout_ms,omitempty"`
	MaxBodyBytes             int64  `json:"max_body_bytes,omitempty"`
	WebhookToleranceSeconds  int    `json:"webhook_tolerance_seconds,omitempty"`
	MaxAttempts              int    `json:"max_attempts,omitempty"`
	RetryBaseMS              int    `json:"retry_base_ms,omitempty"`
	EventRetentionDays       int    `json:"event_retention_days,omitempty"`
	ReconciliationMaxAgeDays int    `json:"reconciliation_max_age_days,omitempty"`
}

func (c Config) WithDefaults() Config {
	if c.Provider == "" {
		c.Provider = ProviderStripe
	}
	if c.APIBaseURL == "" {
		c.APIBaseURL = "https://api.stripe.com"
	}
	if c.RequestTimeoutMS == 0 {
		c.RequestTimeoutMS = 5000
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = 128 << 10
	}
	if c.WebhookToleranceSeconds == 0 {
		c.WebhookToleranceSeconds = 300
	}
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 3
	}
	if c.RetryBaseMS == 0 {
		c.RetryBaseMS = 100
	}
	if c.EventRetentionDays == 0 {
		c.EventRetentionDays = 400
	}
	if c.ReconciliationMaxAgeDays == 0 {
		c.ReconciliationMaxAgeDays = 90
	}
	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Provider != ProviderStripe {
		return fmt.Errorf("hosted_payments.provider must be stripe")
	}
	if !environmentName.MatchString(c.APIKeyEnv) {
		return fmt.Errorf("hosted_payments.api_key_env must name an environment variable")
	}
	if !environmentName.MatchString(c.WebhookSecretEnv) {
		return fmt.Errorf("hosted_payments.webhook_secret_env must name an environment variable")
	}
	if c.APIKeyEnv == c.WebhookSecretEnv {
		return fmt.Errorf("hosted payment API and webhook secrets must use separate environment variables")
	}
	if c.WebhookPreviousSecretEnv != "" {
		if !environmentName.MatchString(c.WebhookPreviousSecretEnv) {
			return fmt.Errorf("hosted_payments.webhook_previous_secret_env must name an environment variable")
		}
		if c.WebhookPreviousSecretEnv == c.APIKeyEnv || c.WebhookPreviousSecretEnv == c.WebhookSecretEnv {
			return fmt.Errorf("hosted payment current, previous, and API secrets must use separate environment variables")
		}
	}
	if err := validateServiceURL("hosted_payments.api_base_url", c.APIBaseURL, true); err != nil {
		return err
	}
	apiURL, _ := url.Parse(c.APIBaseURL)
	apiHost := apiURL.Hostname()
	apiLoopback := apiHost == "localhost" || net.ParseIP(apiHost) != nil && net.ParseIP(apiHost).IsLoopback()
	if c.APIBaseURL != "https://api.stripe.com" && !apiLoopback {
		return fmt.Errorf("hosted_payments.api_base_url must be https://api.stripe.com outside loopback fixtures")
	}
	if c.LiveMode && c.APIBaseURL != "https://api.stripe.com" {
		return fmt.Errorf("hosted_payments.api_base_url must be https://api.stripe.com in live mode")
	}
	if err := validateServiceURL("hosted_payments.public_base_url", c.PublicBaseURL, false); err != nil {
		return err
	}
	if c.RequestTimeoutMS < 500 || c.RequestTimeoutMS > 30_000 {
		return fmt.Errorf("hosted_payments.request_timeout_ms must be between 500 and 30000")
	}
	if c.MaxBodyBytes < 1024 || c.MaxBodyBytes > 1<<20 {
		return fmt.Errorf("hosted_payments.max_body_bytes must be between 1024 and 1048576")
	}
	if c.WebhookToleranceSeconds < 60 || c.WebhookToleranceSeconds > 900 {
		return fmt.Errorf("hosted_payments.webhook_tolerance_seconds must be between 60 and 900")
	}
	if c.MaxAttempts < 1 || c.MaxAttempts > 5 {
		return fmt.Errorf("hosted_payments.max_attempts must be between 1 and 5")
	}
	if c.RetryBaseMS < 25 || c.RetryBaseMS > 1000 {
		return fmt.Errorf("hosted_payments.retry_base_ms must be between 25 and 1000")
	}
	if c.EventRetentionDays < 30 || c.EventRetentionDays > 2555 {
		return fmt.Errorf("hosted_payments.event_retention_days must be between 30 and 2555")
	}
	if c.ReconciliationMaxAgeDays < 1 || c.ReconciliationMaxAgeDays > c.EventRetentionDays {
		return fmt.Errorf("hosted_payments.reconciliation_max_age_days must be positive and no greater than event_retention_days")
	}
	return nil
}

func validateServiceURL(name, raw string, allowPath bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s must be an absolute URL without credentials, query, or fragment", name)
	}
	if !allowPath && u.Path != "" && u.Path != "/" {
		return fmt.Errorf("%s must not contain a path", name)
	}
	if u.Scheme == "https" {
		return nil
	}
	host := u.Hostname()
	if u.Scheme == "http" && (host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()) {
		return nil
	}
	return fmt.Errorf("%s must use HTTPS (HTTP is allowed only on loopback)", name)
}

func (c Config) requestTimeout() time.Duration {
	return time.Duration(c.RequestTimeoutMS) * time.Millisecond
}

func (c Config) retryBase() time.Duration {
	return time.Duration(c.RetryBaseMS) * time.Millisecond
}

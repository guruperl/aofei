// Package publisherauth implements publisher/App-scoped request credentials
// for the direct SSP SDK/server path.
package publisherauth

import "fmt"

const (
	defaultRequestSkewSeconds       = 300
	defaultRefreshSeconds           = 30
	defaultMaxAgeSeconds            = 120
	defaultRotationMaxOverlapSecond = 24 * 60 * 60
)

// Config controls the default-off SDK/server request-authentication boundary.
type Config struct {
	Enabled                   bool `json:"enabled,omitempty"`
	RequestSkewSeconds        int  `json:"request_skew_seconds,omitempty"`
	CredentialRefreshSeconds  int  `json:"credential_refresh_seconds,omitempty"`
	CredentialMaxAgeSeconds   int  `json:"credential_max_age_seconds,omitempty"`
	RotationMaxOverlapSeconds int  `json:"rotation_max_overlap_seconds,omitempty"`
}

// WithDefaults supplies the bounded P03 runtime defaults.
func (c Config) WithDefaults() Config {
	if c.RequestSkewSeconds == 0 {
		c.RequestSkewSeconds = defaultRequestSkewSeconds
	}
	if c.CredentialRefreshSeconds == 0 {
		c.CredentialRefreshSeconds = defaultRefreshSeconds
	}
	if c.CredentialMaxAgeSeconds == 0 {
		c.CredentialMaxAgeSeconds = defaultMaxAgeSeconds
	}
	if c.RotationMaxOverlapSeconds == 0 {
		c.RotationMaxOverlapSeconds = defaultRotationMaxOverlapSecond
	}
	return c
}

// Validate rejects configurations that cannot provide bounded freshness,
// replay, or revocation propagation.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.RequestSkewSeconds < 1 || c.RequestSkewSeconds > 900 {
		return fmt.Errorf("direct_ssp_auth.request_skew_seconds must be between 1 and 900")
	}
	if c.CredentialRefreshSeconds < 1 || c.CredentialRefreshSeconds > 300 {
		return fmt.Errorf("direct_ssp_auth.credential_refresh_seconds must be between 1 and 300")
	}
	if c.CredentialMaxAgeSeconds < c.CredentialRefreshSeconds || c.CredentialMaxAgeSeconds > 3600 {
		return fmt.Errorf("direct_ssp_auth.credential_max_age_seconds must cover the refresh interval and be at most 3600")
	}
	if c.RotationMaxOverlapSeconds < 1 || c.RotationMaxOverlapSeconds > 7*24*60*60 {
		return fmt.Errorf("direct_ssp_auth.rotation_max_overlap_seconds must be between 1 and 604800")
	}
	return nil
}

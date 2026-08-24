package dsp

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/guruperl/aofei/acl"
)

const (
	directSSPLegacyAllow = "allow"
	directSSPLegacyDeny  = "deny"
)

var (
	directSSPTokenEnvironmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	directSSPTokenKeyIDPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)
)

// DirectSSPTokenKeyConfig names one deployment-owned v2 signing epoch. KeyEnv
// is a secret reference; the key value never belongs in JSON.
type DirectSSPTokenKeyConfig struct {
	KeyID  string `json:"key_id,omitempty"`
	Epoch  uint32 `json:"epoch,omitempty"`
	KeyEnv string `json:"key_env,omitempty"`
}

// DirectSSPTokenConfig controls the bounded current/previous v2 verifier and
// the explicit historical-token compatibility gate.
type DirectSSPTokenConfig struct {
	Enabled        bool                     `json:"enabled,omitempty"`
	LegacyReadMode string                   `json:"legacy_read_mode,omitempty"`
	Current        DirectSSPTokenKeyConfig  `json:"current,omitempty"`
	Previous       *DirectSSPTokenKeyConfig `json:"previous,omitempty"`
}

func (c DirectSSPTokenConfig) withDefaults() DirectSSPTokenConfig {
	if c.LegacyReadMode == "" {
		c.LegacyReadMode = directSSPLegacyAllow
	}
	return c
}

func (c DirectSSPTokenConfig) validate() error {
	if c.LegacyReadMode != directSSPLegacyAllow && c.LegacyReadMode != directSSPLegacyDeny {
		return fmt.Errorf("direct_ssp_tokens.legacy_read_mode must be allow or deny")
	}
	currentConfigured := !directSSPTokenKeyConfigEmpty(c.Current)
	if !c.Enabled {
		if c.LegacyReadMode == directSSPLegacyDeny {
			return fmt.Errorf("direct_ssp_tokens cannot deny legacy reads while v2 is disabled")
		}
		if currentConfigured {
			if err := validateDirectSSPTokenKeyConfig("current", c.Current); err != nil {
				return err
			}
		}
		if c.Previous != nil {
			return fmt.Errorf("direct_ssp_tokens.previous requires v2 to be enabled")
		}
		return nil
	}
	if !currentConfigured {
		return fmt.Errorf("direct_ssp_tokens.current is required when v2 is enabled")
	}
	if err := validateDirectSSPTokenKeyConfig("current", c.Current); err != nil {
		return err
	}
	if c.Previous == nil {
		return nil
	}
	if err := validateDirectSSPTokenKeyConfig("previous", *c.Previous); err != nil {
		return err
	}
	if c.Current.KeyID == c.Previous.KeyID && c.Current.Epoch == c.Previous.Epoch {
		return fmt.Errorf("direct_ssp_tokens current and previous selectors must differ")
	}
	if c.Current.KeyEnv == c.Previous.KeyEnv {
		return fmt.Errorf("direct_ssp_tokens current and previous key_env must differ")
	}
	return nil
}

func directSSPTokenKeyConfigEmpty(key DirectSSPTokenKeyConfig) bool {
	return key.KeyID == "" && key.Epoch == 0 && key.KeyEnv == ""
}

func validateDirectSSPTokenKeyConfig(name string, key DirectSSPTokenKeyConfig) error {
	if !directSSPTokenKeyIDPattern.MatchString(key.KeyID) {
		return fmt.Errorf("direct_ssp_tokens.%s.key_id must be 1-32 URL-safe characters", name)
	}
	if key.Epoch == 0 {
		return fmt.Errorf("direct_ssp_tokens.%s.epoch must be positive", name)
	}
	if !directSSPTokenEnvironmentName.MatchString(key.KeyEnv) || len(key.KeyEnv) > 128 {
		return fmt.Errorf("direct_ssp_tokens.%s.key_env must be a valid environment variable name of at most 128 bytes", name)
	}
	return nil
}

func newDirectSSPTokenCodec(config DirectSSPTokenConfig) (*acl.DirectTokenCodec, error) {
	config = config.withDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}
	allowLegacy := config.LegacyReadMode == directSSPLegacyAllow
	if !config.Enabled {
		return acl.NewDirectTokenCodec(acl.DirectTokenKey{}, nil, allowLegacy)
	}
	current, err := loadDirectSSPTokenKey(config.Current)
	if err != nil {
		return nil, fmt.Errorf("load direct SSP current token key: %w", err)
	}
	var previous *acl.DirectTokenKey
	if config.Previous != nil {
		loaded, err := loadDirectSSPTokenKey(*config.Previous)
		if err != nil {
			return nil, fmt.Errorf("load direct SSP previous token key: %w", err)
		}
		previous = &loaded
	}
	return acl.NewDirectTokenCodec(current, previous, allowLegacy)
}

func loadDirectSSPTokenKey(config DirectSSPTokenKeyConfig) (acl.DirectTokenKey, error) {
	secret, err := decodeDirectSSPTokenKey(os.Getenv(config.KeyEnv))
	if err != nil {
		return acl.DirectTokenKey{}, fmt.Errorf("%s: %w", config.KeyEnv, err)
	}
	return acl.DirectTokenKey{ID: config.KeyID, Epoch: config.Epoch, Secret: secret}, nil
}

func decodeDirectSSPTokenKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("environment variable is empty")
	}
	for _, decode := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		hex.DecodeString,
	} {
		if key, err := decode(raw); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	return nil, fmt.Errorf("must be a base64 or hexadecimal 32-byte key")
}

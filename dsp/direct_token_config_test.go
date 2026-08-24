package dsp

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/guruperl/aofei/acl"
)

func TestDirectSSPTokenConfigDefaultsToLegacyOnly(t *testing.T) {
	config := (DirectSSPTokenConfig{}).withDefaults()
	if config.Enabled || config.LegacyReadMode != directSSPLegacyAllow {
		t.Fatalf("defaults = enabled:%t legacy:%q", config.Enabled, config.LegacyReadMode)
	}
	codec, err := newDirectSSPTokenCodec(config)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := acl.PackDirectToken(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, version, err := codec.UnpackSite(legacy); err != nil || version != acl.DirectTokenLegacy {
		t.Fatalf("default legacy read = version %s error %v", version, err)
	}
	if _, err := codec.PackSite(42, 9); err == nil {
		t.Fatal("disabled v2 config emitted a token")
	}
}

func TestDirectSSPTokenConfigLoadsBoundedKeyRing(t *testing.T) {
	currentSecret := []byte(strings.Repeat("c", 32))
	previousSecret := []byte(strings.Repeat("p", 32))
	t.Setenv("DIRECT_SSP_CURRENT_TEST_KEY", base64.StdEncoding.EncodeToString(currentSecret))
	t.Setenv("DIRECT_SSP_PREVIOUS_TEST_KEY", base64.StdEncoding.EncodeToString(previousSecret))
	config := DirectSSPTokenConfig{
		Enabled:        true,
		LegacyReadMode: directSSPLegacyDeny,
		Current: DirectSSPTokenKeyConfig{
			KeyID: "inventory", Epoch: 12, KeyEnv: "DIRECT_SSP_CURRENT_TEST_KEY",
		},
		Previous: &DirectSSPTokenKeyConfig{
			KeyID: "inventory", Epoch: 11, KeyEnv: "DIRECT_SSP_PREVIOUS_TEST_KEY",
		},
	}
	codec, err := newDirectSSPTokenCodec(config)
	if err != nil {
		t.Fatal(err)
	}
	site, err := codec.PackSite(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(site, ".inventory.12.") {
		t.Fatalf("current selector missing from %q", site)
	}
	legacy, err := acl.PackDirectToken(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := codec.UnpackSite(legacy); !errors.Is(err, acl.ErrLegacyDirectTokenDisabled) {
		t.Fatalf("legacy deny error = %v", err)
	}

	previousCodec, err := acl.NewDirectTokenCodec(acl.DirectTokenKey{ID: "inventory", Epoch: 11, Secret: previousSecret}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	previousSite, err := previousCodec.PackSite(42, 9)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := codec.UnpackSite(previousSite); err != nil {
		t.Fatalf("previous epoch during overlap: %v", err)
	}
}

func TestDirectSSPTokenConfigRejectsUnsafePolicies(t *testing.T) {
	validCurrent := DirectSSPTokenKeyConfig{KeyID: "inventory", Epoch: 1, KeyEnv: "DIRECT_SSP_TOKEN_KEY"}
	tests := []struct {
		name   string
		config DirectSSPTokenConfig
	}{
		{name: "unknown legacy mode", config: DirectSSPTokenConfig{LegacyReadMode: "sometimes"}},
		{name: "deny without v2", config: DirectSSPTokenConfig{LegacyReadMode: directSSPLegacyDeny}},
		{name: "enabled without current", config: DirectSSPTokenConfig{Enabled: true, LegacyReadMode: directSSPLegacyAllow}},
		{name: "invalid current key id", config: DirectSSPTokenConfig{Enabled: true, LegacyReadMode: directSSPLegacyAllow, Current: DirectSSPTokenKeyConfig{KeyID: "bad.id", Epoch: 1, KeyEnv: "KEY"}}},
		{name: "zero epoch", config: DirectSSPTokenConfig{Enabled: true, LegacyReadMode: directSSPLegacyAllow, Current: DirectSSPTokenKeyConfig{KeyID: "primary", KeyEnv: "KEY"}}},
		{name: "invalid environment", config: DirectSSPTokenConfig{Enabled: true, LegacyReadMode: directSSPLegacyAllow, Current: DirectSSPTokenKeyConfig{KeyID: "primary", Epoch: 1, KeyEnv: "bad-name"}}},
		{name: "previous while disabled", config: DirectSSPTokenConfig{LegacyReadMode: directSSPLegacyAllow, Current: validCurrent, Previous: &DirectSSPTokenKeyConfig{KeyID: "inventory", Epoch: 2, KeyEnv: "OTHER"}}},
		{name: "duplicate selector", config: DirectSSPTokenConfig{Enabled: true, LegacyReadMode: directSSPLegacyAllow, Current: validCurrent, Previous: &DirectSSPTokenKeyConfig{KeyID: "inventory", Epoch: 1, KeyEnv: "OTHER"}}},
		{name: "reused environment", config: DirectSSPTokenConfig{Enabled: true, LegacyReadMode: directSSPLegacyAllow, Current: validCurrent, Previous: &DirectSSPTokenKeyConfig{KeyID: "inventory", Epoch: 2, KeyEnv: "DIRECT_SSP_TOKEN_KEY"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.withDefaults().validate(); err == nil {
				t.Fatal("unsafe direct SSP token config was accepted")
			}
		})
	}
}

func TestDirectSSPTokenConfigRejectsMissingOrMalformedSecret(t *testing.T) {
	config := DirectSSPTokenConfig{
		Enabled: true, LegacyReadMode: directSSPLegacyAllow,
		Current: DirectSSPTokenKeyConfig{KeyID: "inventory", Epoch: 1, KeyEnv: "DIRECT_SSP_BAD_TEST_KEY"},
	}
	for _, raw := range []string{"", "short", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 31)))} {
		t.Setenv("DIRECT_SSP_BAD_TEST_KEY", raw)
		if _, err := newDirectSSPTokenCodec(config); err == nil {
			t.Fatalf("secret %q was accepted", raw)
		}
	}
}

package dsp

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/guruperl/aofei/acl"
)

func TestDirectSSPTokenIssuerDefaultsToLegacyWithoutSecrets(t *testing.T) {
	issuer, err := NewDirectSSPTokenIssuer(&Config{})
	if err != nil {
		t.Fatal(err)
	}
	metadata := issuer.Metadata()
	if metadata.TokenVersion != "v1" || metadata.LegacyReadMode != directSSPLegacyAllow || metadata.RequestAuthentication != "compatibility" {
		t.Fatalf("metadata = %#v", metadata)
	}
	site, err := issuer.PackSite(7, 11)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := acl.PackDirectToken(7, 11)
	if site != want {
		t.Fatalf("site token = %q, want %q", site, want)
	}
}

func TestDirectSSPTokenIssuerEmitsV2AndSafeMetadata(t *testing.T) {
	t.Setenv("P03_ISSUER_TEST_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	config := &Config{
		DirectSSPTokens: DirectSSPTokenConfig{
			Enabled: true, LegacyReadMode: directSSPLegacyAllow,
			Current: DirectSSPTokenKeyConfig{KeyID: "primary", Epoch: 3, KeyEnv: "P03_ISSUER_TEST_KEY"},
		},
	}
	config.DirectSSPAuth.Enabled = true
	issuer, err := NewDirectSSPTokenIssuer(config)
	if err != nil {
		t.Fatal(err)
	}
	metadata := issuer.Metadata()
	if metadata.TokenVersion != "v2" || metadata.TokenKeyID != "primary" || metadata.TokenEpoch != 3 || metadata.RequestAuthentication != "required" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if strings.Contains(strings.ToLower(metadata.TokenKeyID), "env") {
		t.Fatalf("metadata exposed environment reference: %#v", metadata)
	}
	site, err := issuer.PackSite(7, 11)
	if err != nil {
		t.Fatal(err)
	}
	slot, err := issuer.PackSlot(7, 11, 13, 19661050)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(site, "pz2.site.primary.3.") || !strings.HasPrefix(slot, "pz2.slot.primary.3.") {
		t.Fatalf("v2 locators = site:%q slot:%q", site, slot)
	}
	if pubID, siteID, version, err := issuer.codec.UnpackSite(site); err != nil || pubID != 7 || siteID != 11 || version != acl.DirectTokenV2 {
		t.Fatalf("unpack site = %d/%d %v %v", pubID, siteID, version, err)
	}
	if slotID, sizeID, version, err := issuer.codec.UnpackSlot(slot, 7, 11); err != nil || slotID != 13 || sizeID != 19661050 || version != acl.DirectTokenV2 {
		t.Fatalf("unpack slot = %d/%d %v %v", slotID, sizeID, version, err)
	}
}

func TestDirectSSPTokenIssuerFailsClosedWithoutEnabledKey(t *testing.T) {
	config := &Config{DirectSSPTokens: DirectSSPTokenConfig{
		Enabled: true, LegacyReadMode: directSSPLegacyAllow,
		Current: DirectSSPTokenKeyConfig{KeyID: "primary", Epoch: 1, KeyEnv: "P03_MISSING_ISSUER_KEY"},
	}}
	t.Setenv("P03_MISSING_ISSUER_KEY", "")
	if _, err := NewDirectSSPTokenIssuer(config); err == nil {
		t.Fatal("NewDirectSSPTokenIssuer error = nil, want missing-key failure")
	}
}

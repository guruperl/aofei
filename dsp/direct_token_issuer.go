package dsp

import (
	"fmt"

	"github.com/guruperl/aofei/acl"
)

// DirectSSPIntegrationMetadata is the display-safe issuer and request-proof
// configuration used by readiness manifests and publisher integration pages.
// It deliberately omits signing-key environment names and values, credential
// public ids, and every private request-signing value.
type DirectSSPIntegrationMetadata struct {
	TokenVersion              string
	TokenKeyID                string
	TokenEpoch                uint32
	LegacyReadMode            string
	RequestAuthentication     string
	CredentialRefreshSeconds  int
	CredentialMaxAgeSeconds   int
	RotationMaxOverlapSeconds int
}

// DirectSSPTokenIssuer is the narrow control-plane boundary for generating
// public browser/App inventory locators with the same immutable codec used by
// the request path. Its codec and key material are not exported.
type DirectSSPTokenIssuer struct {
	codec    *acl.DirectTokenCodec
	metadata DirectSSPIntegrationMetadata
}

// NewDirectSSPTokenIssuer loads the configured issuer for read-only tooling.
// Enabled v2 configurations fail closed when their deployment key is absent or
// malformed; disabled configurations emit historical v1 locators.
func NewDirectSSPTokenIssuer(config *Config) (*DirectSSPTokenIssuer, error) {
	if config == nil {
		return nil, fmt.Errorf("direct SSP token issuer config is nil")
	}
	codec, err := newDirectSSPTokenCodec(config.DirectSSPTokens)
	if err != nil {
		return nil, err
	}
	return newDirectSSPTokenIssuer(config, codec), nil
}

func newDirectSSPTokenIssuer(config *Config, codec *acl.DirectTokenCodec) *DirectSSPTokenIssuer {
	if config == nil {
		return nil
	}
	tokens := config.DirectSSPTokens.withDefaults()
	auth := config.DirectSSPAuth.WithDefaults()
	metadata := DirectSSPIntegrationMetadata{
		TokenVersion:              "v1",
		LegacyReadMode:            tokens.LegacyReadMode,
		RequestAuthentication:     "compatibility",
		CredentialRefreshSeconds:  auth.CredentialRefreshSeconds,
		CredentialMaxAgeSeconds:   auth.CredentialMaxAgeSeconds,
		RotationMaxOverlapSeconds: auth.RotationMaxOverlapSeconds,
	}
	if tokens.Enabled {
		metadata.TokenVersion = "v2"
		metadata.TokenKeyID = tokens.Current.KeyID
		metadata.TokenEpoch = tokens.Current.Epoch
	}
	if auth.Enabled {
		metadata.RequestAuthentication = "required"
	}
	return &DirectSSPTokenIssuer{codec: codec, metadata: metadata}
}

// DirectSSPTokenIssuer returns the controller-owned public-token issuer. It is
// safe to place in Summer's in-process dependency storage; the object exposes
// no verifier or secret accessor.
func (self *Controller) DirectSSPTokenIssuer() *DirectSSPTokenIssuer {
	if self == nil {
		return nil
	}
	return newDirectSSPTokenIssuer(self.C, self.directTokens)
}

// Metadata returns a copy of display-safe integration metadata.
func (self *DirectSSPTokenIssuer) Metadata() DirectSSPIntegrationMetadata {
	if self == nil {
		return DirectSSPIntegrationMetadata{}
	}
	return self.metadata
}

// PackSite emits the configured public site locator.
func (self *DirectSSPTokenIssuer) PackSite(pubID, siteID uint32) (string, error) {
	if self == nil || self.codec == nil {
		return "", fmt.Errorf("direct SSP token issuer is unavailable")
	}
	if self.metadata.TokenVersion == "v1" {
		return acl.PackDirectToken(pubID, siteID)
	}
	return self.codec.PackSite(pubID, siteID)
}

// PackSlot emits the configured public slot locator. V2 binds the complete
// publisher/site/slot/size tuple; v1 preserves the historical slot/size shape.
func (self *DirectSSPTokenIssuer) PackSlot(pubID, siteID, slotID, sizeID uint32) (string, error) {
	if self == nil || self.codec == nil {
		return "", fmt.Errorf("direct SSP token issuer is unavailable")
	}
	if self.metadata.TokenVersion == "v1" {
		return acl.PackDirectToken(slotID, sizeID)
	}
	return self.codec.PackSlot(pubID, siteID, slotID, sizeID)
}

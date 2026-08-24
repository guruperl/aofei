package acl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const directTokenV2Prefix = "pz2"

var (
	directTokenKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)

	// ErrInvalidDirectToken identifies malformed, unverifiable, or mis-scoped
	// direct SSP inventory tokens without exposing key material.
	ErrInvalidDirectToken = errors.New("invalid direct token")
	// ErrLegacyDirectTokenDisabled identifies a historical token rejected by
	// the explicit legacy-read gate.
	ErrLegacyDirectTokenDisabled = errors.New("legacy direct token is disabled")
)

// DirectTokenVersion identifies the accepted direct SSP inventory token
// contract. It is suitable for fixed-cardinality migration metrics.
type DirectTokenVersion uint8

const (
	DirectTokenUnknown DirectTokenVersion = iota
	DirectTokenLegacy
	DirectTokenV2
)

func (v DirectTokenVersion) String() string {
	switch v {
	case DirectTokenLegacy:
		return "legacy"
	case DirectTokenV2:
		return "v2"
	default:
		return "unknown"
	}
}

// DirectTokenKey is one deployment-owned v2 signing epoch. Secret must be
// exactly 32 bytes and must come from deployment secret state, not a cache,
// database row, generated tag, or checked-in config value.
type DirectTokenKey struct {
	ID     string
	Epoch  uint32
	Secret []byte
}

type directTokenSelector struct {
	id    string
	epoch uint32
}

// DirectTokenCodec emits with one current key and verifies against a bounded
// current/previous key ring. It is immutable after construction and safe for
// concurrent request-path use.
type DirectTokenCodec struct {
	current     DirectTokenKey
	keys        map[directTokenSelector][]byte
	allowLegacy bool
}

// NewDirectTokenCodec constructs the v2 issuer/verifier. A zero current key is
// permitted only for a legacy-only compatibility codec.
func NewDirectTokenCodec(current DirectTokenKey, previous *DirectTokenKey, allowLegacy bool) (*DirectTokenCodec, error) {
	codec := &DirectTokenCodec{
		allowLegacy: allowLegacy,
		keys:        make(map[directTokenSelector][]byte, 2),
	}
	if directTokenKeyEmpty(current) {
		if previous != nil {
			return nil, fmt.Errorf("direct token previous key requires a current key")
		}
		return codec, nil
	}
	if err := validateDirectTokenKey(current); err != nil {
		return nil, fmt.Errorf("direct token current key: %w", err)
	}
	codec.current = copyDirectTokenKey(current)
	currentSelector := directTokenSelector{id: current.ID, epoch: current.Epoch}
	codec.keys[currentSelector] = append([]byte(nil), current.Secret...)

	if previous != nil {
		if err := validateDirectTokenKey(*previous); err != nil {
			return nil, fmt.Errorf("direct token previous key: %w", err)
		}
		previousSelector := directTokenSelector{id: previous.ID, epoch: previous.Epoch}
		if previousSelector == currentSelector {
			return nil, fmt.Errorf("direct token current and previous selectors must differ")
		}
		if hmac.Equal(previous.Secret, current.Secret) {
			return nil, fmt.Errorf("direct token current and previous secrets must differ")
		}
		codec.keys[previousSelector] = append([]byte(nil), previous.Secret...)
	}
	return codec, nil
}

// NewLegacyDirectTokenCodec returns the historical read-only compatibility
// policy used when the P03 v2 boundary has not been enabled.
func NewLegacyDirectTokenCodec() *DirectTokenCodec {
	codec, _ := NewDirectTokenCodec(DirectTokenKey{}, nil, true)
	return codec
}

func directTokenKeyEmpty(key DirectTokenKey) bool {
	return key.ID == "" && key.Epoch == 0 && len(key.Secret) == 0
}

func validateDirectTokenKey(key DirectTokenKey) error {
	if !directTokenKeyIDPattern.MatchString(key.ID) {
		return fmt.Errorf("key id must be 1-32 URL-safe characters")
	}
	if key.Epoch == 0 {
		return fmt.Errorf("epoch must be positive")
	}
	if len(key.Secret) != 32 {
		return fmt.Errorf("secret must be exactly 32 bytes")
	}
	return nil
}

func copyDirectTokenKey(key DirectTokenKey) DirectTokenKey {
	key.Secret = append([]byte(nil), key.Secret...)
	return key
}

// PackSite emits a v2 site locator bound to publisher and site identity.
func (c *DirectTokenCodec) PackSite(pubID, siteID uint32) (string, error) {
	if pubID == 0 || siteID == 0 {
		return "", fmt.Errorf("direct token site identity must be positive")
	}
	var payload [8]byte
	binary.BigEndian.PutUint32(payload[0:4], pubID)
	binary.BigEndian.PutUint32(payload[4:8], siteID)
	return c.packV2("site", payload[:])
}

// PackSlot emits a v2 slot locator bound to the complete publisher, site,
// slot, and size tuple. A valid slot cannot be moved under another site token.
func (c *DirectTokenCodec) PackSlot(pubID, siteID, slotID, sizeID uint32) (string, error) {
	if pubID == 0 || siteID == 0 || slotID == 0 || sizeID == 0 {
		return "", fmt.Errorf("direct token slot identity must be positive")
	}
	var payload [16]byte
	binary.BigEndian.PutUint32(payload[0:4], pubID)
	binary.BigEndian.PutUint32(payload[4:8], siteID)
	binary.BigEndian.PutUint32(payload[8:12], slotID)
	binary.BigEndian.PutUint32(payload[12:16], sizeID)
	return c.packV2("slot", payload[:])
}

func (c *DirectTokenCodec) packV2(kind string, payload []byte) (string, error) {
	if c == nil || directTokenKeyEmpty(c.current) {
		return "", fmt.Errorf("direct token v2 issuer is not configured")
	}
	unsigned := strings.Join([]string{
		directTokenV2Prefix,
		kind,
		c.current.ID,
		strconv.FormatUint(uint64(c.current.Epoch), 10),
		base64.RawURLEncoding.EncodeToString(payload),
	}, ".")
	mac := directTokenMAC(c.current.Secret, unsigned)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac), nil
}

// UnpackSite verifies and decodes either a v2 site locator or, while the
// explicit compatibility gate remains open, one historical packed token.
func (c *DirectTokenCodec) UnpackSite(text string) (pubID, siteID uint32, version DirectTokenVersion, err error) {
	version = directTokenVersion(text)
	if version == DirectTokenLegacy {
		if c == nil || !c.allowLegacy {
			return 0, 0, version, ErrLegacyDirectTokenDisabled
		}
		pubID, siteID, err = UnpackDirectToken(text)
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrInvalidDirectToken, err)
		}
		return
	}
	if version != DirectTokenV2 {
		return 0, 0, version, ErrInvalidDirectToken
	}
	payload, unpackErr := c.unpackV2(text, "site", 8)
	if unpackErr != nil {
		return 0, 0, version, unpackErr
	}
	pubID = binary.BigEndian.Uint32(payload[0:4])
	siteID = binary.BigEndian.Uint32(payload[4:8])
	if pubID == 0 || siteID == 0 {
		return 0, 0, version, ErrInvalidDirectToken
	}
	return pubID, siteID, version, nil
}

// UnpackSlot verifies and decodes a slot locator for the already decoded
// publisher/site identity. V2 locators reject cross-site mix-and-match.
func (c *DirectTokenCodec) UnpackSlot(text string, pubID, siteID uint32) (slotID, sizeID uint32, version DirectTokenVersion, err error) {
	version = directTokenVersion(text)
	if version == DirectTokenLegacy {
		if c == nil || !c.allowLegacy {
			return 0, 0, version, ErrLegacyDirectTokenDisabled
		}
		slotID, sizeID, err = UnpackDirectToken(text)
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrInvalidDirectToken, err)
		}
		return
	}
	if version != DirectTokenV2 {
		return 0, 0, version, ErrInvalidDirectToken
	}
	payload, unpackErr := c.unpackV2(text, "slot", 16)
	if unpackErr != nil {
		return 0, 0, version, unpackErr
	}
	if binary.BigEndian.Uint32(payload[0:4]) != pubID || binary.BigEndian.Uint32(payload[4:8]) != siteID {
		return 0, 0, version, fmt.Errorf("%w: slot scope does not match site", ErrInvalidDirectToken)
	}
	slotID = binary.BigEndian.Uint32(payload[8:12])
	sizeID = binary.BigEndian.Uint32(payload[12:16])
	if slotID == 0 || sizeID == 0 {
		return 0, 0, version, ErrInvalidDirectToken
	}
	return slotID, sizeID, version, nil
}

func directTokenVersion(text string) DirectTokenVersion {
	if strings.HasPrefix(text, directTokenV2Prefix+".") {
		return DirectTokenV2
	}
	// Lower-case versioned prefixes are outside the historical upper-case
	// base32 alphabet. Never route an unknown future version to the legacy
	// decoder.
	if strings.HasPrefix(text, "pz") {
		return DirectTokenUnknown
	}
	return DirectTokenLegacy
}

func (c *DirectTokenCodec) unpackV2(text, wantKind string, wantPayloadBytes int) ([]byte, error) {
	if c == nil {
		return nil, ErrInvalidDirectToken
	}
	parts := strings.Split(text, ".")
	if len(parts) != 6 || parts[0] != directTokenV2Prefix || parts[1] != wantKind || !directTokenKeyIDPattern.MatchString(parts[2]) {
		return nil, ErrInvalidDirectToken
	}
	epoch, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil || epoch == 0 || strconv.FormatUint(epoch, 10) != parts[3] {
		return nil, ErrInvalidDirectToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil || len(payload) != wantPayloadBytes || base64.RawURLEncoding.EncodeToString(payload) != parts[4] {
		return nil, ErrInvalidDirectToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[5])
	if err != nil || len(signature) != sha256.Size || base64.RawURLEncoding.EncodeToString(signature) != parts[5] {
		return nil, ErrInvalidDirectToken
	}
	key := c.keys[directTokenSelector{id: parts[2], epoch: uint32(epoch)}]
	if len(key) != 32 {
		return nil, ErrInvalidDirectToken
	}
	unsigned := strings.Join(parts[:5], ".")
	if !hmac.Equal(signature, directTokenMAC(key, unsigned)) {
		return nil, ErrInvalidDirectToken
	}
	return payload, nil
}

func directTokenMAC(key []byte, unsigned string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("aofei-direct-ssp-token-v2\x00"))
	_, _ = mac.Write([]byte(unsigned))
	return mac.Sum(nil)
}

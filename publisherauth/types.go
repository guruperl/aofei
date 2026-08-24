package publisherauth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	PermissionCredentialRead   = "publisher.credential.read"
	PermissionCredentialIssue  = "publisher.credential.issue"
	PermissionCredentialRotate = "publisher.credential.rotate"
	PermissionCredentialRevoke = "publisher.credential.revoke"

	HeaderCredential = "X-W8M-PZ-Credential"
	HeaderTimestamp  = "X-W8M-PZ-Timestamp"
	HeaderNonce      = "X-W8M-PZ-Nonce"
	HeaderSignature  = "X-W8M-PZ-Signature"
)

var (
	ErrForbidden   = errors.New("publisher credential action forbidden")
	ErrNotFound    = errors.New("publisher credential not found")
	ErrConflict    = errors.New("publisher credential state conflict")
	ErrRequired    = errors.New("publisher request proof is required")
	ErrInvalid     = errors.New("publisher request proof is invalid")
	ErrStale       = errors.New("publisher request proof is stale")
	ErrScope       = errors.New("publisher request proof scope does not match inventory")
	ErrReplay      = errors.New("publisher request proof was already used")
	ErrUnavailable = errors.New("publisher request authentication is unavailable")

	publicIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

// Actor is derived from a verified S02 session. Permissions and RecentMFA are
// copied from server-owned identity state, never request form claims.
type Actor struct {
	Role        string
	ID          uint64
	Permissions map[string]bool
	RecentMFA   bool
}

func (a Actor) Can(permission string) bool {
	return a.Permissions["*"] || a.Permissions[permission]
}

func (a Actor) validate(permission string, recentMFA bool, pubID uint64) error {
	if (a.Role != "admin" && a.Role != "pub") || a.ID == 0 || pubID == 0 || !a.Can(permission) || recentMFA && !a.RecentMFA {
		return ErrForbidden
	}
	if a.Role == "pub" && a.ID != pubID {
		return ErrForbidden
	}
	return nil
}

// Credential contains only display-safe lifecycle metadata. Private signing
// material is returned separately exactly once.
type Credential struct {
	ID           uint64     `json:"id"`
	PublisherID  uint64     `json:"publisher_id"`
	SiteID       uint64     `json:"site_id"`
	Name         string     `json:"name"`
	PublicID     string     `json:"public_id"`
	Algorithm    string     `json:"algorithm"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RotatedAt    *time.Time `json:"rotated_at,omitempty"`
	OverlapUntil *time.Time `json:"overlap_until,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	ReplacesID   uint64     `json:"replaces_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	State        string     `json:"state"`
}

// Principal is the verified runtime identity. It is distinct from a Summer
// session, advertiser API token, browser locator, or browser cookie.
type Principal struct {
	CredentialID uint64
	PublisherID  uint64
	SiteID       uint64
	PublicID     string
}

// RequestProof is a verified body/freshness proof awaiting inventory scope and
// shared replay checks. Its raw signature is deliberately not retained.
type RequestProof struct {
	principal Principal
	timestamp int64
	nonce     string
}

func (p *RequestProof) Principal() Principal {
	if p == nil {
		return Principal{}
	}
	return p.principal
}

func (p *RequestProof) AuthorizeScope(pubID, siteID uint64) error {
	if p == nil || pubID == 0 || siteID == 0 || p.principal.PublisherID != pubID || p.principal.SiteID != siteID {
		return ErrScope
	}
	return nil
}

func validateLifecycleText(name, reason string) error {
	name = strings.TrimSpace(name)
	reason = strings.TrimSpace(reason)
	if name == "" || len(name) > 128 || strings.ContainsAny(name, "\r\n\x00") {
		return fmt.Errorf("credential name must be a single line of at most 128 bytes")
	}
	if reason == "" || len(reason) > 255 || strings.ContainsAny(reason, "\r\n\x00") {
		return fmt.Errorf("a single-line reason of at most 255 bytes is required")
	}
	return nil
}

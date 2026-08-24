package publisherauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mediocregopher/radix/v4"
)

const (
	credentialAlgorithm = "Ed25519-v1"
	replayTimeout       = time.Second
)

var claimReplayScript = radix.NewEvalScript(`
local result = redis.call('SET', KEYS[1], '1', 'NX', 'EX', ARGV[1])
if result then
  return 1
end
return 0
`)

type verifierRecord struct {
	principal  Principal
	publicKey  ed25519.PublicKey
	validUntil time.Time
}

type verifierSnapshot struct {
	generatedAt time.Time
	records     map[string]verifierRecord
}

// Service owns credential lifecycle state plus an immutable request-path
// verifier snapshot. Request authentication never queries MySQL.
type Service struct {
	config   Config
	db       *sql.DB
	redis    radix.Client
	now      func() time.Time
	random   io.Reader
	snapshot atomic.Pointer[verifierSnapshot]
	snapMu   sync.Mutex
}

// NewService initializes the default-off service and requires a complete
// verifier snapshot before an enabled HTTP process can start.
func NewService(config Config, db *sql.DB, redis radix.Client) (*Service, error) {
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}
	if db == nil || redis == nil {
		return nil, fmt.Errorf("direct SSP authentication requires MySQL and Redis")
	}
	service := &Service{config: config, db: db, redis: redis, now: time.Now, random: rand.Reader}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := service.ReloadSnapshot(ctx); err != nil {
		return nil, fmt.Errorf("load direct SSP credential snapshot: %w", err)
	}
	return service, nil
}

// ReloadSnapshot atomically replaces all active verifier metadata. A failed
// refresh leaves the previous generation in place; request verification stops
// using it after CredentialMaxAgeSeconds.
func (s *Service) ReloadSnapshot(ctx context.Context) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	now := s.currentTime()
	rows, err := s.db.QueryContext(ctx, `
SELECT c.credential_id, c.pub_id, c.site_id, c.public_id, c.public_key,
       c.expires_at, c.rotated_at, c.overlap_until
FROM pub_request_credential c
INNER JOIN pub p ON p.pub_id=c.pub_id AND p.active='Yes'
INNER JOIN pub_site ps ON ps.site_id=c.site_id AND ps.pub_id=c.pub_id
  AND ps.active='Yes' AND ps.site_type='App'
  AND ps.inventory_environment='App'
  AND ps.integration_mode IN ('SDK','ServerAPI')
WHERE c.revoked_at IS NULL AND c.expires_at>?
  AND (c.rotated_at IS NULL OR c.overlap_until>?)
ORDER BY c.credential_id`, now, now)
	if err != nil {
		return err
	}
	defer rows.Close()
	records := make(map[string]verifierRecord)
	for rows.Next() {
		var principal Principal
		var publicKey []byte
		var expiresAt time.Time
		var rotatedAt, overlapUntil sql.NullTime
		if err := rows.Scan(&principal.CredentialID, &principal.PublisherID, &principal.SiteID,
			&principal.PublicID, &publicKey, &expiresAt, &rotatedAt, &overlapUntil); err != nil {
			return err
		}
		if principal.CredentialID == 0 || principal.PublisherID == 0 || principal.SiteID == 0 ||
			!publicIDPattern.MatchString(principal.PublicID) || len(publicKey) != ed25519.PublicKeySize {
			return fmt.Errorf("credential %d has invalid verifier metadata", principal.CredentialID)
		}
		validUntil := expiresAt.UTC()
		if rotatedAt.Valid {
			if !overlapUntil.Valid {
				return fmt.Errorf("credential %d has an unbounded rotation state", principal.CredentialID)
			}
			if overlapUntil.Time.UTC().Before(validUntil) {
				validUntil = overlapUntil.Time.UTC()
			}
		}
		if prior, exists := records[principal.PublicID]; exists {
			return fmt.Errorf("credentials %d and %d have duplicate public ids", prior.principal.CredentialID, principal.CredentialID)
		}
		records[principal.PublicID] = verifierRecord{
			principal: principal, publicKey: append(ed25519.PublicKey(nil), publicKey...), validUntil: validUntil,
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.installSnapshot(&verifierSnapshot{generatedAt: s.currentTime(), records: records})
	return nil
}

func (s *Service) installSnapshot(snapshot *verifierSnapshot) {
	s.snapMu.Lock()
	s.snapshot.Store(snapshot)
	s.snapMu.Unlock()
}

// SnapshotGeneratedAt reports only snapshot freshness metadata.
func (s *Service) SnapshotGeneratedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	snapshot := s.snapshot.Load()
	if snapshot == nil {
		return time.Time{}
	}
	return snapshot.generatedAt
}

// VerifyRequest checks canonical headers, body binding, freshness, credential
// activity, and the Ed25519 signature without mutating replay state.
func (s *Service) VerifyRequest(r *http.Request, body []byte) (*RequestProof, error) {
	if s == nil {
		return nil, ErrUnavailable
	}
	if r == nil {
		return nil, ErrInvalid
	}
	if r.Method != http.MethodPost || r.URL == nil || r.URL.EscapedPath() != "/pz" || r.URL.RawQuery != "" {
		return nil, ErrInvalid
	}
	credential, err := singleHeader(r.Header, HeaderCredential)
	if err != nil {
		return nil, err
	}
	timestampText, err := singleHeader(r.Header, HeaderTimestamp)
	if err != nil {
		return nil, err
	}
	nonce, err := singleHeader(r.Header, HeaderNonce)
	if err != nil {
		return nil, err
	}
	signatureText, err := singleHeader(r.Header, HeaderSignature)
	if err != nil {
		return nil, err
	}
	publicID, err := parseCredentialReference(credential)
	if err != nil {
		return nil, ErrInvalid
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil || strconv.FormatInt(timestamp, 10) != timestampText {
		return nil, ErrInvalid
	}
	if err := validateNonce(nonce); err != nil {
		return nil, ErrInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(signatureText)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != signatureText {
		return nil, ErrInvalid
	}
	now := s.currentTime()
	skewSeconds := int64(s.config.RequestSkewSeconds)
	if timestamp < now.Unix()-skewSeconds || timestamp > now.Unix()+skewSeconds {
		return nil, ErrStale
	}
	snapshot := s.snapshot.Load()
	maxAge := time.Duration(s.config.CredentialMaxAgeSeconds) * time.Second
	refresh := time.Duration(s.config.CredentialRefreshSeconds) * time.Second
	if snapshot == nil || snapshot.generatedAt.IsZero() || snapshot.generatedAt.After(now.Add(refresh)) || now.Sub(snapshot.generatedAt) > maxAge {
		return nil, ErrUnavailable
	}
	record, ok := snapshot.records[publicID]
	if !ok || !now.Before(record.validUntil) {
		return nil, ErrInvalid
	}
	if !ed25519.Verify(record.publicKey, canonicalRequest(credential, timestampText, nonce, body), signature) {
		return nil, ErrInvalid
	}
	return &RequestProof{principal: record.principal, timestamp: timestamp, nonce: nonce}, nil
}

// ClaimReplay atomically consumes the verified nonce in shared Redis for the
// remainder of its accepted timestamp window.
func (s *Service) ClaimReplay(ctx context.Context, proof *RequestProof) error {
	if s == nil || s.redis == nil || proof == nil || proof.principal.CredentialID == 0 {
		return ErrUnavailable
	}
	now := s.currentTime()
	validUntil := time.Unix(proof.timestamp, 0).UTC().Add(time.Duration(s.config.RequestSkewSeconds) * time.Second)
	remaining := validUntil.Sub(now)
	if remaining <= 0 {
		return ErrStale
	}
	ttlSeconds := int64((remaining + time.Second - 1) / time.Second)
	digest := sha256.Sum256([]byte("w8m-pz-replay-v1\x00" + proof.principal.PublicID + "\x00" + proof.nonce))
	key := "pz:auth:replay:" + base64.RawURLEncoding.EncodeToString(digest[:])
	claimCtx, cancel := context.WithTimeout(ctx, replayTimeout)
	defer cancel()
	var claimed int
	if err := s.redis.Do(claimCtx, claimReplayScript.Cmd(&claimed, []string{key}, strconv.FormatInt(ttlSeconds, 10))); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if claimed != 1 {
		return ErrReplay
	}
	return nil
}

func singleHeader(headers http.Header, name string) (string, error) {
	values := headers.Values(name)
	if len(values) == 0 {
		return "", ErrRequired
	}
	if len(values) != 1 || values[0] == "" || values[0] != strings.TrimSpace(values[0]) || strings.Contains(values[0], ",") {
		return "", ErrInvalid
	}
	return values[0], nil
}

func (s *Service) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

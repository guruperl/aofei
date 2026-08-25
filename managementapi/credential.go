package managementapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mediocregopher/radix/v4"
)

var publicIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type Service struct {
	config Config
	db     *sql.DB
	redis  radix.Client
	key    []byte
	now    func() time.Time
	random io.Reader
}

func NewService(config Config, db *sql.DB, redis radix.Client) (*Service, error) {
	config = config.WithDefaults(config.CacheActivationSeconds)
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}
	if db == nil || redis == nil {
		return nil, fmt.Errorf("management API requires MySQL and Redis")
	}
	key, err := decodeKey(os.Getenv(config.KeyEnv))
	if err != nil {
		return nil, fmt.Errorf("management API key %s: %w", config.KeyEnv, err)
	}
	return &Service{config: config, db: db, redis: redis, key: key, now: time.Now, random: rand.Reader}, nil
}

func (s *Service) Handler() http.Handler {
	if s == nil {
		return nil
	}
	return newHandler(s)
}

func decodeKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("environment variable is empty")
	}
	for _, decode := range []func(string) ([]byte, error){base64.StdEncoding.DecodeString, hex.DecodeString} {
		if key, err := decode(raw); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	return nil, fmt.Errorf("must be a base64 or hexadecimal 32-byte key")
}

func (s *Service) digest(domain, raw string) []byte {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(domain))
	mac.Write([]byte{0})
	mac.Write([]byte(raw))
	return mac.Sum(nil)
}

func (s *Service) Authenticate(ctx context.Context, bearer string) (Principal, error) {
	parts := strings.SplitN(bearer, "_", 4)
	if len(parts) != 4 || parts[0] != "w8m" || parts[1] != "v1" || !publicIDPattern.MatchString(parts[2]) || len(parts[3]) < 40 || len(parts[3]) > 64 {
		return Principal{}, ErrUnauthorized
	}
	var principal Principal
	var digest []byte
	var scopes string
	var revoked sql.NullTime
	err := s.db.QueryRowContext(ctx, `
SELECT credential_id, adv_id, credential_name, token_digest, permissions, expires_at, revoked_at
FROM api_credential WHERE public_id=?`, parts[2]).Scan(
		&principal.CredentialID, &principal.AdvertiserID, &principal.Name,
		&digest, &scopes, &principal.ExpiresAt, &revoked)
	if err == sql.ErrNoRows {
		return Principal{}, ErrUnauthorized
	}
	if err != nil {
		return Principal{}, err
	}
	principal.ExpiresAt = principal.ExpiresAt.UTC()
	want := s.digest("management-api-token-v1", bearer)
	if len(digest) != sha256.Size || subtle.ConstantTimeCompare(digest, want) != 1 || revoked.Valid || !s.now().UTC().Before(principal.ExpiresAt) {
		return Principal{}, ErrUnauthorized
	}
	principal.Scopes = scopesMap(scopes)
	_, _ = s.db.ExecContext(ctx, `UPDATE api_credential SET last_used_at=? WHERE credential_id=? AND (last_used_at IS NULL OR last_used_at < ?)`, s.now().UTC(), principal.CredentialID, s.now().UTC().Add(-time.Minute))
	return principal, nil
}

func (s *Service) IssueCredential(ctx context.Context, actor Actor, advID uint64, name string, scopes []string, expiresAt time.Time, reason string) (Credential, string, error) {
	if err := validateLifecycle(actor, advID, name, reason); err != nil {
		return Credential{}, "", err
	}
	name = strings.TrimSpace(name)
	reason = strings.TrimSpace(reason)
	scopes, err := validateScopes(scopes)
	if err != nil {
		return Credential{}, "", err
	}
	now := s.now().UTC()
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(now.Add(5*time.Minute)) || expiresAt.After(now.Add(366*24*time.Hour)) {
		return Credential{}, "", fmt.Errorf("credential expiry must be more than five minutes and at most 366 days in the future")
	}
	publicBytes := make([]byte, 16)
	secret := make([]byte, 32)
	if _, err := io.ReadFull(s.random, publicBytes); err != nil {
		return Credential{}, "", err
	}
	if _, err := io.ReadFull(s.random, secret); err != nil {
		return Credential{}, "", err
	}
	publicID := hex.EncodeToString(publicBytes)
	token := "w8m_v1_" + publicID + "_" + base64.RawURLEncoding.EncodeToString(secret)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Credential{}, "", err
	}
	defer tx.Rollback()
	if err := advertiserExists(ctx, tx, advID); err != nil {
		return Credential{}, "", err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO api_credential
  (adv_id, credential_name, public_id, token_digest, permissions, expires_at,
   created_by_role, created_by_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		advID, name, publicID, s.digest("management-api-token-v1", token), strings.Join(scopes, ","), expiresAt,
		actor.Role, actor.ID, now)
	if err != nil {
		return Credential{}, "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Credential{}, "", err
	}
	if err := insertAPIAudit(ctx, tx, apiAudit{Actor: actor, AdvID: advID, Event: "CredentialIssued", ObjectType: "credential", ObjectID: uint64(id), NewState: "Active", Reason: reason, Outcome: "Success", CreatedAt: now}); err != nil {
		return Credential{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return Credential{}, "", err
	}
	return Credential{ID: uint64(id), AdvertiserID: advID, Name: name, PublicID: publicID, Scopes: scopes, ExpiresAt: expiresAt, CreatedAt: now}, token, nil
}

func (s *Service) RotateCredential(ctx context.Context, actor Actor, advID, credentialID uint64, reason string) (Credential, string, error) {
	if err := validateLifecycle(actor, advID, "rotation", reason); err != nil {
		return Credential{}, "", err
	}
	reason = strings.TrimSpace(reason)
	secret := make([]byte, 32)
	if _, err := io.ReadFull(s.random, secret); err != nil {
		return Credential{}, "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Credential{}, "", err
	}
	defer tx.Rollback()
	credential, err := loadCredentialForUpdate(ctx, tx, advID, credentialID)
	if err != nil {
		return Credential{}, "", err
	}
	if credential.RevokedAt != nil || !s.now().UTC().Before(credential.ExpiresAt) {
		return Credential{}, "", fmt.Errorf("credential is revoked or expired")
	}
	token := "w8m_v1_" + credential.PublicID + "_" + base64.RawURLEncoding.EncodeToString(secret)
	now := s.now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE api_credential SET token_digest=?, rotated_at=? WHERE credential_id=? AND adv_id=?`, s.digest("management-api-token-v1", token), now, credentialID, advID); err != nil {
		return Credential{}, "", err
	}
	if err := insertAPIAudit(ctx, tx, apiAudit{Actor: actor, AdvID: advID, Event: "CredentialRotated", ObjectType: "credential", ObjectID: credentialID, PriorState: "Active", NewState: "Rotated", Reason: reason, Outcome: "Success", CreatedAt: now}); err != nil {
		return Credential{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return Credential{}, "", err
	}
	return credential, token, nil
}

func (s *Service) RevokeCredential(ctx context.Context, actor Actor, advID, credentialID uint64, reason string) error {
	if err := validateLifecycle(actor, advID, "revocation", reason); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	now := s.now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE api_credential SET revoked_at=? WHERE credential_id=? AND adv_id=? AND revoked_at IS NULL`, now, credentialID, advID)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		return ErrNotFound
	}
	if err := insertAPIAudit(ctx, tx, apiAudit{Actor: actor, AdvID: advID, Event: "CredentialRevoked", ObjectType: "credential", ObjectID: credentialID, PriorState: "Active", NewState: "Revoked", Reason: reason, Outcome: "Success", CreatedAt: now}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ListCredentials(ctx context.Context, advID uint64) ([]Credential, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT credential_id, adv_id, credential_name, public_id, permissions,
       expires_at, revoked_at, created_at, last_used_at
FROM api_credential WHERE adv_id=? ORDER BY credential_id`, advID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Credential, 0)
	for rows.Next() {
		var item Credential
		var scopes string
		var revoked, used sql.NullTime
		if err := rows.Scan(&item.ID, &item.AdvertiserID, &item.Name, &item.PublicID, &scopes, &item.ExpiresAt, &revoked, &item.CreatedAt, &used); err != nil {
			return nil, err
		}
		item.Scopes = strings.Split(scopes, ",")
		item.ExpiresAt = item.ExpiresAt.UTC()
		item.CreatedAt = item.CreatedAt.UTC()
		if revoked.Valid {
			value := revoked.Time.UTC()
			item.RevokedAt = &value
		}
		if used.Valid {
			value := used.Time.UTC()
			item.LastUsedAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func validateLifecycle(actor Actor, advID uint64, name, reason string) error {
	if (actor.Role != "admin" && actor.Role != "adv") || actor.ID == 0 || advID == 0 {
		return fmt.Errorf("verified advertiser or administrator actor is required")
	}
	if actor.Role == "adv" && actor.ID != advID {
		return fmt.Errorf("advertiser cannot manage another account")
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 || strings.ContainsAny(name, "\r\n\x00") {
		return fmt.Errorf("credential name is required and must be a single line of at most 128 bytes")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 255 || strings.ContainsAny(reason, "\r\n\x00") {
		return fmt.Errorf("a single-line reason of at most 255 bytes is required")
	}
	return nil
}

func advertiserExists(ctx context.Context, tx *sql.Tx, advID uint64) error {
	var found uint64
	if err := tx.QueryRowContext(ctx, `SELECT adv_id FROM adv WHERE adv_id=?`, advID).Scan(&found); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func loadCredentialForUpdate(ctx context.Context, tx *sql.Tx, advID, credentialID uint64) (Credential, error) {
	var credential Credential
	var scopes string
	var revoked sql.NullTime
	err := tx.QueryRowContext(ctx, `
SELECT credential_id, adv_id, credential_name, public_id, permissions,
       expires_at, revoked_at, created_at
FROM api_credential WHERE credential_id=? AND adv_id=? FOR UPDATE`, credentialID, advID).Scan(
		&credential.ID, &credential.AdvertiserID, &credential.Name, &credential.PublicID,
		&scopes, &credential.ExpiresAt, &revoked, &credential.CreatedAt)
	if err == sql.ErrNoRows {
		return Credential{}, ErrNotFound
	}
	credential.Scopes = strings.Split(scopes, ",")
	credential.ExpiresAt = credential.ExpiresAt.UTC()
	credential.CreatedAt = credential.CreatedAt.UTC()
	if revoked.Valid {
		value := revoked.Time.UTC()
		credential.RevokedAt = &value
	}
	return credential, err
}

type apiAudit struct {
	Actor                                 Actor
	CredentialID                          uint64
	AdvID                                 uint64
	Event, ObjectType                     string
	ObjectID                              uint64
	IdempotencyHash                       []byte
	PriorState, NewState, Reason, Outcome string
	CreatedAt                             time.Time
}

func insertAPIAudit(ctx context.Context, tx *sql.Tx, audit apiAudit) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO api_audit
 (event_name, actor_role, actor_id, credential_id, adv_id, object_type,
  object_id, idempotency_hash, prior_state, new_state, reason, outcome, created_at)
VALUES (?, ?, NULLIF(?, 0), NULLIF(?, 0), ?, ?, NULLIF(?, 0), ?, ?, ?, ?, ?, ?)`,
		audit.Event, audit.Actor.Role, audit.Actor.ID, audit.CredentialID, audit.AdvID,
		audit.ObjectType, audit.ObjectID, audit.IdempotencyHash, audit.PriorState,
		audit.NewState, audit.Reason, audit.Outcome, audit.CreatedAt)
	return err
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

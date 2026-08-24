package publisherauth

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const maxCredentialLifetime = 366 * 24 * time.Hour

// IssueCredential creates one App-scoped keypair. The private seed is returned
// once and only the public Ed25519 verifier enters MySQL and runtime snapshots.
func (s *Service) IssueCredential(ctx context.Context, actor Actor, pubID, siteID uint64, name string, expiresAt time.Time, reason string) (Credential, string, error) {
	if s == nil || s.db == nil {
		return Credential{}, "", ErrUnavailable
	}
	if err := actor.validate(PermissionCredentialIssue, true, pubID); err != nil {
		return Credential{}, "", err
	}
	if siteID == 0 || validateLifecycleText(name, reason) != nil {
		return Credential{}, "", fmt.Errorf("credential lifecycle input is invalid")
	}
	name, reason = strings.TrimSpace(name), strings.TrimSpace(reason)
	now := s.currentTime()
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(now.Add(5*time.Minute)) || expiresAt.After(now.Add(maxCredentialLifetime)) {
		return Credential{}, "", fmt.Errorf("credential expiry must be more than five minutes and at most 366 days in the future")
	}
	publicID, publicKey, privateCredential, err := s.generateCredential()
	if err != nil {
		return Credential{}, "", err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Credential{}, "", err
	}
	defer tx.Rollback()
	if err := requireApprovedAppSite(ctx, tx, pubID, siteID); err != nil {
		return Credential{}, "", err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO pub_request_credential
 (pub_id, site_id, credential_name, public_id, public_key, algorithm,
  expires_at, created_by_role, created_by_id, created_at)
VALUES (?,?,?,?,?,'Ed25519-v1',?,?,?,?)`,
		pubID, siteID, name, publicID, publicKey, expiresAt, actor.Role, actor.ID, now)
	if err != nil {
		return Credential{}, "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Credential{}, "", err
	}
	credential := Credential{
		ID: uint64(id), PublisherID: pubID, SiteID: siteID, Name: name,
		PublicID: publicID, Algorithm: credentialAlgorithm, ExpiresAt: expiresAt, CreatedAt: now, State: "Active",
	}
	if err := s.insertCredentialAudit(ctx, tx, actor, credential, "PublisherCredentialIssued", "Absent", "Active", reason); err != nil {
		return Credential{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return Credential{}, "", err
	}
	s.mutateSnapshot(func(records map[string]verifierRecord) {
		records[publicID] = verifierRecord{
			principal: Principal{CredentialID: credential.ID, PublisherID: pubID, SiteID: siteID, PublicID: publicID},
			publicKey: append(ed25519.PublicKey(nil), publicKey...), validUntil: expiresAt,
		}
	})
	return credential, privateCredential, nil
}

// RotateCredential creates a replacement keypair. The predecessor is either
// revoked immediately or remains valid only through the requested bounded
// overlap, never beyond its original expiry.
func (s *Service) RotateCredential(ctx context.Context, actor Actor, pubID, credentialID uint64, overlap time.Duration, reason string) (Credential, string, error) {
	if s == nil || s.db == nil {
		return Credential{}, "", ErrUnavailable
	}
	if err := actor.validate(PermissionCredentialRotate, true, pubID); err != nil {
		return Credential{}, "", err
	}
	if credentialID == 0 || overlap < 0 || overlap > time.Duration(s.config.RotationMaxOverlapSeconds)*time.Second || validateLifecycleText("rotation", reason) != nil {
		return Credential{}, "", fmt.Errorf("credential rotation input is invalid")
	}
	reason = strings.TrimSpace(reason)
	now := s.currentTime()
	publicID, publicKey, privateCredential, err := s.generateCredential()
	if err != nil {
		return Credential{}, "", err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Credential{}, "", err
	}
	defer tx.Rollback()
	prior, err := loadCredentialForUpdate(ctx, tx, pubID, credentialID)
	if err != nil {
		return Credential{}, "", err
	}
	if prior.RevokedAt != nil || prior.RotatedAt != nil || !now.Before(prior.ExpiresAt) {
		return Credential{}, "", ErrConflict
	}
	if err := requireApprovedAppSite(ctx, tx, pubID, prior.SiteID); err != nil {
		return Credential{}, "", err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO pub_request_credential
 (pub_id, site_id, credential_name, public_id, public_key, algorithm,
  expires_at, replaces_credential_id, created_by_role, created_by_id, created_at)
VALUES (?,?,?,?,?,'Ed25519-v1',?,?,?,?,?)`,
		pubID, prior.SiteID, prior.Name, publicID, publicKey, prior.ExpiresAt,
		prior.ID, actor.Role, actor.ID, now)
	if err != nil {
		return Credential{}, "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Credential{}, "", err
	}
	overlapUntil := now.Add(overlap)
	if overlapUntil.After(prior.ExpiresAt) {
		overlapUntil = prior.ExpiresAt
	}
	var revokedAt any
	if overlap == 0 {
		revokedAt = now
	}
	updateResult, err := tx.ExecContext(ctx, `
UPDATE pub_request_credential
SET rotated_at=?, overlap_until=?, revoked_at=?
WHERE credential_id=? AND pub_id=? AND rotated_at IS NULL AND revoked_at IS NULL`,
		now, overlapUntil, revokedAt, prior.ID, pubID)
	if err != nil {
		return Credential{}, "", err
	}
	if rows, err := updateResult.RowsAffected(); err != nil || rows != 1 {
		return Credential{}, "", ErrConflict
	}
	credential := Credential{
		ID: uint64(id), PublisherID: pubID, SiteID: prior.SiteID, Name: prior.Name,
		PublicID: publicID, Algorithm: credentialAlgorithm, ExpiresAt: prior.ExpiresAt,
		ReplacesID: prior.ID, CreatedAt: now, State: "Active",
	}
	nextState := "Overlap"
	if overlap == 0 {
		nextState = "Revoked"
	}
	if err := s.insertCredentialAudit(ctx, tx, actor, prior, "PublisherCredentialRotated", "Active", nextState, reason); err != nil {
		return Credential{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return Credential{}, "", err
	}
	s.mutateSnapshot(func(records map[string]verifierRecord) {
		if overlap == 0 {
			delete(records, prior.PublicID)
		} else if old, ok := records[prior.PublicID]; ok {
			old.validUntil = overlapUntil
			records[prior.PublicID] = old
		}
		records[publicID] = verifierRecord{
			principal: Principal{CredentialID: credential.ID, PublisherID: pubID, SiteID: prior.SiteID, PublicID: publicID},
			publicKey: append(ed25519.PublicKey(nil), publicKey...), validUntil: prior.ExpiresAt,
		}
	})
	return credential, privateCredential, nil
}

// RevokeCredential withdraws one credential from the local snapshot
// immediately and from every other node after the bounded refresh interval.
func (s *Service) RevokeCredential(ctx context.Context, actor Actor, pubID, credentialID uint64, reason string) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	if err := actor.validate(PermissionCredentialRevoke, true, pubID); err != nil {
		return err
	}
	if credentialID == 0 || validateLifecycleText("revocation", reason) != nil {
		return fmt.Errorf("credential revocation input is invalid")
	}
	reason = strings.TrimSpace(reason)
	now := s.currentTime()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	credential, err := loadCredentialForUpdate(ctx, tx, pubID, credentialID)
	if err != nil {
		return err
	}
	if credential.RevokedAt != nil {
		return ErrConflict
	}
	result, err := tx.ExecContext(ctx, `
UPDATE pub_request_credential SET revoked_at=?
WHERE credential_id=? AND pub_id=? AND revoked_at IS NULL`, now, credentialID, pubID)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		return ErrConflict
	}
	if err := s.insertCredentialAudit(ctx, tx, actor, credential, "PublisherCredentialRevoked", credentialState(credential, now), "Revoked", reason); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.mutateSnapshot(func(records map[string]verifierRecord) { delete(records, credential.PublicID) })
	return nil
}

// ListCredentials returns display-safe metadata within the verified publisher
// scope. It never returns a public verifier or private signing seed.
func (s *Service) ListCredentials(ctx context.Context, actor Actor, pubID uint64) ([]Credential, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	if err := actor.validate(PermissionCredentialRead, false, pubID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT credential_id, pub_id, site_id, credential_name, public_id, algorithm,
       expires_at, rotated_at, overlap_until, revoked_at,
       COALESCE(replaces_credential_id,0), created_at
FROM pub_request_credential WHERE pub_id=? ORDER BY credential_id`, pubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Credential, 0)
	for rows.Next() {
		credential, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		credential.State = credentialState(credential, s.currentTime())
		items = append(items, credential)
	}
	return items, rows.Err()
}

func (s *Service) generateCredential() (string, ed25519.PublicKey, string, error) {
	publicIDBytes := make([]byte, 16)
	if _, err := io.ReadFull(s.random, publicIDBytes); err != nil {
		return "", nil, "", err
	}
	publicKey, privateKey, err := ed25519.GenerateKey(s.random)
	if err != nil {
		return "", nil, "", err
	}
	publicID := hex.EncodeToString(publicIDBytes)
	privateCredential := credentialPrefix + publicID + "_" + base64.RawURLEncoding.EncodeToString(privateKey.Seed())
	return publicID, publicKey, privateCredential, nil
}

func requireApprovedAppSite(ctx context.Context, tx *sql.Tx, pubID, siteID uint64) error {
	var found uint64
	err := tx.QueryRowContext(ctx, `
SELECT ps.site_id
FROM pub_site ps INNER JOIN pub p ON p.pub_id=ps.pub_id
WHERE ps.site_id=? AND ps.pub_id=? AND p.active='Yes' AND ps.active='Yes'
  AND ps.site_type='App' AND ps.inventory_environment='App'
  AND ps.integration_mode IN ('SDK','ServerAPI')
FOR SHARE`, siteID, pubID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func loadCredentialForUpdate(ctx context.Context, tx *sql.Tx, pubID, credentialID uint64) (Credential, error) {
	row := tx.QueryRowContext(ctx, `
SELECT credential_id, pub_id, site_id, credential_name, public_id, algorithm,
       expires_at, rotated_at, overlap_until, revoked_at,
       COALESCE(replaces_credential_id,0), created_at
FROM pub_request_credential
WHERE credential_id=? AND pub_id=? FOR UPDATE`, credentialID, pubID)
	credential, err := scanCredential(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrNotFound
	}
	return credential, err
}

type rowScanner interface{ Scan(...any) error }

func scanCredential(row rowScanner) (Credential, error) {
	var credential Credential
	var rotated, overlap, revoked sql.NullTime
	err := row.Scan(
		&credential.ID, &credential.PublisherID, &credential.SiteID, &credential.Name,
		&credential.PublicID, &credential.Algorithm, &credential.ExpiresAt,
		&rotated, &overlap, &revoked, &credential.ReplacesID, &credential.CreatedAt,
	)
	if err != nil {
		return Credential{}, err
	}
	credential.ExpiresAt = credential.ExpiresAt.UTC()
	credential.CreatedAt = credential.CreatedAt.UTC()
	credential.RotatedAt = nullTime(rotated)
	credential.OverlapUntil = nullTime(overlap)
	credential.RevokedAt = nullTime(revoked)
	return credential, nil
}

func nullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	normalized := value.Time.UTC()
	return &normalized
}

func credentialState(credential Credential, now time.Time) string {
	if credential.RevokedAt != nil {
		return "Revoked"
	}
	if !now.Before(credential.ExpiresAt) {
		return "Expired"
	}
	if credential.RotatedAt != nil {
		if credential.OverlapUntil != nil && now.Before(*credential.OverlapUntil) {
			return "Overlap"
		}
		return "Rotated"
	}
	return "Active"
}

func (s *Service) insertCredentialAudit(ctx context.Context, tx *sql.Tx, actor Actor, credential Credential, event, prior, next, reason string) error {
	digest := sha256.Sum256([]byte("publisher-credential:" + credential.PublicID))
	_, err := tx.ExecContext(ctx, `
INSERT INTO auth_security_audit
 (event_name, actor_role, actor_id, subject_role, subject_id, permission_name,
  resource_role, resource_id, object_hash, prior_state, new_state, reason,
  outcome, created_at)
VALUES (?,?,?,'pub',?,?,?,?,?,?,?,?,'Success',?)`,
		event, actor.Role, actor.ID, credential.PublisherID,
		permissionForEvent(event), "site", credential.SiteID, digest[:], prior, next, reason, s.currentTime())
	return err
}

func permissionForEvent(event string) string {
	switch event {
	case "PublisherCredentialIssued":
		return PermissionCredentialIssue
	case "PublisherCredentialRotated":
		return PermissionCredentialRotate
	case "PublisherCredentialRevoked":
		return PermissionCredentialRevoke
	default:
		return "publisher.credential.unknown"
	}
}

func (s *Service) mutateSnapshot(mutate func(map[string]verifierRecord)) {
	if s == nil || mutate == nil {
		return
	}
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	current := s.snapshot.Load()
	if current == nil {
		return
	}
	records := make(map[string]verifierRecord, len(current.records)+1)
	for key, value := range current.records {
		records[key] = value
	}
	mutate(records)
	s.snapshot.Store(&verifierSnapshot{generatedAt: current.generatedAt, records: records})
}

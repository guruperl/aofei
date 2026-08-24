package publisherauth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/mediocregopher/radix/v4"
)

func TestRequestProofBindsBodyFreshnessScopeAndSharedReplay(t *testing.T) {
	service, privateCredential, now := requestTestService(t, 7, 42, 11)
	body := []byte(`{"platform":"sdk","site":"fixture"}`)
	nonce := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x44}, 16))
	headers, err := SignRequest(privateCredential, now, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	request.Header = headers
	proof, err := service.VerifyRequest(request, body)
	if err != nil {
		t.Fatal(err)
	}
	if got := proof.Principal(); got.CredentialID != 7 || got.PublisherID != 42 || got.SiteID != 11 {
		t.Fatalf("principal = %#v", got)
	}
	if err := proof.AuthorizeScope(42, 12); !errors.Is(err, ErrScope) {
		t.Fatalf("cross-App scope error = %v", err)
	}
	if err := proof.AuthorizeScope(42, 11); err != nil {
		t.Fatal(err)
	}
	if err := service.ClaimReplay(context.Background(), proof); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("w8m-pz-replay-v1\x00" + proof.principal.PublicID + "\x00" + proof.nonce))
	replayKey := "pz:auth:replay:" + base64.RawURLEncoding.EncodeToString(digest[:])
	if err := service.redis.Do(context.Background(), radix.Cmd(nil, "SET", replayKey, "opaque-non-secret-label", "EX", "300")); err != nil {
		t.Fatal(err)
	}
	if err := service.ClaimReplay(context.Background(), proof); !errors.Is(err, ErrReplay) {
		t.Fatalf("arbitrary replay marker value became authoritative: %v", err)
	}

	tampered := append([]byte(nil), body...)
	tampered[len(tampered)-2] ^= 1
	if _, err := service.VerifyRequest(request, tampered); !errors.Is(err, ErrInvalid) {
		t.Fatalf("body tamper error = %v", err)
	}
	staleHeaders, err := SignRequest(privateCredential, now.Add(-301*time.Second), nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header = staleHeaders
	if _, err := service.VerifyRequest(request, body); !errors.Is(err, ErrStale) {
		t.Fatalf("stale proof error = %v", err)
	}
	request.Header.Del(HeaderCredential)
	if _, err := service.VerifyRequest(request, body); !errors.Is(err, ErrRequired) {
		t.Fatalf("missing credential error = %v", err)
	}
}

func TestRequestProofRejectsStaleSnapshotExpiredCredentialAndMalformedHeaders(t *testing.T) {
	service, privateCredential, now := requestTestService(t, 7, 42, 11)
	body := []byte(`{"platform":"sdk"}`)
	nonce := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x45}, 16))
	headers, err := SignRequest(privateCredential, now, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	request.Header = headers

	snapshot := service.snapshot.Load()
	service.snapshot.Store(&verifierSnapshot{generatedAt: now.Add(-121 * time.Second), records: snapshot.records})
	if _, err := service.VerifyRequest(request, body); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("stale snapshot error = %v", err)
	}
	service.snapshot.Store(&verifierSnapshot{generatedAt: now, records: map[string]verifierRecord{}})
	if _, err := service.VerifyRequest(request, body); !errors.Is(err, ErrInvalid) {
		t.Fatalf("withdrawn credential error = %v", err)
	}
	service.snapshot.Store(snapshot)
	request.Header.Add(HeaderSignature, headers.Get(HeaderSignature))
	if _, err := service.VerifyRequest(request, body); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate header error = %v", err)
	}
}

func TestSnapshotReloadCannotUndoConcurrentLifecycleMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	publicID := "00112233445566778899aabbccddeeff"
	publicKey := bytes.Repeat([]byte{0x31}, ed25519.PublicKeySize)
	service := &Service{db: db, now: func() time.Time { return now }}
	service.snapshot.Store(&verifierSnapshot{generatedAt: now, records: map[string]verifierRecord{
		publicID: {
			principal: Principal{CredentialID: 7, PublisherID: 42, SiteID: 11, PublicID: publicID},
			publicKey: publicKey, validUntil: now.Add(time.Hour),
		},
	}})
	mock.ExpectQuery("SELECT c.credential_id, c.pub_id, c.site_id").
		WithArgs(now, now).
		WillDelayFor(200 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{
			"credential_id", "pub_id", "site_id", "public_id", "public_key",
			"expires_at", "rotated_at", "overlap_until",
		}).AddRow(7, 42, 11, publicID, publicKey, now.Add(time.Hour), nil, nil))

	reloadDone := make(chan error, 1)
	go func() { reloadDone <- service.ReloadSnapshot(context.Background()) }()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		if err := mock.ExpectationsWereMet(); err == nil {
			break
		}
		select {
		case <-deadline.C:
			t.Fatal("snapshot reload did not begin its MySQL query")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	mutationDone := make(chan struct{})
	go func() {
		service.mutateSnapshot(func(records map[string]verifierRecord) { delete(records, publicID) })
		close(mutationDone)
	}()
	select {
	case <-mutationDone:
		t.Fatal("lifecycle snapshot mutation bypassed an in-flight reload")
	case <-time.After(25 * time.Millisecond):
	}
	if err := <-reloadDone; err != nil {
		t.Fatal(err)
	}
	<-mutationDone
	if _, ok := service.snapshot.Load().records[publicID]; ok {
		t.Fatal("stale reload undid the lifecycle snapshot mutation")
	}
}

func TestRequestProofConcurrentReplayHasOneWinner(t *testing.T) {
	service, privateCredential, now := requestTestService(t, 7, 42, 11)
	body := []byte(`{"platform":"sdk","id":"concurrent"}`)
	nonce := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x46}, 16))
	headers, err := SignRequest(privateCredential, now, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	request.Header = headers
	proof, err := service.VerifyRequest(request, body)
	if err != nil {
		t.Fatal(err)
	}
	var accepted atomic.Int32
	var replayed atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := service.ClaimReplay(context.Background(), proof)
			switch {
			case err == nil:
				accepted.Add(1)
			case errors.Is(err, ErrReplay):
				replayed.Add(1)
			default:
				t.Errorf("claim replay: %v", err)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 || replayed.Load() != 31 {
		t.Fatalf("accepted/replayed = %d/%d", accepted.Load(), replayed.Load())
	}
}

func TestCredentialLifecycleIsScopedAuditedAndRotatable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	service := &Service{
		config: (Config{Enabled: true}).WithDefaults(), db: db, now: func() time.Time { return now },
		random: bytes.NewReader(append(
			bytes.Repeat([]byte{0x51}, 16+ed25519.SeedSize),
			bytes.Repeat([]byte{0x52}, 16+ed25519.SeedSize)...,
		)),
	}
	service.snapshot.Store(&verifierSnapshot{generatedAt: now, records: map[string]verifierRecord{}})
	actor := Actor{Role: "pub", ID: 42, RecentMFA: true, Permissions: map[string]bool{"publisher.credential.*": false,
		PermissionCredentialRead: true, PermissionCredentialIssue: true, PermissionCredentialRotate: true, PermissionCredentialRevoke: true}}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT ps.site_id").WithArgs(uint64(11), uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"site_id"}).AddRow(11))
	mock.ExpectExec("INSERT INTO pub_request_credential").WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectExec("INSERT INTO auth_security_audit").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	issued, privateCredential, err := service.IssueCredential(context.Background(), actor, 42, 11, "primary app", now.Add(24*time.Hour), "issue approved App credential")
	if err != nil {
		t.Fatal(err)
	}
	if issued.ID != 7 || issued.SiteID != 11 || !bytes.HasPrefix([]byte(privateCredential), []byte(credentialPrefix)) {
		t.Fatalf("issued credential = %#v / malformed private value", issued)
	}
	if _, seed, err := parsePrivateCredential(privateCredential); err != nil || len(seed) != ed25519.SeedSize {
		t.Fatalf("private credential parse = %d, %v", len(seed), err)
	}
	if _, ok := service.snapshot.Load().records[issued.PublicID]; !ok {
		t.Fatal("issued verifier was not installed locally")
	}

	mock.ExpectBegin()
	credentialRows := sqlmock.NewRows([]string{
		"credential_id", "pub_id", "site_id", "credential_name", "public_id", "algorithm",
		"expires_at", "rotated_at", "overlap_until", "revoked_at", "replaces_credential_id", "created_at",
	}).AddRow(issued.ID, issued.PublisherID, issued.SiteID, issued.Name, issued.PublicID, issued.Algorithm,
		issued.ExpiresAt, nil, nil, nil, 0, issued.CreatedAt)
	mock.ExpectQuery("SELECT credential_id, pub_id, site_id").WithArgs(issued.ID, uint64(42)).WillReturnRows(credentialRows)
	mock.ExpectQuery("SELECT ps.site_id").WithArgs(uint64(11), uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"site_id"}).AddRow(11))
	mock.ExpectExec("INSERT INTO pub_request_credential").WillReturnResult(sqlmock.NewResult(8, 1))
	mock.ExpectExec("UPDATE pub_request_credential").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO auth_security_audit").WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	replacement, replacementPrivate, err := service.RotateCredential(context.Background(), actor, 42, issued.ID, 10*time.Minute, "bounded App key overlap")
	if err != nil {
		t.Fatal(err)
	}
	if replacement.ID != 8 || replacement.ReplacesID != issued.ID || replacementPrivate == privateCredential {
		t.Fatalf("replacement = %#v", replacement)
	}
	if old := service.snapshot.Load().records[issued.PublicID]; !old.validUntil.Equal(now.Add(10 * time.Minute)) {
		t.Fatalf("old valid until = %s", old.validUntil)
	}

	mock.ExpectBegin()
	replacementRows := sqlmock.NewRows([]string{
		"credential_id", "pub_id", "site_id", "credential_name", "public_id", "algorithm",
		"expires_at", "rotated_at", "overlap_until", "revoked_at", "replaces_credential_id", "created_at",
	}).AddRow(replacement.ID, replacement.PublisherID, replacement.SiteID, replacement.Name, replacement.PublicID, replacement.Algorithm,
		replacement.ExpiresAt, nil, nil, nil, replacement.ReplacesID, replacement.CreatedAt)
	mock.ExpectQuery("SELECT credential_id, pub_id, site_id").WithArgs(replacement.ID, uint64(42)).WillReturnRows(replacementRows)
	mock.ExpectExec("UPDATE pub_request_credential SET revoked_at").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO auth_security_audit").WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectCommit()
	if err := service.RevokeCredential(context.Background(), actor, 42, replacement.ID, "revoke replacement after compromise"); err != nil {
		t.Fatal(err)
	}
	if _, ok := service.snapshot.Load().records[replacement.PublicID]; ok {
		t.Fatal("revoked verifier remained in local snapshot")
	}

	crossAccount := actor
	crossAccount.ID = 43
	if _, _, err := service.IssueCredential(context.Background(), crossAccount, 42, 11, "forbidden", now.Add(time.Hour), "cross-account attempt"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-account issue error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialStateDistinguishesExpiryAndCompletedRotation(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	if got := credentialState(Credential{ExpiresAt: now}, now); got != "Expired" {
		t.Fatalf("expired state=%q", got)
	}
	rotatedAt, overlapUntil := now.Add(-time.Hour), now.Add(-time.Minute)
	if got := credentialState(Credential{ExpiresAt: now.Add(time.Hour), RotatedAt: &rotatedAt, OverlapUntil: &overlapUntil}, now); got != "Rotated" {
		t.Fatalf("rotated state=%q", got)
	}
	overlapUntil = now.Add(time.Minute)
	if got := credentialState(Credential{ExpiresAt: now.Add(time.Hour), RotatedAt: &rotatedAt, OverlapUntil: &overlapUntil}, now); got != "Overlap" {
		t.Fatalf("overlap state=%q", got)
	}
}

func requestTestService(t *testing.T, credentialID, pubID, siteID uint64) (*Service, string, time.Time) {
	t.Helper()
	server := miniredis.RunT(t)
	redis, err := (radix.PoolConfig{Size: 8}).New(context.Background(), "tcp", server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { redis.Close() })
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	seed := bytes.Repeat([]byte{0x32}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicID := "00112233445566778899aabbccddeeff"
	service := &Service{
		config: (Config{Enabled: true}).WithDefaults(), redis: redis, now: func() time.Time { return now },
	}
	service.snapshot.Store(&verifierSnapshot{generatedAt: now, records: map[string]verifierRecord{
		publicID: {
			principal: Principal{CredentialID: credentialID, PublisherID: pubID, SiteID: siteID, PublicID: publicID},
			publicKey: privateKey.Public().(ed25519.PublicKey), validUntil: now.Add(time.Hour),
		},
	}})
	privateCredential := credentialPrefix + publicID + "_" + base64.RawURLEncoding.EncodeToString(seed)
	return service, privateCredential, now
}

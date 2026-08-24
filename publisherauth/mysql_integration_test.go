package publisherauth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mediocregopher/radix/v4"
)

const integrationPublisherID = uint64(4000000001)

func TestMySQLPublisherCredentialLifecycle(t *testing.T) {
	dsn := os.Getenv("PUBLISHER_AUTH_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("PUBLISHER_AUTH_INTEGRATION_DSN is unset")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	cleanupPublisherAuthIntegration(t, db)
	t.Cleanup(func() { cleanupPublisherAuthIntegration(t, db) })
	if _, err := db.Exec(`INSERT INTO pub (pub_id,email,passwd,active,created) VALUES (?,'p03-integration@example.invalid','integration-only','Yes',UTC_TIMESTAMP())`, integrationPublisherID); err != nil {
		t.Fatal(err)
	}
	result, err := db.Exec(`INSERT INTO pub_site (pub_id,site_name,site_url,site_type,inventory_environment,canonical_identity,integration_mode,active,created) VALUES (?,'P03 integration App','p03.example.invalid','App','App','p03.example.invalid','SDK','Yes',UTC_TIMESTAMP())`, integrationPublisherID)
	if err != nil {
		t.Fatal(err)
	}
	siteID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	redisServer := miniredis.RunT(t)
	redisClient, err := (radix.PoolConfig{Size: 4}).New(context.Background(), "tcp", redisServer.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = redisClient.Close() })
	service, err := NewService(Config{Enabled: true}, db, redisClient)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	service.now = func() time.Time { return now }
	actor := Actor{Role: "pub", ID: integrationPublisherID, RecentMFA: true, Permissions: map[string]bool{
		PermissionCredentialRead: true, PermissionCredentialIssue: true,
		PermissionCredentialRotate: true, PermissionCredentialRevoke: true,
	}}
	issued, privateValue, err := service.IssueCredential(context.Background(), actor, integrationPublisherID, uint64(siteID), "integration", now.Add(24*time.Hour), "P03 disposable lifecycle proof")
	if err != nil {
		t.Fatal(err)
	}
	var storedPublicKey []byte
	if err := db.QueryRow(`SELECT public_key FROM pub_request_credential WHERE credential_id=?`, issued.ID).Scan(&storedPublicKey); err != nil {
		t.Fatal(err)
	}
	if len(storedPublicKey) != 32 || bytes.Contains(storedPublicKey, []byte(privateValue)) {
		t.Fatal("database did not retain exactly one public verifier")
	}
	listed, err := service.ListCredentials(context.Background(), actor, integrationPublisherID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != issued.ID || listed[0].State != "Active" {
		t.Fatalf("listed credentials=%#v", listed)
	}

	body := []byte(`{"platform":"sdk","site":"integration"}`)
	nonce, err := NewRequestNonce()
	if err != nil {
		t.Fatal(err)
	}
	headers, err := SignRequest(privateValue, now, nonce, body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/pz", bytes.NewReader(body))
	request.Header = headers
	proof, err := service.VerifyRequest(request, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := proof.AuthorizeScope(integrationPublisherID, uint64(siteID)); err != nil {
		t.Fatal(err)
	}
	if err := service.ClaimReplay(context.Background(), proof); err != nil {
		t.Fatal(err)
	}
	if err := service.ClaimReplay(context.Background(), proof); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay error=%v", err)
	}

	replacement, replacementPrivate, err := service.RotateCredential(context.Background(), actor, integrationPublisherID, issued.ID, time.Minute, "P03 disposable bounded rotation")
	if err != nil {
		t.Fatal(err)
	}
	if replacement.ReplacesID != issued.ID || replacementPrivate == privateValue {
		t.Fatalf("replacement=%#v", replacement)
	}
	if err := service.RevokeCredential(context.Background(), actor, integrationPublisherID, replacement.ID, "P03 disposable revocation"); err != nil {
		t.Fatal(err)
	}
	replacementNonce, err := NewRequestNonce()
	if err != nil {
		t.Fatal(err)
	}
	replacementHeaders, err := SignRequest(replacementPrivate, now, replacementNonce, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header = replacementHeaders
	if _, err := service.VerifyRequest(request, body); !errors.Is(err, ErrInvalid) {
		t.Fatalf("revoked verifier error=%v", err)
	}

	var audits int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_security_audit WHERE subject_role='pub' AND subject_id=? AND event_name IN ('PublisherCredentialIssued','PublisherCredentialRotated','PublisherCredentialRevoked')`, integrationPublisherID).Scan(&audits); err != nil || audits != 3 {
		t.Fatalf("audit count=%d err=%v", audits, err)
	}
	if _, err := db.Exec(`UPDATE auth_security_audit SET outcome='Failure' WHERE subject_role='pub' AND subject_id=? LIMIT 1`, integrationPublisherID); err == nil {
		t.Fatal("publisher credential lifecycle audit was mutable")
	}
}

func cleanupPublisherAuthIntegration(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SET @aofei_auth_audit_retention=1`); err != nil {
		t.Fatal(err)
	}
	_, deleteErr := conn.ExecContext(ctx, `DELETE FROM auth_security_audit WHERE subject_role='pub' AND subject_id=? AND event_name IN ('PublisherCredentialIssued','PublisherCredentialRotated','PublisherCredentialRevoked')`, integrationPublisherID)
	_, resetErr := conn.ExecContext(ctx, `SET @aofei_auth_audit_retention=0`)
	if deleteErr != nil || resetErr != nil {
		t.Fatalf("audit cleanup delete=%v reset=%v", deleteErr, resetErr)
	}
	for _, statement := range []string{
		`DELETE FROM pub_request_credential WHERE pub_id=4000000001 ORDER BY credential_id DESC`,
		`DELETE FROM pub_site WHERE pub_id=4000000001`,
		`DELETE FROM pub WHERE pub_id=4000000001`,
	} {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatalf("cleanup %q: %v", statement, err)
		}
	}
}

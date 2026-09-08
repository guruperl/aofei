package main

import (
	"os"
	"strings"
	"testing"
)

func TestAccountIdentifierProtectionSchemaIsAdditiveAndPasswordSafe(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(data)
	for _, fragment := range []string{
		"`email_hmac` binary(32) DEFAULT NULL",
		"`email_cipher` varbinary(512) DEFAULT NULL",
		"`login_hmac` binary(32) DEFAULT NULL",
		"`login_cipher` varbinary(512) DEFAULT NULL",
		"`activation_token_digest` binary(32) DEFAULT NULL",
		"`activation_token_expires` datetime(6) DEFAULT NULL",
		"`reset_token_digest` binary(32) DEFAULT NULL",
		"`reset_token_expires` datetime(6) DEFAULT NULL",
		"UNIQUE KEY `adv_email_hmac` (`email_hmac`)",
		"UNIQUE KEY `pub_email_hmac` (`email_hmac`)",
		"UNIQUE KEY `admin_login_hmac` (`login_hmac`)",
		"UNIQUE KEY `agent_login_hmac` (`login_hmac`)",
		"UNIQUE KEY `analyst_login_hmac` (`login_hmac`)",
		"UNIQUE KEY `adv_activation_token_digest` (`activation_token_digest`)",
		"UNIQUE KEY `adv_reset_token_digest` (`reset_token_digest`)",
		"UNIQUE KEY `pub_activation_token_digest` (`activation_token_digest`)",
		"UNIQUE KEY `pub_reset_token_digest` (`reset_token_digest`)",
	} {
		if !strings.Contains(schema, fragment) {
			t.Errorf("schema is missing %q", fragment)
		}
	}
	if strings.Contains(schema, "passwd_hmac") {
		t.Fatal("schema contains a fast deterministic password verifier")
	}
	for _, rawTokenColumn := range []string{"`activation_token`", "`reset_token`"} {
		if strings.Contains(schema, rawTokenColumn) {
			t.Fatalf("schema contains raw account-action token column %s", rawTokenColumn)
		}
	}
	if strings.Contains(strings.ToLower(schema), "p.passwd=i_passwd") {
		t.Fatal("schema still compares a plaintext password in MySQL")
	}
}

func TestAccountIdentifierMigrationIsAdditiveOnly(t *testing.T) {
	data, err := os.ReadFile("s07_account_identifier_migration.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := strings.ToLower(string(data))
	for _, forbidden := range []string{"drop column", "modify email_hmac", "modify login_hmac", "add column passwd_hmac"} {
		if strings.Contains(migration, forbidden) {
			t.Errorf("additive migration contains %q", forbidden)
		}
	}
}

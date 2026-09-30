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

func TestAccountIdentifierRetirementMigrationIsGuardedAndOneWay(t *testing.T) {
	data, err := os.ReadFile("s07_account_identifier_retirement.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := strings.ToLower(string(data))
	preflight := strings.Index(migration, "prepare s07_retirement_preflight_statement")
	firstMutation := strings.Index(migration, "drop procedure proc_adv;")
	if preflight < 0 || firstMutation < 0 || preflight > firstMutation {
		t.Fatal("retirement migration must execute its fail-closed preflight before DDL")
	}
	for _, fragment := range []string{
		"@s07_retirement_source_shape_ok",
		"@s07_retirement_indexes_ok",
		"@s07_retirement_protected_indexes_ok",
		"@s07_retirement_trigger_shape_ok",
		"@s07_retirement_dependent_views_absent",
		"@s07_retirement_legacy_procedures_ok",
		"@s07_retirement_rows_complete",
		"@s07_password_digest_absent",
		"drop procedure proc_adv_as;",
		"drop procedure proc_pub;",
		"drop procedure proc_pub_as;",
		"drop column email",
		"drop column login",
		"modify column email_hmac binary(32) not null",
		"modify column email_cipher varbinary(512) not null",
		"modify column login_hmac binary(32) not null",
		"modify column login_cipher varbinary(512) not null",
	} {
		if !strings.Contains(migration, fragment) {
			t.Errorf("retirement migration is missing %q", fragment)
		}
	}
	if strings.Contains(migration, "add column passwd_hmac") {
		t.Fatal("retirement migration adds a fast deterministic password verifier")
	}
}

package main

import (
	"os"
	"strings"
	"testing"
)

func TestPublisherRequestCredentialBaselineStoresOnlyPublicVerifierMetadata(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		"CREATE TABLE `pub_request_credential`",
		"`public_key` binary(32) NOT NULL",
		"`algorithm` enum('Ed25519-v1')",
		"`pub_request_credential_scope_fk`",
		"`pub_request_credential_rotation_chk`",
		"CREATE TABLE `auth_security_audit`",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("baseline is missing %q", required)
		}
	}
	table := schemaTableDefinition(text, "pub_request_credential")
	for _, forbidden := range []string{"private_key", "private_seed", "raw_signature", "request_nonce", "request_body"} {
		if strings.Contains(table, forbidden) {
			t.Errorf("publisher credential table contains prohibited secret/proof field %q", forbidden)
		}
	}
}

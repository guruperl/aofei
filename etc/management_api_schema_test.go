package main

import (
	"os"
	"strings"
	"testing"
)

func TestManagementAPIBaselineOwnsDigestsIdempotencyVersionsAndAudit(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{
		"CREATE TABLE `api_credential`", "`token_digest` binary(32)",
		"CREATE TABLE `api_idempotency`", "`idempotency_hash` binary(32)",
		"`claim_token` binary(16)",
		"UNIQUE KEY `api_idempotency_once`", "CREATE TABLE `api_operation`",
		"`publication_token` binary(16)",
		"CREATE TABLE `api_audit`", "trig_api_audit_no_update", "trig_api_audit_no_delete",
		"trig_campaign_api_version", "trig_item_api_version", "trig_creative_api_version",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("baseline is missing %q", required)
		}
	}
	for _, forbidden := range []string{"w8m_v1_", "Authorization: Bearer"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("baseline contains bearer material %q", forbidden)
		}
	}
}

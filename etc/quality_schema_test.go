package main

import (
	"os"
	"strings"
	"testing"
)

func TestTrafficQualitySchemaIsVersionedScopedAndImmutable(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(data)
	for _, table := range []string{
		"quality_rule", "quality_decision", "quality_evidence", "quality_case",
		"quality_case_event", "quality_enforcement", "quality_billing",
		"quality_counter", "quality_audit",
	} {
		if schemaTableDefinition(schema, table) == "" {
			t.Errorf("traffic-quality schema is missing %s", table)
		}
	}
	rule := schemaTableDefinition(schema, "quality_rule")
	for _, required := range []string{
		"`rule_key` varchar(64)", "`rule_version` int unsigned", "quality_rule_version",
		"'Replay','ImpossibleSequence','InvalidOriginApp','MalformedIdentity','AbnormalRate','AbnormalCTR','Automation','PartnerPolicy'",
		"'Observe','Flag','Throttle','Reject','Quarantine'", "quality_rule_scope_chk",
		"quality_rule_evidence_retention_chk", "quality_rule_false_positive_chk",
	} {
		if !strings.Contains(rule, required) {
			t.Errorf("quality_rule is missing %q", required)
		}
	}
	decision := schemaTableDefinition(schema, "quality_decision")
	for _, required := range []string{
		"`event_digest` binary(32) NOT NULL", "`partner_digest` binary(32)",
		"`evidence_state` enum('Complete','Partial','Missing')", "`billing_disposition` enum('Observe','Exclude','Hold','Reverse')",
		"quality_decision_once", "quality_decision_evidence_expiry",
		"quality_decision_evidence_chk", "quality_decision_action_chk",
		"quality_decision_billing_chk",
	} {
		if !strings.Contains(decision, required) {
			t.Errorf("quality_decision is missing %q", required)
		}
	}
	enforcement := schemaTableDefinition(schema, "quality_enforcement")
	for _, required := range []string{"quality_enforcement_canary_chk", "BETWEEN 1 AND 9999", "IN (0,10000)"} {
		if !strings.Contains(enforcement, required) {
			t.Errorf("quality_enforcement is missing %q", required)
		}
	}
	for _, prohibited := range []string{"ip_address", "user_agent", "cookie", "email", "bearer", "auction_id", "device_id", "raw_evidence"} {
		if strings.Contains(decision, prohibited) || strings.Contains(schemaTableDefinition(schema, "quality_evidence"), prohibited) {
			t.Errorf("traffic-quality evidence contains prohibited field %q", prohibited)
		}
	}
	for _, required := range []string{
		"quality_rule_versioned_update", "quality_rule_immutable_delete",
		"quality_decision_immutable_update", "quality_decision_immutable_delete",
		"quality_evidence_immutable_update", "quality_evidence_retained_delete",
		"quality_case_event_immutable_update", "quality_case_event_immutable_delete",
		"quality_enforcement_protected_update", "quality_billing_protected_update",
		"traffic-quality enforcement identity is immutable",
		"traffic-quality billing identity is immutable",
		"quality_audit_immutable_update", "quality_audit_immutable_delete",
		"@aofei_quality_retention",
	} {
		if !strings.Contains(schema, required) {
			t.Errorf("traffic-quality immutability/retention contract is missing %q", required)
		}
	}
}

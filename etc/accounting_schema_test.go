package main

import (
	"os"
	"strings"
	"testing"
)

func TestAccountingBaselineHasV2ContractAndNoCredentialColumns(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(data)
	for _, required := range []string{
		"CREATE TABLE `acct_contract`", "'usd-cpm-impression-v2'",
		"CREATE TABLE `acct_statement`", "CREATE TABLE `acct_adjustment`",
		"CREATE TABLE `acct_audit`", "acct_audit_immutable_update",
		"acct_statement_protected_update", "acct_statement_immutable_delete",
		"money_migration_evidence_immutable_update", "money_migration_evidence_immutable_delete",
	} {
		if !strings.Contains(schema, required) {
			t.Errorf("accounting schema is missing %q", required)
		}
	}
	for _, required := range []string{
		"CREATE TABLE `hosted_binding`", "CREATE TABLE `hosted_operation`",
		"CREATE TABLE `hosted_provider_object`", "CREATE TABLE `hosted_event`",
		"CREATE TABLE `hosted_reconciliation`", "CREATE TABLE `hosted_audit`",
		"hosted_operation_identity_immutable", "hosted_event_immutable_update",
		"hosted_reconciliation_identity_immutable", "hosted_reconciliation_no_delete",
		"hosted_audit_immutable_delete",
	} {
		if !strings.Contains(schema, required) {
			t.Errorf("hosted payment schema is missing %q", required)
		}
	}
	for _, prohibited := range []string{"card_number", "routing_number", "bank_account", "payment_method_details", "raw_payload", "raw_signature", "api_key"} {
		for _, table := range []string{"hosted_binding", "hosted_operation", "hosted_provider_object", "hosted_event", "hosted_reconciliation", "hosted_audit"} {
			if strings.Contains(strings.ToLower(schemaTableDefinition(schema, table)), prohibited) {
				t.Errorf("hosted table %s contains prohibited credential/payload field %q", table, prohibited)
			}
		}
	}
	for table, required := range map[string]string{
		"ledger_log": "UNIQUE KEY `timely` (`timely`)",
		"daily_log":  "UNIQUE KEY `daily` (`daily`)",
	} {
		if definition := schemaTableDefinition(schema, table); !strings.Contains(definition, required) {
			t.Errorf("%s is missing singleton-failover guard %q", table, required)
		}
	}
	for _, prohibited := range []string{"`cardnumber`", "`routing_number`", "`account_number`"} {
		if strings.Contains(schema, prohibited) {
			t.Errorf("active baseline still contains unsafe credential column %s", prohibited)
		}
	}
	for _, retired := range []string{"TRIGGER `trig_payment`", "VIEW `view_payment`"} {
		if strings.Contains(schema, retired) {
			t.Errorf("active baseline still contains retired funding object %q", retired)
		}
	}
	for _, table := range []string{"pay_alipay", "pay_cc", "pay_cheque", "pay_payment", "pay_wechat"} {
		definition := schemaTableDefinition(schema, table)
		if definition == "" {
			t.Errorf("missing inactive compatibility table %s", table)
			continue
		}
		for _, prohibited := range []string{"`cardnumber`", "`routing_number`", "`account_number`", "`accountype`", "`bank`", "`expire`", "`sender_name`", "`sender_id`", "`ip`"} {
			if strings.Contains(definition, prohibited) {
				t.Errorf("compatibility table %s still contains identity/credential field %s", table, prohibited)
			}
		}
	}
}

func schemaTableDefinition(schema, table string) string {
	start := strings.Index(schema, "CREATE TABLE `"+table+"` (")
	if start < 0 {
		return ""
	}
	rest := schema[start:]
	end := strings.Index(rest, ") ENGINE=")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

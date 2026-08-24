package main

import (
	"os"
	"strings"
	"testing"
)

func TestReportingSchemaIsScopedPseudonymousAndNonFinancial(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(data)
	delivery := schemaTableDefinition(schema, "report_delivery")
	for _, required := range []string{
		"`demand_source` enum('Local','Fallback','Always','MiddlemanUnknown')",
		"`country_id` int unsigned", "`device_type` tinyint unsigned",
		"`device_key` smallint unsigned GENERATED ALWAYS",
		"`inventory_environment` enum('Unknown','Web','App','CTV','DOOH','Other')",
		"`refresh_seconds` smallint unsigned", "report_delivery_refresh_chk",
		"`seller_type` enum('Unknown','Publisher','Intermediary')", "`seller_id` varchar(64)",
		"`dimension_hash` binary(32) NOT NULL",
		"`spend_usd` decimal(20,6)", "`revenue_usd` decimal(20,6)",
		"`cost_usd` decimal(20,6)", "`margin_usd` decimal(20,6)",
		"`accounting_version` varchar(32)", "report_delivery_interval_dimension",
	} {
		if !strings.Contains(delivery, required) {
			t.Errorf("report_delivery is missing %q", required)
		}
	}
	for _, prohibited := range []string{"email", "user_id", "device_id", "ip_address", "consent", "credential"} {
		if strings.Contains(delivery, prohibited) {
			t.Errorf("report_delivery contains prohibited identity field %q", prohibited)
		}
	}
	exposure := schemaTableDefinition(schema, "report_exposure")
	for _, required := range []string{"`subject_hash` binary(32) NOT NULL", "`expires_at` datetime(6) NOT NULL", "report_exposure_expiry", "report_exposure_subject", "report_exposure_immutable_update"} {
		if !strings.Contains(exposure+schema, required) {
			t.Errorf("report exposure contract is missing %q", required)
		}
	}
	experiment := schemaTableDefinition(schema, "report_experiment")
	for _, required := range []string{"`assignment_algorithm_version` smallint unsigned NOT NULL DEFAULT '1'", "report_experiment_algorithm_chk", "report_experiment_assignment_immutable_update", "`retention_hours` int unsigned NOT NULL", "report_experiment_retention_chk"} {
		if !strings.Contains(experiment+schema, required) {
			t.Errorf("report experiment retention contract is missing %q", required)
		}
	}
	for _, required := range []string{"report_experiment_variant_immutable_update", "report_experiment_variant_immutable_delete"} {
		if !strings.Contains(schema, required) {
			t.Errorf("report experiment compatibility contract is missing %q", required)
		}
	}
	for _, prohibited := range []string{"subject_id", "user_id", "email", "cookie", "ifa"} {
		if strings.Contains(exposure, prohibited) {
			t.Errorf("report_exposure contains raw identity field %q", prohibited)
		}
	}
	outcome := schemaTableDefinition(schema, "report_experiment_outcome")
	for _, required := range []string{
		"`exposure_id` bigint unsigned NOT NULL", "`metric_value` decimal(20,6) NOT NULL",
		"`idempotency_key` binary(32) NOT NULL", "report_experiment_outcome_idempotency",
		"report_experiment_outcome_immutable_update",
	} {
		if !strings.Contains(outcome+schema, required) {
			t.Errorf("report experiment outcome contract is missing %q", required)
		}
	}
	for _, prohibited := range []string{"subject_id", "user_id", "email", "cookie", "ifa", "subject_hash"} {
		if strings.Contains(outcome, prohibited) {
			t.Errorf("report_experiment_outcome contains identity field %q", prohibited)
		}
	}
}

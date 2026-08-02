package main

import (
	"os"
	"strings"
	"testing"
)

func TestActionMeasurementSchemaIsAnalyticalScopedAndExpiring(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(data)
	for _, table := range []string{"measurement_touch", "measurement_action"} {
		definition := schemaTableDefinition(schema, table)
		if definition == "" {
			t.Fatalf("missing %s table", table)
		}
		for _, required := range []string{"`expires_at` datetime(6) NOT NULL", "`privacy_mode`", "`privacy_reason`"} {
			if !strings.Contains(definition, required) {
				t.Errorf("%s is missing %q", table, required)
			}
		}
		for _, prohibited := range []string{"delivery_reservation", "acct_statement", "acct_adjustment", "raw_consent", "user_id", "device_id", "email"} {
			if strings.Contains(definition, prohibited) {
				t.Errorf("%s unexpectedly contains %q", table, prohibited)
			}
		}
	}
	action := schemaTableDefinition(schema, "measurement_action")
	for _, required := range []string{
		"UNIQUE KEY `measurement_action_adv_event` (`adv_id`,`event_id`)",
		"enum('conversion','purchase','download','video_complete','custom')",
		"enum('click','view','unattributed')",
		"`action_pseudonym` char(64)",
		"measurement_action_purchase_value_chk",
	} {
		if !strings.Contains(action, required) {
			t.Errorf("measurement_action is missing %q", required)
		}
	}
}

package main

import (
	"os"
	"strings"
	"testing"
)

func TestA03AuthoritativeMoneyColumnsAreExact(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(data)
	for table, fields := range map[string][]string{
		"adv_item":       {"`cost` decimal(12,6)"},
		"pub_slot":       {"`bidfloor` decimal(12,6)"},
		"adv_balance":    {"`limit_spend` decimal(20,9)", "`current_spend` decimal(20,9)"},
		"his_balance":    {"`budget_old` decimal(20,9)", "`budget_add` decimal(20,9)", "`budget_new` decimal(20,9)"},
		"ledger_log":     {"`spend` decimal(20,9)"},
		"ledger_adv":     {"`spend` decimal(20,9)"},
		"ledger_pub":     {"`spend` decimal(20,9)"},
		"ledger_pub_adv": {"`spend` decimal(20,9)"},
		"ledger_mid":     {"`charge_spend` decimal(20,9)", "`pay_spend` decimal(20,9)", "`margin_spend` decimal(20,9)"},
		"daily_log":      {"`spend` decimal(20,9)"},
		"daily_adv":      {"`spend` decimal(20,9)"},
		"daily_pub":      {"`spend` decimal(20,9)"},
		"daily_pub_adv":  {"`spend` decimal(20,9)"},
		"daily_mid":      {"`charge_spend` decimal(20,9)", "`pay_spend` decimal(20,9)", "`margin_spend` decimal(20,9)"},
	} {
		definition := schemaTableDefinition(schema, table)
		for _, field := range fields {
			if !strings.Contains(definition, field) {
				t.Errorf("%s is missing exact field %q", table, field)
			}
		}
		if strings.Contains(definition, " float") || strings.Contains(definition, " double") {
			t.Errorf("%s retains a binary floating-point monetary column", table)
		}
	}
	evidence := schemaTableDefinition(schema, "money_migration_evidence")
	for _, required := range []string{"`legacy_value` varchar(255)", "`converted_value` decimal(20,9)", "`conversion_rule` enum", "money_migration_source"} {
		if !strings.Contains(evidence, required) {
			t.Errorf("money migration evidence is missing %q", required)
		}
	}
}

package main

import (
	"os"
	"strings"
	"testing"
)

func TestAccountAddressSchemaSupportsCanonicalIPv6(t *testing.T) {
	data, err := os.ReadFile("step4_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	definition := schemaTableDefinition(string(data), "add_address")
	if !strings.Contains(definition, "`ip` varchar(45) DEFAULT NULL") {
		t.Fatalf("add_address.ip is not the reviewed IPv4/IPv6 text shape:\n%s", definition)
	}
	if strings.Contains(definition, "`ip` varchar(15)") {
		t.Fatal("add_address.ip retains the legacy IPv4-only width")
	}
}

func TestAccountAddressIPv6MigrationWidensOnlyLegacyShape(t *testing.T) {
	data, err := os.ReadFile("s06_account_address_ipv6_migration.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := strings.ToLower(string(data))
	for _, required := range []string{
		"character_maximum_length=15",
		"modify column ip varchar(45) default null",
	} {
		if !strings.Contains(migration, required) {
			t.Errorf("IPv6 migration is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"drop column", "delete from", "update add_address", "insert into add_address",
	} {
		if strings.Contains(migration, forbidden) {
			t.Errorf("IPv6 migration contains data mutation %q", forbidden)
		}
	}
}

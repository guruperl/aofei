package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDocumentedUserAgentDependencyMatchesModule(t *testing.T) {
	const current = "github.com/mileusna/useragent"
	const stale = "github.com/mssola/user_agent"
	for _, path := range []string{"../go.mod", "../memory-bank/tech-stack.md", "../advice/device.go", "../advice/type.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, current) {
			t.Errorf("%s does not name the active user-agent package %s", path, current)
		}
		if strings.Contains(text, stale) {
			t.Errorf("%s names retired user-agent package %s", path, stale)
		}
	}
}

func TestIdentityDocumentListsExactInteractiveAccountRoles(t *testing.T) {
	data, err := os.ReadFile("../docs/identity-access-security.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	start := strings.Index(text, "The exact interactive account-role inventory is:")
	end := strings.Index(text, "These five keys are the complete")
	if start < 0 || end <= start {
		t.Fatal("identity account-role inventory markers are missing or out of order")
	}
	rows := regexp.MustCompile(`(?m)^\| \x60([a-z]+)\x60 \| \x60([^\x60]+)\x60 \|`).FindAllStringSubmatch(text[start:end], -1)
	want := map[string]string{
		"admin":   "admin.admin_id",
		"adv":     "adv.adv_id",
		"pub":     "pub.pub_id",
		"agent":   "agent.agent_id",
		"analyst": "analyst.analyst_id",
	}
	if len(rows) != len(want) {
		t.Fatalf("documented account-role rows = %v, want exactly %v", rows, mapKeys(want))
	}
	for _, row := range rows {
		account, ok := want[row[1]]
		if !ok {
			t.Errorf("identity document lists unknown account role %q", row[1])
			continue
		}
		if row[2] != account {
			t.Errorf("identity role %s account = %q, want %q", row[1], row[2], account)
		}
		delete(want, row[1])
	}
	if len(want) != 0 {
		t.Fatalf("identity document is missing account roles %v", mapKeys(want))
	}
	for _, boundary := range []string{"`web` is", "`ssp` is", "`service` and", "`w8m_v1` management"} {
		if !strings.Contains(text, boundary) {
			t.Errorf("identity document does not distinguish non-account principal %s", boundary)
		}
	}
}

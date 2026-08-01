package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSummerExampleUsesBcryptPasswordHashField(t *testing.T) {
	data, err := os.ReadFile("summer.example.json")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(strings.ToUpper(text), "SHA1(") {
		t.Fatal("summer example must not use SHA1 password SQL")
	}
	if !strings.Contains(text, `"Password_hash": "passwd"`) {
		t.Fatal("summer example should verify stored password hashes through Password_hash")
	}

	var config struct {
		Roles map[string]struct {
			Issuers map[string]struct {
				PasswordHash string   `json:"Password_hash"`
				OutPars      []string `json:"OutPars"`
			} `json:"Issuers"`
		} `json:"Roles"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"adv":   {"adv_id", "a_email", "a_company", "a_contact", "a_timezone_id", "passwd"},
		"pub":   {"pub_id", "p_email", "p_company", "p_contact", "p_timezone_id", "passwd"},
		"agent": {"agent_id", "agent_login", "agent_level", "passwd"},
		"admin": {"admin_id", "admin_login", "passwd"},
	}
	for role, fields := range want {
		issuer := config.Roles[role].Issuers["db"]
		if issuer.PasswordHash != "passwd" {
			t.Errorf("%s db issuer Password_hash = %q, want passwd", role, issuer.PasswordHash)
		}
		if !reflect.DeepEqual(issuer.OutPars, fields) {
			t.Errorf("%s db issuer OutPars = %#v, want %#v", role, issuer.OutPars, fields)
		}
	}
}

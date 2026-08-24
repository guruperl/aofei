package main

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
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
		Identity struct {
			Enabled            bool
			KeyEnv             string
			RequiredTOTP       []string
			AuditRetentionDays int
		}
		Roles map[string]struct {
			Permissions  []string
			RequireGrant bool
			Issuers      map[string]struct {
				PasswordHash          string   `json:"Password_hash"`
				LegacyPasswordUpgrade bool     `json:"Legacy_password_upgrade"`
				OutPars               []string `json:"OutPars"`
			} `json:"Issuers"`
		} `json:"Roles"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"adv":     {"adv_id", "a_email", "a_company", "a_contact", "a_timezone_id", "passwd"},
		"pub":     {"pub_id", "p_email", "p_company", "p_contact", "p_timezone_id", "passwd"},
		"agent":   {"agent_id", "agent_login", "agent_level", "passwd"},
		"admin":   {"admin_id", "admin_login", "passwd"},
		"analyst": {"analyst_id", "analyst_login", "passwd"},
	}
	if len(config.Roles) != len(want) {
		t.Fatalf("summer example roles = %v, want exactly admin, adv, pub, agent, analyst", mapKeys(config.Roles))
	}
	for role, fields := range want {
		configured, ok := config.Roles[role]
		if !ok {
			t.Fatalf("summer example is missing account role %q", role)
		}
		issuer := configured.Issuers["db"]
		if issuer.PasswordHash != "passwd" {
			t.Errorf("%s db issuer Password_hash = %q, want passwd", role, issuer.PasswordHash)
		}
		if !issuer.LegacyPasswordUpgrade {
			t.Errorf("%s db issuer must atomically upgrade legacy passwords during migration", role)
		}
		if !reflect.DeepEqual(issuer.OutPars, fields) {
			t.Errorf("%s db issuer OutPars = %#v, want %#v", role, issuer.OutPars, fields)
		}
	}
	if config.Identity.Enabled {
		t.Fatal("example identity boundary must require an explicit production enablement step")
	}
	if config.Identity.KeyEnv != "GENELET_IDENTITY_KEY" || config.Identity.AuditRetentionDays < 365 {
		t.Fatalf("identity key/retention policy is incomplete: %#v", config.Identity)
	}
	if len(config.Roles["analyst"].Permissions) == 0 || !config.Roles["analyst"].RequireGrant {
		t.Fatal("analyst role must be permission-limited and grant-scoped")
	}
	totpRoles := make(map[string]bool, len(config.Identity.RequiredTOTP))
	for _, role := range config.Identity.RequiredTOTP {
		totpRoles[role] = true
	}
	if len(totpRoles) != len(want) {
		t.Fatalf("RequiredTOTP roles = %v, want every interactive account role", mapKeys(totpRoles))
	}
	for role := range want {
		if !totpRoles[role] {
			t.Errorf("RequiredTOTP is missing interactive account role %q", role)
		}
	}
}

func mapKeys[M ~map[string]V, V any](values M) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func TestAofeiExampleKeepsManagementAPIDisabledAndKeyOutOfJSON(t *testing.T) {
	data, err := os.ReadFile("aofei.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		ManagementAPI struct {
			Enabled bool   `json:"enabled"`
			KeyEnv  string `json:"key_env"`
		} `json:"management_api"`
		DirectSSPAuth struct {
			Enabled                   bool `json:"enabled"`
			RequestSkewSeconds        int  `json:"request_skew_seconds"`
			CredentialRefreshSeconds  int  `json:"credential_refresh_seconds"`
			CredentialMaxAgeSeconds   int  `json:"credential_max_age_seconds"`
			RotationMaxOverlapSeconds int  `json:"rotation_max_overlap_seconds"`
		} `json:"direct_ssp_auth"`
		TrafficQuality struct {
			Enabled                   bool   `json:"enabled"`
			DigestKeyEnv              string `json:"digest_key_env"`
			EnforcementRefreshSeconds int    `json:"enforcement_refresh_seconds"`
			EnforcementMaxAgeSeconds  int    `json:"enforcement_max_age_seconds"`
		} `json:"traffic_quality"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.ManagementAPI.Enabled || config.ManagementAPI.KeyEnv != "MANAGEMENT_API_HMAC_KEY" {
		t.Fatalf("management API must require explicit enablement and an environment key: %#v", config.ManagementAPI)
	}
	if strings.Contains(string(data), "w8m_v1_") {
		t.Fatal("management API example contains a bearer credential")
	}
	if config.DirectSSPAuth.Enabled || config.DirectSSPAuth.RequestSkewSeconds != 300 ||
		config.DirectSSPAuth.CredentialRefreshSeconds != 30 || config.DirectSSPAuth.CredentialMaxAgeSeconds != 120 ||
		config.DirectSSPAuth.RotationMaxOverlapSeconds != 86400 {
		t.Fatalf("direct SSP request authentication must remain default-off and bounded: %#v", config.DirectSSPAuth)
	}
	if strings.Contains(string(data), "w8m_pz_v1_") {
		t.Fatal("direct SSP example contains a private publisher credential")
	}
	if config.TrafficQuality.Enabled || config.TrafficQuality.DigestKeyEnv != "TRAFFIC_QUALITY_DIGEST_KEY" ||
		config.TrafficQuality.EnforcementRefreshSeconds != 30 || config.TrafficQuality.EnforcementMaxAgeSeconds != 120 {
		t.Fatalf("traffic quality must require explicit enablement, an environment key, and bounded snapshot ages: %#v", config.TrafficQuality)
	}
	if strings.Contains(string(data), `"traffic_quality_key"`) {
		t.Fatal("traffic-quality example contains key material")
	}
}

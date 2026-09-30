package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetiredAccountReleaseAdmission(t *testing.T) {
	cases := []struct {
		name                      string
		capable, enabled, retired bool
		routines                  int
		result                    string
		queryFailure              bool
		alterShape                bool
		wantPass                  bool
	}{
		{name: "verified retired profile", capable: true, enabled: true, retired: true, routines: 2, result: "2|2|0|10|5\n", wantPass: true},
		{name: "old release", enabled: true, retired: true, routines: 2, result: "2|2|0|10|5\n"},
		{name: "protection disabled", capable: true, retired: true, routines: 2, result: "2|2|0|10|5\n"},
		{name: "retirement disabled", capable: true, enabled: true, routines: 2, result: "2|2|0|10|5\n"},
		{name: "partial retirement", capable: true, enabled: true, retired: true, routines: 2, result: "2|2|1|10|5\n"},
		{name: "missing protected column", capable: true, enabled: true, retired: true, routines: 2, result: "2|2|0|9|5\n"},
		{name: "wrong retained routines", capable: true, enabled: true, retired: true, routines: 2, result: "2|1|0|10|5\n"},
		{name: "query failed", capable: true, enabled: true, retired: true, routines: 2, queryFailure: true},
		{name: "unknown routine profile", capable: true, enabled: true, retired: true, routines: 3, result: "2|2|0|10|5\n"},
		{name: "unrelated schema drift", capable: true, enabled: true, retired: true, routines: 2, result: "2|2|0|10|5\n", alterShape: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := prepareEngineFixture(t, true)
			f.engine.Environment.Database.Routines = tc.routines
			f.runner.environment = f.engine.Environment
			f.runner.retiredAccountResult = tc.result
			f.runner.failRetiredAccountQuery = tc.queryFailure
			var config map[string]any
			raw, err := os.ReadFile(f.engine.Environment.Paths.SummerConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(raw, &config); err != nil {
				t.Fatal(err)
			}
			config["AccountProtection"] = map[string]bool{"Enabled": tc.enabled, "PlaintextRetired": tc.retired}
			writeJSONFixture(t, f.engine.Environment.Paths.SummerConfig, config)
			manifest, err := loadReleaseManifest(filepath.Join(f.candidate, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			manifest.Contracts.Database.Routines = 6
			manifest.Contracts.SupportsAccountIdentifierRetirement = tc.capable
			if tc.alterShape {
				manifest.Contracts.Database.Tables++
			}
			rewriteRetiredReleaseFixture(t, f.candidate, manifest)
			// Exercise the public deployment preflight, including checksum and
			// strict manifest decoding, without activation or production data.
			_, err = f.engine.Preflight(context.Background(), f.candidate)
			if (err == nil) != tc.wantPass {
				t.Fatalf("admission error=%v wantPass=%v", err, tc.wantPass)
			}
			if f.runner.restarts != 0 || f.runner.starts != 0 || f.runner.stops != 0 {
				t.Fatal("preflight mutated service")
			}
			selected, err := filepath.EvalSymlinks(f.engine.Environment.Paths.CurrentLink)
			if err != nil || selected != f.prior {
				t.Fatalf("preflight changed selection: %s %v", selected, err)
			}
			entries, err := os.ReadDir(f.engine.HistoryDir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("preflight wrote history: %v %v", entries, err)
			}
		})
	}
}

func TestRetiredAccountCapabilityManifestCompatibility(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		m := testReleaseManifest(ReleaseSchemaVersion, ReleaseKind)
		m.Contracts.Database.Routines = 6
		m.Contracts.SupportsAccountIdentifierRetirement = true
		if legacy {
			m.SchemaVersion = 1
			m.Kind = LegacyReleaseKind
		}
		if (m.Validate() == nil) == legacy {
			t.Fatalf("retirement capability legacy=%v was misjudged", legacy)
		}
	}
	m := testReleaseManifest(ReleaseSchemaVersion, ReleaseKind)
	m.Contracts.SupportsAccountIdentifierRetirement = true
	if m.Validate() == nil {
		t.Fatal("capability with unknown baseline routine profile admitted")
	}
}

func rewriteRetiredReleaseFixture(t *testing.T, root string, manifest ReleaseManifest) {
	t.Helper()
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err = os.WriteFile(filepath.Join(root, "manifest.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "checksums.sha256")
	checksums, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(checksums), "\n")
	for i, line := range lines {
		if strings.HasSuffix(line, "  manifest.json") {
			lines[i] = fmt.Sprintf("%x  manifest.json", sha256.Sum256(raw))
		}
	}
	if err = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
}

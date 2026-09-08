package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testEnvironment(t *testing.T) Environment {
	t.Helper()
	root := t.TempDir()
	current := filepath.Join(root, "current")
	return Environment{
		SchemaVersion: EnvironmentSchemaVersion,
		Environment:   "example-production",
		HostFQDN:      "app.example.test",
		Operator:      Operator{User: "operator", UID: os.Geteuid()},
		Origins: Origins{
			Canonical: "https://app.example.test",
			Accepted:  []string{"https://app.example.test", "https://example.test"},
		},
		Paths: Paths{
			ReleaseRoot: filepath.Join(root, "releases"), CurrentLink: current,
			LockFile:            filepath.Join(root, "deploy.lock"),
			AofeiConfig:         filepath.Join(root, "config", "aofei.json"),
			SummerConfig:        filepath.Join(root, "config", "summer.json"),
			BootstrapBackupRoot: filepath.Join(root, "backups"),
		},
		Service: Service{
			Scope: "user", Name: "example.service", UnitPath: filepath.Join(root, "service.unit"),
			ExecStart:              filepath.Join(current, "bin", "unify"),
			WorkingDirectory:       filepath.Join(current, "assets", "pzdesign"),
			SecretEnvironmentFiles: []string{filepath.Join(root, "config", "secret.env")},
		},
		Health: Health{
			Attempts: 2, IntervalSeconds: 0,
			Origin: []Probe{
				{URL: "http://127.0.0.1:8080/healthz", Status: 204},
				{URL: "http://127.0.0.1:8080/readyz", Status: 204},
			},
			Public: []Probe{{URL: "https://app.example.test/", Status: 200}},
		},
		Dependencies: Dependencies{Docker: []DockerDependency{{
			Name: "example-db", ConfiguredImage: "database:stable",
			ImageID:          "sha256:" + strings.Repeat("a", 64),
			RepositoryDigest: "database@sha256:" + strings.Repeat("b", 64),
			PublishedPorts:   []PublishedPort{{Container: "3306/tcp", Host: "127.0.0.1:3307"}},
		}}},
		Database: Database{
			Container: "example-db", Name: "application", Tables: 10, Routines: 2, Triggers: 3,
			AuthoritativeMoneyColumns: 4, AccountingVersion: "money-v1",
		},
		Retention: Retention{MinimumReleases: 2, AutomaticDeletion: false},
	}
}

func TestEnvironmentValidationIsStrictAndTargetNeutral(t *testing.T) {
	environment := testEnvironment(t)
	if err := environment.Validate(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "environment.json")
	data, err := json.Marshal(environment)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data[:len(data)-1], []byte(`,"unexpected":true}`)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEnvironment(path); err == nil {
		t.Fatal("unknown environment field passed")
	}
}

func TestEnvironmentValidationRejectsEffectfulAmbiguity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Environment)
	}{
		{name: "unaccepted canonical origin", mutate: func(value *Environment) { value.Origins.Accepted = []string{"https://example.test"} }},
		{name: "external direct health", mutate: func(value *Environment) { value.Health.Origin[0].URL = "http://example.test/healthz" }},
		{name: "service bypasses selection", mutate: func(value *Environment) { value.Service.ExecStart = "/opt/application/unify" }},
		{name: "relative path", mutate: func(value *Environment) { value.Paths.LockFile = "deploy.lock" }},
		{name: "overlapping path", mutate: func(value *Environment) { value.Paths.BootstrapBackupRoot = filepath.Dir(value.Paths.AofeiConfig) }},
		{name: "automatic deletion", mutate: func(value *Environment) { value.Retention.AutomaticDeletion = true }},
		{name: "non-loopback Docker port", mutate: func(value *Environment) { value.Dependencies.Docker[0].PublishedPorts[0].Host = "0.0.0.0:3307" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := testEnvironment(t)
			test.mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("invalid environment passed")
			}
		})
	}
}

func TestReleaseManifestCompatibilityIsReadOldWriteCurrent(t *testing.T) {
	manifest := testReleaseManifest(ReleaseSchemaVersion, ReleaseKind)
	manifest.GoVersion = "go version go1.23.5 linux/amd64"
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	legacy := testReleaseManifest(1, LegacyReleaseKind)
	if err := legacy.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []ReleaseManifest{
		testReleaseManifest(1, ReleaseKind),
		testReleaseManifest(ReleaseSchemaVersion, LegacyReleaseKind),
		testReleaseManifest(3, "future-backend"),
	} {
		if err := changed.Validate(); err == nil {
			t.Fatalf("unsupported release passed: version=%d kind=%q", changed.SchemaVersion, changed.Kind)
		}
	}
}

func testReleaseManifest(version int, kind string) ReleaseManifest {
	aofei := strings.Repeat("a", 40)
	pzdesign := strings.Repeat("b", 40)
	genelet := strings.Repeat("c", 40)
	return ReleaseManifest{
		SchemaVersion: version, Kind: kind,
		ReleaseID: "aofei-" + aofei[:12] + "_pzdesign-" + pzdesign[:12] + "_genelet-" + genelet[:12],
		BuiltAt:   "2026-09-08T00:00:00Z", GoVersion: "go1.23.5",
		Sources: ReleaseSources{
			Aofei:    Source{Commit: aofei, Branch: "main", Upstream: "origin/main", Remote: "git@example.test:aofei.git"},
			Pzdesign: Source{Commit: pzdesign, Branch: "main", Upstream: "origin/main", Remote: "git@example.test:pzdesign.git"},
			Genelet:  Source{Commit: genelet, Branch: "main", Upstream: "origin/main", Remote: "git@example.test:genelet.git"},
		},
		Contracts: ReleaseContracts{Database: DatabaseShape{Tables: 10, Routines: 2, Triggers: 3}, AccountingVersion: "money-v1"},
		Assets:    ReleaseAssets{ComponentCount: 1}, ArtifactCount: 6,
	}
}

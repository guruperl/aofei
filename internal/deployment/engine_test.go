package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type deploymentRunner struct {
	environment Environment
	restarts    int
	failConfig  bool
}

func (runner *deploymentRunner) Run(_ context.Context, command string, arguments, _ []string) ([]byte, error) {
	switch command {
	case "hostname":
		return []byte(runner.environment.HostFQDN + "\n"), nil
	case "id":
		if len(arguments) == 1 && arguments[0] == "-un" {
			return []byte(runner.environment.Operator.User + "\n"), nil
		}
		if len(arguments) == 1 && arguments[0] == "-u" {
			return []byte(fmt.Sprint(runner.environment.Operator.UID) + "\n"), nil
		}
	case "docker":
		return runner.docker(arguments)
	case "systemctl":
		return runner.systemctl(arguments)
	default:
		if strings.HasSuffix(filepath.ToSlash(command), "/bin/config-preflight") {
			if runner.failConfig {
				return nil, errors.New("synthetic config failure")
			}
			return []byte("production_config_preflight=passed\n"), nil
		}
	}
	return nil, fmt.Errorf("unexpected command: %s %q", command, arguments)
}

func (runner *deploymentRunner) docker(arguments []string) ([]byte, error) {
	dependency := runner.environment.Dependencies.Docker[0]
	switch {
	case len(arguments) == 2 && arguments[0] == "inspect" && arguments[1] == dependency.Name:
		return []byte("{}\n"), nil
	case containsSequence(arguments, "{{.State.Running}}"):
		return []byte("true\n"), nil
	case containsSequence(arguments, "{{.Config.Image}}"):
		return []byte(dependency.ConfiguredImage + "\n"), nil
	case containsSequence(arguments, "{{.Image}}"):
		return []byte(dependency.ImageID + "\n"), nil
	case len(arguments) >= 2 && arguments[0] == "image" && arguments[1] == "inspect":
		data, _ := json.Marshal([]string{dependency.RepositoryDigest})
		return append(data, '\n'), nil
	case len(arguments) == 3 && arguments[0] == "port":
		return []byte(dependency.PublishedPorts[0].Host + "\n"), nil
	case len(arguments) > 2 && arguments[0] == "exec":
		database := runner.environment.Database
		return []byte(fmt.Sprintf("%d|%d|%d|%d|%s|%s|1|%d|%d\n",
			database.Tables, database.Views, database.Routines, database.Triggers,
			database.AccountingVersion, database.AccountingVersion,
			database.AuthoritativeMoneyColumns, database.AuthoritativeMoneyColumns)), nil
	}
	return nil, fmt.Errorf("unexpected Docker command: %q", arguments)
}

func (runner *deploymentRunner) systemctl(arguments []string) ([]byte, error) {
	if containsSequence(arguments, "restart") {
		runner.restarts++
		return nil, nil
	}
	if containsSequence(arguments, "daemon-reload") {
		return nil, nil
	}
	if containsSequence(arguments, "is-active") {
		return []byte("active\n"), nil
	}
	if containsSequence(arguments, "EnvironmentFiles") {
		var output strings.Builder
		for _, path := range runner.environment.Service.SecretEnvironmentFiles {
			fmt.Fprintf(&output, "%s (ignore_errors=no)\n", path)
		}
		return []byte(output.String()), nil
	}
	if containsSequence(arguments, "show") {
		return []byte(fmt.Sprintf("%d\n", 100+runner.restarts*100)), nil
	}
	return nil, fmt.Errorf("unexpected systemctl command: %q", arguments)
}

type selectionProber struct {
	current       string
	failingTarget string
}

type cancelingSelectionProber struct {
	selectionProber
	cancel   context.CancelFunc
	canceled bool
}

func (prober *cancelingSelectionProber) Status(ctx context.Context, rawURL string) (int, error) {
	selected, _ := filepath.EvalSymlinks(prober.current)
	if !prober.canceled && selected == prober.failingTarget {
		prober.canceled = true
		prober.cancel()
		return 0, context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return prober.selectionProber.Status(ctx, rawURL)
}

type unrecoveredSelectionProber struct {
	current       string
	failingTarget string
	poisoned      bool
}

func (prober *unrecoveredSelectionProber) Status(ctx context.Context, rawURL string) (int, error) {
	selected, _ := filepath.EvalSymlinks(prober.current)
	if selected == prober.failingTarget {
		prober.poisoned = true
	}
	if prober.poisoned {
		return 503, nil
	}
	return selectionProber{current: prober.current}.Status(ctx, rawURL)
}

func (prober selectionProber) Status(_ context.Context, rawURL string) (int, error) {
	if prober.failingTarget != "" {
		selected, _ := filepath.EvalSymlinks(prober.current)
		if selected == prober.failingTarget {
			return 503, nil
		}
	}
	if strings.Contains(rawURL, "/healthz") || strings.Contains(rawURL, "/readyz") {
		return 204, nil
	}
	return 200, nil
}

type engineFixture struct {
	engine    *Engine
	runner    *deploymentRunner
	prior     string
	candidate string
	output    *bytes.Buffer
}

func prepareEngineFixture(t *testing.T, current bool) engineFixture {
	t.Helper()
	environment := testEnvironment(t)
	t.Cleanup(func() { _ = makeTreeWritable(environment.Paths.ReleaseRoot) })
	for _, directory := range []string{
		filepath.Dir(environment.Paths.AofeiConfig), environment.Paths.ReleaseRoot,
		filepath.Dir(environment.Service.UnitPath), filepath.Join(filepath.Dir(environment.Paths.CurrentLink), "history"),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Dir(environment.Paths.CurrentLink), 0o700); err != nil {
		t.Fatal(err)
	}
	history := filepath.Join(filepath.Dir(environment.Paths.CurrentLink), "history")
	if err := os.Chmod(history, 0o700); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(environment.Paths.CurrentLink, "assets", "pzdesign")
	writeJSONFixture(t, environment.Paths.AofeiConfig, map[string]any{"document_root": filepath.Join(project, "www")})
	writeJSONFixture(t, environment.Paths.SummerConfig, map[string]any{
		"ProjectRoot": project, "Template": filepath.Join(project, "tmpls"), "DocumentRoot": filepath.Join(project, "www"),
	})
	if err := os.WriteFile(environment.Service.SecretEnvironmentFiles[0], []byte("SECRET_REFERENCE=fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	unit := []byte("[Service]\nWorkingDirectory=" + environment.Service.WorkingDirectory +
		"\nEnvironment=AOFEI=" + environment.Paths.AofeiConfig +
		"\nEnvironment=SUMMER=" + environment.Paths.SummerConfig +
		"\nExecStart=" + environment.Service.ExecStart + "\n")
	unitTemplate := filepath.Join(filepath.Dir(environment.Paths.CurrentLink), "unit-template.service")
	if err := os.WriteFile(unitTemplate, unit, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(environment.Service.UnitPath, unit, 0o644); err != nil {
		t.Fatal(err)
	}
	priorFixture, priorManifest, _ := writeReleaseFixtureWithIdentity(t, environment.Paths.ReleaseRoot, true, "abc")
	prior := filepath.Join(environment.Paths.ReleaseRoot, priorManifest.ReleaseID)
	if err := os.Rename(priorFixture, prior); err != nil {
		t.Fatal(err)
	}
	if current {
		if err := os.Symlink(prior, environment.Paths.CurrentLink); err != nil {
			t.Fatal(err)
		}
	}
	candidate, _, _ := writeReleaseFixtureWithIdentity(t, t.TempDir(), false, "def")
	engine, err := NewEngine(environment, unitTemplate, history)
	if err != nil {
		t.Fatal(err)
	}
	runner := &deploymentRunner{environment: environment}
	output := &bytes.Buffer{}
	engine.Runner = runner
	engine.Prober = selectionProber{current: environment.Paths.CurrentLink}
	engine.Inspector = fixtureInspector{}
	engine.Output = output
	engine.Now = func() time.Time { return time.Date(2026, 9, 8, 12, 0, 0, 123, time.UTC) }
	engine.Sleep = func(context.Context, time.Duration) error { return nil }
	return engineFixture{engine: engine, runner: runner, prior: prior, candidate: candidate, output: output}
}

func TestNewEngineRejectsTargetInputStateOverlap(t *testing.T) {
	environment := testEnvironment(t)
	if _, err := NewEngine(environment, environment.Paths.AofeiConfig, filepath.Join(t.TempDir(), "history")); err == nil {
		t.Fatal("unit template overlapping mutable config passed")
	}
}

func TestPreflightFailureDoesNotMutateSelectionReleaseOrHistory(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	fixture.runner.failConfig = true
	beforeSelection, err := os.Readlink(fixture.engine.Environment.Paths.CurrentLink)
	if err != nil {
		t.Fatal(err)
	}
	beforeReleases := directoryNames(t, fixture.engine.Environment.Paths.ReleaseRoot)
	beforeHistory := directoryNames(t, fixture.engine.HistoryDir)
	if _, err := fixture.engine.Preflight(context.Background(), fixture.candidate); err == nil {
		t.Fatal("failed application config preflight passed")
	}
	afterSelection, _ := os.Readlink(fixture.engine.Environment.Paths.CurrentLink)
	if afterSelection != beforeSelection || strings.Join(directoryNames(t, fixture.engine.Environment.Paths.ReleaseRoot), "|") != strings.Join(beforeReleases, "|") ||
		strings.Join(directoryNames(t, fixture.engine.HistoryDir), "|") != strings.Join(beforeHistory, "|") {
		t.Fatal("preflight failure mutated deployment state")
	}
}

func TestPreflightRejectsInstalledUnitAndPriorReleaseDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(engineFixture)
	}{
		{name: "unit", mutate: func(fixture engineFixture) {
			_ = os.WriteFile(fixture.engine.Environment.Service.UnitPath, []byte("[Service]\nExecStart=/changed\n"), 0o644)
		}},
		{name: "prior release", mutate: func(fixture engineFixture) {
			_ = makeTreeWritable(fixture.prior)
			_ = os.WriteFile(filepath.Join(fixture.prior, "assets", "pzdesign", "www", "index.html"), []byte("changed"), 0o644)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := prepareEngineFixture(t, true)
			test.mutate(fixture)
			if _, err := fixture.engine.Preflight(context.Background(), fixture.candidate); err == nil {
				t.Fatal("drifted rollback input passed preflight")
			}
		})
	}
}

func TestPreflightRejectsServiceEnvironmentFileDriftWithoutMutation(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	fixture.runner.environment.Service.SecretEnvironmentFiles = nil
	beforeSelection, err := os.Readlink(fixture.engine.Environment.Paths.CurrentLink)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.engine.Preflight(context.Background(), fixture.candidate); err == nil {
		t.Fatal("service environment-file drift passed preflight")
	}
	afterSelection, _ := os.Readlink(fixture.engine.Environment.Paths.CurrentLink)
	if beforeSelection != afterSelection || len(directoryNames(t, fixture.engine.HistoryDir)) != 0 {
		t.Fatal("environment-file preflight failure mutated deployment state")
	}
}

func TestBootstrapPreflightRejectsUnsafeLegacyUnitWithoutMutation(t *testing.T) {
	fixture := prepareEngineFixture(t, false)
	if err := os.Chmod(fixture.engine.Environment.Service.UnitPath, 0o664); err != nil {
		t.Fatal(err)
	}
	if err := fixture.engine.Bootstrap(context.Background(), fixture.candidate); err == nil {
		t.Fatal("peer-writable legacy unit passed bootstrap preflight")
	}
	if _, err := os.Lstat(fixture.engine.Environment.Paths.CurrentLink); !os.IsNotExist(err) ||
		len(directoryNames(t, fixture.engine.HistoryDir)) != 0 ||
		len(directoryNames(t, fixture.engine.Environment.Paths.BootstrapBackupRoot)) != 0 {
		t.Fatal("unsafe bootstrap preflight mutated deployment state")
	}
}

func TestDeployRejectsSymlinkLockBeforeSelectionMutation(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	target := filepath.Join(t.TempDir(), "lock-target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, fixture.engine.Environment.Paths.LockFile); err != nil {
		t.Fatal(err)
	}
	if err := fixture.engine.Deploy(context.Background(), fixture.candidate); err == nil {
		t.Fatal("symlink deployment lock passed")
	}
	selected, _ := filepath.EvalSymlinks(fixture.engine.Environment.Paths.CurrentLink)
	if selected != fixture.prior || len(directoryNames(t, fixture.engine.HistoryDir)) != 0 {
		t.Fatal("lock rejection mutated deployment state")
	}
}

func TestDeployFailureRestoresVerifiedPriorReleaseAndRecordsRollback(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	candidateManifest, err := loadReleaseManifest(filepath.Join(fixture.candidate, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidateTarget := filepath.Join(fixture.engine.Environment.Paths.ReleaseRoot, candidateManifest.ReleaseID)
	fixture.engine.Prober = selectionProber{current: fixture.engine.Environment.Paths.CurrentLink, failingTarget: candidateTarget}
	if err := fixture.engine.Deploy(context.Background(), fixture.candidate); err == nil || !strings.Contains(err.Error(), "restored the prior release") {
		t.Fatalf("deploy failure = %v", err)
	}
	selected, err := filepath.EvalSymlinks(fixture.engine.Environment.Paths.CurrentLink)
	if err != nil || selected != fixture.prior {
		t.Fatalf("selected release = %q, %v", selected, err)
	}
	record := readOnlyHistoryRecord(t, fixture.engine.HistoryDir)
	if record.Result != "rolled_back" || record.SelectedRelease != fixture.prior || record.Checks.RollbackRecovery != "passed" || fixture.runner.restarts != 2 {
		t.Fatalf("rollback record = %#v; restarts=%d", record, fixture.runner.restarts)
	}
}

func TestDeployCancellationStillRestoresPriorRelease(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	candidateManifest, err := loadReleaseManifest(filepath.Join(fixture.candidate, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	fixture.engine.Prober = &cancelingSelectionProber{
		selectionProber: selectionProber{
			current: fixture.engine.Environment.Paths.CurrentLink,
			failingTarget: filepath.Join(fixture.engine.Environment.Paths.ReleaseRoot,
				candidateManifest.ReleaseID),
		},
		cancel: cancel,
	}
	if err := fixture.engine.Deploy(ctx, fixture.candidate); err == nil || !strings.Contains(err.Error(), "restored the prior release") {
		t.Fatalf("deploy cancellation = %v", err)
	}
	selected, err := filepath.EvalSymlinks(fixture.engine.Environment.Paths.CurrentLink)
	if err != nil || selected != fixture.prior {
		t.Fatalf("selected release = %q, %v", selected, err)
	}
	record := readOnlyHistoryRecord(t, fixture.engine.HistoryDir)
	if record.Result != "rolled_back" || record.Checks.RollbackRecovery != "passed" {
		t.Fatalf("cancellation rollback record = %#v", record)
	}
}

func TestDeployRecordsUnrecoveredRollbackFailure(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	candidateManifest, err := loadReleaseManifest(filepath.Join(fixture.candidate, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixture.engine.Prober = &unrecoveredSelectionProber{
		current: fixture.engine.Environment.Paths.CurrentLink,
		failingTarget: filepath.Join(fixture.engine.Environment.Paths.ReleaseRoot,
			candidateManifest.ReleaseID),
	}
	if err := fixture.engine.Deploy(context.Background(), fixture.candidate); err == nil || !strings.Contains(err.Error(), "prior release did not recover") {
		t.Fatalf("deploy failure = %v", err)
	}
	record := readOnlyHistoryRecord(t, fixture.engine.HistoryDir)
	if record.Result != "rollback_failed" || record.Checks.RollbackRecovery != "failed" || record.CompletedAt == "" {
		t.Fatalf("failed rollback record = %#v", record)
	}
}

func TestDeploySuccessSelectsImmutableReleaseAndRecordsHealth(t *testing.T) {
	fixture := prepareEngineFixture(t, true)
	if err := fixture.engine.Deploy(context.Background(), fixture.candidate); err != nil {
		t.Fatal(err)
	}
	selected, err := filepath.EvalSymlinks(fixture.engine.Environment.Paths.CurrentLink)
	if err != nil || selected == fixture.prior {
		t.Fatalf("selected release = %q, %v", selected, err)
	}
	info, err := os.Stat(selected)
	if err != nil || info.Mode().Perm()&0o222 != 0 {
		t.Fatal("selected release is not immutable")
	}
	record := readOnlyHistoryRecord(t, fixture.engine.HistoryDir)
	if record.Result != "succeeded" || record.SelectedRelease != selected || record.Checks.Activation != "passed" {
		t.Fatalf("success record = %#v", record)
	}
}

func TestBootstrapProjectsConfigsAndInstallsTargetUnit(t *testing.T) {
	fixture := prepareEngineFixture(t, false)
	legacyUnit := []byte("[Service]\nExecStart=/legacy/unify\n")
	if err := os.WriteFile(fixture.engine.Environment.Service.UnitPath, legacyUnit, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fixture.engine.Bootstrap(context.Background(), fixture.candidate); err != nil {
		t.Fatal(err)
	}
	selected, err := filepath.EvalSymlinks(fixture.engine.Environment.Paths.CurrentLink)
	if err != nil || selected == "" {
		t.Fatal("bootstrap did not select a release")
	}
	installed, _ := os.ReadFile(fixture.engine.Environment.Service.UnitPath)
	template, _ := os.ReadFile(fixture.engine.UnitTemplate)
	if !bytes.Equal(installed, template) {
		t.Fatal("bootstrap did not install the target unit")
	}
	if err := fixture.engine.verifyReleaseConfigPaths(); err != nil {
		t.Fatal(err)
	}
	backups := directoryNames(t, fixture.engine.Environment.Paths.BootstrapBackupRoot)
	if len(backups) != 1 {
		t.Fatalf("bootstrap backups = %v", backups)
	}
}

func TestBootstrapFailureRestoresLegacyInputsAndRecordsRollback(t *testing.T) {
	fixture := prepareEngineFixture(t, false)
	legacyUnit := []byte("[Service]\nExecStart=/legacy/unify\n")
	if err := os.WriteFile(fixture.engine.Environment.Service.UnitPath, legacyUnit, 0o644); err != nil {
		t.Fatal(err)
	}
	aofeiBefore, _ := os.ReadFile(fixture.engine.Environment.Paths.AofeiConfig)
	summerBefore, _ := os.ReadFile(fixture.engine.Environment.Paths.SummerConfig)
	candidateManifest, err := loadReleaseManifest(filepath.Join(fixture.candidate, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixture.engine.Prober = selectionProber{
		current:       fixture.engine.Environment.Paths.CurrentLink,
		failingTarget: filepath.Join(fixture.engine.Environment.Paths.ReleaseRoot, candidateManifest.ReleaseID),
	}
	if err := fixture.engine.Bootstrap(context.Background(), fixture.candidate); err == nil || !strings.Contains(err.Error(), "restored the legacy service") {
		t.Fatalf("bootstrap failure = %v", err)
	}
	if _, err := os.Lstat(fixture.engine.Environment.Paths.CurrentLink); !os.IsNotExist(err) {
		t.Fatal("failed bootstrap retained a current selection")
	}
	aofeiAfter, _ := os.ReadFile(fixture.engine.Environment.Paths.AofeiConfig)
	summerAfter, _ := os.ReadFile(fixture.engine.Environment.Paths.SummerConfig)
	unitAfter, _ := os.ReadFile(fixture.engine.Environment.Service.UnitPath)
	if !bytes.Equal(aofeiBefore, aofeiAfter) || !bytes.Equal(summerBefore, summerAfter) || !bytes.Equal(legacyUnit, unitAfter) {
		t.Fatal("failed bootstrap did not restore legacy inputs")
	}
	record := readOnlyHistoryRecord(t, fixture.engine.HistoryDir)
	if record.Result != "rolled_back" || record.SelectedRelease != "legacy-direct-binary" || record.Checks.RollbackRecovery != "passed" {
		t.Fatalf("bootstrap rollback record = %#v", record)
	}
}

func writeJSONFixture(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func directoryNames(t *testing.T, path string) []string {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	result := make([]string, len(entries))
	for index, entry := range entries {
		result[index] = entry.Name()
	}
	return result
}

func readOnlyHistoryRecord(t *testing.T, history string) historyRecord {
	t.Helper()
	entries, err := os.ReadDir(history)
	if err != nil || len(entries) != 1 {
		t.Fatalf("history entries = %d, %v", len(entries), err)
	}
	var record historyRecord
	if err := decodeStrictFile(filepath.Join(history, entries[0].Name()), &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func containsSequence(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

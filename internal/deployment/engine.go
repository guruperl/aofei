package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type CommandRunner interface {
	Run(context.Context, string, []string, []string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command string, arguments, environment []string) ([]byte, error) {
	process := exec.CommandContext(ctx, command, arguments...)
	process.Env = append(os.Environ(), environment...)
	return process.Output()
}

type ProbeClient interface {
	Status(context.Context, string) (int, error)
}

type HTTPProbeClient struct {
	Client *http.Client
}

func (client HTTPProbeClient) Status(ctx context.Context, rawURL string) (int, error) {
	httpClient := client.Client
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout:   5 * time.Second,
			Transport: &http.Transport{Proxy: nil},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Cache-Control", "no-cache")
	response, err := httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	return response.StatusCode, nil
}

type Engine struct {
	Environment  Environment
	UnitTemplate string
	HistoryDir   string
	Runner       CommandRunner
	Prober       ProbeClient
	Inspector    ProvenanceInspector
	Output       io.Writer
	Now          func() time.Time
	Sleep        func(context.Context, time.Duration) error
}

func NewEngine(environment Environment, unitTemplate, historyDir string) (*Engine, error) {
	if err := environment.Validate(); err != nil {
		return nil, err
	}
	for _, path := range []string{unitTemplate, historyDir} {
		if err := validateAbsolutePath(path); err != nil {
			return nil, err
		}
	}
	if pathsOverlap(unitTemplate, historyDir) || pathsOverlap(environment.Paths.ReleaseRoot, unitTemplate) ||
		pathsOverlap(environment.Paths.ReleaseRoot, historyDir) {
		return nil, fmt.Errorf("engine input paths collide with deployment state")
	}
	engine := &Engine{
		Environment:  environment,
		UnitTemplate: unitTemplate,
		HistoryDir:   historyDir,
		Runner:       ExecRunner{},
		Prober:       HTTPProbeClient{},
		Inspector:    BuildInfoInspector{},
		Output:       os.Stdout,
		Now:          time.Now,
		Sleep: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
	return engine, nil
}

func (engine *Engine) Preflight(ctx context.Context, bundle string) (VerifiedRelease, error) {
	if err := engine.ready(); err != nil {
		return VerifiedRelease{}, err
	}
	release, err := VerifyRelease(bundle, engine.Inspector)
	if err != nil {
		return VerifiedRelease{}, err
	}
	if err := engine.verifyTargetInputs(ctx, release); err != nil {
		return VerifiedRelease{}, err
	}
	if _, err := fmt.Fprintf(engine.Output, "environment_preflight=passed release_id=%s\n", release.Manifest.ReleaseID); err != nil {
		return VerifiedRelease{}, err
	}
	return release, nil
}

func (engine *Engine) Status(ctx context.Context) error {
	if err := engine.ready(); err != nil {
		return err
	}
	if err := engine.verifyOperatorAndHost(ctx); err != nil {
		return err
	}
	selected := "none"
	if _, err := os.Lstat(engine.Environment.Paths.CurrentLink); err == nil {
		release, err := engine.selectedRelease()
		if err != nil {
			return err
		}
		selected = release.Root
	} else if !os.IsNotExist(err) {
		return err
	}
	active, err := engine.run(ctx, "systemctl", "--user", "is-active", engine.Environment.Service.Name)
	if err != nil || strings.TrimSpace(string(active)) != "active" {
		return fmt.Errorf("service is not active")
	}
	pid, err := engine.servicePID(ctx)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(engine.Output, "environment=%s\nservice=%s\npid=%d\nselected_release=%s\n",
		engine.Environment.Environment, strings.TrimSpace(string(active)), pid, selected); err != nil {
		return err
	}
	if err := engine.waitForProbes(ctx, engine.Environment.Health.Origin); err != nil {
		return fmt.Errorf("origin health failed")
	}
	if err := engine.probeOnce(ctx, engine.Environment.Health.Public); err != nil {
		return fmt.Errorf("public smoke failed")
	}
	_, err = fmt.Fprintln(engine.Output, "origin_health=passed\npublic_smoke=passed")
	return err
}

func (engine *Engine) Deploy(ctx context.Context, bundle string) error {
	if _, err := engine.Preflight(ctx, bundle); err != nil {
		return err
	}
	if _, err := engine.selectedRelease(); err != nil {
		return fmt.Errorf("deploy requires a verified current release: %w", err)
	}
	return engine.withLock(func() error {
		candidate, err := engine.Preflight(ctx, bundle)
		if err != nil {
			return err
		}
		previous, err := engine.selectedRelease()
		if err != nil {
			return fmt.Errorf("current release is not a verified rollback target: %w", err)
		}
		if err := engine.requireInstalledUnit(); err != nil {
			return err
		}
		target, err := engine.installRelease(candidate)
		if err != nil {
			return err
		}
		if previous.Root == target.Root {
			_, err := fmt.Fprintf(engine.Output, "deployment=noop release_id=%s\n", target.Manifest.ReleaseID)
			return err
		}
		oldPID, err := engine.servicePID(ctx)
		if err != nil {
			return err
		}
		record, err := engine.beginHistory(target, previous.Root, oldPID)
		if err != nil {
			return err
		}
		if err := engine.switchCurrent(target.Root); err != nil {
			return engine.rollbackDeploy(record, previous, oldPID, err)
		}
		newPID, activationErr := engine.restartAndVerify(ctx, oldPID)
		if activationErr == nil {
			if err := engine.finishHistory(record, "succeeded", target.Root, newPID, "passed", "not_required"); err != nil {
				return err
			}
			_, err = fmt.Fprintf(engine.Output, "deployment=passed release_id=%s pid=%d\n", target.Manifest.ReleaseID, newPID)
			return err
		}
		return engine.rollbackDeploy(record, previous, oldPID, activationErr)
	})
}

func (engine *Engine) Bootstrap(ctx context.Context, bundle string) error {
	if _, err := engine.Preflight(ctx, bundle); err != nil {
		return err
	}
	if _, err := os.Lstat(engine.Environment.Paths.CurrentLink); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("bootstrap requires an absent current symlink")
	}
	return engine.withLock(func() error {
		candidate, err := engine.Preflight(ctx, bundle)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(engine.Environment.Paths.CurrentLink); err == nil || !os.IsNotExist(err) {
			return fmt.Errorf("bootstrap requires an absent current symlink")
		}
		active, err := engine.run(ctx, "systemctl", "--user", "is-active", engine.Environment.Service.Name)
		if err != nil || strings.TrimSpace(string(active)) != "active" {
			return fmt.Errorf("legacy service is not active")
		}
		if err := engine.waitForProbes(ctx, engine.Environment.Health.Origin); err != nil {
			return fmt.Errorf("legacy service is not healthy")
		}
		oldPID, err := engine.servicePID(ctx)
		if err != nil {
			return err
		}
		target, err := engine.installRelease(candidate)
		if err != nil {
			return err
		}
		backup, err := engine.backupBootstrapInputs()
		if err != nil {
			return err
		}
		record, err := engine.beginHistory(target, "legacy-direct-binary", oldPID)
		if err != nil {
			return err
		}
		if err := engine.switchCurrent(target.Root); err != nil {
			return engine.rollbackBootstrap(backup, record, oldPID, err)
		}
		if err := engine.writeReleaseConfigs(); err != nil {
			return engine.rollbackBootstrap(backup, record, oldPID, err)
		}
		if err := copyRegular(engine.UnitTemplate, engine.Environment.Service.UnitPath, 0o644); err != nil {
			return engine.rollbackBootstrap(backup, record, oldPID, err)
		}
		if _, err := engine.run(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return engine.rollbackBootstrap(backup, record, oldPID, err)
		}
		newPID, err := engine.restartAndVerify(ctx, oldPID)
		if err != nil {
			return engine.rollbackBootstrap(backup, record, oldPID, err)
		}
		if err := engine.finishHistory(record, "succeeded", target.Root, newPID, "passed", "not_required"); err != nil {
			return err
		}
		_, err = fmt.Fprintf(engine.Output, "deployment=passed release_id=%s pid=%d\n", target.Manifest.ReleaseID, newPID)
		return err
	})
}

func (engine *Engine) rollbackDeploy(record *historyRecord, previous VerifiedRelease, oldPID int, cause error) error {
	recoveryCtx, cancelRecovery := engine.recoveryContext()
	defer cancelRecovery()
	if err := engine.switchCurrent(previous.Root); err != nil {
		historyErr := engine.finishFailedHistory(record, "unknown", "rollback_failed")
		return errors.Join(fmt.Errorf("release activation failed: %w", cause), fmt.Errorf("rollback selection failed: %w", err), historyErr)
	}
	rollbackPID, err := engine.restartAndVerify(recoveryCtx, oldPID)
	if err != nil {
		historyErr := engine.finishFailedHistory(record, previous.Root, "rollback_failed")
		return errors.Join(fmt.Errorf("release activation failed: %w", cause), fmt.Errorf("prior release did not recover: %w", err), historyErr)
	}
	if err := engine.finishHistory(record, "rolled_back", previous.Root, rollbackPID, "failed", "passed"); err != nil {
		return errors.Join(fmt.Errorf("release activation failed: %w", cause), err)
	}
	return fmt.Errorf("release activation failed and restored the prior release: %w", cause)
}

func (engine *Engine) rollbackBootstrap(backup string, record *historyRecord, oldPID int, cause error) error {
	recoveryCtx, cancelRecovery := engine.recoveryContext()
	defer cancelRecovery()
	var rollbackErrors []error
	for _, item := range []struct {
		source      string
		destination string
		mode        os.FileMode
	}{
		{filepath.Join(backup, "aofei.json"), engine.Environment.Paths.AofeiConfig, 0o600},
		{filepath.Join(backup, "summer.json"), engine.Environment.Paths.SummerConfig, 0o600},
		{filepath.Join(backup, "service.unit"), engine.Environment.Service.UnitPath, 0o644},
	} {
		if err := copyRegular(item.source, item.destination, item.mode); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	selection := "legacy-direct-binary"
	if err := engine.removeCurrent(); err != nil {
		selection = "unknown"
		rollbackErrors = append(rollbackErrors, err)
	}
	if _, err := engine.run(recoveryCtx, "systemctl", "--user", "daemon-reload"); err != nil {
		rollbackErrors = append(rollbackErrors, err)
	}
	rollbackPID, recoveryErr := engine.restartAndVerify(recoveryCtx, oldPID)
	if recoveryErr != nil {
		rollbackErrors = append(rollbackErrors, recoveryErr)
	}
	if len(rollbackErrors) > 0 {
		historyErr := engine.finishFailedHistory(record, selection, "rollback_failed")
		return errors.Join(fmt.Errorf("bootstrap activation failed: %w", cause), fmt.Errorf("legacy rollback failed: %w", errors.Join(rollbackErrors...)), historyErr)
	}
	if err := engine.finishHistory(record, "rolled_back", selection, rollbackPID, "failed", "passed"); err != nil {
		return errors.Join(fmt.Errorf("bootstrap activation failed: %w", cause), err)
	}
	return fmt.Errorf("bootstrap activation failed and restored the legacy service: %w", cause)
}

func (engine *Engine) ready() error {
	if engine.Runner == nil || engine.Prober == nil || engine.Inspector == nil || engine.Output == nil || engine.Now == nil || engine.Sleep == nil {
		return fmt.Errorf("deployment engine dependencies are incomplete")
	}
	if err := engine.Environment.Validate(); err != nil {
		return err
	}
	if err := validateRegularFile(engine.UnitTemplate, -1, false); err != nil {
		return fmt.Errorf("unit template: %w", err)
	}
	if info, err := os.Stat(engine.UnitTemplate); err != nil || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("unit template is writable by peers")
	}
	if err := engine.validateUnitTemplate(); err != nil {
		return err
	}
	if err := validateOwnedDirectory(engine.HistoryDir, engine.Environment.Operator.UID); err != nil {
		return fmt.Errorf("history directory: %w", err)
	}
	return nil
}

func (engine *Engine) verifyTargetInputs(ctx context.Context, release VerifiedRelease) error {
	if err := engine.verifyOperatorAndHost(ctx); err != nil {
		return err
	}
	if err := engine.verifyOwnerFiles(); err != nil {
		return err
	}
	if err := engine.verifyDockerDependencies(ctx); err != nil {
		return err
	}
	if err := engine.verifyDatabaseContract(ctx); err != nil {
		return err
	}
	if err := engine.verifyReleaseContract(release.Manifest); err != nil {
		return err
	}
	if _, err := engine.run(ctx, filepath.Join(release.Root, "bin", "config-preflight"), "-s", engine.Environment.Paths.AofeiConfig); err != nil {
		return fmt.Errorf("application config preflight failed")
	}
	if err := engine.requireSafeUnit(); err != nil {
		return err
	}
	if _, err := os.Lstat(engine.Environment.Paths.CurrentLink); err == nil {
		if _, err := engine.selectedRelease(); err != nil {
			return fmt.Errorf("current release: %w", err)
		}
		if err := engine.verifyReleaseConfigPaths(); err != nil {
			return err
		}
		if err := engine.requireInstalledUnit(); err != nil {
			return err
		}
		if err := engine.verifyActiveService(ctx); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (engine *Engine) verifyActiveService(ctx context.Context) error {
	active, err := engine.run(ctx, "systemctl", "--user", "is-active", engine.Environment.Service.Name)
	if err != nil || strings.TrimSpace(string(active)) != "active" {
		return fmt.Errorf("current service is not active")
	}
	if _, err := engine.servicePID(ctx); err != nil {
		return err
	}
	if err := engine.waitForProbes(ctx, engine.Environment.Health.Origin); err != nil {
		return fmt.Errorf("current origin is not healthy")
	}
	if err := engine.probeOnce(ctx, engine.Environment.Health.Public); err != nil {
		return fmt.Errorf("current public smoke failed")
	}
	return nil
}

func (engine *Engine) verifyOperatorAndHost(ctx context.Context) error {
	host, err := engine.run(ctx, "hostname", "-f")
	if err != nil || strings.TrimSpace(string(host)) != engine.Environment.HostFQDN {
		return fmt.Errorf("wrong deployment host")
	}
	name, err := engine.run(ctx, "id", "-un")
	if err != nil || strings.TrimSpace(string(name)) != engine.Environment.Operator.User {
		return fmt.Errorf("wrong deployment user")
	}
	uid, err := engine.run(ctx, "id", "-u")
	if err != nil || strings.TrimSpace(string(uid)) != strconv.Itoa(engine.Environment.Operator.UID) {
		return fmt.Errorf("wrong deployment uid")
	}
	return nil
}

func (engine *Engine) verifyOwnerFiles() error {
	paths := append([]string{engine.Environment.Paths.AofeiConfig, engine.Environment.Paths.SummerConfig}, engine.Environment.Service.SecretEnvironmentFiles...)
	for _, path := range paths {
		if err := validateRegularFile(path, engine.Environment.Operator.UID, true); err != nil {
			return fmt.Errorf("owner-managed file: %w", err)
		}
	}
	for _, path := range []string{engine.Environment.Paths.AofeiConfig, engine.Environment.Paths.SummerConfig} {
		var value map[string]any
		if err := decodeStrictFile(path, &value); err != nil {
			return fmt.Errorf("runtime config is not valid JSON")
		}
	}
	return nil
}

func (engine *Engine) verifyDockerDependencies(ctx context.Context) error {
	for _, dependency := range engine.Environment.Dependencies.Docker {
		if _, err := engine.run(ctx, "docker", "inspect", dependency.Name); err != nil {
			return fmt.Errorf("Docker dependency is missing: %s", dependency.Name)
		}
		running, err := engine.run(ctx, "docker", "inspect", "-f", "{{.State.Running}}", dependency.Name)
		if err != nil || strings.TrimSpace(string(running)) != "true" {
			return fmt.Errorf("Docker dependency is not running: %s", dependency.Name)
		}
		image, err := engine.run(ctx, "docker", "inspect", "-f", "{{.Config.Image}}", dependency.Name)
		if err != nil || strings.TrimSpace(string(image)) != dependency.ConfiguredImage {
			return fmt.Errorf("Docker configured image drifted: %s", dependency.Name)
		}
		imageID, err := engine.run(ctx, "docker", "inspect", "-f", "{{.Image}}", dependency.Name)
		if err != nil || strings.TrimSpace(string(imageID)) != dependency.ImageID {
			return fmt.Errorf("Docker image id drifted: %s", dependency.Name)
		}
		digests, err := engine.run(ctx, "docker", "image", "inspect", "--format", "{{json .RepoDigests}}", dependency.ConfiguredImage)
		var values []string
		if err != nil || json.Unmarshal(bytes.TrimSpace(digests), &values) != nil || !contains(values, dependency.RepositoryDigest) {
			return fmt.Errorf("Docker repository digest is unavailable: %s", dependency.Name)
		}
		for _, port := range dependency.PublishedPorts {
			actual, err := engine.run(ctx, "docker", "port", dependency.Name, port.Container)
			if err != nil || strings.TrimSpace(string(actual)) != port.Host {
				return fmt.Errorf("Docker published port drifted: %s", dependency.Name)
			}
		}
	}
	return nil
}

const databaseContractSQL = `SELECT CONCAT(
    (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type="BASE TABLE"),"|",
    (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type="VIEW"),"|",
    (SELECT COUNT(*) FROM information_schema.routines WHERE routine_schema=DATABASE()),"|",
    (SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema=DATABASE()),"|",
    COALESCE((SELECT MAX(unit_version) FROM acct_contract WHERE contract_id=1),"missing"),"|",
    COALESCE((SELECT column_default FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name="report_delivery" AND column_name="accounting_version"),"missing"),"|",
    (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name="money_migration_evidence"),"|",
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND ((table_name="adv_item" AND column_name="cost") OR (table_name="pub_slot" AND column_name="bidfloor") OR (table_name="adv_balance" AND column_name IN ("limit_spend","current_spend")) OR (table_name="his_balance" AND column_name IN ("budget_old","budget_add","budget_new")) OR (table_name IN ("ledger_log","ledger_adv","ledger_pub","ledger_pub_adv","daily_log","daily_adv","daily_pub","daily_pub_adv") AND column_name="spend") OR (table_name IN ("ledger_mid","daily_mid") AND column_name IN ("charge_spend","pay_spend","margin_spend")) OR (table_name IN ("mid_route_group","mid_route_bidder") AND column_name="min_margin_cpm"))),"|",
    (SELECT SUM(data_type="decimal") FROM information_schema.columns WHERE table_schema=DATABASE() AND ((table_name="adv_item" AND column_name="cost") OR (table_name="pub_slot" AND column_name="bidfloor") OR (table_name="adv_balance" AND column_name IN ("limit_spend","current_spend")) OR (table_name="his_balance" AND column_name IN ("budget_old","budget_add","budget_new")) OR (table_name IN ("ledger_log","ledger_adv","ledger_pub","ledger_pub_adv","daily_log","daily_adv","daily_pub","daily_pub_adv") AND column_name="spend") OR (table_name IN ("ledger_mid","daily_mid") AND column_name IN ("charge_spend","pay_spend","margin_spend")) OR (table_name IN ("mid_route_group","mid_route_bidder") AND column_name="min_margin_cpm")))
  );`

func (engine *Engine) verifyDatabaseContract(ctx context.Context) error {
	database := engine.Environment.Database
	output, err := engine.run(ctx, "docker", "exec", database.Container, "sh", "-c",
		`MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql --batch --skip-column-names -uroot "$1" -e "$2"`, "sh", database.Name, databaseContractSQL)
	if err != nil {
		return fmt.Errorf("database contract query failed")
	}
	expected := fmt.Sprintf("%d|%d|%d|%d|%s|%s|1|%d|%d",
		database.Tables, database.Views, database.Routines, database.Triggers,
		database.AccountingVersion, database.AccountingVersion,
		database.AuthoritativeMoneyColumns, database.AuthoritativeMoneyColumns)
	if strings.TrimSpace(string(output)) != expected {
		return fmt.Errorf("database contract drifted")
	}
	return nil
}

func (engine *Engine) verifyReleaseContract(manifest ReleaseManifest) error {
	database := engine.Environment.Database
	shape := manifest.Contracts.Database
	if shape.Tables != database.Tables || shape.Views != database.Views ||
		shape.Routines != database.Routines || shape.Triggers != database.Triggers ||
		manifest.Contracts.AccountingVersion != database.AccountingVersion {
		return fmt.Errorf("release contract does not match the environment")
	}
	return nil
}

func (engine *Engine) verifyReleaseConfigPaths() error {
	project := filepath.Join(engine.Environment.Paths.CurrentLink, "assets", "pzdesign")
	var aofei map[string]any
	if err := decodeStrictFile(engine.Environment.Paths.AofeiConfig, &aofei); err != nil || aofei["document_root"] != filepath.Join(project, "www") {
		return fmt.Errorf("AOFEI static root does not point at the atomic current release")
	}
	var summer map[string]any
	if err := decodeStrictFile(engine.Environment.Paths.SummerConfig, &summer); err != nil ||
		summer["ProjectRoot"] != project || summer["Template"] != filepath.Join(project, "tmpls") || summer["DocumentRoot"] != filepath.Join(project, "www") {
		return fmt.Errorf("SUMMER assets do not point at the atomic current release")
	}
	return nil
}

func (engine *Engine) selectedRelease() (VerifiedRelease, error) {
	info, err := os.Lstat(engine.Environment.Paths.CurrentLink)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return VerifiedRelease{}, fmt.Errorf("current release path is not a symlink")
	}
	resolved, err := filepath.EvalSymlinks(engine.Environment.Paths.CurrentLink)
	if err != nil {
		return VerifiedRelease{}, err
	}
	root := filepath.Clean(engine.Environment.Paths.ReleaseRoot)
	if filepath.Dir(resolved) != root || !releaseIDPattern.MatchString(filepath.Base(resolved)) {
		return VerifiedRelease{}, fmt.Errorf("selected release is outside the release root")
	}
	return VerifyRelease(resolved, engine.Inspector)
}

func (engine *Engine) requireInstalledUnit() error {
	if err := engine.requireSafeUnit(); err != nil {
		return err
	}
	template, err := os.ReadFile(engine.UnitTemplate)
	if err != nil {
		return err
	}
	installed, err := os.ReadFile(engine.Environment.Service.UnitPath)
	if err != nil || !bytes.Equal(template, installed) {
		return fmt.Errorf("installed unit differs from its target template")
	}
	return nil
}

func (engine *Engine) requireSafeUnit() error {
	if err := validateRegularFile(engine.Environment.Service.UnitPath, engine.Environment.Operator.UID, false); err != nil {
		return fmt.Errorf("installed unit: %w", err)
	}
	if info, err := os.Stat(engine.Environment.Service.UnitPath); err != nil || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("installed unit is writable by peers")
	}
	return nil
}

func (engine *Engine) validateUnitTemplate() error {
	data, err := readSmallRegular(engine.UnitTemplate, 1<<20)
	if err != nil {
		return err
	}
	want := map[string]string{
		"ExecStart=":          "ExecStart=" + engine.Environment.Service.ExecStart,
		"WorkingDirectory=":   "WorkingDirectory=" + engine.Environment.Service.WorkingDirectory,
		"Environment=AOFEI=":  "Environment=AOFEI=" + engine.Environment.Paths.AofeiConfig,
		"Environment=SUMMER=": "Environment=SUMMER=" + engine.Environment.Paths.SummerConfig,
	}
	counts := map[string]int{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		for prefix, exact := range want {
			if strings.HasPrefix(line, prefix) {
				if line != exact {
					return fmt.Errorf("unit template does not match the environment manifest")
				}
				counts[prefix]++
			}
		}
	}
	for prefix := range want {
		if counts[prefix] != 1 {
			return fmt.Errorf("unit template does not match the environment manifest")
		}
	}
	return nil
}

func (engine *Engine) installRelease(source VerifiedRelease) (VerifiedRelease, error) {
	root := engine.Environment.Paths.ReleaseRoot
	if err := ensurePrivateDirectory(root, engine.Environment.Operator.UID, 0o750); err != nil {
		return VerifiedRelease{}, err
	}
	target := filepath.Join(root, source.Manifest.ReleaseID)
	if info, err := os.Lstat(target); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return VerifiedRelease{}, fmt.Errorf("existing release target is not a directory")
		}
		existing, verifyErr := VerifyRelease(target, engine.Inspector)
		if verifyErr != nil {
			return VerifiedRelease{}, verifyErr
		}
		left, _ := fileDigest(filepath.Join(source.Root, "checksums.sha256"))
		right, _ := fileDigest(filepath.Join(existing.Root, "checksums.sha256"))
		if left == "" || left != right {
			return VerifiedRelease{}, fmt.Errorf("existing release id has different contents")
		}
		return existing, nil
	} else if !os.IsNotExist(err) {
		return VerifiedRelease{}, err
	}
	stage, err := os.MkdirTemp(root, ".staging.")
	if err != nil {
		return VerifiedRelease{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = makeTreeWritable(stage)
			_ = os.RemoveAll(stage)
		}
	}()
	if err := copyTree(source.Root, stage); err != nil {
		return VerifiedRelease{}, err
	}
	staged, err := VerifyRelease(stage, engine.Inspector)
	if err != nil {
		return VerifiedRelease{}, err
	}
	if err := makeTreeImmutable(stage); err != nil {
		return VerifiedRelease{}, err
	}
	if err := os.Rename(stage, target); err != nil {
		return VerifiedRelease{}, err
	}
	cleanup = false
	if err := syncDirectory(root); err != nil {
		return VerifiedRelease{}, err
	}
	staged.Root = target
	return staged, nil
}

func (engine *Engine) switchCurrent(target string) error {
	if filepath.Dir(target) != filepath.Clean(engine.Environment.Paths.ReleaseRoot) || !releaseIDPattern.MatchString(filepath.Base(target)) {
		return fmt.Errorf("selected release is outside the release root")
	}
	current := engine.Environment.Paths.CurrentLink
	if err := ensurePrivateDirectory(filepath.Dir(current), engine.Environment.Operator.UID, 0o750); err != nil {
		return err
	}
	temporary := current + ".next." + strconv.Itoa(os.Getpid())
	if _, err := os.Lstat(temporary); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("temporary release link already exists")
	}
	previous, hadPrevious := "", false
	if info, err := os.Lstat(current); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("current release path is not a symlink")
		}
		previous, err = os.Readlink(current)
		if err != nil {
			return err
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(target, temporary); err != nil {
		return err
	}
	if err := os.Rename(temporary, current); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := syncDirectory(filepath.Dir(current)); err != nil {
		if hadPrevious {
			restore := temporary + ".restore"
			if linkErr := os.Symlink(previous, restore); linkErr == nil {
				_ = os.Rename(restore, current)
				_ = syncDirectory(filepath.Dir(current))
			}
		} else {
			_ = os.Remove(current)
			_ = syncDirectory(filepath.Dir(current))
		}
		return err
	}
	return nil
}

func (engine *Engine) removeCurrent() error {
	current := engine.Environment.Paths.CurrentLink
	if err := os.Remove(current); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDirectory(filepath.Dir(current))
}

func (engine *Engine) restartAndVerify(ctx context.Context, oldPID int) (int, error) {
	if _, err := engine.run(ctx, "systemctl", "--user", "restart", engine.Environment.Service.Name); err != nil {
		return 0, err
	}
	active, err := engine.run(ctx, "systemctl", "--user", "is-active", engine.Environment.Service.Name)
	if err != nil || strings.TrimSpace(string(active)) != "active" {
		return 0, fmt.Errorf("service is not active")
	}
	if err := engine.waitForProbes(ctx, engine.Environment.Health.Origin); err != nil {
		return 0, err
	}
	newPID, err := engine.servicePID(ctx)
	if err != nil || newPID == oldPID {
		return 0, fmt.Errorf("service process identity did not change")
	}
	if err := engine.probeOnce(ctx, engine.Environment.Health.Public); err != nil {
		return 0, err
	}
	return newPID, nil
}

func (engine *Engine) servicePID(ctx context.Context) (int, error) {
	output, err := engine.run(ctx, "systemctl", "--user", "show", engine.Environment.Service.Name, "-p", "MainPID", "--value")
	if err != nil {
		return 0, fmt.Errorf("service PID is unavailable")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil || pid < 1 {
		return 0, fmt.Errorf("service PID is invalid")
	}
	return pid, nil
}

func (engine *Engine) probeOnce(ctx context.Context, probes []Probe) error {
	for _, probe := range probes {
		status, err := engine.Prober.Status(ctx, probe.URL)
		if err != nil || status != probe.Status {
			return fmt.Errorf("health probe failed")
		}
	}
	return nil
}

func (engine *Engine) waitForProbes(ctx context.Context, probes []Probe) error {
	for attempt := 1; attempt <= engine.Environment.Health.Attempts; attempt++ {
		if err := engine.probeOnce(ctx, probes); err == nil {
			return nil
		}
		if attempt < engine.Environment.Health.Attempts {
			if err := engine.Sleep(ctx, time.Duration(engine.Environment.Health.IntervalSeconds)*time.Second); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("health retry policy exhausted")
}

func (engine *Engine) recoveryContext() (context.Context, context.CancelFunc) {
	seconds := engine.Environment.Health.Attempts*(engine.Environment.Health.IntervalSeconds+5) + 30
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}

func (engine *Engine) backupBootstrapInputs() (string, error) {
	root := engine.Environment.Paths.BootstrapBackupRoot
	if err := ensurePrivateDirectory(root, engine.Environment.Operator.UID, 0o700); err != nil {
		return "", err
	}
	backup := filepath.Join(root, engine.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.Mkdir(backup, 0o700); err != nil {
		return "", err
	}
	for _, item := range []struct {
		source string
		name   string
		mode   os.FileMode
	}{
		{engine.Environment.Paths.AofeiConfig, "aofei.json", 0o600},
		{engine.Environment.Paths.SummerConfig, "summer.json", 0o600},
		{engine.Environment.Service.UnitPath, "service.unit", 0o644},
	} {
		if err := copyRegular(item.source, filepath.Join(backup, item.name), item.mode); err != nil {
			return "", err
		}
	}
	return backup, nil
}

func (engine *Engine) writeReleaseConfigs() error {
	project := filepath.Join(engine.Environment.Paths.CurrentLink, "assets", "pzdesign")
	updates := []struct {
		path   string
		values map[string]string
	}{
		{engine.Environment.Paths.AofeiConfig, map[string]string{"document_root": filepath.Join(project, "www")}},
		{engine.Environment.Paths.SummerConfig, map[string]string{
			"ProjectRoot": project, "Template": filepath.Join(project, "tmpls"), "DocumentRoot": filepath.Join(project, "www"),
		}},
	}
	for _, update := range updates {
		var document map[string]any
		if err := decodeStrictFile(update.path, &document); err != nil {
			return err
		}
		for key, value := range update.values {
			document[key] = value
		}
		if err := writeJSONAtomic(update.path, document, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (engine *Engine) withLock(work func() error) error {
	path := engine.Environment.Paths.LockFile
	if err := ensurePrivateDirectory(filepath.Dir(path), engine.Environment.Operator.UID, 0o750); err != nil {
		return err
	}
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return fmt.Errorf("deployment lock is unavailable")
	}
	defer file.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || int(stat.Uid) != engine.Environment.Operator.UID || stat.Nlink != 1 {
		return fmt.Errorf("deployment lock metadata changed")
	}
	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("another deployment holds the lock")
	}
	defer unix.Flock(int(file.Fd()), unix.LOCK_UN) //nolint:errcheck
	return work()
}

func (engine *Engine) run(ctx context.Context, command string, arguments ...string) ([]byte, error) {
	return engine.Runner.Run(ctx, command, arguments, nil)
}

type historyRecord struct {
	Path             string        `json:"-"`
	SchemaVersion    int           `json:"schema_version"`
	Environment      string        `json:"environment"`
	Host             string        `json:"host"`
	ReleaseID        string        `json:"release_id"`
	ManifestSHA256   string        `json:"manifest_sha256"`
	PreviousRelease  string        `json:"previous_release"`
	AttemptedRelease string        `json:"attempted_release"`
	SelectedRelease  string        `json:"selected_release"`
	Result           string        `json:"result"`
	StartedAt        string        `json:"started_at"`
	CompletedAt      string        `json:"completed_at,omitempty"`
	OldPID           int           `json:"old_pid"`
	FinalPID         int           `json:"final_pid,omitempty"`
	Checks           historyChecks `json:"checks"`
}

type historyChecks struct {
	EnvironmentPreflight string `json:"environment_preflight"`
	Activation           string `json:"activation"`
	RollbackRecovery     string `json:"rollback_recovery"`
	SelectedOriginHealth string `json:"selected_origin_health"`
	SelectedReadiness    string `json:"selected_origin_readiness"`
	SelectedPublicSmoke  string `json:"selected_public_smoke"`
}

func (engine *Engine) beginHistory(release VerifiedRelease, previous string, oldPID int) (*historyRecord, error) {
	now := engine.Now().UTC()
	record := &historyRecord{
		SchemaVersion:    2,
		Environment:      engine.Environment.Environment,
		Host:             engine.Environment.HostFQDN,
		ReleaseID:        release.Manifest.ReleaseID,
		ManifestSHA256:   release.Digest,
		PreviousRelease:  previous,
		AttemptedRelease: release.Root,
		SelectedRelease:  previous,
		Result:           "started",
		StartedAt:        now.Format(time.RFC3339Nano),
		OldPID:           oldPID,
		Checks: historyChecks{
			EnvironmentPreflight: "passed", Activation: "pending", RollbackRecovery: "not_required",
			SelectedOriginHealth: "pending", SelectedReadiness: "pending", SelectedPublicSmoke: "pending",
		},
	}
	record.Path = filepath.Join(engine.HistoryDir, now.Format("20060102T150405.000000000Z")+"-"+release.Manifest.ReleaseID+"-deployment.json")
	if _, err := os.Lstat(record.Path); err == nil || !os.IsNotExist(err) {
		return nil, fmt.Errorf("deployment history record already exists")
	}
	if err := writeJSONAtomicExclusive(record.Path, record, 0o600); err != nil {
		return nil, err
	}
	return record, nil
}

func (engine *Engine) finishHistory(record *historyRecord, result, selected string, finalPID int, activation, rollback string) error {
	record.Result = result
	record.SelectedRelease = selected
	record.CompletedAt = engine.Now().UTC().Format(time.RFC3339Nano)
	record.FinalPID = finalPID
	record.Checks.Activation = activation
	record.Checks.RollbackRecovery = rollback
	record.Checks.SelectedOriginHealth = "passed"
	record.Checks.SelectedReadiness = "passed"
	record.Checks.SelectedPublicSmoke = "passed"
	if err := writeJSONAtomic(record.Path, record, 0o600); err != nil {
		return err
	}
	_, err := fmt.Fprintf(engine.Output, "deployment_history=%s\n", record.Path)
	return err
}

func (engine *Engine) finishFailedHistory(record *historyRecord, selected, result string) error {
	record.Result = result
	record.SelectedRelease = selected
	record.CompletedAt = engine.Now().UTC().Format(time.RFC3339Nano)
	record.Checks.Activation = "failed"
	record.Checks.RollbackRecovery = "failed"
	record.Checks.SelectedOriginHealth = "unknown"
	record.Checks.SelectedReadiness = "unknown"
	record.Checks.SelectedPublicSmoke = "unknown"
	if err := writeJSONAtomic(record.Path, record, 0o600); err != nil {
		return err
	}
	_, err := fmt.Fprintf(engine.Output, "deployment_history=%s\n", record.Path)
	return err
}

func validateRegularFile(path string, ownerUID int, requirePrivateMode bool) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("required regular file is missing")
	}
	if ownerUID >= 0 {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || int(stat.Uid) != ownerUID {
			return fmt.Errorf("file owner changed")
		}
	}
	if requirePrivateMode && info.Mode().Perm() != 0o600 && info.Mode().Perm() != 0o400 {
		return fmt.Errorf("file mode is not private")
	}
	return nil
}

func validateOwnedDirectory(path string, ownerUID int) error {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("directory is missing or writable by peers")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != ownerUID {
		return fmt.Errorf("directory owner changed")
	}
	return nil
}

func ensurePrivateDirectory(path string, ownerUID int, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o027 != 0 {
		return fmt.Errorf("directory is not private")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != ownerUID {
		return fmt.Errorf("directory owner changed")
	}
	return nil
}

func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == source {
			return nil
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.Mkdir(destination, info.Mode().Perm()|0o700)
		}
		return copyRegular(current, destination, info.Mode().Perm())
	})
}

func copyRegular(source, destination string, mode os.FileMode) error {
	fd, err := unix.Open(source, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	input := os.NewFile(uintptr(fd), source)
	if input == nil {
		_ = unix.Close(fd)
		return fmt.Errorf("source file is unavailable")
	}
	defer input.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Nlink != 1 {
		return fmt.Errorf("source is not a singly-linked regular file")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return err
	}
	output, err := os.CreateTemp(filepath.Dir(destination), ".deployment-copy.")
	if err != nil {
		return err
	}
	temporary := output.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()
	if err := output.Chmod(mode); err != nil {
		_ = output.Close()
		return err
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := os.Rename(temporary, destination); err != nil {
		return err
	}
	removeTemporary = false
	return syncDirectory(filepath.Dir(destination))
}

func makeTreeImmutable(root string) error {
	var paths []string
	if err := filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
		if err == nil {
			paths = append(paths, path)
		}
		return err
	}); err != nil {
		return err
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.Chmod(path, info.Mode().Perm()&^0o222); err != nil {
			return err
		}
	}
	return nil
}

func makeTreeWritable(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode().Perm()|0o700)
	})
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".deployment.")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func writeJSONAtomicExclusive(path string, value any, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	syncErr := file.Sync()
	closeErr := file.Close()
	if encodeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return errors.Join(encodeErr, syncErr, closeErr)
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

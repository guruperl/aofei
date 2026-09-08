// Package deployment implements the reusable immutable-release deployment
// contract for Aofei/Pzdesign/Genelet website realizations.
package deployment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	EnvironmentSchemaVersion = 1
	ReleaseSchemaVersion     = 2
	ReleaseKind              = "aofei-http-backend"
	LegacyReleaseKind        = "w8m-http-backend"
)

var (
	identifierPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	userPattern             = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	servicePattern          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]*\.service$`)
	hex40Pattern            = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hex64Pattern            = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	repositoryDigestPattern = regexp.MustCompile(`^[^@[:space:]]+@sha256:[0-9a-f]{64}$`)
	dockerImagePattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]{0,511}$`)
	releaseIDPattern        = regexp.MustCompile(`^aofei-([0-9a-f]{12})_pzdesign-([0-9a-f]{12})_genelet-([0-9a-f]{12})$`)
)

type Environment struct {
	SchemaVersion int          `json:"schema_version"`
	Environment   string       `json:"environment"`
	HostFQDN      string       `json:"host_fqdn"`
	Operator      Operator     `json:"operator"`
	Origins       Origins      `json:"origins"`
	Paths         Paths        `json:"paths"`
	Service       Service      `json:"service"`
	Front         *Front       `json:"front,omitempty"`
	Health        Health       `json:"health"`
	Dependencies  Dependencies `json:"dependencies"`
	Database      Database     `json:"database"`
	Retention     Retention    `json:"retention"`
}

type Operator struct {
	User string `json:"user"`
	UID  int    `json:"uid"`
}

type Origins struct {
	Canonical string   `json:"canonical"`
	Accepted  []string `json:"accepted"`
}

type Paths struct {
	ReleaseRoot         string `json:"release_root"`
	CurrentLink         string `json:"current_link"`
	LockFile            string `json:"lock_file"`
	AofeiConfig         string `json:"aofei_config"`
	SummerConfig        string `json:"summer_config"`
	BootstrapBackupRoot string `json:"bootstrap_backup_root"`
}

type Service struct {
	Scope                  string   `json:"scope"`
	Name                   string   `json:"name"`
	UnitPath               string   `json:"unit_path"`
	ExecStart              string   `json:"exec_start"`
	WorkingDirectory       string   `json:"working_directory"`
	SecretEnvironmentFiles []string `json:"secret_environment_files"`
}

// Front is target-owned metadata. The backend deployer accepts it so one
// strict environment document can also drive an independently authorized front
// workflow, but it never mutates or probes these paths.
type Front struct {
	Server              string   `json:"server"`
	ApacheSite          string   `json:"apache_site"`
	ApacheInclude       string   `json:"apache_include"`
	LegacyDocumentRoot  string   `json:"legacy_document_root"`
	ReleaseRoot         string   `json:"release_root"`
	CurrentLink         string   `json:"current_link"`
	BackendOrigin       string   `json:"backend_origin"`
	PreservedUploadRoot string   `json:"preserved_upload_root"`
	HiddenPublicPaths   []string `json:"hidden_public_paths"`
}

type Health struct {
	Attempts        int     `json:"attempts"`
	IntervalSeconds int     `json:"interval_seconds"`
	Origin          []Probe `json:"origin"`
	Public          []Probe `json:"public"`
}

type Probe struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
}

type Dependencies struct {
	Docker []DockerDependency `json:"docker"`
}

type DockerDependency struct {
	Name             string          `json:"name"`
	ConfiguredImage  string          `json:"configured_image"`
	ImageID          string          `json:"image_id"`
	RepositoryDigest string          `json:"repository_digest"`
	PublishedPorts   []PublishedPort `json:"published_ports"`
}

type PublishedPort struct {
	Container string `json:"container"`
	Host      string `json:"host"`
}

type Database struct {
	Container                 string `json:"container"`
	Name                      string `json:"name"`
	Tables                    int    `json:"tables"`
	Views                     int    `json:"views"`
	Routines                  int    `json:"routines"`
	Triggers                  int    `json:"triggers"`
	AuthoritativeMoneyColumns int    `json:"authoritative_money_columns"`
	AccountingVersion         string `json:"accounting_version"`
}

type Retention struct {
	MinimumReleases   int  `json:"minimum_releases"`
	AutomaticDeletion bool `json:"automatic_deletion"`
}

type ReleaseManifest struct {
	SchemaVersion int              `json:"schema_version"`
	Kind          string           `json:"kind"`
	ReleaseID     string           `json:"release_id"`
	BuiltAt       string           `json:"built_at"`
	GoVersion     string           `json:"go_version"`
	Sources       ReleaseSources   `json:"sources"`
	Contracts     ReleaseContracts `json:"contracts"`
	Assets        ReleaseAssets    `json:"assets"`
	ArtifactCount int              `json:"artifact_count"`
}

type ReleaseSources struct {
	Aofei    Source `json:"aofei"`
	Pzdesign Source `json:"pzdesign"`
	Genelet  Source `json:"genelet"`
}

type Source struct {
	Commit   string `json:"commit"`
	Branch   string `json:"branch"`
	Upstream string `json:"upstream"`
	Remote   string `json:"remote"`
}

type ReleaseContracts struct {
	Database          DatabaseShape `json:"database"`
	AccountingVersion string        `json:"accounting_version"`
}

type DatabaseShape struct {
	Tables   int `json:"tables"`
	Views    int `json:"views"`
	Routines int `json:"routines"`
	Triggers int `json:"triggers"`
}

type ReleaseAssets struct {
	ComponentCount int `json:"component_count"`
}

func LoadEnvironment(path string) (Environment, error) {
	var environment Environment
	if err := decodeStrictFile(path, &environment); err != nil {
		return Environment{}, fmt.Errorf("environment manifest: %w", err)
	}
	if err := environment.Validate(); err != nil {
		return Environment{}, fmt.Errorf("environment manifest: %w", err)
	}
	return environment, nil
}

func loadReleaseManifest(path string) (ReleaseManifest, error) {
	var manifest ReleaseManifest
	if err := decodeStrictFile(path, &manifest); err != nil {
		return ReleaseManifest{}, fmt.Errorf("release manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return ReleaseManifest{}, fmt.Errorf("release manifest: %w", err)
	}
	return manifest, nil
}

func decodeStrictFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func (environment Environment) Validate() error {
	if environment.SchemaVersion != EnvironmentSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", environment.SchemaVersion)
	}
	if !identifierPattern.MatchString(environment.Environment) {
		return fmt.Errorf("invalid environment name")
	}
	if environment.HostFQDN == "" || strings.ContainsAny(environment.HostFQDN, `/\\ 	\r\n`) {
		return fmt.Errorf("invalid host_fqdn")
	}
	if !userPattern.MatchString(environment.Operator.User) || environment.Operator.UID < 0 {
		return fmt.Errorf("invalid operator")
	}
	canonical, err := validateOrigin(environment.Origins.Canonical, true)
	if err != nil {
		return fmt.Errorf("canonical origin: %w", err)
	}
	accepted := make(map[string]bool, len(environment.Origins.Accepted))
	for _, raw := range environment.Origins.Accepted {
		origin, err := validateOrigin(raw, true)
		if err != nil || accepted[origin] {
			return fmt.Errorf("invalid accepted origin")
		}
		accepted[origin] = true
	}
	if !accepted[canonical] {
		return fmt.Errorf("canonical origin is not accepted")
	}
	pathValues := []string{
		environment.Paths.ReleaseRoot, environment.Paths.CurrentLink,
		environment.Paths.LockFile, environment.Paths.AofeiConfig,
		environment.Paths.SummerConfig, environment.Paths.BootstrapBackupRoot,
		environment.Service.UnitPath, environment.Service.ExecStart,
		environment.Service.WorkingDirectory,
	}
	pathValues = append(pathValues, environment.Service.SecretEnvironmentFiles...)
	for _, path := range pathValues {
		if err := validateAbsolutePath(path); err != nil {
			return err
		}
	}
	mutablePaths := append([]string{
		environment.Paths.CurrentLink, environment.Paths.LockFile,
		environment.Paths.AofeiConfig, environment.Paths.SummerConfig,
		environment.Paths.BootstrapBackupRoot, environment.Service.UnitPath,
	}, environment.Service.SecretEnvironmentFiles...)
	for index, path := range mutablePaths {
		if pathsOverlap(environment.Paths.ReleaseRoot, path) {
			return fmt.Errorf("mutable deployment path overlaps the release root")
		}
		for _, other := range mutablePaths[index+1:] {
			if pathsOverlap(path, other) {
				return fmt.Errorf("deployment paths collide")
			}
		}
	}
	if environment.Service.Scope != "user" || !servicePattern.MatchString(environment.Service.Name) {
		return fmt.Errorf("unsupported service contract")
	}
	if environment.Service.ExecStart != filepath.Join(environment.Paths.CurrentLink, "bin", "unify") ||
		environment.Service.WorkingDirectory != filepath.Join(environment.Paths.CurrentLink, "assets", "pzdesign") {
		return fmt.Errorf("service paths do not follow the atomic selection")
	}
	if hasDuplicate(environment.Service.SecretEnvironmentFiles) {
		return fmt.Errorf("duplicate secret environment file")
	}
	if environment.Health.Attempts < 1 || environment.Health.Attempts > 300 ||
		environment.Health.IntervalSeconds < 0 || environment.Health.IntervalSeconds > 60 ||
		len(environment.Health.Origin) == 0 || len(environment.Health.Public) == 0 {
		return fmt.Errorf("invalid health retry policy")
	}
	healthPaths := map[string]int{}
	for _, probe := range environment.Health.Origin {
		if err := validateProbe(probe, false, nil); err != nil {
			return fmt.Errorf("origin health: %w", err)
		}
		parsed, _ := url.Parse(probe.URL)
		healthPaths[parsed.Path]++
	}
	if healthPaths["/healthz"] != 1 || healthPaths["/readyz"] != 1 || len(environment.Health.Origin) != 2 {
		return fmt.Errorf("origin health must contain exact healthz and readyz probes")
	}
	for _, probe := range environment.Health.Public {
		if err := validateProbe(probe, true, accepted); err != nil {
			return fmt.Errorf("public health: %w", err)
		}
	}
	seenDependencies := map[string]bool{}
	for _, dependency := range environment.Dependencies.Docker {
		if !identifierPattern.MatchString(dependency.Name) || seenDependencies[dependency.Name] || !dockerImagePattern.MatchString(dependency.ConfiguredImage) ||
			!hex64Pattern.MatchString(dependency.ImageID) || !repositoryDigestPattern.MatchString(dependency.RepositoryDigest) {
			return fmt.Errorf("invalid Docker dependency")
		}
		seenDependencies[dependency.Name] = true
		for _, port := range dependency.PublishedPorts {
			if err := validatePublishedPort(port); err != nil {
				return fmt.Errorf("Docker dependency %s: %w", dependency.Name, err)
			}
		}
	}
	if !identifierPattern.MatchString(environment.Database.Container) || !identifierPattern.MatchString(environment.Database.Name) ||
		environment.Database.Tables < 0 || environment.Database.Views < 0 ||
		environment.Database.Routines < 0 || environment.Database.Triggers < 0 ||
		environment.Database.AuthoritativeMoneyColumns < 0 || environment.Database.AccountingVersion == "" {
		return fmt.Errorf("invalid database contract")
	}
	if !seenDependencies[environment.Database.Container] {
		return fmt.Errorf("database container is not a declared dependency")
	}
	if environment.Retention.MinimumReleases < 2 || environment.Retention.AutomaticDeletion {
		return fmt.Errorf("unsafe retention contract")
	}
	if environment.Front != nil {
		for _, path := range []string{environment.Front.ApacheSite, environment.Front.ApacheInclude,
			environment.Front.LegacyDocumentRoot, environment.Front.ReleaseRoot,
			environment.Front.CurrentLink, environment.Front.PreservedUploadRoot} {
			if err := validateAbsolutePath(path); err != nil {
				return fmt.Errorf("front: %w", err)
			}
		}
		if environment.Front.Server == "" {
			return fmt.Errorf("front server is required")
		}
		if _, err := validateOrigin(environment.Front.BackendOrigin, false); err != nil {
			return fmt.Errorf("front backend origin: %w", err)
		}
	}
	return nil
}

func (manifest ReleaseManifest) Validate() error {
	legacy := manifest.SchemaVersion == 1 && manifest.Kind == LegacyReleaseKind
	current := manifest.SchemaVersion == ReleaseSchemaVersion && manifest.Kind == ReleaseKind
	if !legacy && !current {
		return fmt.Errorf("unsupported release schema/kind")
	}
	matches := releaseIDPattern.FindStringSubmatch(manifest.ReleaseID)
	if len(matches) != 4 {
		return fmt.Errorf("invalid release_id")
	}
	sources := []Source{manifest.Sources.Aofei, manifest.Sources.Pzdesign, manifest.Sources.Genelet}
	for _, source := range sources {
		if !hex40Pattern.MatchString(source.Commit) || !safeSourceText(source.Branch, 256) ||
			!safeSourceText(source.Upstream, 512) || !safeSourceRemote(source.Remote) {
			return fmt.Errorf("invalid source provenance")
		}
	}
	if matches[1] != manifest.Sources.Aofei.Commit[:12] ||
		matches[2] != manifest.Sources.Pzdesign.Commit[:12] ||
		matches[3] != manifest.Sources.Genelet.Commit[:12] {
		return fmt.Errorf("release_id does not match source commits")
	}
	if _, err := time.Parse(time.RFC3339, manifest.BuiltAt); err != nil {
		return fmt.Errorf("invalid built_at")
	}
	if !safeDisplayText(manifest.GoVersion, 128) || manifest.ArtifactCount < 2 ||
		manifest.Assets.ComponentCount < 0 || manifest.Contracts.AccountingVersion == "" ||
		manifest.Contracts.Database.Tables < 0 || manifest.Contracts.Database.Views < 0 ||
		manifest.Contracts.Database.Routines < 0 || manifest.Contracts.Database.Triggers < 0 {
		return fmt.Errorf("incomplete release contract")
	}
	return nil
}

func validateAbsolutePath(path string) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || path == string(filepath.Separator) || strings.ContainsAny(path, "\x00\r\n\t ") {
		return fmt.Errorf("invalid absolute path %q", path)
	}
	return nil
}

func validateOrigin(raw string, public bool) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Path != "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid origin")
	}
	if public && parsed.Scheme != "https" {
		return "", fmt.Errorf("public origin must use HTTPS")
	}
	if !public && parsed.Scheme != "http" {
		return "", fmt.Errorf("origin must use HTTP")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func validateProbe(probe Probe, public bool, accepted map[string]bool) error {
	if probe.Status < 100 || probe.Status > 599 {
		return fmt.Errorf("invalid expected status")
	}
	parsed, err := url.Parse(probe.URL)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("invalid URL")
	}
	if public {
		if parsed.Scheme != "https" || !accepted[parsed.Scheme+"://"+parsed.Host] {
			return fmt.Errorf("URL is outside accepted origins")
		}
		return nil
	}
	if parsed.Scheme != "http" {
		return fmt.Errorf("direct origin must use HTTP")
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("direct origin is not loopback")
	}
	return nil
}

func validatePublishedPort(port PublishedPort) error {
	parts := strings.Split(port.Container, "/")
	if len(parts) != 2 || (parts[1] != "tcp" && parts[1] != "udp") {
		return fmt.Errorf("invalid container port")
	}
	value, err := strconv.Atoi(parts[0])
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("invalid container port")
	}
	host, rawPort, err := net.SplitHostPort(port.Host)
	if err != nil {
		return fmt.Errorf("invalid host port")
	}
	ip := net.ParseIP(host)
	hostPort, numberErr := strconv.Atoi(rawPort)
	if ip == nil || !ip.IsLoopback() || numberErr != nil || hostPort < 1 || hostPort > 65535 {
		return fmt.Errorf("host port is not loopback")
	}
	return nil
}

func hasDuplicate(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func withinPath(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func pathsOverlap(left, right string) bool {
	return withinPath(left, right) || withinPath(right, left)
}

func safeSourceText(value string, limit int) bool {
	return value != "" && len(value) <= limit && !strings.ContainsAny(value, "\x00\r\n\t ")
}

func safeDisplayText(value string, limit int) bool {
	return value != "" && len(value) <= limit && !strings.ContainsAny(value, "\x00\r\n\t")
}

func safeSourceRemote(value string) bool {
	if !safeSourceText(value, 2048) {
		return false
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		return err == nil && parsed.Scheme != "" && parsed.Host != "" && parsed.User == nil
	}
	return strings.Count(value, "@") <= 1 && strings.Contains(value, ":")
}

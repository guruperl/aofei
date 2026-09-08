package deployment

import (
	"bufio"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type ProvenanceInspector interface {
	Inspect(path string) (ExecutableProvenance, error)
}

type BuildInfoInspector struct{}

type ExecutableProvenance struct {
	Revision  string
	GoVersion string
}

func (BuildInfoInspector) Inspect(path string) (ExecutableProvenance, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return ExecutableProvenance{}, err
	}
	provenance := ExecutableProvenance{}
	goos, goarch := "", ""
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && hex40Pattern.MatchString(setting.Value) {
			provenance.Revision = setting.Value
		}
		if setting.Key == "vcs.modified" && setting.Value == "true" {
			return ExecutableProvenance{}, fmt.Errorf("binary was built from a modified worktree")
		}
		switch setting.Key {
		case "GOOS":
			goos = setting.Value
		case "GOARCH":
			goarch = setting.Value
		}
	}
	if provenance.Revision != "" && info.GoVersion != "" && goos != "" && goarch != "" {
		provenance.GoVersion = fmt.Sprintf("go version %s %s/%s", info.GoVersion, goos, goarch)
		return provenance, nil
	}
	return ExecutableProvenance{}, fmt.Errorf("build provenance is incomplete")
}

type VerifiedRelease struct {
	Root     string
	Manifest ReleaseManifest
	Digest   string
}

func VerifyRelease(root string, inspector ProvenanceInspector) (VerifiedRelease, error) {
	if inspector == nil {
		return VerifiedRelease{}, fmt.Errorf("provenance inspector is required")
	}
	canonical, err := canonicalDirectory(root)
	if err != nil {
		return VerifiedRelease{}, err
	}
	if err := validateReleaseTree(canonical); err != nil {
		return VerifiedRelease{}, err
	}
	manifestPath := filepath.Join(canonical, "manifest.json")
	manifest, err := loadReleaseManifest(manifestPath)
	if err != nil {
		return VerifiedRelease{}, err
	}
	releaseID, err := readSmallRegular(filepath.Join(canonical, "RELEASE_ID"), 256)
	if err != nil || string(releaseID) != manifest.ReleaseID+"\n" {
		return VerifiedRelease{}, fmt.Errorf("RELEASE_ID does not match the manifest")
	}
	checksums, err := readChecksums(filepath.Join(canonical, "checksums.sha256"))
	if err != nil {
		return VerifiedRelease{}, err
	}
	files, err := releaseFiles(canonical)
	if err != nil {
		return VerifiedRelease{}, err
	}
	if len(checksums) != len(files) {
		return VerifiedRelease{}, fmt.Errorf("checksum inventory is incomplete")
	}
	for _, relative := range files {
		want, ok := checksums[relative]
		if !ok {
			return VerifiedRelease{}, fmt.Errorf("checksum is missing for %s", relative)
		}
		got, err := fileDigest(filepath.Join(canonical, filepath.FromSlash(relative)))
		if err != nil || got != want {
			return VerifiedRelease{}, fmt.Errorf("checksum verification failed for %s", relative)
		}
	}
	for _, executable := range []string{"bin/unify", "bin/config-preflight"} {
		if err := requireExecutable(filepath.Join(canonical, filepath.FromSlash(executable))); err != nil {
			return VerifiedRelease{}, err
		}
	}
	if manifest.SchemaVersion == ReleaseSchemaVersion {
		if err := requireExecutable(filepath.Join(canonical, "bin", "aofei-deploy")); err != nil {
			return VerifiedRelease{}, err
		}
	}
	if err := requireDirectory(filepath.Join(canonical, "assets", "pzdesign", "tmpls")); err != nil {
		return VerifiedRelease{}, err
	}
	if err := requireDirectory(filepath.Join(canonical, "assets", "pzdesign", "www")); err != nil {
		return VerifiedRelease{}, err
	}
	components, artifacts, err := countReleaseInventory(canonical)
	if err != nil {
		return VerifiedRelease{}, err
	}
	if components != manifest.Assets.ComponentCount || artifacts != manifest.ArtifactCount {
		return VerifiedRelease{}, fmt.Errorf("release inventory does not match the manifest")
	}
	provenance := []struct {
		path     string
		revision string
	}{
		{path: "bin/unify", revision: manifest.Sources.Pzdesign.Commit},
		{path: "bin/config-preflight", revision: manifest.Sources.Aofei.Commit},
	}
	if manifest.SchemaVersion == ReleaseSchemaVersion {
		provenance = append(provenance, struct {
			path     string
			revision string
		}{path: "bin/aofei-deploy", revision: manifest.Sources.Aofei.Commit})
	}
	for _, executable := range provenance {
		provenance, err := inspector.Inspect(filepath.Join(canonical, filepath.FromSlash(executable.path)))
		if err != nil || provenance.Revision != executable.revision ||
			(manifest.SchemaVersion == ReleaseSchemaVersion && provenance.GoVersion != manifest.GoVersion) {
			return VerifiedRelease{}, fmt.Errorf("%s build provenance does not match the manifest", executable.path)
		}
	}
	digest, err := fileDigest(manifestPath)
	if err != nil {
		return VerifiedRelease{}, err
	}
	return VerifiedRelease{Root: canonical, Manifest: manifest, Digest: digest}, nil
}

func canonicalDirectory(root string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("release path must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve release: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("release is not a directory")
	}
	return filepath.Clean(resolved), nil
}

func validateReleaseTree(root string) error {
	return filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("release contains a symlink")
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("release contains a non-regular object")
		}
		if info.Mode().IsRegular() {
			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok || stat.Nlink != 1 {
				return fmt.Errorf("release contains a multiply-linked file")
			}
		}
		if info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("release contains a group/world-writable path")
		}
		return nil
	})
}

func readChecksums(pathname string) (map[string]string, error) {
	info, err := os.Lstat(pathname)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 16<<20 {
		return nil, fmt.Errorf("invalid checksum manifest")
	}
	file, err := os.Open(pathname)
	if err != nil {
		return nil, fmt.Errorf("open checksums: %w", err)
	}
	defer file.Close()
	checksums := map[string]string{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		separator := strings.Index(line, "  ")
		if separator != 64 || len(line) <= separator+2 {
			return nil, fmt.Errorf("invalid checksum row")
		}
		digest, relative := line[:separator], line[separator+2:]
		if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 || strings.ToLower(digest) != digest || !safeReleasePath(relative) || checksums[relative] != "" {
			return nil, fmt.Errorf("invalid checksum row")
		}
		checksums[relative] = digest
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	if len(checksums) == 0 {
		return nil, fmt.Errorf("checksum inventory is empty")
	}
	return checksums, nil
}

func safeReleasePath(relative string) bool {
	if relative == "" || strings.ContainsAny(relative, "\\\x00\r\n\t ") || strings.HasPrefix(relative, "/") || path.Clean(relative) != relative {
		return false
	}
	return relative == "manifest.json" || relative == "RELEASE_ID" ||
		strings.HasPrefix(relative, "bin/") || strings.HasPrefix(relative, "assets/")
}

func releaseFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative != "checksums.sha256" {
			if !safeReleasePath(relative) {
				return fmt.Errorf("unexpected release file %s", relative)
			}
			files = append(files, relative)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func countReleaseInventory(root string) (components, artifacts int, err error) {
	err = filepath.WalkDir(filepath.Join(root, "bin"), func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			artifacts++
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	err = filepath.WalkDir(filepath.Join(root, "assets"), func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		artifacts++
		if entry.Name() == "component.json" {
			relative, relErr := filepath.Rel(filepath.Join(root, "assets", "pzdesign", "summer"), current)
			if relErr == nil && len(strings.Split(filepath.ToSlash(relative), "/")) == 2 {
				components++
			}
		}
		return nil
	})
	return components, artifacts, err
}

func requireExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("required executable is missing: %s", filepath.Base(path))
	}
	return nil
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("required asset directory is missing")
	}
	return nil
}

func readSmallRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limit {
		return nil, fmt.Errorf("invalid regular file %s", filepath.Base(path))
	}
	return os.ReadFile(path)
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

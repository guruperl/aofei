package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type fixtureInspector struct{}

func (fixtureInspector) Inspect(executable string) (ExecutableProvenance, error) {
	manifest, err := loadReleaseManifest(filepath.Join(filepath.Dir(filepath.Dir(executable)), "manifest.json"))
	if err != nil {
		return ExecutableProvenance{}, err
	}
	switch filepath.Base(executable) {
	case "unify":
		return ExecutableProvenance{Revision: manifest.Sources.Pzdesign.Commit, GoVersion: manifest.GoVersion}, nil
	case "config-preflight", "aofei-deploy":
		return ExecutableProvenance{Revision: manifest.Sources.Aofei.Commit, GoVersion: manifest.GoVersion}, nil
	}
	return ExecutableProvenance{}, fmt.Errorf("unknown executable")
}

type wrongGoInspector struct{ fixtureInspector }

func (inspector wrongGoInspector) Inspect(executable string) (ExecutableProvenance, error) {
	provenance, err := inspector.fixtureInspector.Inspect(executable)
	provenance.GoVersion = "go version go0.0.0 invalid/invalid"
	return provenance, err
}

func writeReleaseFixture(t *testing.T, parent string, legacy bool) (string, ReleaseManifest, fixtureInspector) {
	return writeReleaseFixtureWithIdentity(t, parent, legacy, "abc")
}

func writeReleaseFixtureWithIdentity(t *testing.T, parent string, legacy bool, identity string) (string, ReleaseManifest, fixtureInspector) {
	t.Helper()
	if len(identity) != 3 {
		t.Fatal("release fixture identity must contain three characters")
	}
	version, kind := ReleaseSchemaVersion, ReleaseKind
	if legacy {
		version, kind = 1, LegacyReleaseKind
	}
	manifest := testReleaseManifest(version, kind)
	manifest.Sources.Aofei.Commit = strings.Repeat(identity[0:1], 40)
	manifest.Sources.Pzdesign.Commit = strings.Repeat(identity[1:2], 40)
	manifest.Sources.Genelet.Commit = strings.Repeat(identity[2:3], 40)
	manifest.ReleaseID = "aofei-" + manifest.Sources.Aofei.Commit[:12] + "_pzdesign-" + manifest.Sources.Pzdesign.Commit[:12] + "_genelet-" + manifest.Sources.Genelet.Commit[:12]
	root := filepath.Join(parent, manifest.ReleaseID+fmt.Sprint(legacy))
	files := map[string][]byte{
		"bin/unify":            []byte("unify"),
		"bin/config-preflight": []byte("preflight"),
		"assets/pzdesign/summer/example/component.json": []byte("{}\n"),
		"assets/pzdesign/tmpls/index.html":              []byte("template\n"),
		"assets/pzdesign/www/index.html":                []byte("static\n"),
	}
	if !legacy {
		files["bin/aofei-deploy"] = []byte("deployer")
	} else {
		manifest.ArtifactCount--
	}
	for relative, data := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(relative, "bin/") {
			mode = 0o755
		}
		if err := os.WriteFile(path, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestData = append(manifestData, '\n')
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "RELEASE_ID"), []byte(manifest.ReleaseID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var paths []string
	for relative := range files {
		paths = append(paths, relative)
	}
	paths = append(paths, "manifest.json", "RELEASE_ID")
	sort.Strings(paths)
	var checksums strings.Builder
	for _, relative := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		fmt.Fprintf(&checksums, "%s  %s\n", hex.EncodeToString(digest[:]), relative)
	}
	if err := os.WriteFile(filepath.Join(root, "checksums.sha256"), []byte(checksums.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	inspector := fixtureInspector{}
	return root, manifest, inspector
}

func TestVerifyReleaseAcceptsCurrentAndLegacyBundles(t *testing.T) {
	parent := t.TempDir()
	for _, legacy := range []bool{false, true} {
		root, manifest, inspector := writeReleaseFixture(t, parent, legacy)
		verified, err := VerifyRelease(root, inspector)
		if err != nil {
			t.Fatal(err)
		}
		if verified.Manifest.ReleaseID != manifest.ReleaseID || verified.Digest == "" {
			t.Fatal("verified release identity is incomplete")
		}
	}
}

func TestVerifyCurrentReleaseRejectsToolchainProvenanceMismatch(t *testing.T) {
	root, _, _ := writeReleaseFixture(t, t.TempDir(), false)
	if _, err := VerifyRelease(root, wrongGoInspector{}); err == nil {
		t.Fatal("current release with mismatched toolchain provenance passed")
	}
}

func TestVerifyReleaseRejectsIncompleteOrMutableBundles(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string)
	}{
		{name: "changed asset", mutate: func(root string) {
			_ = os.WriteFile(filepath.Join(root, "assets/pzdesign/www/index.html"), []byte("changed"), 0o644)
		}},
		{name: "extra file", mutate: func(root string) {
			_ = os.WriteFile(filepath.Join(root, "assets/pzdesign/www/extra"), []byte("extra"), 0o644)
		}},
		{name: "symlink", mutate: func(root string) { _ = os.Symlink("index.html", filepath.Join(root, "assets/pzdesign/www/link")) }},
		{name: "peer writable", mutate: func(root string) { _ = os.Chmod(filepath.Join(root, "bin/unify"), 0o775) }},
		{name: "peer writable root", mutate: func(root string) { _ = os.Chmod(root, 0o775) }},
		{name: "hard link", mutate: func(root string) {
			_ = os.Link(filepath.Join(root, "assets/pzdesign/www/index.html"), filepath.Join(root, "assets/pzdesign/www/hard-link"))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, _, inspector := writeReleaseFixture(t, t.TempDir(), false)
			test.mutate(root)
			if _, err := VerifyRelease(root, inspector); err == nil {
				t.Fatal("invalid release passed")
			}
		})
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	dummyCommitA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	dummyCommitB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	dummyCommitC = "cccccccccccccccccccccccccccccccccccccccc"
)

func makeFullManifest(version, rootCommit, modCommit string) *ReleaseManifest {
	var modules []ModuleEntry
	for dir, mod := range canonicalModules {
		modules = append(modules, ModuleEntry{
			Directory:    dir,
			Module:       mod,
			Version:      version,
			Tag:          dir + "/" + version,
			SourceCommit: modCommit,
		})
	}
	return &ReleaseManifest{
		Release:      version,
		SourceCommit: rootCommit,
		Modules:      modules,
	}
}

func TestValidateManifest_Full14InitialVersionPasses(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	norm, err := ValidateManifest(manifest, nil, "")
	if err != nil {
		t.Fatalf("expected valid initial manifest, got error: %v", err)
	}
	if len(norm.Modules) != 14 {
		t.Fatalf("expected 14 modules, got %d", len(norm.Modules))
	}
}

func TestValidateManifest_SemverV2Regex(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"valid standard", "v2.0.0", true},
		{"valid minor patch", "v2.10.3", true},
		{"valid prerelease alpha", "v2.0.0-alpha", true},
		{"valid prerelease dotted", "v2.0.0-alpha.1", true},
		{"valid prerelease alphanumeric", "v2.0.0-beta.2.abc-3", true},
		{"valid prerelease zero dot", "v2.0.0-0.3.7", true},
		{"invalid leading zero in numeric prerelease segment", "v2.0.0-01", false},
		{"invalid leading zero in second numeric prerelease segment", "v2.0.0-alpha.01", false},
		{"invalid empty prerelease segment", "v2.0.0-..", false},
		{"invalid double dot prerelease segment", "v2.0.0-a..b", false},
		{"invalid trailing dot prerelease", "v2.0.0-alpha.", false},
		{"invalid leading dot prerelease", "v2.0.0-.alpha", false},
		{"invalid build metadata rejected", "v2.0.0+build123", false},
		{"invalid build metadata with prerelease rejected", "v2.0.0-alpha+build", false},
		{"invalid v1 major", "v1.2.3", false},
		{"invalid missing v prefix", "2.0.0", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := makeFullManifest(tc.version, dummyCommitA, dummyCommitA)
			_, err := ValidateManifest(m, nil, "")
			if tc.valid && err != nil {
				t.Fatalf("expected version %q to be valid, got: %v", tc.version, err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected version %q to be invalid, got nil error", tc.version)
			}
		})
	}
}

func TestValidateManifest_DirectoryCanonicalChecks(t *testing.T) {
	nonCanonicalDirs := []struct {
		name string
		dir  string
	}{
		{"relative dot prefix", "./foundation"},
		{"traversal segment", "plugins/../foundation"},
		{"parent traversal", "../foundation"},
		{"double slash", "plugins//email"},
		{"trailing slash", "foundation/"},
		{"backslash path", "plugins\\database"},
		{"absolute path slash", "/foundation"},
	}

	for _, tc := range nonCanonicalDirs {
		t.Run(tc.name, func(t *testing.T) {
			m := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
			for i := range m.Modules {
				if m.Modules[i].Directory == "foundation" || m.Modules[i].Directory == "plugins/database" || m.Modules[i].Directory == "plugins/email" {
					m.Modules[i].Directory = tc.dir
					break
				}
			}
			_, err := ValidateManifest(m, nil, "")
			if err == nil {
				t.Fatalf("expected non-canonical directory %q to be rejected, got nil", tc.dir)
			}
		})
	}
}

func TestValidateManifest_PreviousManifestValidation(t *testing.T) {
	cur := makeFullManifest("v2.0.1", dummyCommitB, dummyCommitB)

	// Case 1: Bad previous manifest (e.g. duplicate directory in previous)
	badPrev := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	badPrev.Modules[0] = badPrev.Modules[1]
	_, err := ValidateManifest(cur, badPrev, "")
	if err == nil {
		t.Fatal("expected error when previous manifest is invalid, got nil")
	}
	if !strings.Contains(err.Error(), "previous manifest is invalid") {
		t.Fatalf("expected 'previous manifest is invalid' error, got: %v", err)
	}

	// Case 2: Valid previous manifest, incremental release passes
	goodPrev := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	_, err = ValidateManifest(cur, goodPrev, "")
	if err != nil {
		t.Fatalf("expected validation with good previous manifest to succeed, got: %v", err)
	}
}

func TestValidateManifest_MissingModulesFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	manifest.Modules = manifest.Modules[:13] // 13 modules only
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for missing modules, got nil")
	}
}

func TestValidateManifest_ExtraModulesFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	manifest.Modules = append(manifest.Modules, ModuleEntry{
		Directory:    "extra",
		Module:       "github.com/bamboo-services/bamboo-base-go/extra/v2",
		Version:      "v2.0.0",
		Tag:          "extra/v2.0.0",
		SourceCommit: dummyCommitA,
	})
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for extra module, got nil")
	}
}

func TestValidateManifest_DuplicateModuleFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	manifest.Modules[0] = manifest.Modules[1] // Duplicate entry
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for duplicate module, got nil")
	}
}

func TestValidateManifest_PathTraversalFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	for i, m := range manifest.Modules {
		if m.Directory == "foundation" {
			manifest.Modules[i].Directory = "../foundation"
			break
		}
	}
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestValidateManifest_RootPlaceholderFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	manifest.Modules[0].Directory = "."
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for root placeholder, got nil")
	}
}

func TestValidateManifest_WrongV2TagFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	for i, m := range manifest.Modules {
		if m.Directory == "plugins/grpc" {
			// e.g. plugins/grpc/v2/v2.0.0 extra /v2 tag
			manifest.Modules[i].Tag = "plugins/grpc/v2/v2.0.0"
			break
		}
	}
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for wrong v2 tag, got nil")
	}
}

func TestValidateManifest_WrongModulePathFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	manifest.Modules[0].Module = "github.com/bamboo-services/bamboo-base-go/wrong/v2"
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for mismatched module path, got nil")
	}
}

func TestValidateManifest_InvalidCommitHashFails(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", "HEAD", dummyCommitA)
	_, err := ValidateManifest(manifest, nil, "")
	if err == nil {
		t.Fatal("expected error for HEAD commit, got nil")
	}

	manifest2 := makeFullManifest("v2.0.0", dummyCommitA, "master")
	_, err = ValidateManifest(manifest2, nil, "")
	if err == nil {
		t.Fatal("expected error for branch commit, got nil")
	}
}

func TestValidateManifest_AppAndEmailBumpedOthersRetainedPasses(t *testing.T) {
	v200 := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)

	// v2.0.1 release where only application and email bumped to v2.0.1 and commitB
	// other modules retain v2.0.0 tag and dummyCommitA
	var newModules []ModuleEntry
	for _, m := range v200.Modules {
		if m.Directory == "application" || m.Directory == "plugins/email" {
			newModules = append(newModules, ModuleEntry{
				Directory:    m.Directory,
				Module:       m.Module,
				Version:      "v2.0.1",
				Tag:          m.Directory + "/v2.0.1",
				SourceCommit: dummyCommitB,
			})
		} else {
			newModules = append(newModules, m)
		}
	}
	v201 := &ReleaseManifest{
		Release:      "v2.0.1",
		SourceCommit: dummyCommitB, // root release candidate commit
		Modules:      newModules,
	}

	norm, err := ValidateManifest(v201, v200, "")
	if err != nil {
		t.Fatalf("expected valid incremental release, got: %v", err)
	}

	for _, m := range norm.Modules {
		if m.Directory == "application" || m.Directory == "plugins/email" {
			if m.Version != "v2.0.1" || m.SourceCommit != dummyCommitB {
				t.Fatalf("expected application/email to be v2.0.1 / commitB, got %s / %s", m.Version, m.SourceCommit)
			}
		} else {
			if m.Version != "v2.0.0" || m.SourceCommit != dummyCommitA {
				t.Fatalf("expected %s to retain v2.0.0 / commitA, got %s / %s", m.Directory, m.Version, m.SourceCommit)
			}
		}
	}
}

func TestValidateManifest_SameTagAlteredCommitRejected(t *testing.T) {
	prev := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)

	// Attempting to re-publish an existing tag (foundation/v2.0.0) with altered commit
	cur := makeFullManifest("v2.0.1", dummyCommitB, dummyCommitA)
	for i, m := range cur.Modules {
		if m.Directory == "foundation" {
			cur.Modules[i].Version = "v2.0.0"
			cur.Modules[i].Tag = "foundation/v2.0.0"
			cur.Modules[i].SourceCommit = dummyCommitC // changed commit for same tag foundation/v2.0.0
			break
		}
	}

	_, err := ValidateManifest(cur, prev, "")
	if err == nil {
		t.Fatal("expected rejection when existing tag has altered sourceCommit, got nil")
	}
	if !strings.Contains(err.Error(), "same-tag sourceCommit mutation rejected") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateManifest_RealGoModMatching(t *testing.T) {
	tmpDir := t.TempDir()

	for dir, mod := range canonicalModules {
		modDir := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(modDir, 0755); err != nil {
			t.Fatal(err)
		}
		goModContent := "module " + mod + "\n\ngo 1.25\n"
		if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	_, err := ValidateManifest(manifest, nil, tmpDir)
	if err != nil {
		t.Fatalf("expected real go.mod matching to pass, got: %v", err)
	}

	// Corrupt one go.mod
	badMod := filepath.Join(tmpDir, "foundation", "go.mod")
	if err := os.WriteFile(badMod, []byte("module github.com/other/mod\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = ValidateManifest(manifest, nil, tmpDir)
	if err == nil {
		t.Fatal("expected failure when go.mod declares mismatched module path, got nil")
	}
}

func TestCLIOutputSerialization(t *testing.T) {
	manifest := makeFullManifest("v2.0.0", dummyCommitA, dummyCommitA)
	tmpFile := filepath.Join(t.TempDir(), "manifest.json")
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	readM, err := readManifest(tmpFile)
	if err != nil {
		t.Fatalf("readManifest failed: %v", err)
	}
	norm, err := ValidateManifest(readM, nil, "")
	if err != nil {
		t.Fatalf("ValidateManifest failed: %v", err)
	}
	if norm.Release != "v2.0.0" {
		t.Fatalf("expected v2.0.0, got %s", norm.Release)
	}
}

func TestGenerateFromModulesFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "v2-modules.txt")
	content := `# comments
foundation github.com/bamboo-services/bamboo-base-go/foundation/v2
application github.com/bamboo-services/bamboo-base-go/application/v2
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := generateFromModulesFile(tmpFile, "v2.0.0", dummyCommitA)
	if err != nil {
		t.Fatalf("generateFromModulesFile failed: %v", err)
	}
	if len(manifest.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(manifest.Modules))
	}
	if manifest.Modules[0].Tag != "foundation/v2.0.0" {
		t.Fatalf("expected foundation/v2.0.0, got %s", manifest.Modules[0].Tag)
	}
}

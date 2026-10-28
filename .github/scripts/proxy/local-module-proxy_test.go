package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEscapeModulePath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"github.com/foo/bar", "github.com/foo/bar"},
		{"github.com/Azure/azure-sdk-for-go", "github.com/!azure/azure-sdk-for-go"},
	}
	for _, tc := range tests {
		got := escapeModulePath(tc.in)
		if got != tc.want {
			t.Errorf("escapeModulePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsExcludedPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{".git/config", true},
		{".GIT/config", true},
		{".pi/agent/foo", true},
		{".github/workflows/foo.yml", true},
		{"node_modules/pkg/index.js", true},
		{".env", true},
		{".ENV", true},
		{".env.local", true},
		{".ENV.LOCAL", true},
		{"foo.env.local", true},
		{"configs/app.env", true},
		{"sub/credentials", true},
		{"sub/cert.key", true},
		{"sub/cert.KEY", true},
		{"sub/cert.pem", true},
		{"keys/id_rsa", true},
		{"keys/ID_RSA", true},
		{"keys/id_ecdsa", true},
		{"certs/bundle.p12", true},
		{"certs/bundle.pfx", true},
		{"certs/keystore.jks", true},
		{"src/main.go", false},
		{"README.md", false},
		{"go.mod", false},
	}
	for _, tc := range cases {
		got := isExcludedPath(tc.path)
		if got != tc.want {
			t.Errorf("isExcludedPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestCreateModuleZipExcludesNestedAndSensitive(t *testing.T) {
	tmpDir := t.TempDir()

	// Root module files
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test/mod\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "code.go"), []byte("package mod\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sensitive files
	if err := os.WriteFile(filepath.Join(tmpDir, "secret.key"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "id_rsa"), []byte("private-key"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cert.p12"), []byte("cert"), 0644); err != nil {
		t.Fatal(err)
	}

	// Excluded dir
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Nested module
	subDir := filepath.Join(tmpDir, "nested")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module test/mod/nested\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.go"), []byte("package nested\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outZip := filepath.Join(t.TempDir(), "v2.0.0.zip")
	_, count, realGoFiles, goModBytes, err := createModuleZip(tmpDir, "test/mod", "v2.0.0", outZip)
	if err != nil {
		t.Fatalf("createModuleZip failed: %v", err)
	}

	zr, err := zip.OpenReader(outZip)
	if err != nil {
		t.Fatalf("OpenReader failed: %v", err)
	}
	defer zr.Close()

	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		if !strings.HasPrefix(f.Name, "test/mod@v2.0.0/") {
			t.Errorf("expected entry prefix 'test/mod@v2.0.0/', got %s", f.Name)
		}
		if strings.Contains(f.Name, "secret.key") || strings.Contains(f.Name, "id_rsa") || strings.Contains(f.Name, "cert.p12") {
			t.Errorf("sensitive file was included in zip: %s", f.Name)
		}
		if strings.Contains(f.Name, ".git") {
			t.Errorf(".git entry was included in zip: %s", f.Name)
		}
		if strings.Contains(f.Name, "nested") {
			t.Errorf("nested module was included in zip: %s", f.Name)
		}
	}

	if count != 2 { // go.mod and code.go
		t.Errorf("expected 2 files packaged, got %d (found: %v)", count, names)
	}
	if realGoFiles != 1 {
		t.Errorf("expected 1 real Go file, got %d", realGoFiles)
	}
	if !strings.Contains(string(goModBytes), "module test/mod") {
		t.Errorf("expected goModBytes to contain module declaration, got %s", string(goModBytes))
	}
}

func TestRejectGoModAndReadmeOnlyShell(t *testing.T) {
	// Candidate module containing only go.mod and README.md (no real .go source files)
	repoRoot := t.TempDir()
	modDir := filepath.Join(repoRoot, "shellmod")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module test/shellmod\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "README.md"), []byte("# Shell module\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	target := moduleTarget{
		relDir:     "shellmod",
		modulePath: "test/shellmod",
		version:    "v2.0.0",
	}

	err := buildModuleProxy(repoRoot, target, outDir)
	if err == nil {
		t.Fatal("expected buildModuleProxy to fail for shell module with only go.mod and README.md, but got nil")
	}
	if !strings.Contains(err.Error(), "real Go source file") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Verify no partial consumable artifacts left (.zip, .mod, list, .info)
	destDir := filepath.Join(outDir, escapeModulePath(target.modulePath), "@v")
	for _, name := range []string{"v2.0.0.zip", "v2.0.0.mod", "list", "v2.0.0.info"} {
		path := filepath.Join(destDir, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected %s to NOT exist after failure, but it was left in destDir", name)
		}
	}
}

func TestRejectTestFilesOnly(t *testing.T) {
	// Candidate module containing only _test.go (no non-_test.go source)
	repoRoot := t.TempDir()
	modDir := filepath.Join(repoRoot, "testonly")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module test/testonly\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "foo_test.go"), []byte("package testonly\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	target := moduleTarget{
		relDir:     "testonly",
		modulePath: "test/testonly",
		version:    "v2.0.0",
	}

	err := buildModuleProxy(repoRoot, target, outDir)
	if err == nil {
		t.Fatal("expected buildModuleProxy to fail for test-only module")
	}
	destDir := filepath.Join(outDir, escapeModulePath(target.modulePath), "@v")
	for _, name := range []string{"v2.0.0.zip", "v2.0.0.mod", "list", "v2.0.0.info"} {
		if _, err := os.Stat(filepath.Join(destDir, name)); err == nil {
			t.Errorf("consumable artifact %s was left after error", name)
		}
	}
}

func TestGoModConsistencyDetection(t *testing.T) {
	repoRoot := t.TempDir()
	modDir := filepath.Join(repoRoot, "validmod")
	if err := os.MkdirAll(modDir, 0755); err != nil {
		t.Fatal(err)
	}
	goModContent := []byte("module test/validmod\n\ngo 1.25\n")
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), goModContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "lib.go"), []byte("package validmod\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	target := moduleTarget{
		relDir:     "validmod",
		modulePath: "test/validmod",
		version:    "v2.0.0",
	}

	if err := buildModuleProxy(repoRoot, target, outDir); err != nil {
		t.Fatalf("buildModuleProxy failed unexpectedly: %v", err)
	}

	destDir := filepath.Join(outDir, escapeModulePath(target.modulePath), "@v")
	for _, name := range []string{"v2.0.0.zip", "v2.0.0.mod", "list", "v2.0.0.info"} {
		if _, err := os.Stat(filepath.Join(destDir, name)); err != nil {
			t.Errorf("expected consumable artifact %s to exist, err: %v", name, err)
		}
	}
}

func TestEmptySourceRejection(t *testing.T) {
	emptyDir := t.TempDir()
	outZip := filepath.Join(t.TempDir(), "v2.0.0.zip")
	_, count, realGo, _, err := createModuleZip(emptyDir, "test/empty", "v2.0.0", outZip)
	if err != nil {
		t.Fatalf("unexpected error creating zip: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 files, got %d", count)
	}
	if realGo != 0 {
		t.Errorf("expected 0 real Go files, got %d", realGo)
	}
}

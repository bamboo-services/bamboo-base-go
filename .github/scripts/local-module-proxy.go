package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type moduleTarget struct {
	relDir     string
	modulePath string
	version    string
}

type moduleInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

func main() {
	var repoRoot string
	var outDir string
	var modulesFlag string
	var version string

	flag.StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
	flag.StringVar(&outDir, "out-dir", "", "Path to output proxy directory")
	flag.StringVar(&modulesFlag, "modules", "foundation,application", "Comma-separated list of relative directories to include in proxy")
	flag.StringVar(&version, "version", "v2.0.0", "Module version candidate to publish in local proxy")
	flag.Parse()

	if outDir == "" {
		fmt.Fprintln(os.Stderr, "error: -out-dir is required")
		os.Exit(1)
	}

	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving repo-root: %v\n", err)
		os.Exit(1)
	}

	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving out-dir: %v\n", err)
		os.Exit(1)
	}

	modDirs := strings.Split(modulesFlag, ",")
	var targets []moduleTarget
	for _, dir := range modDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		modPath, err := readModulePath(filepath.Join(absRepoRoot, trimmed))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			fmt.Fprintf(os.Stderr, "error reading module path in %s: %v\n", trimmed, err)
			os.Exit(1)
		}
		targets = append(targets, moduleTarget{
			relDir:     trimmed,
			modulePath: modPath,
			version:    version,
		})
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "error: no valid modules found to host in proxy")
		os.Exit(1)
	}

	for _, target := range targets {
		if err := buildModuleProxy(absRepoRoot, target, absOutDir); err != nil {
			fmt.Fprintf(os.Stderr, "error building proxy for %s (%s): %v\n", target.relDir, target.modulePath, err)
			os.Exit(1)
		}
	}
}

func readModulePath(dir string) (string, error) {
	modFile := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(modFile)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("module declaration not found in %s", modFile)
}

func escapeModulePath(path string) string {
	var sb strings.Builder
	for i := 0; i < len(path); i++ {
		b := path[i]
		if b >= 'A' && b <= 'Z' {
			sb.WriteByte('!')
			sb.WriteByte(b + ('a' - 'A'))
		} else {
			sb.WriteByte(b)
		}
	}
	return sb.String()
}

func isExcludedPath(rel string) bool {
	clean := filepath.ToSlash(rel)
	parts := strings.Split(clean, "/")
	for _, part := range parts {
		lowerPart := strings.ToLower(part)
		if lowerPart == ".git" || lowerPart == ".pi" || lowerPart == ".github" || lowerPart == "node_modules" || lowerPart == ".idea" || lowerPart == ".vscode" {
			return true
		}
	}
	base := strings.ToLower(filepath.Base(clean))

	// Match .env files and extensions (e.g. .env, .env.local, .env.production, foo.env.local)
	if base == ".env" || strings.HasPrefix(base, ".env.") || strings.HasSuffix(base, ".env") || strings.Contains(base, ".env.") {
		return true
	}

	// Match sensitive filenames and key files
	if base == "credentials" || base == "id_rsa" || base == "id_dsa" || base == "id_ecdsa" || base == "id_ed25519" {
		return true
	}

	// Match sensitive certificate / key extensions
	sensitiveExts := []string{".key", ".pem", ".p12", ".pfx", ".jks"}
	for _, ext := range sensitiveExts {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}

	return false
}

func buildModuleProxy(repoRoot string, target moduleTarget, outDir string) error {
	modDir := filepath.Join(repoRoot, target.relDir)
	escapedMod := escapeModulePath(target.modulePath)
	destDir := filepath.Join(outDir, filepath.FromSlash(escapedMod), "@v")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create proxy dir: %w", err)
	}

	// Track created artifacts to clean up on error, preventing partial consumable artifacts
	var createdFiles []string
	cleanUpOnError := func() {
		for _, f := range createdFiles {
			_ = os.Remove(f)
		}
	}

	goModContent, err := os.ReadFile(filepath.Join(modDir, "go.mod"))
	if err != nil {
		cleanUpOnError()
		return fmt.Errorf("read go.mod: %w", err)
	}

	zipPath := filepath.Join(destDir, target.version+".zip")
	modTime, fileCount, realGoFiles, zipGoModContent, err := createModuleZip(modDir, target.modulePath, target.version, zipPath)
	if err != nil {
		cleanUpOnError()
		return fmt.Errorf("create zip: %w", err)
	}
	createdFiles = append(createdFiles, zipPath)

	if fileCount == 0 {
		cleanUpOnError()
		return errors.New("empty module source: no valid files found to package")
	}

	if realGoFiles == 0 {
		cleanUpOnError()
		return errors.New("candidate module must contain at least one non-_test.go real Go source file")
	}

	// Verify go.mod content packaged in zip matches separate go.mod read to avoid inconsistent metadata
	if !bytes.Equal(goModContent, zipGoModContent) {
		cleanUpOnError()
		return errors.New("concurrent modification detected: go.mod in zip differs from standalone go.mod")
	}

	modPath := filepath.Join(destDir, target.version+".mod")
	if err := os.WriteFile(modPath, goModContent, 0644); err != nil {
		cleanUpOnError()
		return fmt.Errorf("write .mod: %w", err)
	}
	createdFiles = append(createdFiles, modPath)

	listPath := filepath.Join(destDir, "list")
	listContent := target.version + "\n"
	if err := os.WriteFile(listPath, []byte(listContent), 0644); err != nil {
		cleanUpOnError()
		return fmt.Errorf("write list: %w", err)
	}
	createdFiles = append(createdFiles, listPath)

	info := moduleInfo{
		Version: target.version,
		Time:    modTime.UTC(),
	}
	infoBytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		cleanUpOnError()
		return fmt.Errorf("marshal info: %w", err)
	}
	infoPath := filepath.Join(destDir, target.version+".info")
	if err := os.WriteFile(infoPath, append(infoBytes, '\n'), 0644); err != nil {
		cleanUpOnError()
		return fmt.Errorf("write .info: %w", err)
	}
	createdFiles = append(createdFiles, infoPath)

	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		cleanUpOnError()
		return fmt.Errorf("hash zip: %w", err)
	}
	hash := sha256.Sum256(zipData)
	fmt.Fprintf(os.Stdout, "proxy: generated %s@%s (files: %d, sha256: %s)\n", target.modulePath, target.version, fileCount, hex.EncodeToString(hash[:]))
	return nil
}

func createModuleZip(sourceDir, modPath, version, destZipPath string) (time.Time, int, int, []byte, error) {
	tmpZip := destZipPath + ".tmp"
	f, err := os.Create(tmpZip)
	if err != nil {
		return time.Time{}, 0, 0, nil, err
	}
	defer func() {
		f.Close()
		_ = os.Remove(tmpZip)
	}()

	zw := zip.NewWriter(f)
	prefix := fmt.Sprintf("%s@%s/", modPath, version)
	var latestTime time.Time
	fileCount := 0
	realGoFiles := 0
	var zipGoModContent []byte

	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if isExcludedPath(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Exclude nested submodules (directory containing another go.mod)
		if d.IsDir() {
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}

		// Don't package symlinks or non-regular files
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}

		zipEntryName := prefix + filepath.ToSlash(rel)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = zipEntryName
		header.Method = zip.Deflate

		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		fileReader, err := os.Open(path)
		if err != nil {
			return err
		}

		cleanRel := filepath.ToSlash(rel)
		if cleanRel == "go.mod" {
			var buf bytes.Buffer
			mw := io.MultiWriter(w, &buf)
			_, err = io.Copy(mw, fileReader)
			fileReader.Close()
			if err != nil {
				return err
			}
			zipGoModContent = buf.Bytes()
		} else {
			_, err = io.Copy(w, fileReader)
			fileReader.Close()
			if err != nil {
				return err
			}
		}

		// Count real Go source files (excluding _test.go)
		baseName := filepath.Base(cleanRel)
		if strings.HasSuffix(baseName, ".go") && !strings.HasSuffix(baseName, "_test.go") {
			realGoFiles++
		}

		fileCount++
		return nil
	})

	if err != nil {
		return time.Time{}, 0, 0, nil, err
	}

	if err := zw.Close(); err != nil {
		return time.Time{}, 0, 0, nil, err
	}
	if err := f.Close(); err != nil {
		return time.Time{}, 0, 0, nil, err
	}

	if err := os.Rename(tmpZip, destZipPath); err != nil {
		return time.Time{}, 0, 0, nil, err
	}

	if latestTime.IsZero() {
		latestTime = time.Now()
	}

	return latestTime, fileCount, realGoFiles, zipGoModContent, nil
}

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Allowed 14 v2 modules and their canonical paths
var canonicalModules = map[string]string{
	"foundation":                         "github.com/bamboo-services/bamboo-base-go/foundation/v2",
	"application":                        "github.com/bamboo-services/bamboo-base-go/application/v2",
	"plugins/database":                   "github.com/bamboo-services/bamboo-base-go/plugins/database/v2",
	"plugins/cache":                      "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2",
	"plugins/email":                      "github.com/bamboo-services/bamboo-base-go/plugins/email/v2",
	"plugins/web-gin":                    "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2",
	"plugins/grpc":                       "github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2",
	"plugins/cron":                       "github.com/bamboo-services/bamboo-base-go/plugins/cron/v2",
	"plugins/async":                      "github.com/bamboo-services/bamboo-base-go/plugins/async/v2",
	"plugins/database/adapter/mysql":     "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/mysql/v2",
	"plugins/database/adapter/postgres":  "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2",
	"plugins/database/adapter/sqlite":    "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2",
	"plugins/database/adapter/sqlserver": "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlserver/v2",
	"plugins/database/adapter/oracle":    "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/oracle/v2",
}

var (
	semverV2Regex   = regexp.MustCompile(`^v2\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(?:0|[1-9]\d*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*)?$`)
	commitHashRegex = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// ModuleEntry 表示单个 v2 模块在发布清单中的元数据条目。
type ModuleEntry struct {
	Directory    string `json:"directory"`
	Module       string `json:"module"`
	Version      string `json:"version"`
	Tag          string `json:"tag"`
	SourceCommit string `json:"sourceCommit"`
}

// ReleaseManifest 表示版本发布的完整清单，包含顶层版本、源码提交及各子模块条目。
type ReleaseManifest struct {
	Release      string        `json:"release"`
	SourceCommit string        `json:"sourceCommit"`
	Modules      []ModuleEntry `json:"modules"`
}

func main() {
	var manifestPath string
	var previousPath string
	var sourceRoot string
	var outputPath string
	var modulesFile string
	var versionStr string
	var commitStr string

	flag.StringVar(&manifestPath, "manifest", "", "Path to release manifest to validate")
	flag.StringVar(&previousPath, "previous", "", "Optional path to previous release manifest")
	flag.StringVar(&sourceRoot, "source-root", "", "Optional path to source repo root for go.mod verification")
	flag.StringVar(&outputPath, "output", "", "Optional output path to write normalized manifest (defaults to stdout)")
	flag.StringVar(&modulesFile, "modules-file", "", "Optional path to modules.txt to generate manifest from")
	flag.StringVar(&versionStr, "version", "", "Version for generated manifest (used with -modules-file)")
	flag.StringVar(&commitStr, "commit", "", "Source commit for generated manifest (used with -modules-file)")
	flag.Parse()

	var manifest *ReleaseManifest
	var err error

	if modulesFile != "" {
		if versionStr == "" || commitStr == "" {
			fmt.Fprintln(os.Stderr, "error: -version and -commit are required when using -modules-file")
			os.Exit(1)
		}
		manifest, err = generateFromModulesFile(modulesFile, versionStr, commitStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating manifest from %s: %v\n", modulesFile, err)
			os.Exit(1)
		}
	} else {
		if manifestPath == "" {
			fmt.Fprintln(os.Stderr, "error: -manifest or -modules-file is required")
			os.Exit(1)
		}
		manifest, err = readManifest(manifestPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading manifest: %v\n", err)
			os.Exit(1)
		}
	}

	var previous *ReleaseManifest
	if previousPath != "" {
		prev, err := readManifest(previousPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading previous manifest: %v\n", err)
			os.Exit(1)
		}
		previous = prev
	}

	normalized, err := ValidateManifest(manifest, previous, sourceRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "manifest validation failed: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serializing normalized manifest: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if outputPath != "" {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output file %s: %v\n", outputPath, err)
			os.Exit(1)
		}
	} else {
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "error writing to stdout: %v\n", err)
			os.Exit(1)
		}
	}
}

func readManifest(manifestPath string) (*ReleaseManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest ReleaseManifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid json in %s: %w", manifestPath, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("extra content after json in %s", manifestPath)
	}
	return &manifest, nil
}

func generateFromModulesFile(modulesPath, version, commit string) (*ReleaseManifest, error) {
	data, err := os.ReadFile(modulesPath)
	if err != nil {
		return nil, err
	}
	var entries []ModuleEntry
	lines := strings.Split(string(data), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			entries = append(entries, ModuleEntry{
				Directory:    parts[0],
				Module:       parts[1],
				Version:      version,
				Tag:          parts[0] + "/" + version,
				SourceCommit: commit,
			})
		}
	}
	return &ReleaseManifest{
		Release:      version,
		SourceCommit: commit,
		Modules:      entries,
	}, nil
}

// ValidateManifest 校验发布清单的合法性，可选比对上一版本清单并验证源码树中的 go.mod 声明。
func ValidateManifest(manifest *ReleaseManifest, previous *ReleaseManifest, sourceRoot string) (*ReleaseManifest, error) {
	if manifest == nil {
		return nil, errors.New("manifest is nil")
	}

	if !semverV2Regex.MatchString(manifest.Release) {
		return nil, fmt.Errorf("invalid release version format: %q", manifest.Release)
	}

	manifestCommit := strings.ToLower(strings.TrimSpace(manifest.SourceCommit))
	if !commitHashRegex.MatchString(manifestCommit) {
		return nil, fmt.Errorf("invalid sourceCommit in release: %q (must be 40-char full hex sha, not HEAD or branch)", manifest.SourceCommit)
	}

	if len(manifest.Modules) != len(canonicalModules) {
		return nil, fmt.Errorf("expected exactly %d modules, got %d", len(canonicalModules), len(manifest.Modules))
	}

	seenDirs := make(map[string]bool)
	seenModules := make(map[string]bool)
	seenTags := make(map[string]bool)

	var normalizedModules []ModuleEntry

	for _, entry := range manifest.Modules {
		rawDir := entry.Directory
		if strings.Contains(rawDir, "\\") {
			return nil, fmt.Errorf("directory %q contains backslashes", rawDir)
		}
		if path.IsAbs(rawDir) || filepath.IsAbs(rawDir) {
			return nil, fmt.Errorf("directory %q must not be absolute", rawDir)
		}
		cleanDir := path.Clean(rawDir)
		if cleanDir != rawDir {
			return nil, fmt.Errorf("directory %q is not in canonical path form (cleaned: %q)", rawDir, cleanDir)
		}
		if cleanDir == "." || cleanDir == ".." || strings.HasPrefix(cleanDir, "../") {
			return nil, fmt.Errorf("invalid directory path: %q", rawDir)
		}

		expectedMod, ok := canonicalModules[cleanDir]
		if !ok {
			return nil, fmt.Errorf("directory %q is not in the allowed 14 v2 modules allowlist", cleanDir)
		}

		if entry.Module != expectedMod {
			return nil, fmt.Errorf("module mismatch for %q: expected %q, got %q", cleanDir, expectedMod, entry.Module)
		}

		if seenDirs[cleanDir] {
			return nil, fmt.Errorf("duplicate directory %q in manifest", cleanDir)
		}
		seenDirs[cleanDir] = true

		if seenModules[entry.Module] {
			return nil, fmt.Errorf("duplicate module %q in manifest", entry.Module)
		}
		seenModules[entry.Module] = true

		if !semverV2Regex.MatchString(entry.Version) {
			return nil, fmt.Errorf("invalid module version %q in %s: must be valid v2 semver", entry.Version, cleanDir)
		}

		expectedTag := cleanDir + "/" + entry.Version
		if entry.Tag != expectedTag {
			return nil, fmt.Errorf("invalid tag %q for %s: expected exact %q (dir/version, without extra v2)", entry.Tag, cleanDir, expectedTag)
		}

		if seenTags[entry.Tag] {
			return nil, fmt.Errorf("duplicate tag %q in manifest", entry.Tag)
		}
		seenTags[entry.Tag] = true

		modCommit := strings.ToLower(strings.TrimSpace(entry.SourceCommit))
		if !commitHashRegex.MatchString(modCommit) {
			return nil, fmt.Errorf("invalid sourceCommit %q in %s: must be 40-char full hex sha", entry.SourceCommit, cleanDir)
		}

		if sourceRoot != "" {
			modFilePath := filepath.Join(sourceRoot, filepath.FromSlash(cleanDir), "go.mod")
			declaredMod, err := readDeclaredModule(modFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed verifying go.mod at %s: %w", modFilePath, err)
			}
			if declaredMod != expectedMod {
				return nil, fmt.Errorf("go.mod at %s declares module %q, expected %q", modFilePath, declaredMod, expectedMod)
			}
		}

		normalizedModules = append(normalizedModules, ModuleEntry{
			Directory:    cleanDir,
			Module:       expectedMod,
			Version:      entry.Version,
			Tag:          expectedTag,
			SourceCommit: modCommit,
		})
	}

	if len(seenDirs) != len(canonicalModules) {
		return nil, fmt.Errorf("missing modules: expected %d, got %d", len(canonicalModules), len(seenDirs))
	}

	if previous != nil {
		validPrev, err := ValidateManifest(previous, nil, "")
		if err != nil {
			return nil, fmt.Errorf("previous manifest is invalid: %w", err)
		}

		prevByTag := make(map[string]ModuleEntry, len(validPrev.Modules))
		for _, prevEntry := range validPrev.Modules {
			prevByTag[prevEntry.Tag] = prevEntry
		}

		for _, cur := range normalizedModules {
			if prev, exists := prevByTag[cur.Tag]; exists {
				if prev.Module != cur.Module {
					return nil, fmt.Errorf("module identity mutation for tag %q: was %q, now %q", cur.Tag, prev.Module, cur.Module)
				}
				if prev.Version != cur.Version {
					return nil, fmt.Errorf("version mutation for tag %q: was %q, now %q", cur.Tag, prev.Version, cur.Version)
				}
				if prev.SourceCommit != cur.SourceCommit {
					return nil, fmt.Errorf("same-tag sourceCommit mutation rejected for %q: previously %s, now %s (must never alter published identity)", cur.Tag, prev.SourceCommit, cur.SourceCommit)
				}
			}
		}
	}

	sort.Slice(normalizedModules, func(i, j int) bool {
		return normalizedModules[i].Directory < normalizedModules[j].Directory
	})

	return &ReleaseManifest{
		Release:      manifest.Release,
		SourceCommit: manifestCommit,
		Modules:      normalizedModules,
	}, nil
}

func readDeclaredModule(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("no module declaration in %s", goModPath)
}

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveRepoContractAndWhitelist(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	report, err := CheckRepo(repoRoot, DefaultWhitelist())
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	if report.ModuleCount != 14 {
		t.Fatalf("expected 14 workspace modules, got %d", report.ModuleCount)
	}
	if len(report.Violations) > 0 {
		for _, v := range report.Violations {
			t.Errorf("violation: [%s] %s (%s) @ %s — %s", v.Kind, v.Symbol, v.PkgPath, v.Location, v.Detail)
		}
	}
	if len(report.UnusedWhitelist) > 0 {
		for _, u := range report.UnusedWhitelist {
			t.Errorf("unused whitelist entry: [%s] %s (%s)", u.ID, u.Symbol, u.PkgPath)
		}
	}
}

func TestWhitelistValidationRejectsGlobsAndDuplicates(t *testing.T) {
	cases := []struct {
		name    string
		entries []WhitelistEntry
	}{
		{
			name: "glob star in symbol",
			entries: []WhitelistEntry{
				{ID: "W99", Contract: "c", Kind: KindMethod, PkgPath: "foundation/apperr", Symbol: "(*Error).*", Reason: "r"},
			},
		},
		{
			name: "glob wildcard in pkg",
			entries: []WhitelistEntry{
				{ID: "W99", Contract: "c", Kind: KindFunc, PkgPath: "plugins/*", Symbol: "Foo", Reason: "r"},
			},
		},
		{
			name: "question mark wildcard",
			entries: []WhitelistEntry{
				{ID: "W99", Contract: "c", Kind: KindFunc, PkgPath: "foundation/apperr", Symbol: "Foo?", Reason: "r"},
			},
		},
		{
			name: "duplicate entries",
			entries: []WhitelistEntry{
				{ID: "W01", Contract: "c", Kind: KindMethod, PkgPath: "foundation/apperr", Symbol: "(*Error).Unwrap", Reason: "r1"},
				{ID: "W01", Contract: "c", Kind: KindMethod, PkgPath: "foundation/apperr", Symbol: "(*Error).Unwrap", Reason: "r2"},
			},
		},
		{
			name: "missing reason",
			entries: []WhitelistEntry{
				{ID: "W01", Contract: "c", Kind: KindMethod, PkgPath: "foundation/apperr", Symbol: "(*Error).Unwrap", Reason: ""},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateWhitelist(tc.entries); err == nil {
				t.Fatalf("expected ValidateWhitelist error for %s, got nil", tc.name)
			}
		})
	}
}

func TestCheckerRejectsBypassPatterns(t *testing.T) {
	tmp := t.TempDir()
	goMod := "module github.com/bamboo-services/bamboo-base-go/fixture/v2\n\ngo 1.25.0\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	badGo := `package fixture

type ErrAlias = error

type CustomErr error

type CustomIface interface {
	Error() string
}

type WorkerIface interface {
	Run() error
}

type HookFn func() error

type BadHolder struct {
	Cause error
	Done  chan error
}

type impl struct{}

func (i *impl) Run() error { return nil }

func DirectError() error { return nil }

func AliasError() ErrAlias { return nil }

func CustomErrReturn() CustomErr { return nil }

func CustomIfaceReturn() CustomIface { return nil }

func CallbackParam(cb func() error) {
	_ = cb
}

func RunAnon() {
	fn := func() error { return nil }
	_ = fn
}
`
	if err := os.WriteFile(filepath.Join(tmp, "bad.go"), []byte(badGo), 0o600); err != nil {
		t.Fatalf("write bad.go: %v", err)
	}

	staleWhitelist := []WhitelistEntry{
		{
			ID:       "W99",
			Contract: "stale",
			Kind:     KindFunc,
			PkgPath:  "fixture",
			Symbol:   "DoesNotExist",
			Reason:   "test unused detection",
		},
	}

	report, err := CheckPatterns(tmp, []string{"."}, staleWhitelist)
	if err != nil {
		t.Fatalf("CheckPatterns failed: %v", err)
	}

	wantSymbols := []string{
		"ErrAlias",
		"CustomErr",
		"CustomIface",
		"WorkerIface.Run",
		"HookFn",
		"BadHolder.Cause",
		"BadHolder.Done",
		"(*impl).Run",
		"DirectError",
		"AliasError",
		"CustomErrReturn",
		"CustomIfaceReturn",
		"CallbackParam:param-callback",
		"RunAnon/lit#1",
	}

	got := make(map[string]bool, len(report.Violations))
	for _, v := range report.Violations {
		got[v.Symbol] = true
	}
	for _, want := range wantSymbols {
		if !got[want] {
			t.Errorf("expected violation for symbol %q, got violations: %+v", want, report.Violations)
		}
	}

	if len(report.UnusedWhitelist) != 1 || report.UnusedWhitelist[0].Symbol != "DoesNotExist" {
		t.Fatalf("expected 1 unused whitelist entry DoesNotExist, got %+v", report.UnusedWhitelist)
	}
}

func TestCompileNegativeUntypedErrorCannotSatisfyStrongSignature(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	foundationDir := filepath.Join(repoRoot, "foundation")

	tmp := t.TempDir()
	goMod := "module example.com/negcompile\n\ngo 1.25.0\n\nrequire github.com/bamboo-services/bamboo-base-go/foundation/v2 v2.0.0\n\nreplace github.com/bamboo-services/bamboo-base-go/foundation/v2 => " + foundationDir + "\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	src := `package negcompile

import (
	"errors"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
)

func badExit() *apperr.Error {
	return errors.New("raw untyped error")
}
`
	if err := os.WriteFile(filepath.Join(tmp, "neg.go"), []byte(src), 0o600); err != nil {
		t.Fatalf("write neg.go: %v", err)
	}

	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected compile failure when returning errors.New as *apperr.Error, but go build succeeded")
	}
	if !strings.Contains(string(out), "cannot use") {
		t.Fatalf("expected type mismatch compile error, got: %s", string(out))
	}
}

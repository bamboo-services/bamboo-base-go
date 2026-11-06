package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVocabularyValidation(t *testing.T) {
	first := errorEntry{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "active"}
	second := errorEntry{50001, "FRAMEWORK_COMPONENT_NOT_FOUND", "foundation/apperr", "FrameworkComponentNotFound", "组件未找到", "active"}
	tests := []struct {
		name   string
		mutate func([]errorEntry) []errorEntry
	}{
		{"duplicate code", func(e []errorEntry) []errorEntry { e[1].code = e[0].code; return e }},
		{"duplicate output", func(e []errorEntry) []errorEntry { e[1].output = e[0].output; return e }},
		{"duplicate symbol", func(e []errorEntry) []errorEntry { e[1].name = e[0].name; return e }},
		{"wrong owner", func(e []errorEntry) []errorEntry { e[0].owner = "plugins/unknown"; return e }},
		{"wrong prefix", func(e []errorEntry) []errorEntry { e[0].output = "NOT_FOUND"; return e }},
		{"borrowed prefix", func(e []errorEntry) []errorEntry {
			e[0].owner, e[0].output, e[0].name = "plugins/web-gin/httpresult", "HTTP_BIND_PARAMETER_ERROR", "ParameterError"
			return e
		}},
		{"framework client group", func(e []errorEntry) []errorEntry { e[0].code = 40000; return e }},
		{"framework generic symbol", func(e []errorEntry) []errorEntry { e[0].name = "InternalError"; return e }},
		{"business reserved slot", func(e []errorEntry) []errorEntry { e[1].code = 50050; return e }},
		{"invalid group", func(e []errorEntry) []errorEntry { e[0].code = 60000; return e }},
		{"gap", func(e []errorEntry) []errorEntry { e[1].code = 50002; return e }},
		{"out of order", func(e []errorEntry) []errorEntry { return []errorEntry{e[1], e[0]} }},
		{"empty message", func(e []errorEntry) []errorEntry { e[0].message = ""; return e }},
		{"invalid state", func(e []errorEntry) []errorEntry { e[0].state = "deleted"; return e }},
		{"unexported symbol", func(e []errorEntry) []errorEntry { e[1].name = "frameworkError"; return e }},
	}
	if err := validateVocabulary([]errorEntry{first, second}); err != nil {
		t.Fatalf("valid entries: %v", err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateVocabulary(tc.mutate([]errorEntry{first, second})); err == nil {
				t.Fatal("expected invalid vocabulary to fail closed")
			}
		})
	}
}

func TestRepositoryDefinitions(t *testing.T) {
	entry := errorEntry{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "active"}
	declaration := `var FrameworkInternalError = &ErrorCode{Code: 50000, Output: "FRAMEWORK_INTERNAL_ERROR", Message: "框架内部错误"}`
	tests := []struct {
		name   string
		source string
		extra  string
		wantOK bool
	}{
		{"exact match", declaration, "", true},
		{"missing definition", "", "", false},
		{"code drift", strings.Replace(declaration, "50000", "50001", 1), "", false},
		{"output drift", strings.Replace(declaration, "FRAMEWORK_INTERNAL_ERROR", "FRAMEWORK_UNKNOWN_ERROR", 1), "", false},
		{"message drift", strings.Replace(declaration, "框架内部错误", "另一种错误", 1), "", false},
		{"unregistered definition", declaration + `
var FrameworkExtra = &ErrorCode{Code: 50001, Output: "FRAMEWORK_EXTRA", Message: "其他错误"}`, "", false},
		{"alias", declaration + "\nvar FrameworkAlias = FrameworkInternalError", "", false},
		{"constructor", "var FrameworkInternalError = makeCode()", "", false},
		{"positional fields", `var FrameworkInternalError = &ErrorCode{50000, "FRAMEWORK_INTERNAL_ERROR", "框架内部错误"}`, "", false},
		{"duplicate fields", `var FrameworkInternalError = &ErrorCode{Code:50000, Code:50001, Output:"FRAMEWORK_INTERNAL_ERROR", Message:"框架内部错误"}`, "", false},
		{"outside catalog", "", declaration, false},
		{"outside alias", declaration, "var FrameworkAlias = FrameworkInternalError", false},
		{"octal numeric bypass", strings.Replace(declaration, "50000", "050000", 1), "", false},
		{"local literal bypass", declaration, `func another() { _ = &ErrorCode{Code:50001, Output:"FRAMEWORK_EXTRA", Message:"其他错误"} }`, false},
		{"type alias bypass", declaration, "type Another = ErrorCode", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := createFixture(t, []errorEntry{entry}, tc.source)
			if tc.extra != "" {
				writeFixture(t, root, "foundation/apperr/extra.go", "package apperr\n"+tc.extra)
			}
			err := checkRepository(root)
			if (err == nil) != tc.wantOK {
				t.Fatalf("checkRepository() = %v, wantOK %v", err, tc.wantOK)
			}
		})
	}
}

func TestRetiredEntryKeepsItsNumber(t *testing.T) {
	entries := []errorEntry{
		{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "retired"},
		{50001, "FRAMEWORK_COMPONENT_NOT_FOUND", "foundation/apperr", "FrameworkComponentNotFound", "组件未找到", "active"},
	}
	source := `var FrameworkComponentNotFound = &ErrorCode{Code:50001, Output:"FRAMEWORK_COMPONENT_NOT_FOUND", Message:"组件未找到"}`
	root := createFixture(t, entries, source)
	if err := checkRepository(root); err != nil {
		t.Fatalf("retired entry must reserve its slot: %v", err)
	}
	writeFixture(t, root, "foundation/apperr/error_code.go", "package apperr\n"+source+`
var FrameworkInternalError = &ErrorCode{Code:50000, Output:"FRAMEWORK_INTERNAL_ERROR", Message:"框架内部错误"}`)
	if err := checkRepository(root); err == nil {
		t.Fatal("retired declaration must be rejected")
	}
}

func TestVocabularyParserRejectsMalformedInput(t *testing.T) {
	for _, source := range []string{
		"", "<!-- error-code-vocabulary:end --><!-- error-code-vocabulary:start -->",
		"<!-- error-code-vocabulary:start --><!-- error-code-vocabulary:end -->",
		"<!-- error-code-vocabulary:start -->\n|50000|FRAMEWORK_ERROR|\n<!-- error-code-vocabulary:end -->",
		"<!-- error-code-vocabulary:start -->\nnot a table\n<!-- error-code-vocabulary:end -->",
	} {
		if _, err := parseVocabulary(source); err == nil {
			t.Fatalf("expected malformed source to fail: %q", source)
		}
	}
}

func TestImportedDefinitionsCannotBypassCatalog(t *testing.T) {
	entry := errorEntry{50000, "FRAMEWORK_INTERNAL_ERROR", "foundation/apperr", "FrameworkInternalError", "框架内部错误", "active"}
	source := `var FrameworkInternalError = &ErrorCode{Code:50000, Output:"FRAMEWORK_INTERNAL_ERROR", Message:"框架内部错误"}`
	for name, extra := range map[string]string{
		"dot import": `import . "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var Extra = &ErrorCode{Code:50000, Output:"FRAMEWORK_INTERNAL_ERROR", Message:"框架内部错误"}`,
		"qualified alias": `import app "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var Extra = app.FrameworkInternalError`,
		"typed global": `import app "github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
var Extra *app.ErrorCode`,
	} {
		t.Run(name, func(t *testing.T) {
			root := createFixture(t, []errorEntry{entry}, source)
			writeFixture(t, root, "plugins/cache/extra.go", "package cache\n"+extra)
			if err := checkRepository(root); err == nil {
				t.Fatal("unregistered imported definition must fail closed")
			}
		})
	}
}

func createFixture(t *testing.T, entries []errorEntry, source string) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"application", "foundation/apperr", "plugins"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var table strings.Builder
	table.WriteString("<!-- error-code-vocabulary:start -->\n")
	for _, entry := range entries {
		fmt.Fprintf(&table, "| %d | `%s` | `%s` | `%s` | %s | %s |\n", entry.code, entry.output, entry.owner, entry.name, entry.message, entry.state)
	}
	table.WriteString("<!-- error-code-vocabulary:end -->\n")
	writeFixture(t, root, vocabularyPath, table.String())
	writeFixture(t, root, "foundation/apperr/error_code.go", "package apperr\n"+source)
	return root
}

func writeFixture(t *testing.T, root, path, source string) {
	t.Helper()
	path = filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

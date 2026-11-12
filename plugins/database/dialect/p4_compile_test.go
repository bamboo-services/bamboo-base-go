package dialect_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestP4CompileContract(t *testing.T) {
	for _, tc := range []struct {
		name        string
		diagnostics []string
	}{
		{"positive", nil},
		{"negative_adapter", []string{"wrong type for method Open", "*apperr.Error"}},
		{"negative_return", []string{"cannot use", "*apperr.Error"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, "go", "build", "./testdata/p4_compile/"+tc.name).CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("compiler timed out: %s", output)
			}
			if len(tc.diagnostics) == 0 {
				if err != nil {
					t.Fatalf("positive fixture/environment failed: %v\n%s", err, output)
				}
				return
			}
			if err == nil {
				t.Fatal("legacy/native error fixture unexpectedly compiles")
			}
			text := string(output)
			for _, marker := range []string{"dial tcp", "connection refused", "missing go.sum", "no required module", "cannot find module"} {
				if strings.Contains(text, marker) {
					t.Fatalf("environment failure is not a negative proof: %s", text)
				}
			}
			for _, want := range tc.diagnostics {
				if !strings.Contains(text, want) {
					t.Fatalf("missing diagnostic %q: %s", want, text)
				}
			}
		})
	}
}

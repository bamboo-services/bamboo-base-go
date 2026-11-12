package postgres_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/postgres/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"gorm.io/gorm"
)

// sensitivePassword 用于验证脱敏的敏感口令。
const sensitivePassword = "SuperSecretPasswordP@ss123!"

var _ func(dialect.Adapter) (gorm.Dialector, *apperr.Error) = dialect.Adapter.Open

// TestOpen_SignatureContract 验证 Open 的签名为 (gorm.Dialector, *apperr.Error)。
func TestOpen_SignatureContract(t *testing.T) {
	ad := postgres.New("host=localhost user=postgres password=test dbname=postgres port=5432 sslmode=disable")

	open := ad.Open
	dialector, err := open()
	if err != nil {
		t.Fatalf("unexpected Open() error: %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}
}

// TestOpen_ConfigErrorMapping 逐分支验证配置非法错误映射到 dialect.AdapterConfigInvalid，
// 并保留脱敏后的根因说明、错误码与安全中文描述。
func TestOpen_ConfigErrorMapping(t *testing.T) {
	tests := []struct {
		name        string
		ad          dialect.Adapter
		wantMessage string
		wantCause   string
	}{
		{
			name:        "empty DSN",
			ad:          postgres.New("   "),
			wantMessage: "PostgreSQL 连接串不能为空",
			wantCause:   "dsn cannot be empty",
		},
		{
			name:        "URL without connection info",
			ad:          postgres.New("postgres://"),
			wantMessage: "PostgreSQL 连接串格式非法：URL 缺少连接信息",
			wantCause:   "invalid DSN format",
		},
		{
			name:        "URL without host",
			ad:          postgres.New("postgres://postgres:" + sensitivePassword + "@/dbname"),
			wantMessage: "PostgreSQL 连接串格式非法：主机地址为空",
			wantCause:   "host cannot be empty",
		},
		{
			name:        "plain text DSN",
			ad:          postgres.New("invalid_dsn_plain_text_" + sensitivePassword),
			wantMessage: "PostgreSQL 连接串格式非法：需为 URL 或 key=value 格式",
			wantCause:   "invalid DSN format",
		},
		{
			name: "empty port from env",
			ad: func() dialect.Adapter {
				_ = os.Unsetenv("DATABASE_DSN")
				t.Setenv("DATABASE_PASS", sensitivePassword)
				t.Setenv("DATABASE_PORT", "   ")
				return postgres.FromEnv()
			}(),
			wantMessage: "PostgreSQL 端口配置不能为空",
			wantCause:   "port cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialector, err := tt.ad.Open()
			if err == nil {
				t.Fatal("expected Open() to return error, got nil")
			}
			if dialector != nil {
				t.Fatalf("expected nil dialector on error, got %v", dialector)
			}
			if err.Code != *dialect.AdapterConfigInvalid {
				t.Fatalf("unexpected error code: got %+v, want %+v", err.Code, *dialect.AdapterConfigInvalid)
			}
			if err.Code.Code != 50036 {
				t.Fatalf("unexpected numeric code: got %d, want 50036", err.Code.Code)
			}
			if string(err.Code.Output) != "DATABASE_DIALECT_ADAPTER_CONFIG_INVALID" {
				t.Fatalf("unexpected output: got %q", err.Code.Output)
			}
			if string(err.ErrorMessage) != tt.wantMessage {
				t.Fatalf("unexpected message: got %q, want %q", err.ErrorMessage, tt.wantMessage)
			}
			if err.Err == nil {
				t.Fatal("expected desensitized root cause to be preserved")
			}
			if !strings.Contains(err.Err.Error(), tt.wantCause) {
				t.Fatalf("root cause mismatch: got %q, want contains %q", err.Err.Error(), tt.wantCause)
			}
			assertNoPasswordLeak(t, err)
		})
	}
}

// TestOpen_DeferredErrorIsStable 验证延迟交付的错误在重复 Open 时保持稳定。
func TestOpen_DeferredErrorIsStable(t *testing.T) {
	ad := postgres.New("invalid_dsn_plain_text_" + sensitivePassword)

	_, first := ad.Open()
	_, second := ad.Open()
	if first == nil || second == nil {
		t.Fatal("expected both Open() calls to return error")
	}
	if first != second {
		t.Fatal("expected the same deferred error instance across repeated Open() calls")
	}
	if first.Code != second.Code || first.ErrorMessage != second.ErrorMessage {
		t.Fatalf("deferred error drifted: %+v vs %+v", first, second)
	}
}

// TestOpen_SuccessReturnsNilError 验证成功路径返回真正的 nil 错误与非空 Dialector。
func TestOpen_SuccessReturnsNilError(t *testing.T) {
	ad := postgres.New("host=localhost user=postgres password=test dbname=postgres port=5432 sslmode=disable")

	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("expected nil error on success, got %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}
}

// TestOpen_NoNetworkAccess 验证构造与 Open 阶段均不触发网络连接。
func TestOpen_NoNetworkAccess(t *testing.T) {
	ad := postgres.New("host=192.0.2.1 user=postgres password=pass dbname=postgres port=5432 sslmode=disable")

	dialector, err := ad.Open()
	if err != nil {
		t.Fatalf("Open should not perform network access, got error: %v", err)
	}
	if dialector == nil {
		t.Fatal("expected non-nil dialector")
	}
}

// assertNoPasswordLeak 验证错误文本、补充描述与 JSON 序列化均不包含敏感口令。
func assertNoPasswordLeak(t *testing.T, err *apperr.Error) {
	t.Helper()

	if strings.Contains(err.Error(), sensitivePassword) {
		t.Fatalf("error text leaked password: %q", err.Error())
	}
	if strings.Contains(string(err.ErrorMessage), sensitivePassword) {
		t.Fatalf("error message leaked password: %q", err.ErrorMessage)
	}
	if err.Err != nil && strings.Contains(err.Err.Error(), sensitivePassword) {
		t.Fatalf("root cause leaked password: %q", err.Err.Error())
	}
	payload, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("failed to marshal error: %v", marshalErr)
	}
	if strings.Contains(string(payload), sensitivePassword) {
		t.Fatalf("json payload leaked password: %q", string(payload))
	}
}

// TestOpen_ErrorChainPreserved 验证错误链完整：结构化错误可被 Find 命中，
// 脱敏后的根因可通过 Unwrap 与 errors.Is 追溯。
func TestOpen_ErrorChainPreserved(t *testing.T) {
	ad := postgres.New("invalid_plain_text_dsn")

	_, err := ad.Open()
	if err == nil {
		t.Fatal("expected Open() to return error, got nil")
	}
	if err.Err == nil {
		t.Fatal("expected desensitized root cause to be preserved")
	}
	if errors.Unwrap(err) != err.Err {
		t.Fatal("expected Unwrap to expose the root cause")
	}
	if !errors.Is(err, err.Err) {
		t.Fatal("expected errors.Is to match the root cause")
	}
	found := apperr.Find(err)
	if found == nil {
		t.Fatal("expected apperr.Find to locate the structured error")
	}
	if found.Code != err.Code {
		t.Fatalf("unexpected code from apperr.Find: got %+v, want %+v", found.Code, err.Code)
	}
}

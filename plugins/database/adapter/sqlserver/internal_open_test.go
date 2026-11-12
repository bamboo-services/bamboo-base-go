package sqlserver

import (
	"strings"
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"gorm.io/gorm"
)

// 持久编译断言：锁定适配器内部错误字段与关键方法的强类型签名。
var (
	_ *apperr.Error                          = (&adapter{}).err
	_ func() (gorm.Dialector, *apperr.Error) = (&adapter{}).Open
	_ func(string) *apperr.Error             = validateDSN
	_ func() string                          = (&adapter{}).Name
	_ func() (gorm.Dialector, *apperr.Error) = New("sqlserver://sa:pass@localhost:1433").Open
	_ dialect.Adapter                        = (*adapter)(nil)
)

// TestOpen_NilReceiver 验证 nil 接收器调用 Open 时返回契约错误而非 panic。
func TestOpen_NilReceiver(t *testing.T) {
	var a *adapter

	dialector, err := a.Open()
	if err == nil {
		t.Fatal("expected error from nil receiver, got nil")
	}
	if dialector != nil {
		t.Fatalf("expected nil dialector, got %v", dialector)
	}
	if err.Code != *dialect.AdapterInvalid {
		t.Fatalf("unexpected error code: got %+v, want %+v", err.Code, *dialect.AdapterInvalid)
	}
	if err.Code.Code != 50035 {
		t.Fatalf("unexpected numeric code: got %d, want 50035", err.Code.Code)
	}
	if string(err.Code.Output) != "DATABASE_DIALECT_ADAPTER_INVALID" {
		t.Fatalf("unexpected output: got %q", err.Code.Output)
	}
	if !strings.Contains(string(err.ErrorMessage), "适配器实例为空") {
		t.Fatalf("unexpected message: got %q", err.ErrorMessage)
	}
	if err.Err == nil {
		t.Fatal("expected root cause to explain the nil receiver")
	}
}

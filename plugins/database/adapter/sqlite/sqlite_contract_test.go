package sqlite

import (
	"testing"

	"github.com/bamboo-services/bamboo-base-go/foundation/v2/apperr"
	"github.com/bamboo-services/bamboo-base-go/plugins/database/v2/dialect"
	"gorm.io/gorm"
)

// 编译期断言：适配器内部错误字段必须是强类型 *apperr.Error，
// 禁止退化为原生 error 以免丢失错误码与业务数据。
var _ *apperr.Error = (&adapter{}).err

// 编译期断言：Open 方法签名必须与 [dialect.Adapter] 契约完全一致，
// 返回强类型错误指针而不是原生 error。
var _ func(*adapter) (gorm.Dialector, *apperr.Error) = (*adapter).Open

// 编译期断言：适配器必须实现 [dialect.Adapter] 契约。
var _ dialect.Adapter = (*adapter)(nil)

// 说明：SQLite 适配器刻意不提供 validateDSN 之类的 DSN 字符串格式校验函数。
// SQLite 的 DSN 既可以是 ":memory:"，也可以是文件路径或带查询串的 URI，
// 任何格式判断都会误伤合法用法，因此本包只对"空白 DSN/空白路径"这一类
// 配置缺失做拒绝，其余字符串一律原样交给驱动处理。

// TestOpen_NilReceiverUsesAdapterInvalid 验证 nil 接收器调用 Open 时
// 返回适配器无效错误码，且不返回任何 Dialector。
func TestOpen_NilReceiverUsesAdapterInvalid(t *testing.T) {
	var adp *adapter

	dialector, err := adp.Open()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if err.Code.Code != dialect.AdapterInvalid.Code {
		t.Fatalf("错误码 = %+v, 期望 %+v", err.Code, *dialect.AdapterInvalid)
	}
	if dialector != nil {
		t.Fatalf("失败时 dialector 应为 nil，实际为 %v", dialector)
	}
}

// TestOpen_BlankDSNUsesAdapterConfigInvalid 验证空白 DSN 使用配置非法错误码，
// 且错误原因不携带 DSN 明文。
func TestOpen_BlankDSNUsesAdapterConfigInvalid(t *testing.T) {
	for _, dsn := range []string{"", "   ", "\t\n"} {
		dialector, err := New(dsn).Open()
		if err == nil {
			t.Fatalf("DSN %q 期望返回错误，实际为 nil", dsn)
		}
		if err.Code.Code != dialect.AdapterConfigInvalid.Code {
			t.Fatalf("错误码 = %+v, 期望 %+v", err.Code, *dialect.AdapterConfigInvalid)
		}
		if dialector != nil {
			t.Fatalf("失败时 dialector 应为 nil，实际为 %v", dialector)
		}
	}
}

// TestNew_KeepsArbitraryDSNUnchanged 验证 SQLite 适配器不做 DSN 字符串格式校验：
// 任意非空白 DSN 都能构造出 Dialector。
func TestNew_KeepsArbitraryDSNUnchanged(t *testing.T) {
	for _, dsn := range []string{":memory:", "file:test.db?cache=shared", "任意 非标准 DSN"} {
		dialector, err := New(dsn).Open()
		if err != nil {
			t.Fatalf("DSN %q 期望构造成功，实际失败: %v", dsn, err)
		}
		if dialector == nil {
			t.Fatalf("DSN %q 期望返回非空 Dialector", dsn)
		}
	}
}

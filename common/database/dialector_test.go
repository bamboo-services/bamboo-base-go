package xDatabase_test

import (
	"testing"

	xDatabase "github.com/bamboo-services/bamboo-base-go/common/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// mockDialector 测试用最小 Dialector 实现，仅验证注册表存取。
type mockDialector struct{}

func (mockDialector) Name() string { return "mock" }
func (mockDialector) Initialize(*gorm.DB) error {
	return nil
}
func (mockDialector) Migrator(*gorm.DB) gorm.Migrator                       { return nil }
func (mockDialector) DataTypeOf(*schema.Field) string                       { return "" }
func (mockDialector) DefaultValueOf(*schema.Field) clause.Expression        { return nil }
func (mockDialector) BindVarTo(clause.Writer, *gorm.Statement, interface{}) {}
func (mockDialector) QuoteTo(clause.Writer, string)                         {}
func (mockDialector) Explain(sql string, vars ...interface{}) string        { return "" }

// mockFactory 返回 mockDialector 的工厂。
func mockFactory(dsn string) gorm.Dialector { return mockDialector{} }

// TestRegisterDialector_StoreAndFetch 验证注册后可经 DialectorFactory 取回工厂。
func TestRegisterDialector_StoreAndFetch(t *testing.T) {
	const d = xDatabase.Driver("test-mock")
	xDatabase.RegisterDialector(d, mockFactory)

	got, ok := xDatabase.GetDialectorFactory(d)
	if !ok {
		t.Fatal("注册后 DialectorFactory 未命中")
	}
	if got("dsn") == nil {
		t.Error("工厂返回 nil")
	}
}

// TestDialectorFactory_Unregistered 验证未注册驱动返回 ok=false。
func TestDialectorFactory_Unregistered(t *testing.T) {
	_, ok := xDatabase.GetDialectorFactory("no-such-driver")
	if ok {
		t.Error("未注册驱动应返回 ok=false")
	}
}

// TestRegisterDialector_DuplicatePanic 验证重复注册同一枚举会 panic。
func TestRegisterDialector_DuplicatePanic(t *testing.T) {
	const d = xDatabase.Driver("test-dup")
	xDatabase.RegisterDialector(d, mockFactory)

	defer func() {
		if recover() == nil {
			t.Error("重复注册应 panic")
		}
	}()
	xDatabase.RegisterDialector(d, mockFactory)
}

// TestRegisterDialector_InvalidPanic 验证注册空驱动或 none 驱动会 panic。
func TestRegisterDialector_InvalidPanic(t *testing.T) {
	cases := []xDatabase.Driver{"", xDatabase.DriverNone}
	for _, d := range cases {
		t.Run(string(d), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("注册驱动 %q 应 panic", d)
				}
			}()
			xDatabase.RegisterDialector(d, mockFactory)
		})
	}
}

// TestRegisterDialector_NilFactoryPanic 验证注册 nil 工厂会 panic。
func TestRegisterDialector_NilFactoryPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("注册 nil 工厂应 panic")
		}
	}()
	xDatabase.RegisterDialector(xDatabase.Driver("test-nil"), nil)
}

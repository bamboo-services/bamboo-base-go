package dialect

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm"
)

// Driver 代表支持的数据库方言驱动名称。
type Driver string

const (
	// DriverMySQL 代表 MySQL 数据库驱动名称。
	DriverMySQL Driver = "mysql"
	// DriverPostgres 代表 PostgreSQL 数据库驱动名称。
	DriverPostgres Driver = "postgres"
	// DriverSQLite 代表 SQLite 数据库驱动名称。
	DriverSQLite Driver = "sqlite"
	// DriverOracle 代表 Oracle 数据库驱动名称。
	DriverOracle Driver = "oracle"
	// DriverSQLServer 代表 SQL Server 数据库驱动名称。
	DriverSQLServer Driver = "sqlserver"
)

var (
	// ErrDriverNotLoaded 在请求的数据库驱动未被注册或未被加载时返回。
	ErrDriverNotLoaded = errors.New("dialect: driver not loaded")
	// ErrNilAdapter 在注册的驱动工厂返回了 nil 或 typed nil 适配器时返回。
	ErrNilAdapter = errors.New("dialect: factory returned nil adapter")
)

// Adapter 定义数据库驱动适配器接口，由各具体数据库驱动子模块实现。
//
// 该接口将数据库插件核心与具体驱动实现解耦。实现方应保证 [Adapter.Open]
// 仅完成 [gorm.Dialector] 的构造，不产生实际网络 I/O。
type Adapter interface {
	// Name 返回适配器对应的驱动名称，例如 "mysql" 或 "postgres"。
	Name() string
	// Open 构造并返回 [gorm.Dialector] 实例或在配置非法时返回错误。
	Open() (gorm.Dialector, error)
}

// Factory 代表用于构造 [Adapter] 实例的工厂函数类型。
type Factory func() Adapter

// registry 内部方言注册表结构，负责并发安全的驱动注册与查找。
type registry struct {
	mu        sync.RWMutex
	factories map[Driver]Factory
}

// newRegistry 创建一个独立的空注册表实例，主要用于测试以防污染全局表。
func newRegistry() *registry {
	return &registry{
		factories: make(map[Driver]Factory),
	}
}

// isTypedNil 判断接口底层是否为 nil（包含 typed nil）。
func isTypedNil(a Adapter) bool {
	if a == nil {
		return true
	}
	v := reflect.ValueOf(a)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

func validateDriver(driver Driver) error {
	trimmed := strings.TrimSpace(string(driver))
	if trimmed == "" {
		return errors.New("dialect: driver cannot be empty")
	}
	if strings.EqualFold(trimmed, "none") {
		return errors.New("dialect: driver cannot be none")
	}
	return nil
}

// register 向当前注册表注册驱动工厂。
func (r *registry) register(driver Driver, factory Factory) {
	if err := validateDriver(driver); err != nil {
		panic(err)
	}
	if factory == nil {
		panic(errors.New("dialect: factory cannot be nil"))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[driver]; exists {
		panic(fmt.Errorf("dialect: driver %q already registered", driver))
	}
	r.factories[driver] = factory
}

// lookup 从当前注册表查找并实例化驱动适配器，工厂调用不在锁内进行。
func (r *registry) lookup(driver Driver) (Adapter, error) {
	r.mu.RLock()
	factory, exists := r.factories[driver]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrDriverNotLoaded, driver)
	}

	adapter := factory()
	if isTypedNil(adapter) {
		return nil, ErrNilAdapter
	}
	return adapter, nil
}

// defaultRegistry 包级全局默认注册表，供 init 与包级函数使用。
var defaultRegistry = newRegistry()

// Register 将指定数据库方言的驱动工厂注册到全局注册表。
//
// 若 driver 为空、包含仅空白字符、等于 "none"、factory 为 nil，
// 或同一 driver 已被注册过，将直接触发 panic。该函数通常在各驱动适配器的 init 函数中调用。
func Register(driver Driver, factory Factory) {
	defaultRegistry.register(driver, factory)
}

// Lookup 根据驱动名称从全局注册表查找并调用工厂构造适配器。
//
// 若请求的驱动未注册，返回包含 [ErrDriverNotLoaded] 的错误；
// 若工厂函数返回 nil 或底层为 nil 的接口值，返回 [ErrNilAdapter]。
// 工厂构造调用在全局读写锁外执行，避免外部工厂内部逻辑造成锁重入或死锁。
func Lookup(driver Driver) (Adapter, error) {
	return defaultRegistry.lookup(driver)
}

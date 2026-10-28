// Package dialect 定义数据库方言适配器接口、驱动标识与全局注册表。
//
// 本包为应用框架提供统一的数据库驱动抽象契约，解耦底层具体 GORM 驱动实现
// （如 MySQL、PostgreSQL、SQLite、Oracle、SQL Server）。各具体驱动适配器
// 通过实现 [Adapter] 接口并在包初始化阶段向本包注册工厂，使得核心数据库插件
// 能够在运行时通过方言名称动态查找并装配对应的 GORM 方言。
//
// # 核心类型与契约
//
// 本包的核心抽象包括以下类型与接口：
//
//   - [Driver]：强类型数据库驱动标识字符串，预定义了主流数据库驱动常量
//     （[DriverMySQL]、[DriverPostgres]、[DriverSQLite]、[DriverOracle]、[DriverSQLServer]）。
//   - [Adapter]：数据库驱动适配器接口，约定驱动名称 [Adapter.Name] 与
//     构造方言实例的 [Adapter.Open] 方法。
//   - [Factory]：构造 [Adapter] 实例的工厂函数签名。
//
// # 驱动注册与查找
//
// 驱动通过全局注册表进行注册与查找：
//
//   - 驱动注册：各驱动适配器子包通常在自身的 init 函数中调用 [Register] 完成注册；
//     若驱动名称为空、为 "none"、工厂为 nil 或重复注册，将立即触发 panic。
//   - 动态查找：上层组件通过 [Lookup] 传入驱动标识查找对应适配器。
//     查找过程在只读锁保护下获取工厂，并在锁外执行工厂闭包构造 [Adapter] 实例，
//     避免在工厂执行期间持有锁引发死锁风险。
//
// # 错误语义
//
// 驱动查找可能返回以下哨兵错误：
//
//   - [ErrDriverNotLoaded]：请求的驱动未被注册或未被匿名导入加载时返回。
//   - [ErrNilAdapter]：已注册驱动的工厂函数返回了 nil 或包含类型信息的 typed nil 时返回。
//
// # 并发与使用边界
//
// 全局注册表内部使用读写互斥锁 [sync.RWMutex] 保护驱动映射表，支持多 goroutine 并发注册与查找。
// [Register] 应当在应用程序初始化或包 init 阶段调用，[Lookup] 可在任意生命周期阶段并发调用。
// 本包属于纯抽象接口与注册层，不建立任何底层网络连接，也不执行任何真实 SQL 操作。
package dialect

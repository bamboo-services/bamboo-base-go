// Package ctxkey 定义基于 [typed.String] 派生的强类型上下文键，
// 以及围绕这些键的请求元数据读取函数。
//
// 本包属于 foundation 领域原语基座，只负责键的表达、校验与编解码，
// 以及少量基础元数据的直接读取；键值的写入由协议层入口
// 与上层容器在装配共享组件时完成。
//
// # 核心类型
//
// [Key] 是上下文与插件标识键的统一载体：底层复用 [typed.String]
// 的受控键名语法校验与六大通道编解码，杜绝魔法字符串与命名漂移。
// [Exposer] 约定插件向容器交付已声明底层句柄的方式，
// 配合组件类键完成共享组件的注入。
//
// # 内置上下文键目录
//
// 目录覆盖两类键：
//
//   - 元数据键：[RequestKey]、[ErrorCodeKey]、[ErrorMessageKey] 与
//     [UserStartTimeKey]，由协议层在入口处写入请求标识与起始时间等
//     基础信息；
//   - 组件键：[DatabaseKey]、[RedisClientKey]、[SnowflakeNodeKey]、
//     [EmailClientKey] 与 [CacheManagerKey]，供上层容器注入对应共享
//     组件实例。
//
// [Nil] 是空键常量，等价于 [Key] 的零值。
//
// # 编解码契约
//
// [Key] 以值接收器满足 [typed.Operable]，以指针接收器满足
// [typed.Convertible]，覆盖 Text、Binary、JSON、XML（含属性）、
// SQL 与 CLI Flag 六大通道：
//
//   - 编码方向（Marshal*、[Key.Value]）先校验键名，空键或非法键
//     返回错误；
//   - 解码方向（Unmarshal*、[Key.Scan]、[Key.Set]）先在候选值上
//     完成校验再写回，失败时原键值保持不变；JSON null 与 SQL NULL
//     均被拒绝；
//   - 指针接收器方法在 nil 接收器上返回明确错误，不会 panic。
//
// # 元数据读取
//
// [IsDebugMode] 读取调试模式环境变量；[CalcOverheadTime] 仅在调试
// 模式下计算请求耗时（微秒）；[GetRequestKey] 与 [GetErrorMessage]
// 从上下文读取请求标识与错误消息字符串。这些函数对 nil 上下文、
// 键缺失与值类型不匹配均稳健处理，不返回错误。
//
// # 并发与资源边界
//
// [Key] 是不可变的字符串值类型，可并发读取与传递。
// 本包不持有外部资源与后台任务，全部函数在当前调用内完成。
package ctxkey

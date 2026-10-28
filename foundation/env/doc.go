// Package env 提供环境变量的强类型读取入口与框架预置配置键目录，
// 消除配置读取中的魔法字符串与命名漂移。
//
// # 核心类型与调用路径
//
// 键常量（如 [Env]、[Port]、[DatabaseDSN]）统一采用 [Key] 类型；
// [Key] 从 [typed.String] 派生，继承受控标识符校验与六通道编解码契约。
// [Lookup] 以 [Key] 为入参读取进程环境变量，返回 [typed.String]，
// 供 [typed.String.Int]、[typed.String.Bool]、[typed.String.Duration]
// 等方法链式解析为 Go 原生类型。
//
// # 读取语义
//
// [Lookup] 严格区分两种状态：
//
//   - 未设置：ok 为 false，值为空 [typed.String]；
//   - 设置为空串：ok 为 true，值同样为空 [typed.String]。
//
// 返回的原始文本完全保留用户输入，不做受控键名语法过滤，
// 连接串、带符号路径与密码等任意内容均可原样读取；
// 需要语法校验时，由调用方按需调用 [typed.String.IsValid]。
//
// # 预置键目录
//
// 框架级标准环境变量以 [Key] 常量分组预置：
//
//   - 系统与预算：[Env]、[AppName]、[Host]、[StartTimeout] 等；
//   - 数据库：[DatabaseDriver]、[DatabaseDSN]、[DatabasePath] 等；
//   - 缓存与 NoSQL：[NoSQLDriver]、[NoSQLHost]、[NoSQLPrefix] 等；
//   - 雪花算法：[SnowflakeDatacenterID]、[SnowflakeNodeID]；
//   - 日志：[LogLevel]、[LogPath]、[LogCompress] 等；
//   - 第三方服务：[SmsAccessKey]、[EmailHost]、[OssEndpoint] 等。
//
// # 典型用法
//
// 读取监听端口并解析为整数，未设置或解析失败时回退默认值：
//
//	port := 8080
//	if raw, ok := env.Lookup(env.Port); ok {
//		if v, err := raw.Int(); err == nil {
//			port = v
//		}
//	}
//
// # 键的编解码
//
// [Key] 与 [typed.String] 是不同的命名类型，相互赋值需显式转换。
// 各编解码方法与 [typed.String] 对应方法行为一致：编码前先校验键名，
// 解码仅在候选值校验成功后写回，失败保留原值；
// 指针接收器为 nil 时返回 "env.Key: nil receiver" 错误。
//
// # 并发与资源
//
// 本包无共享可变状态，[Lookup] 每次调用即时读取进程环境变量。
// 进程环境变量的并发修改需由调用方自行串行化。
package env

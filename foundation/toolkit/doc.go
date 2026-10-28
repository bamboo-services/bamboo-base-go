// Package toolkit 提供加解密、字符串、时间、类型解析与格式校验等
// 通用工具函数集合。
//
// 本包属于 foundation 层的领域原语基座，供 application 与 plugins 层
// 复用。全部能力以无状态的纯函数提供，调用方传入明文输入，
// 直接获得计算结果；不涉及配置读取、外部 I/O 与进程级副作用。
//
// # 核心能力与调用路径
//
// 按职责划分为九组：
//
//   - 通用泛型工具：[Ptr]、[Val]、[Contains] 与 [ToBool]；
//   - 哈希摘要：[SHA256]、[SHA256Bytes]、[MD5] 与 [MD5Bytes]；
//   - 密码加密与校验：[EncryptPassword]、[VerifyPassword] 及其
//     Must 与 String 便捷变体；
//   - 安全密钥：[GenerateLongKey]、[GenerateKey] 与 [VerifyKey]；
//   - 随机字符串：[RandomString]、[RandomUpperString] 与
//     [RandomNumberString]；
//   - 字符串处理：[Mask]、[Truncate]、[CamelToSnake]、[SnakeToCamel]、
//     [RemoveSpaces] 等；
//   - 时间处理：[Now]、[MustParse]、[StartOfWeek]、[DiffDays] 等，
//     以及 [TimeFormatDate] 等布局常量；
//   - 类型解析：[ParseInt]、[ParseUint]、[ParseFloat64]、[ParseBool]
//     与 [ParseString]；
//   - 格式校验与反射工具：[IsPhone]、[IsUUID]、[IsJSON]、
//     [GetFunctionName] 与 [GetMethodName]。
//
// # 典型用法
//
// 密码入库前经 [EncryptPasswordString] 加密，登录时用
// [IsPasswordValid] 比对；加密侧会先对明文做 Base64 预编码，
// 校验侧自动保持同一约定：
//
//	hash, err := toolkit.EncryptPasswordString("my-password")
//	if err != nil {
//		return err
//	}
//	ok := toolkit.IsPasswordValid("my-password", hash)
//
// 外部输入的动态值经 [ParseInt] 等函数转换为目标类型，
// 以 (值, 是否成功) 形式返回，避免 panic：
//
//	port, ok := toolkit.ParseInt(configValue)
//	if !ok {
//		// 处理非法输入
//	}
//
// # 输入与零值
//
// 类型解析与校验函数接受 any 输入：类型不支持、nil 或超出目标范围时
// 一律返回失败，不触发 panic。[Val] 对 nil 指针返回目标类型的零值。
// 字符串截断与脱敏函数按字节处理，多字节字符的边界行为见各函数说明。
//
// # 错误处理
//
// 校验类函数（IsXxx、VerifyXxx）直接返回布尔值，不区分失败原因。
// 解析类函数以第二个 bool 返回值表示成败。仅 [EncryptPassword]、
// [EncryptPasswordString] 与 [Parse] 返回 error；对应的 Must 前缀
// 变体（[MustEncryptPassword]、[MustParse] 等）在失败时 panic，
// 只应用于初始化阶段已保证合法的输入。
//
// # 并发与资源边界
//
// 本包所有函数均为纯计算，不持有内部可变状态，可被多个 goroutine
// 并发调用，无需额外同步。函数内部不打开文件、网络连接或后台任务，
// 调用方自行管理上下游资源的生命周期。
//
// # 使用边界
//
// [MD5] 已不具备抗碰撞安全性，仅用于缓存键、指纹比对等非安全场景；
// 安全摘要请使用 [SHA256]。[RandomString] 系列基于 math/rand
// 全局源，适用于验证码等非安全场景；密码学随机需求请使用 crypto/rand。
// 校验类函数基于简化规则（正则或结构检查），覆盖常见合法输入，
// 不等同于对应国家标准的完整校验，具体边界见各函数说明。
package toolkit

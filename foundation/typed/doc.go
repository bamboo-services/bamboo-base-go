// Package typed 提供强类型字符串原语 [String]，既可作为受控机器键、
// 常量词汇的基底，也可作为环境变量等原始文本的解析载体，
// 是 foundation 层各领域键类型的公共契约。
//
// 本包仅依赖标准库，无内部状态与后台资源。
//
// # 核心类型与调用路径
//
// [String] 以原生 string 为基底，零值为空字符串。
// 受控词汇场景先以 [String.IsValid] 校验语法，再经 [String.MarshalText]、
// [String.MarshalJSON] 等编码方法输出；外部输入经 [String.UnmarshalText]、
// [String.Scan] 等解码方法完成等价校验后写回。
// 原始文本场景直接使用 [String.Int]、[String.Bool]、[String.Duration]
// 等方法解析为 Go 原生类型，不强制键名语法。
//
// [Operable] 约束领域词汇派生类型的基础操作；
// [Convertible] 汇总六通道编解码契约；*String 及其派生类型实现这组契约后，
// 即可与标准库序列化设施协作。
//
// # 受控标识符语法
//
// [String.IsValid] 按字节判定，全部条件满足才视为合法：
//
//   - 长度：1 至 [MaxKeyBytes] 字节；
//   - 首字符：ASCII 字母（A-Z 或 a-z）；
//   - 其余字符：ASCII 字母、数字、下划线 '_' 或连字符 '-'；
//   - 末字符：不能是下划线或连字符；
//   - 分隔符：不能连续出现（"__"、"--"、"_-"、"-_" 均非法）。
//
// 校验按原始字节执行，不忽略首尾空白，非 ASCII 内容一律不合法。
//
// # 六通道编解码契约
//
// 编码方法 [String.MarshalText]、[String.MarshalBinary]、
// [String.MarshalJSON]、[String.MarshalXML]、[String.MarshalXMLAttr] 与
// [String.Value] 使用值接收器，输出前先执行 [String.IsValid] 校验，
// 不合法时返回错误且不产生部分输出。
//
// 解码方法 [String.UnmarshalText]、[String.UnmarshalBinary]、
// [String.UnmarshalJSON]、[String.UnmarshalXML]、[String.UnmarshalXMLAttr]、
// [String.Scan] 与 [String.Set] 使用指针接收器，仅当候选值通过
// [String.IsValid] 校验后才写回，任何失败均不改变接收器原值。
// 各通道的额外约束：JSON 拒绝 null 与非字符串，SQL 仅接受 string 与
// []byte 且拒绝 NULL，XML 元素拒绝嵌套子元素，CLI Flag 由 [String.Set]
// 与 [String.String] 共同实现 [flag.Value]。
// 接收器为 nil 时返回 [ErrNilReceiver]，可用 errors.Is 识别。
//
// # 原生类型解析
//
// [String.Int]、[String.Uint8]、[String.Float64]、[String.Bool]、
// [String.Duration] 等方法先去除首尾空白，再按目标类型的严格位宽解析，
// 失败时原样返回 strconv 与 time 的下游错误。
// 布尔解析忽略大小写，真值接受 "true"、"1"、"yes"、"on"、"enabled"，
// 假值接受 "false"、"0"、"no"、"off"、"disabled"，其余输入报错。
//
// # 典型用法
//
// 从文本恢复一个合法键，再编码为 JSON 字符串：
//
//	func encodeKey(raw string) ([]byte, error) {
//		var value typed.String
//		if err := value.UnmarshalText([]byte(raw)); err != nil {
//			return nil, err
//		}
//		return value.MarshalJSON()
//	}
//
// # 并发与资源
//
// [String] 是不可变值类型，未被写回的值可在多个 goroutine 间安全共享；
// 解码方法经指针接收器写回，跨 goroutine 复用同一变量时由调用方同步。
package typed

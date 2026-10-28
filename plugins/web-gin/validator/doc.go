// Package validator 提供基于 go-playground/validator/v10 的独立参数验证引擎与中文化支持。
//
// 本包为 Web 插件提供私有隔离的参数校验器 [Engine]，内建针对中文环境的默认错误翻译，
// 并扩展了 URL、UUID、枚举、雪花算法 ID 及正则表达式等常用领域验证规则。
// 校验引擎严格禁止使用全局单例，避免不同应用插件实例或租户之间的规则污染。
//
// # 核心类型与架构设计
//
// 本包的核心入口为 [Engine]：
//
//   - 独立引擎：通过 [New] 构造私有的 [*Engine] 实例，
//     内部持有独立的 [*v10.Validate] 与中文翻译器，配置 binding 作为验证标签名；
//   - 字段名提取策略：注册了智能字段名提取函数，优先读取结构体字段的 label 标签，
//     若无则读取 json 标签；若标签设为 "-" 则忽略；
//   - 规则扩展：支持通过 [Engine.RegisterValidation] 动态挂载业务自定义验证规则。
//
// # 典型用法
//
// 下面的示例展示构造验证引擎并执行结构体校验：
//
//	type Account struct {
//		ID    string `json:"id" binding:"required,snowflake"`
//		Email string `json:"email" binding:"required,email"`
//	}
//
//	engine, err := validator.New()
//	if err != nil {
//		return err
//	}
//	if err := engine.Validate(&Account{ID: "1234567890", Email: "test@example.com"}); err != nil {
//		errMap := engine.TranslateError(err)
//		// 处理中文错误映射...
//	}
//
// # 内置领域验证规则
//
// 引擎在初始化阶段自动注册以下 8 种高频领域自定义验证规则及其中文翻译映射：
//
//   - strict_url（[ValidateStrictURL]）：严格 URL 校验，仅允许 http 与 https 协议；
//   - strict_uuid（[ValidateStrictUUID]）：严格 UUID 格式校验；
//   - alphanum_underscore（[ValidateAlphanumUnderscore]）：仅允许包含字母、数字和下划线；
//   - regexp（[ValidateRegexp]）：参数化正则表达式校验，支持传参指定正则模式；
//   - enum_int（[ValidateEnumInt]）：整型枚举校验，参数为空格分隔的允许数值；
//   - enum_string（[ValidateEnumString]）：字符串枚举校验，参数为空格分隔的允许字符串；
//   - enum_float（[ValidateEnumFloat]）：浮点型枚举校验（带 1e-9 容差比较）；
//   - snowflake（[ValidateSnowflake]）：雪花算法 ID 格式校验，仅允许 1~20 位纯数字字符串。
//
// # 错误结构与安全防泄漏
//
// 校验错误通过 [Engine.TranslateError] 转换为结构化的中文字段映射。
// 在处理 HTTP 绑定失败时，[Engine.HandleError] 构造 [ValidationErrorDetail] 列表，
// 并将其包装在标准业务错误 [apperr.BodyError] 中通过 [httpresult.Error] 输出：
//
//   - 字段明细：[ValidationErrorDetail] 包含触发错误的 Field、Tag 以及本地化 Message；
//   - 输入脱敏：[ValidationErrorDetail.Value] 安全默认置空（nil），
//     绝不在错误响应中回显用户提交的密码、私钥或敏感输入，防止信息泄漏。
//
// # 并发安全
//
// [Engine] 完成初始化与规则注册后，其 [Engine.Validate]、[Engine.TranslateError]
// 与 [Engine.HandleError] 方法可被多个 goroutine 安全并发调用。
// [Engine.RegisterValidation] 必须在服务初始化单协程阶段完成，不承诺运行期并发修改规则的安全性。
package validator

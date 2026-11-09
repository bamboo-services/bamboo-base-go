// Package valueerr 归一值校验类失败，产出统一的强类型错误。
//
// 本包属于 foundation 领域原语基座，只依赖 [apperr]，不依赖 HTTP、gRPC
// 或 application 层，因此可以被任意插件与协议层复用。
//
// # 职责边界
//
// 本包只提供显式的分类核心，不重写各处的值校验逻辑：
// 调用方在自己的校验分支里判定分类，再交给 [From] 转接为统一错误。
//
//   - 输入值本身非法归 400 组：[KindInvalidFormat]（40013）、
//     [KindOutOfRange]（40014）、[KindInvalidType]（40015）。
//   - 转换目标或接收器等调用契约非法归 500 组：
//     [KindUnsupportedTarget]（50018）、[KindInvalidReceiver]（50019）。
//
// 分类只依据实际校验分支与调用方显式给出的 [Kind]：
// 不解析错误文本，不按错误来源或调用时机推断分类，
// 也不会因为函数返回了 error 就一律判成 400。
//
// # 责任边界
//
// 本包不判断错误来自请求输入还是服务自身配置。请求边界直接传播 400 类值错误；
// 部署或启动时读取服务自身配置的入口，必须由所属域按配置责任使用自己的 500 码
// 重新包装，并把本包产出的值错误保留为原因：
//
//	valueErr := valueerr.From(valueerr.KindInvalidFormat, "端口号格式非法", err)
//	// 读取服务自身配置的责任边界：用所属域的 500 码显式重新包装，
//	// 并把本包产出的值错误保留为原因。
//	return apperr.New(pluginConfigError, "读取服务配置失败", valueErr)
//
// 管理接口提交配置时的请求校验仍按输入责任处理为 400，不因字段名含"配置"就归 500。
//
// # 无底层原因的失败
//
// [From] 是原因转接器，nil原因始终返回nil，不是失败构造器。
// nil接收器等校验分支没有原生cause时，直接使用 [apperr.New]：
//
//	if receiver == nil {
//		return apperr.New(valueerr.InvalidReceiver, "转换接收器不能为空")
//	}
//
// 不要为调用From伪造原生原因，也不要将From的nil返回误认为校验已经通过。
//
// # 与已有结构化错误的关系
//
// 原因链中已经存在 [apperr.Error] 时，[From] 沿用既有错误码与安全描述，
// 并保留整条原因链：既有错误不会因为链内深层出现其它类型的错误而被改判为值错误，
// 也不会丢失聚合中的兄弟原因。
package valueerr

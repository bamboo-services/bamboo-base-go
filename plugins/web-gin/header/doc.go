// Package header 提供常用的 HTTP 标头常量定义、标头属性判定及认证令牌提取工具。
//
// 本包定义了强类型的 [Header] 字符串类型，集中收敛标准 HTTP 请求标头、
// 响应标头、CORS 控制标头及框架自定义标头常量，并提供基于 RFC 规范的自定义标头检测、
// Bearer 认证令牌提取与刷新令牌解析能力。
//
// # 核心类型与常量
//
// 本包核心类型为 [Header]：
//
//   - 常用常量：预置了 RFC 常见标头（如 [HeaderContentType]、[HeaderAuthorization]）
//     以及分布式追踪标头 [HeaderRequestUUID] 与刷新令牌标头 [HeaderRefreshToken]；
//   - 标头判定：通过 [Header.IsEmpty] 检查标头是否为空，
//     通过 [Header.IsCustom] 判定是否属于以 "X-" 或 "x-" 前缀开头的自定义标头；
//   - 字符串转换：[Header.String] 输出底层原生字符串，可直接用于标准 HTTP 标头读写接口。
//
// # 典型用法
//
// 下面的示例展示从当前请求中提取客户端携带的 Bearer 授权凭据：
//
//	func AuthMiddleware() gin.HandlerFunc {
//		return func(c *gin.Context) {
//			token, err := header.GetAuthorization(c)
//			if err != nil {
//				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
//				return
//			}
//			// 校验 token...
//			c.Next()
//		}
//	}
//
// # 令牌提取契约
//
// 本包提供了安全解析令牌字符串的实用函数：
//
//   - [GetAuthorization]：严格从 [HeaderAuthorization] 请求头中提取凭据。
//     若缺少该标头返回 [ErrMissingAuthorization]；若标头内容未以 "Bearer " 前缀开头，
//     返回 [ErrInvalidAuthorizationFormat]；格式合法时截取并返回前缀后的有效 Token；
//   - [GetToken]：从指定标头（仅支持 [HeaderAuthorization] 或 [HeaderRefreshToken]，
//     传入其他标头时默认回退为 [HeaderAuthorization]）读取值。
//     若存在 "Bearer " 前缀则自动剥离，否则原样返回；若标头不存在或为空则返回空字符串。
//
// # 并发安全
//
// [Header] 类型底层为不可变的 string 类型，所有方法与辅助函数均无内部可变状态，
// 且仅执行只读提取，可被多个 goroutine 安全并发调用。
package header

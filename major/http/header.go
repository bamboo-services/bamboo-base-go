package xHttp

import "strings"

// Header 表示 HTTP 请求或响应标头字段的名称，提供了对自定义标头的识别功能。
type Header string

const (
	// 常见请求头
	HeaderAccept            Header = "Accept"              // 可接受的内容类型
	HeaderAcceptCharset     Header = "Accept-Charset"      // 可接受的字符集
	HeaderAcceptEncoding    Header = "Accept-Encoding"     // 可接受的内容编码
	HeaderAcceptLanguage    Header = "Accept-Language"     // 可接受的语言
	HeaderAuthorization     Header = "Authorization"       // HTTP 授权头字段名
	HeaderCacheControl      Header = "Cache-Control"       // 缓存控制指令
	HeaderConnection        Header = "Connection"          // 连接管理
	HeaderContentLength     Header = "Content-Length"      // 内容长度
	HeaderContentType       Header = "Content-Type"        // 内容类型
	HeaderCookie            Header = "Cookie"              // Cookie 信息
	HeaderHost              Header = "Host"                // 请求主机
	HeaderIfMatch           Header = "If-Match"            // 条件请求 ETag
	HeaderIfModifiedSince   Header = "If-Modified-Since"   // 条件请求修改时间
	HeaderIfNoneMatch       Header = "If-None-Match"       // 条件请求 ETag 不匹配
	HeaderIfUnmodifiedSince Header = "If-Unmodified-Since" // 条件请求未修改时间
	HeaderOrigin            Header = "Origin"              // 跨域来源
	HeaderPragma            Header = "Pragma"              // 兼容缓存控制
	HeaderRange             Header = "Range"               // 断点范围
	HeaderReferer           Header = "Referer"             // 引用来源
	HeaderUserAgent         Header = "User-Agent"          // 用户代理
	HeaderRequestUUID       Header = "X-Request-UUID"      // 请求唯一标识符的响应头字段名，用于跟踪请求的唯一性和溯源性
	HeaderRefreshToken      Header = "X-Refresh-Token"     // 刷新令牌的请求头字段名，通常用于获取新的访问令牌
	HeaderXForwardedFor     Header = "X-Forwarded-For"     // 代理转发 IP
	HeaderXForwardedHost    Header = "X-Forwarded-Host"    // 代理转发 Host
	HeaderXForwardedProto   Header = "X-Forwarded-Proto"   // 代理转发协议
	HeaderXRealIP           Header = "X-Real-IP"           // 真实客户端 IP
	HeaderXRequestedWith    Header = "X-Requested-With"    // Ajax 请求标识

	// 常见响应头
	HeaderAccessControlAllowCredentials Header = "Access-Control-Allow-Credentials" // CORS 允许携带凭据
	HeaderAccessControlAllowHeaders     Header = "Access-Control-Allow-Headers"     // CORS 允许的请求头
	HeaderAccessControlAllowMethods     Header = "Access-Control-Allow-Methods"     // CORS 允许的方法
	HeaderAccessControlAllowOrigin      Header = "Access-Control-Allow-Origin"      // CORS 允许的来源
	HeaderAccessControlExposeHeaders    Header = "Access-Control-Expose-Headers"    // CORS 暴露的响应头
	HeaderAccessControlMaxAge           Header = "Access-Control-Max-Age"           // CORS 预检缓存时间
	HeaderContentDisposition            Header = "Content-Disposition"              // 内容处置
	HeaderContentEncoding               Header = "Content-Encoding"                 // 内容编码
	HeaderContentLanguage               Header = "Content-Language"                 // 内容语言
	HeaderContentLocation               Header = "Content-Location"                 // 内容位置
	HeaderContentRange                  Header = "Content-Range"                    // 内容范围
	HeaderETag                          Header = "ETag"                             // 实体标签
	HeaderExpires                       Header = "Expires"                          // 过期时间
	HeaderLastModified                  Header = "Last-Modified"                    // 最后修改时间
	HeaderLocation                      Header = "Location"                         // 重定向位置
	HeaderServer                        Header = "Server"                           // 服务端信息
	HeaderSetCookie                     Header = "Set-Cookie"                       // 设置 Cookie
	HeaderTransferEncoding              Header = "Transfer-Encoding"                // 传输编码
	HeaderVary                          Header = "Vary"                             // 响应变化
	HeaderWWWAuthenticate               Header = "WWW-Authenticate"                 // 认证挑战
)

// String 返回 Header 的字符串形式表示。
func (h Header) String() string {
	return string(h)
}

// IsEmpty 判断 HTTP 标头是否为空字符串。
func (h Header) IsEmpty() bool {
	return h == ""
}

// IsCustom 判断 HTTP 标头是否为自定义类型。
//
// 根据 RFC 规范，以 "X-" 或 "x-" 前缀开头的标头被视为自定义标头。
func (h Header) IsCustom() bool {
	return strings.HasPrefix(h.String(), "X-") || strings.HasPrefix(h.String(), "x-")
}

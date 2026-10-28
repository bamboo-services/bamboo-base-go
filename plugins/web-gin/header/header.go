package header

import "strings"

// Header 表示 HTTP 请求或响应标头字段的名称类型，支持自定义标头格式检测。
type Header string

// 下列常量定义了标准 RFC 以及框架约定的 HTTP 请求与响应标头字段名称。
const (
	// HeaderAccept 表示客户端可接受的媒体类型的请求标头字段名。
	HeaderAccept Header = "Accept"
	// HeaderAcceptCharset 表示客户端可接受的字符集的请求标头字段名。
	HeaderAcceptCharset Header = "Accept-Charset"
	// HeaderAcceptEncoding 表示客户端支持的内容编码格式的请求标头字段名。
	HeaderAcceptEncoding Header = "Accept-Encoding"
	// HeaderAcceptLanguage 表示客户端优先偏好语言的请求标头字段名。
	HeaderAcceptLanguage Header = "Accept-Language"
	// HeaderAuthorization 表示携带客户端认证凭据的请求标头字段名。
	HeaderAuthorization Header = "Authorization"
	// HeaderCacheControl 表示控制请求与响应缓存机制指令的标头字段名。
	HeaderCacheControl Header = "Cache-Control"
	// HeaderConnection 表示控制当前网络连接选项的标头字段名。
	HeaderConnection Header = "Connection"
	// HeaderContentLength 表示请求体或响应体内容字节长度的标头字段名。
	HeaderContentLength Header = "Content-Length"
	// HeaderContentType 表示指示实体正文媒体类型的标头字段名。
	HeaderContentType Header = "Content-Type"
	// HeaderCookie 表示客户端随请求向服务端发送存储 Cookie 的标头字段名。
	HeaderCookie Header = "Cookie"
	// HeaderHost 表示目标服务器域名及端口号的请求标头字段名。
	HeaderHost Header = "Host"
	// HeaderIfMatch 表示条件请求中用于 ETag 匹配验证的标头字段名。
	HeaderIfMatch Header = "If-Match"
	// HeaderIfModifiedSince 表示仅在指定时间后资源变更才返回的条件请求标头字段名。
	HeaderIfModifiedSince Header = "If-Modified-Since"
	// HeaderIfNoneMatch 表示条件请求中若 ETag 不匹配才返回的标头字段名。
	HeaderIfNoneMatch Header = "If-None-Match"
	// HeaderIfUnmodifiedSince 表示仅在指定时间后资源未变更才执行的条件请求标头字段名。
	HeaderIfUnmodifiedSince Header = "If-Unmodified-Since"
	// HeaderOrigin 表示指示跨源请求或预检请求来源域的标头字段名。
	HeaderOrigin Header = "Origin"
	// HeaderPragma 表示为了向后兼容 HTTP/1.0 缓存控制的标头字段名。
	HeaderPragma Header = "Pragma"
	// HeaderRange 表示客户端请求部分实体范围的断点续传标头字段名。
	HeaderRange Header = "Range"
	// HeaderReferer 表示发起当前请求的前置页面地址的标头字段名。
	HeaderReferer Header = "Referer"
	// HeaderUserAgent 表示发起请求的客户端应用程序软件信息的标头字段名。
	HeaderUserAgent Header = "User-Agent"
	// HeaderRequestUUID 表示服务端注入的全局唯一请求追踪标识标头字段名。
	HeaderRequestUUID Header = "X-Request-UUID"
	// HeaderRefreshToken 表示客户端请求刷新访问凭证时携带的刷新令牌标头字段名。
	HeaderRefreshToken Header = "X-Refresh-Token"
	// HeaderXForwardedFor 表示标识通过 HTTP 代理连接的客户端原始 IP 的标头字段名。
	HeaderXForwardedFor Header = "X-Forwarded-For"
	// HeaderXForwardedHost 表示标识客户端初始请求主机头的标头字段名。
	HeaderXForwardedHost Header = "X-Forwarded-Host"
	// HeaderXForwardedProto 表示标识客户端发起请求所使用底层协议的标头字段名。
	HeaderXForwardedProto Header = "X-Forwarded-Proto"
	// HeaderXRealIP 表示反向代理服务器记录的真实客户端 IP 地址的标头字段名。
	HeaderXRealIP Header = "X-Real-IP"
	// HeaderXRequestedWith 表示标识 Ajax 异步请求的标头字段名。
	HeaderXRequestedWith Header = "X-Requested-With"

	// HeaderAccessControlAllowCredentials 表示 CORS 预检或响应中是否允许携带凭据的标头字段名。
	HeaderAccessControlAllowCredentials Header = "Access-Control-Allow-Credentials"
	// HeaderAccessControlAllowHeaders 表示 CORS 预检中允许跨域请求携带的标头列表的标头字段名。
	HeaderAccessControlAllowHeaders Header = "Access-Control-Allow-Headers"
	// HeaderAccessControlAllowMethods 表示 CORS 预检中允许跨域请求使用的 HTTP 方法列表的标头字段名。
	HeaderAccessControlAllowMethods Header = "Access-Control-Allow-Methods"
	// HeaderAccessControlAllowOrigin 表示 CORS 响应中允许访问资源的一个或多个源的标头字段名。
	HeaderAccessControlAllowOrigin Header = "Access-Control-Allow-Origin"
	// HeaderAccessControlExposeHeaders 表示 CORS 响应中允许脚本读取的安全标头白名单的标头字段名。
	HeaderAccessControlExposeHeaders Header = "Access-Control-Expose-Headers"
	// HeaderAccessControlMaxAge 表示 CORS 预检请求响应结果允许在客户端缓存的最大秒数的标头字段名。
	HeaderAccessControlMaxAge Header = "Access-Control-Max-Age"
	// HeaderContentDisposition 表示指示响应正文展示方式或下载文件名的标头字段名。
	HeaderContentDisposition Header = "Content-Disposition"
	// HeaderContentEncoding 表示用于对响应实体正文进行压缩编码转换的标头字段名。
	HeaderContentEncoding Header = "Content-Encoding"
	// HeaderContentLanguage 表示响应实体正文所面向受众自然语言的标头字段名。
	HeaderContentLanguage Header = "Content-Language"
	// HeaderContentLocation 表示所返回数据对应独立资源候选地址的标头字段名。
	HeaderContentLocation Header = "Content-Location"
	// HeaderContentRange 表示当前响应正文在完整实体中所处字节范围的标头字段名。
	HeaderContentRange Header = "Content-Range"
	// HeaderETag 表示特定版本资源的特定实体的唯一标记标识符的标头字段名。
	HeaderETag Header = "ETag"
	// HeaderExpires 表示响应过期的绝对日期与时间的标头字段名。
	HeaderExpires Header = "Expires"
	// HeaderLastModified 表示源服务器认定资源最后修改日期与时间的标头字段名。
	HeaderLastModified Header = "Last-Modified"
	// HeaderLocation 表示页面重定向或新建资源的目标 URL 的标头字段名。
	HeaderLocation Header = "Location"
	// HeaderServer 表示处理请求的源服务器所使用软件信息的标头字段名。
	HeaderServer Header = "Server"
	// HeaderSetCookie 表示由服务器向客户端发送 Cookie 的标头字段名。
	HeaderSetCookie Header = "Set-Cookie"
	// HeaderTransferEncoding 表示安全传输实体给调用方采用的编码形式的标头字段名。
	HeaderTransferEncoding Header = "Transfer-Encoding"
	// HeaderVary 表示指示源服务器缓存机制依据哪些请求头调整响应的标头字段名。
	HeaderVary Header = "Vary"
	// HeaderWWWAuthenticate 表示服务器在返回 401 响应时指示客户端所适用的认证挑战的标头字段名。
	HeaderWWWAuthenticate Header = "WWW-Authenticate"
)

// String 返回 [Header] 的底层原始字符串表示。
func (h Header) String() string {
	return string(h)
}

// IsEmpty 检查当前 HTTP 标头是否为空字符串。
//
// 若标头未赋值或为空串，返回 true；否则返回 false。
func (h Header) IsEmpty() bool {
	return h == ""
}

// IsCustom 判定当前 HTTP 标头是否属于自定义扩展标头。
//
// 根据 RFC 历史规范，凡是以 "X-" 或 "x-" 大小写前缀开头的标头均被视作自定义标头。
func (h Header) IsCustom() bool {
	return strings.HasPrefix(h.String(), "X-") || strings.HasPrefix(h.String(), "x-")
}

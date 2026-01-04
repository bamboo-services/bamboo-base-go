package xHttp

type HttpHeader string

const (
	HeaderRequestUUID   HttpHeader = "X-Request-UUID" // 请求唯一标识符的响应头字段名，用于跟踪请求的唯一性和溯源性
	HeaderAuthorization HttpHeader = "Authorization"  // HTTP 授权头字段名
)

func (h HttpHeader) String() string {
	return string(h)
}

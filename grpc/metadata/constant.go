package xGrpcMD

type Metadata string

// String 返回元数据的字符串表示形式。
//
// 该方法实现了 `fmt.Stringer` 接口，直接将 `Metadata` 类型转换为 `string` 类型返回。
func (md Metadata) String() string {
	return string(md)
}

const (
	AppAccessID  Metadata = "app_access_id"  // 定义用于传递应用访问标识符的元数据键，通常用于在中间件或拦截器中标识请求方的应用 ID。
	AppSecretKey Metadata = "app_secret_key" // 定义用于传递应用密钥的元数据键，通常用于在中间件或拦截器中验证请求方的身份。
)

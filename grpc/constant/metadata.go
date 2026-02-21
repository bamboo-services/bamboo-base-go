package xGrpcConst

type Metadata string

// String 返回元数据的字符串表示形式。
//
// 该方法实现了 `fmt.Stringer` 接口，直接将 `Trailer` 类型转换为 `string` 类型返回。
func (md Metadata) String() string {
	return string(md)
}

const (
	MetadataAppAccessID  Metadata = "app_access_id"  // 定义用于传递应用访问标识符的元数据键，通常用于在中间件或拦截器中标识请求方的应用 ID。
	MetadataAppSecretKey Metadata = "app_secret_key" // 定义用于传递应用密钥的元数据键，通常用于在中间件或拦截器中验证请求方的身份。
	MetadataRequestUUID  Metadata = "x_request_uuid" // 定义用于传递请求唯一标识符的元数据键，通常用于在中间件或拦截器中标识请求的唯一性。
)

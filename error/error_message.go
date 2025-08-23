package xError

// ErrMessage 自定义错误消息类型
type ErrMessage string

// String 将 ErrMessage 转换为字符串形式的表示。
func (e *ErrMessage) String() string {
	return string(*e)
}

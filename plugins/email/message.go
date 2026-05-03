package xEmail

import (
	"fmt"
	"io"
	"os"
)

// Attachment 邮件附件
type Attachment struct {
	Filename    string    // 附件文件名
	ContentType string    // MIME 类型 (如 "application/pdf")
	Data        io.Reader // 附件数据
}

// Message 邮件消息
type Message struct {
	From         string       // 发件人地址 (可选，为空使用默认配置)
	To           []string     // 收件人地址列表
	Cc           []string     // 抄送列表
	Bcc          []string     // 密送列表
	ReplyTo      string       // 回复地址
	Subject      string       // 邮件主题
	TextBody     string       // 纯文本内容
	HTMLBody     string       // HTML 内容
	Template     string       // 模板名称 (用于 SendTemplate)
	TemplateData any          // 模板数据
	Attachments  []Attachment // 附件列表
}

// NewMessage 创建空邮件消息
func NewMessage() *Message {
	return &Message{}
}

// AttachFile 从文件系统添加附件
func (m *Message) AttachFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("打开附件文件失败: %w", err)
	}

	m.Attachments = append(m.Attachments, Attachment{
		Filename: filename,
		Data:     file,
	})
	return nil
}

// AttachReader 从 io.Reader 添加附件
func (m *Message) AttachReader(name string, r io.Reader) *Message {
	m.Attachments = append(m.Attachments, Attachment{
		Filename: name,
		Data:     r,
	})
	return m
}

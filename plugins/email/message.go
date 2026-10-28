package email

import (
	"fmt"
	"io"
	"os"
)

// Attachment 表示随邮件一同发送的二进制或文本附件。
type Attachment struct {
	Filename    string    // 附件显示文件名（如 "report.pdf"）。
	ContentType string    // 附件的 MIME 类型（如 "application/pdf"，可选）。
	Data        io.Reader // 附件的数据读取流，在发信时按需读取。
}

// Message 表示待发送的邮件消息结构体。
// 零值为合法可用的空邮件结构。
type Message struct {
	// From 为发件人邮箱地址。可选；为空时将使用插件全局配置的默认发件人。
	From string

	// To 为主送收件人邮箱地址切片（至少包含一个收件人）。
	To []string

	// Cc 为抄送收件人邮箱地址切片。
	Cc []string

	// Bcc 为密送收件人邮箱地址切片。
	Bcc []string

	// ReplyTo 为指定的回复邮箱地址。可选。
	ReplyTo string

	// Subject 为邮件主题标题（必填）。
	Subject string

	// TextBody 为纯文本备用正文内容。可选。
	TextBody string

	// HTMLBody 为富文本 HTML 正文内容。可选。
	HTMLBody string

	// Template 为动态模板名称（不含 .html 后缀），配合 [Client.SendTemplate] 使用。
	Template string

	// TemplateData 为用于模板渲染的上下文数据对象。
	TemplateData any

	// Attachments 为随邮件携带的附件列表。
	Attachments []Attachment
}

// NewMessage 构造并返回一个空的 [Message] 消息指针。
func NewMessage() *Message {
	return &Message{}
}

// AttachFile 从指定的文件系统路径打开文件并作为附件追加至邮件中。
// 若文件不存在或无法读取，返回相应 I/O 错误。
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

// AttachReader 将内存中的 [io.Reader] 数据流作为命名附件追加至邮件中。
// 返回当前 [Message] 指针以支持链式调用。
func (m *Message) AttachReader(name string, r io.Reader) *Message {
	m.Attachments = append(m.Attachments, Attachment{
		Filename: name,
		Data:     r,
	})
	return m
}

package xEmail

import (
	"context"
	"fmt"

	mail "github.com/wneessen/go-mail"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// Config 邮件客户端配置
type Config struct {
	Host        string // SMTP 服务器地址
	Port        int    // SMTP 端口
	Username    string // 认证用户名
	Password    string // 认证密码
	FromName    string // 发件人名称
	FromAddr    string // 发件人地址
	TLSType     string // TLS 策略 (none/starttls/tls)
	TemplateDir string // 外部模板目录 (可选)
}

// EmailClient 邮件客户端
type EmailClient struct {
	client *mail.Client
	tmpl   *TemplateManager
	from   string
	fromName string
	log    *xLog.LogNamedLogger
}

// InitClient 初始化邮件客户端（注册节点函数）
//
// 从环境变量读取 SMTP 配置，创建邮件客户端。
// 返回 *EmailClient 实例，可通过 xCtxUtil 获取。
//
// 环境变量:
//   - EMAIL_HOST: SMTP 服务器地址 [必填]
//   - EMAIL_PORT: SMTP 端口 [必填]
//   - EMAIL_USER: 认证用户名 [必填]
//   - EMAIL_PASS: 认证密码 [必填]
//   - EMAIL_FROM: 发件人地址 [必填]
//   - EMAIL_FROM_NAME: 发件人名称 [可选]
//   - EMAIL_TLS: TLS 策略 [可选，默认 starttls]
//   - EMAIL_TEMPLATE_DIR: 外部模板目录 [可选]
func InitClient(ctx context.Context) (any, error) {
	host := xEnv.GetEnvString(xEnv.EmailHost, "")
	port := xEnv.GetEnvInt(xEnv.EmailPort, 0)
	user := xEnv.GetEnvString(xEnv.EmailUser, "")
	pass := xEnv.GetEnvString(xEnv.EmailPass, "")
	from := xEnv.GetEnvString(xEnv.EmailFrom, "")
	fromName := xEnv.GetEnvString(xEnv.EmailFromName, "")
	tlsType := xEnv.GetEnvString(xEnv.EmailTLS, "")
	tmplDir := xEnv.GetEnvString(xEnv.EmailTemplateDir, "")

	// 验证必填字段
	if host == "" {
		return nil, fmt.Errorf("邮件服务器地址 (EMAIL_HOST) 未配置")
	}
	if user == "" {
		return nil, fmt.Errorf("邮件用户名 (EMAIL_USER) 未配置")
	}
	if pass == "" {
		return nil, fmt.Errorf("邮件密码 (EMAIL_PASS) 未配置")
	}
	if from == "" {
		return nil, fmt.Errorf("发件人地址 (EMAIL_FROM) 未配置")
	}

	// 默认 TLS 策略
	if tlsType == "" {
		tlsType = "starttls"
	}
	if port == 0 {
		port = 587
	}

	// 创建 go-mail Client
	opts := []mail.Option{
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(user),
		mail.WithPassword(pass),
	}

	// 设置 TLS 策略
	switch tlsType {
	case "none":
		opts = append(opts, mail.WithTLSPortPolicy(mail.NoTLS))
	case "tls":
		opts = append(opts, mail.WithSSLPort(true), mail.WithTLSPortPolicy(mail.TLSMandatory))
	default:
		// STARTTLS (默认, port 587)
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	}

	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("创建邮件客户端失败: %w", err)
	}

	// 创建模板管理器
	tmpl, err := newTemplateManager(tmplDir)
	if err != nil {
		return nil, fmt.Errorf("初始化模板管理器失败: %w", err)
	}

	logger := xLog.WithName(xLog.NamedEMAIL)

	return &EmailClient{
		client:   client,
		tmpl:     tmpl,
		from:     from,
		fromName: fromName,
		log:      logger,
	}, nil
}

// Send 发送邮件
//
// 根据 Message 构建并发送邮件。支持纯文本和 HTML 内容，以及附件。
func (c *EmailClient) Send(ctx context.Context, msg *Message) error {
	// 验证必填字段
	if len(msg.To) == 0 {
		return fmt.Errorf("收件人地址不能为空")
	}
	if msg.Subject == "" {
		return fmt.Errorf("邮件主题不能为空")
	}

	// 构建 go-mail Msg
	goMsg := mail.NewMsg()

	// 设置发件人
	fromAddr := msg.From
	if fromAddr == "" {
		if c.fromName != "" {
			if err := goMsg.FromFormat(c.fromName, c.from); err != nil {
				return fmt.Errorf("设置发件人失败: %w", err)
			}
		} else {
			if err := goMsg.From(c.from); err != nil {
				return fmt.Errorf("设置发件人失败: %w", err)
			}
		}
	} else {
		if err := goMsg.From(fromAddr); err != nil {
			return fmt.Errorf("设置发件人失败: %w", err)
		}
	}

	// 设置收件人
	if err := goMsg.To(msg.To...); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	// 设置抄送
	if len(msg.Cc) > 0 {
		if err := goMsg.Cc(msg.Cc...); err != nil {
			return fmt.Errorf("设置抄送失败: %w", err)
		}
	}

	// 设置密送
	if len(msg.Bcc) > 0 {
		if err := goMsg.Bcc(msg.Bcc...); err != nil {
			return fmt.Errorf("设置密送失败: %w", err)
		}
	}

	// 设置回复地址
	if msg.ReplyTo != "" {
		goMsg.ReplyTo(msg.ReplyTo)
	}

	// 设置主题
	goMsg.Subject(msg.Subject)

	// 设置正文
	if msg.HTMLBody != "" {
		goMsg.SetBodyString(mail.TypeTextHTML, msg.HTMLBody)
		if msg.TextBody != "" {
			goMsg.AddAlternativeString(mail.TypeTextPlain, msg.TextBody)
		}
	} else if msg.TextBody != "" {
		goMsg.SetBodyString(mail.TypeTextPlain, msg.TextBody)
	}

	// 添加附件
	for _, att := range msg.Attachments {
		if att.Data != nil {
			goMsg.AttachReader(att.Filename, att.Data)
		}
	}

	// 发送
	if err := c.client.DialAndSendWithContext(ctx, goMsg); err != nil {
		c.log.SugarError(ctx, "发送邮件失败", "error", err)
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	c.log.SugarInfo(ctx, "邮件发送成功",
		"to", msg.To,
		"subject", msg.Subject,
	)
	return nil
}

// SendHTML 发送 HTML 邮件
func (c *EmailClient) SendHTML(ctx context.Context, msg *Message) error {
	return c.Send(ctx, msg)
}

// SendTemplate 使用模板渲染并发送 HTML 邮件
func (c *EmailClient) SendTemplate(ctx context.Context, msg *Message) error {
	if msg.Template == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	html, err := c.tmpl.Render(msg.Template, msg.TemplateData)
	if err != nil {
		return fmt.Errorf("渲染邮件模板失败: %w", err)
	}

	msg.HTMLBody = html
	return c.Send(ctx, msg)
}

// AddTemplateDir 添加外部模板目录
func (c *EmailClient) AddTemplateDir(dir string) error {
	return c.tmpl.AddDir(dir)
}

// ListTemplates 返回可用模板名称列表
func (c *EmailClient) ListTemplates() []string {
	return c.tmpl.ListTemplates()
}

package email

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"sync"
	"time"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/env"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/logging"
	"github.com/wneessen/go-mail"
)

// PluginKey 是 Email 插件的固定唯一标识。
const PluginKey ctxkey.Key = "plugin_email"

// 常见错误定义。
var (
	ErrStopped        = errors.New("email: client has been stopped")
	ErrRecipientEmpty = errors.New("email: recipient address cannot be empty")
	ErrSubjectEmpty   = errors.New("email: subject cannot be empty")
	ErrTemplateEmpty  = errors.New("email: template name cannot be empty")
	ErrNotInitialized = errors.New("email: plugin not initialized")
	ErrMissingHost    = errors.New("email: missing SMTP host")
	ErrMissingUser    = errors.New("email: missing SMTP username")
	ErrMissingPass    = errors.New("email: missing SMTP password")
	ErrMissingFrom    = errors.New("email: missing sender address")
	ErrInvalidPort    = errors.New("email: invalid SMTP port (must be between 1 and 65535)")
)

// Config 邮件客户端配置。
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

// Option 定义邮件插件的配置选项。
type Option func(*pluginOptions)

type templateFSEntry struct {
	fsys    fs.FS
	pattern string
}

type pluginOptions struct {
	cfg          Config
	fromEnv      bool
	logger       *logging.Logger
	templateDirs []string
	templateFS   []templateFSEntry
}

// WithConfig 显式指定配置。
func WithConfig(cfg Config) Option {
	return func(o *pluginOptions) {
		o.cfg = cfg
	}
}

// FromEnv 从环境变量读取配置。
//
// 环境变量列表:
//   - EMAIL_HOST: SMTP 服务器地址 [必填]
//   - EMAIL_PORT: SMTP 端口 [必填，缺省 587]
//   - EMAIL_USER: 认证用户名 [必填]
//   - EMAIL_PASS: 认证密码 [必填]
//   - EMAIL_FROM: 发件人地址 [必填]
//   - EMAIL_FROM_NAME: 发件人名称 [可选]
//   - EMAIL_TLS: TLS 策略 [可选，默认 starttls]
//   - EMAIL_TEMPLATE_DIR: 外部模板目录 [可选]
func FromEnv() Option {
	return func(o *pluginOptions) {
		o.fromEnv = true
	}
}

// WithLogger 显式注入应用 Logger 对象。
// 不走 slog.Default / 全局单例，保持 App 隔离。
func WithLogger(l *logging.Logger) Option {
	return func(o *pluginOptions) {
		o.logger = l
	}
}

// WithTemplateDir 预添加外部模板目录。
func WithTemplateDir(dir string) Option {
	return func(o *pluginOptions) {
		if dir != "" {
			o.templateDirs = append(o.templateDirs, dir)
		}
	}
}

// WithTemplateFS 预加载内嵌模板 FS。
func WithTemplateFS(fsys fs.FS, pattern string) Option {
	return func(o *pluginOptions) {
		if fsys != nil && pattern != "" {
			o.templateFS = append(o.templateFS, templateFSEntry{fsys: fsys, pattern: pattern})
		}
	}
}

// Client 提供邮件发送与模板管理 API。
type Client struct {
	plugin *Plugin
}

// Plugin 实现了 application.Plugin 与 ctxkey.Exposer 接口。
type Plugin struct {
	opts pluginOptions

	clientMu sync.RWMutex
	client   *Client
	tmpl     *TemplateManager
	log      *logging.LogNamedLogger
	forceCtx context.Context

	// 关闸与正在执行的发送计数，由同一个 inflightMu 互斥保护
	inflightMu sync.Mutex
	stopped    bool // 仅表示关闸拒绝新准入，不代表停止完成
	inflight   int
	zeroCond   *sync.Cond
	closeDone  chan struct{}
}

// New 构造 Email 插件注册描述与工厂函数。
// 构造、New、工厂均不发生任何网络建连与 SMTP 连接。
func New(opts ...Option) (application.PluginInfo, func() application.Plugin) {
	var po pluginOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&po)
		}
	}

	info := application.PluginInfo{
		Key:      PluginKey,
		Name:     "EmailPlugin",
		Mode:     application.ModeTool,
		Provides: []ctxkey.Key{ctxkey.EmailClientKey},
		Requires: nil,
	}

	factory := func() application.Plugin {
		p := &Plugin{
			opts: po,
		}
		p.zeroCond = sync.NewCond(&p.inflightMu)
		return p
	}

	return info, factory
}

// Start 在 App 启动阶段完成纯配置客户端和模板管理器初始化。
// Start 时不连接 SMTP，只有实际发信时才建连。
func (p *Plugin) Start(ctx context.Context) error {
	if p == nil {
		return ErrNotInitialized
	}

	cfg := p.opts.cfg
	if p.opts.fromEnv {
		getString := func(k env.Key) string {
			if v, ok := env.Lookup(k); ok {
				return v.String()
			}
			return ""
		}
		host := getString(env.EmailHost)
		var port int
		if v, ok := env.Lookup(env.EmailPort); ok {
			p, err := v.Int()
			if err != nil || p < 1 || p > 65535 {
				return ErrInvalidPort
			}
			port = p
		} else {
			port = 587
		}
		user := getString(env.EmailUser)
		pass := getString(env.EmailPass)
		from := getString(env.EmailFrom)
		fromName := getString(env.EmailFromName)
		tlsType := getString(env.EmailTLS)
		tmplDir := getString(env.EmailTemplateDir)

		if host != "" {
			cfg.Host = host
		}
		if port != 0 {
			cfg.Port = port
		}
		if user != "" {
			cfg.Username = user
		}
		if pass != "" {
			cfg.Password = pass
		}
		if from != "" {
			cfg.FromAddr = from
		}
		if fromName != "" {
			cfg.FromName = fromName
		}
		if tlsType != "" {
			cfg.TLSType = tlsType
		}
		if tmplDir != "" {
			cfg.TemplateDir = tmplDir
		}
	}

	// 校验必填配置
	if cfg.Host == "" {
		return ErrMissingHost
	}
	if cfg.Username == "" {
		return ErrMissingUser
	}
	if cfg.Password == "" {
		return ErrMissingPass
	}
	if cfg.FromAddr == "" {
		return ErrMissingFrom
	}

	if cfg.Port <= 0 {
		cfg.Port = 587
	}
	if cfg.TLSType == "" {
		cfg.TLSType = "starttls"
	}

	// 初始化模板管理器
	tmpl, err := newTemplateManager(cfg.TemplateDir)
	if err != nil {
		return fmt.Errorf("email: 初始化模板管理器失败: %w", err)
	}

	for _, dir := range p.opts.templateDirs {
		if err := tmpl.AddDir(dir); err != nil {
			return fmt.Errorf("email: 加载外部模板目录 %s 失败: %w", dir, err)
		}
	}
	for _, entry := range p.opts.templateFS {
		if err := tmpl.AddFS(entry.fsys, entry.pattern); err != nil {
			return fmt.Errorf("email: 加载内嵌模板失败: %w", err)
		}
	}

	// 注入日志器
	var namedLogger *logging.LogNamedLogger
	if p.opts.logger != nil {
		namedLogger = p.opts.logger.WithName(logging.NamedEMAIL)
	}

	forceCtx := application.ForceContext(ctx)
	if forceCtx == nil {
		forceCtx = context.Background()
	}

	p.clientMu.Lock()
	p.opts.cfg = cfg
	p.tmpl = tmpl
	p.log = namedLogger
	p.client = &Client{plugin: p}
	p.forceCtx = forceCtx
	p.clientMu.Unlock()

	return nil
}

// Expose 返回对外暴露的句柄资源。
func (p *Plugin) Expose() map[ctxkey.Key]any {
	p.clientMu.RLock()
	defer p.clientMu.RUnlock()
	return map[ctxkey.Key]any{
		ctxkey.EmailClientKey: p.client,
	}
}

// Stop 关闭发信准入，等待所有已接纳的发送真实退出再清理。
// nil 接收者或重复调用 Stop 必须安全幂等并等待同一完成。
func (p *Plugin) Stop(ctx context.Context) error {
	if p == nil {
		return nil
	}

	p.inflightMu.Lock()
	if p.zeroCond == nil {
		p.zeroCond = sync.NewCond(&p.inflightMu)
	}

	if p.stopped {
		done := p.closeDone
		p.inflightMu.Unlock()
		if done != nil {
			<-done
		}
		return nil
	}
	p.stopped = true
	done := make(chan struct{})
	p.closeDone = done

	// 首个 Stop 同步等待计数真归零（Cond.Wait 自动释锁与重获锁）
	for p.inflight > 0 {
		p.zeroCond.Wait()
	}
	p.inflightMu.Unlock()

	p.clientMu.Lock()
	p.client = nil
	p.clientMu.Unlock()

	close(done)
	return nil
}

// acquire 尝试登记一次发送任务。如果已停止则拒绝准入。
func (p *Plugin) acquire() error {
	p.inflightMu.Lock()
	defer p.inflightMu.Unlock()

	if p.stopped {
		return ErrStopped
	}
	p.inflight++
	return nil
}

// release 释放一次发送计数，并在归零时唤醒 Wait 中的 Stop。
func (p *Plugin) release() {
	p.inflightMu.Lock()
	defer p.inflightMu.Unlock()

	p.inflight--
	if p.inflight == 0 {
		p.zeroCond.Broadcast()
	}
}

// InflightCount 返回当前正在进行的发送任务数（主要用于测试观测）。
func (p *Plugin) InflightCount() int {
	if p == nil {
		return 0
	}
	p.inflightMu.Lock()
	defer p.inflightMu.Unlock()
	return p.inflight
}

// buildMailClient 构建 go-mail.Client 实例（纯配置构建，不发网络包）。
func (p *Plugin) buildMailClient(extraOpts ...mail.Option) (*mail.Client, error) {
	p.clientMu.RLock()
	cfg := p.opts.cfg
	p.clientMu.RUnlock()

	opts := []mail.Option{
		mail.WithPort(cfg.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password),
	}

	switch cfg.TLSType {
	case "none":
		opts = append(opts, mail.WithTLSPortPolicy(mail.NoTLS))
	case "tls":
		opts = append(opts, mail.WithSSLPort(true), mail.WithTLSPortPolicy(mail.TLSMandatory))
	default:
		// starttls (默认)
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	}

	if len(extraOpts) > 0 {
		opts = append(opts, extraOpts...)
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("email: 创建邮件底层客户端失败: %w", err)
	}
	return client, nil
}

// watchedConn 包装真实 net.Conn，绑定发送生命周期并由 context.AfterFunc 强制中断。
type watchedConn struct {
	net.Conn
	mu        sync.Mutex
	stopWatch func() bool
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

func (c *watchedConn) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		if c.stopWatch != nil {
			c.stopWatch()
			c.stopWatch = nil
		}
		c.mu.Unlock()
		c.closeErr = c.Conn.Close()
	})
	return c.closeErr
}

// Send 根据 Message 构建并发送邮件。
// 准入、执行、日志均处于严格受控与监督边界，支持 ctx 取消。
func (c *Client) Send(ctx context.Context, msg *Message) error {
	if c == nil || c.plugin == nil {
		return ErrNotInitialized
	}
	if msg == nil {
		return errors.New("email: message cannot be nil")
	}
	if len(msg.To) == 0 {
		return ErrRecipientEmpty
	}
	if msg.Subject == "" {
		return ErrSubjectEmpty
	}

	// 检查 Context 是否已取消
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// 准入登记
	if err := c.plugin.acquire(); err != nil {
		return err
	}
	defer c.plugin.release()

	// 再次检查 Context
	if err := ctx.Err(); err != nil {
		return err
	}

	c.plugin.clientMu.RLock()
	forceCtx := c.plugin.forceCtx
	c.plugin.clientMu.RUnlock()

	sendCtx, sendCancel := context.WithCancel(ctx)
	defer sendCancel()
	if forceCtx != nil {
		stopAfter := context.AfterFunc(forceCtx, sendCancel)
		defer stopAfter()
	}

	var activeConn net.Conn
	var connMu sync.Mutex

	dialCtxFunc := func(dialCtx context.Context, network, address string) (net.Conn, error) {
		if err := sendCtx.Err(); err != nil {
			return nil, err
		}
		var d net.Dialer
		rawConn, err := d.DialContext(dialCtx, network, address)
		if err != nil {
			return nil, err
		}

		wc := &watchedConn{Conn: rawConn}
		stopWatch := context.AfterFunc(sendCtx, func() {
			_ = rawConn.SetDeadline(time.Now())
			_ = rawConn.Close()
		})
		wc.mu.Lock()
		if wc.closed {
			stopWatch()
		} else {
			wc.stopWatch = stopWatch
		}
		wc.mu.Unlock()
		if err := sendCtx.Err(); err != nil {
			_ = wc.Close()
			return nil, err
		}

		connMu.Lock()
		activeConn = wc
		connMu.Unlock()
		return wc, nil
	}

	mailClient, err := c.plugin.buildMailClient(mail.WithDialContextFunc(dialCtxFunc))
	if err != nil {
		return err
	}
	defer func() {
		connMu.Lock()
		conn := activeConn
		activeConn = nil
		connMu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
	}()

	c.plugin.clientMu.RLock()
	cfg := c.plugin.opts.cfg
	log := c.plugin.log
	c.plugin.clientMu.RUnlock()

	goMsg := mail.NewMsg()

	fromAddr := msg.From
	if fromAddr == "" {
		if cfg.FromName != "" {
			if err := goMsg.FromFormat(cfg.FromName, cfg.FromAddr); err != nil {
				return fmt.Errorf("email: 设置发件人失败: %w", err)
			}
		} else {
			if err := goMsg.From(cfg.FromAddr); err != nil {
				return fmt.Errorf("email: 设置发件人失败: %w", err)
			}
		}
	} else {
		if err := goMsg.From(fromAddr); err != nil {
			return fmt.Errorf("email: 设置发件人失败: %w", err)
		}
	}

	if err := goMsg.To(msg.To...); err != nil {
		return fmt.Errorf("email: 设置收件人失败: %w", err)
	}
	if len(msg.Cc) > 0 {
		if err := goMsg.Cc(msg.Cc...); err != nil {
			return fmt.Errorf("email: 设置抄送失败: %w", err)
		}
	}
	if len(msg.Bcc) > 0 {
		if err := goMsg.Bcc(msg.Bcc...); err != nil {
			return fmt.Errorf("email: 设置密送失败: %w", err)
		}
	}
	if msg.ReplyTo != "" {
		if err := goMsg.ReplyTo(msg.ReplyTo); err != nil {
			return fmt.Errorf("email: 设置回复地址失败: %w", err)
		}
	}

	goMsg.Subject(msg.Subject)

	if msg.HTMLBody != "" {
		goMsg.SetBodyString(mail.TypeTextHTML, msg.HTMLBody)
		if msg.TextBody != "" {
			goMsg.AddAlternativeString(mail.TypeTextPlain, msg.TextBody)
		}
	} else if msg.TextBody != "" {
		goMsg.SetBodyString(mail.TypeTextPlain, msg.TextBody)
	}

	for _, att := range msg.Attachments {
		if att.Data != nil {
			if err := goMsg.AttachReader(att.Filename, att.Data); err != nil {
				return fmt.Errorf("email: 添加附件失败: %w", err)
			}
		}
	}

	// 在当前协程中同生命周期等待发送返回，禁止开无人监督的脱靶协程
	if err := mailClient.DialAndSendWithContext(sendCtx, goMsg); err != nil {
		if log != nil {
			log.SugarError(sendCtx, "发送邮件失败", "error", err)
		}
		return fmt.Errorf("email: 发送邮件失败: %w", err)
	}

	if log != nil {
		log.SugarInfo(sendCtx, "邮件发送成功", "to", msg.To, "subject", msg.Subject)
	}
	return nil
}

// SendHTML 发送 HTML 邮件。
func (c *Client) SendHTML(ctx context.Context, msg *Message) error {
	return c.Send(ctx, msg)
}

// SendTemplate 使用模板管理器渲染并发送邮件。
func (c *Client) SendTemplate(ctx context.Context, msg *Message) error {
	if c == nil || c.plugin == nil {
		return ErrNotInitialized
	}
	if msg == nil {
		return errors.New("email: message cannot be nil")
	}
	if msg.Template == "" {
		return ErrTemplateEmpty
	}

	c.plugin.clientMu.RLock()
	tmpl := c.plugin.tmpl
	c.plugin.clientMu.RUnlock()

	if tmpl == nil {
		return fmt.Errorf("email: 模板管理器未就绪")
	}

	html, err := tmpl.Render(msg.Template, msg.TemplateData)
	if err != nil {
		return fmt.Errorf("email: 渲染邮件模板失败: %w", err)
	}

	msg.HTMLBody = html
	return c.Send(ctx, msg)
}

// AddTemplateDir 添加外部模板目录。
func (c *Client) AddTemplateDir(dir string) error {
	if c == nil || c.plugin == nil {
		return ErrNotInitialized
	}
	c.plugin.clientMu.RLock()
	tmpl := c.plugin.tmpl
	c.plugin.clientMu.RUnlock()
	if tmpl == nil {
		return fmt.Errorf("email: 模板管理器未就绪")
	}
	return tmpl.AddDir(dir)
}

// AddTemplateFS 从 fs.FS 添加内嵌模板。
func (c *Client) AddTemplateFS(fsys fs.FS, pattern string) error {
	if c == nil || c.plugin == nil {
		return ErrNotInitialized
	}
	c.plugin.clientMu.RLock()
	tmpl := c.plugin.tmpl
	c.plugin.clientMu.RUnlock()
	if tmpl == nil {
		return fmt.Errorf("email: 模板管理器未就绪")
	}
	return tmpl.AddFS(fsys, pattern)
}

// ListTemplates 列出可用模板。
func (c *Client) ListTemplates() []string {
	if c == nil || c.plugin == nil {
		return nil
	}
	c.plugin.clientMu.RLock()
	tmpl := c.plugin.tmpl
	c.plugin.clientMu.RUnlock()
	if tmpl == nil {
		return nil
	}
	return tmpl.ListTemplates()
}

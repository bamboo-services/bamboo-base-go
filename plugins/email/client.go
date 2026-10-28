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

// PluginKey 是 Email 插件在应用拓扑图中的固定唯一标识。
const PluginKey ctxkey.Key = "plugin_email"

var (
	// ErrStopped 表示邮件客户端已被停止或正在停机排空，拒绝接纳新的发信请求。
	ErrStopped = errors.New("email: client has been stopped")

	// ErrRecipientEmpty 表示收件人地址列表为空。
	ErrRecipientEmpty = errors.New("email: recipient address cannot be empty")

	// ErrSubjectEmpty 表示邮件主题为空。
	ErrSubjectEmpty = errors.New("email: subject cannot be empty")

	// ErrTemplateEmpty 表示模板发信时指定的模板名称为空。
	ErrTemplateEmpty = errors.New("email: template name cannot be empty")

	// ErrNotInitialized 表示邮件插件尚未完成 Start 阶段的初始化。
	ErrNotInitialized = errors.New("email: plugin not initialized")

	// ErrMissingHost 表示配置中缺少 SMTP 主机地址。
	ErrMissingHost = errors.New("email: missing SMTP host")

	// ErrMissingUser 表示配置中缺少 SMTP 认证用户名。
	ErrMissingUser = errors.New("email: missing SMTP username")

	// ErrMissingPass 表示配置中缺少 SMTP 认证密码。
	ErrMissingPass = errors.New("email: missing SMTP password")

	// ErrMissingFrom 表示配置中缺少发件人邮箱地址。
	ErrMissingFrom = errors.New("email: missing sender address")

	// ErrInvalidPort 表示 SMTP 端口号超出合法范围（必须介于 1 到 65535 之间）。
	ErrInvalidPort = errors.New("email: invalid SMTP port (must be between 1 and 65535)")
)

// Config 包含邮件客户端的连接与认证配置。
type Config struct {
	// Host 为 SMTP 服务器地址（域名或 IP）。
	Host string

	// Port 为 SMTP 服务端口（常用端口如 25、465、587，缺省时默认为 587）。
	Port int

	// Username 为 SMTP 身份验证用户名。
	Username string

	// Password 为 SMTP 身份验证密码或应用授权码。
	Password string

	// FromName 为发件人显示名称（可选，如 "Bamboo Service"）。
	FromName string

	// FromAddr 为发件人邮箱地址（必填）。
	FromAddr string

	// TLSType 为 TLS 连接加密策略（支持 "none"、"starttls" 与 "tls"，缺省默认为 "starttls"）。
	TLSType string

	// TemplateDir 为可选的外部 HTML 模板根目录。
	TemplateDir string
}

// Option 定义邮件插件的全局函数式配置选项。
// 传入 nil 时将被 [New] 安全忽略。
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

// WithConfig 显式指定邮件客户端的基础配置。
func WithConfig(cfg Config) Option {
	return func(o *pluginOptions) {
		o.cfg = cfg
	}
}

// FromEnv 配置插件从环境变量中读取 SMTP 运行参数。
//
// 支持的环境变量键如下：
//   - EMAIL_HOST: SMTP 服务器地址（必填）；
//   - EMAIL_PORT: SMTP 端口（可选，缺省默认为 587）；
//   - EMAIL_USER: 认证用户名（必填）；
//   - EMAIL_PASS: 认证密码（必填）；
//   - EMAIL_FROM: 发件人地址（必填）；
//   - EMAIL_FROM_NAME: 发件人显示名称（可选）；
//   - EMAIL_TLS: TLS 策略（可选，支持 "none"、"starttls"、"tls"，缺省为 "starttls"）；
//   - EMAIL_TEMPLATE_DIR: 外部模板根目录（可选）。
func FromEnv() Option {
	return func(o *pluginOptions) {
		o.fromEnv = true
	}
}

// WithLogger 显式注入应用统一结构化日志器，保持 App 隔离。
// 为 nil 时插件仅在关键失败时输出最小诊断。
func WithLogger(l *logging.Logger) Option {
	return func(o *pluginOptions) {
		o.logger = l
	}
}

// WithTemplateDir 预添加外部 HTML 模板目录，目录内的模板文件在启动时自动加载。
// 若 dir 为空字符串，则忽略。
func WithTemplateDir(dir string) Option {
	return func(o *pluginOptions) {
		if dir != "" {
			o.templateDirs = append(o.templateDirs, dir)
		}
	}
}

// WithTemplateFS 预加载内嵌文件系统的 HTML 模板集合，适用于单二进制内嵌资产部署。
// 若 fsys 为 nil 或 pattern 为空字符串，则忽略。
func WithTemplateFS(fsys fs.FS, pattern string) Option {
	return func(o *pluginOptions) {
		if fsys != nil && pattern != "" {
			o.templateFS = append(o.templateFS, templateFSEntry{fsys: fsys, pattern: pattern})
		}
	}
}

// Client 提供邮件发送、HTML 渲染与模板管理能力。
// 在应用启动后以 [ctxkey.EmailClientKey] 注入共享组件容器。
// [Client] 的所有方法均为并发安全。
type Client struct {
	plugin *Plugin
}

// Plugin 实现 [application.Plugin] 与 [ctxkey.Exposer] 接口，作为 Tool 模式受宿主编排。
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

// New 构造 Email 插件注册描述与工厂函数，主键为 [PluginKey]，Provides 包含 [ctxkey.EmailClientKey]。
// 构造函数与返回的工厂均为纯内存配置，不发起任何 SMTP 网络连接或 DNS 查询。
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

// Start 在应用启动阶段完成配置校验、环境变量合并以及模板管理器初始化。
// 本阶段不与 SMTP 服务器建立网络连接，连接在调用发送时按需延迟建立。
// 若关键配置缺失或模板语法错误，返回相应错误。
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

// Expose 返回对外暴露的句柄资源，将 [Client] 以 [ctxkey.EmailClientKey] 注入共享组件容器。
func (p *Plugin) Expose() map[ctxkey.Key]any {
	p.clientMu.RLock()
	defer p.clientMu.RUnlock()
	return map[ctxkey.Key]any{
		ctxkey.EmailClientKey: p.client,
	}
}

// Stop 关闭发信准入，并阻塞等待所有已接纳的在途发送任务执行完毕后再清理客户端句柄。
// nil 接收者或多次重复调用 Stop 均安全幂等并等待同一关停动作完成。
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

// InflightCount 返回当前正在进行的发送任务数量。
// 主要供监控观测与测试验证使用。本方法并发安全。
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

// Send 根据给定的 [Message] 构建并发送邮件。
// 发送过程严格受 ctx 的超时与取消管辖；若插件已处于停止状态，立即返回 [ErrStopped]。
// 发送在调用方协程内同步完成，不启动不可控的脱靶后台协程。
// 本方法并发安全。
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

// SendHTML 发送包含 HTML 正文的邮件，行为契约与 [Client.Send] 一致。
// 本方法并发安全。
func (c *Client) SendHTML(ctx context.Context, msg *Message) error {
	return c.Send(ctx, msg)
}

// SendTemplate 使用内置或外部模板渲染正文并发送邮件。
// 要求 msg.Template 必须非空（对应模板名称，无需包含 .html 后缀），渲染结果写入 msg.HTMLBody 并调用 [Client.Send]。
// 若模板不存在或渲染失败，返回相应错误。本方法并发安全。
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

// AddTemplateDir 动态挂载外部模板目录。
// 外部目录中的同名模板将覆盖原有的同名模板。本方法并发安全。
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

// AddTemplateFS 动态挂载内嵌文件系统的 HTML 模板集合。
// 内嵌集合中的同名模板将覆盖原有的同名模板。本方法并发安全。
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

// ListTemplates 返回当前已加载的所有可用模板名称列表。
// 若客户端未就绪或未初始化，返回 nil。本方法并发安全。
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

// Package email 提供基于 SMTP 协议的邮件发送与 HTML 模板渲染插件。
//
// 本包作为 Bamboo Base Go 框架的共享组件插件（Tool 模式），
// 封装了底层 SMTP 通信、TLS 加密连接管理与基于 Go html/template 的邮件正文渲染引擎，
// 适用于系统通知、用户注册激活、密码重置及业务报告邮件发送等场景。
//
// # 核心类型与调用路径
//
// 在应用装配阶段，通过 [New] 配置发件服务器参数并注册到 [application.App]：
//
//	info, factory := email.New(
//		email.WithConfig(email.Config{
//			Host:     "smtp.example.com",
//			Port:     587,
//			Username: "sender@example.com",
//			Password: "secure_password",
//			FromAddr: "noreply@example.com",
//			FromName: "Bamboo Service",
//			TLSType:  "starttls",
//		}),
//	)
//	app.RegisterTool(info, factory)
//
// 启动后，插件将 [Client] 句柄以 [ctxkey.EmailClientKey] 注入共享容器。
// 业务逻辑中可通过 [application.Get] 提取客户端实例并调用 [Client.Send] 或 [Client.SendTemplate]。
//
// # 典型用法
//
// 从应用上下文获取邮件客户端，构造消息并发送带附件的邮件：
//
//	client, err := application.Get[*email.Client](ctx, ctxkey.EmailClientKey)
//	if err != nil {
//		return err
//	}
//
//	msg := email.NewMessage()
//	msg.To = []string{"user@domain.com"}
//	msg.Subject = "您的验证码"
//	msg.TextBody = "您的验证码为 123456，有效时间 5 分钟。"
//	if err := client.Send(ctx, msg); err != nil {
//		return err
//	}
//
// # 邮件构建与模板渲染
//
// 通过 [Message] 支持纯文本、HTML 与动态模板混合排版：
//
//   - 动态模板渲染：调用 [Client.SendTemplate] 时，插件自动检索内置或外部 HTML 模板，
//     将 [Message.TemplateData] 结合公共布局（_base.html）渲染为最终邮件正文；
//   - 模板来源扩展：支持通过 [WithTemplateDir]、[Client.AddTemplateDir] 挂载外部目录，
//     或通过 [WithTemplateFS]、[Client.AddTemplateFS] 挂载内嵌 [fs.FS] 单二进制模板；
//   - 附件追加：提供 [Message.AttachFile] 与 [Message.AttachReader]，支持直接挂载数据流。
//
// # 配置与环境变量
//
// 支持显式传入 [Config] 或通过 [FromEnv] 自动解析环境变量：
//
//   - EMAIL_HOST: SMTP 服务器域名或 IP 地址（必填）；
//   - EMAIL_PORT: SMTP 端口（可选，默认 587）；
//   - EMAIL_USER / EMAIL_PASS: SMTP 身份认证凭据（必填）；
//   - EMAIL_FROM / EMAIL_FROM_NAME: 发信人地址与显示名称；
//   - EMAIL_TLS: TLS 策略，支持 "none"、"starttls"（默认）与 "tls"（SMTPS/SSL）；
//   - EMAIL_TEMPLATE_DIR: 外部模板根目录（可选）。
//
// # 连接管理与生命周期
//
// 插件在连接与资源管理上遵循严格原则：
//
//   - 延迟按需建连：[New]、工厂与 [Plugin.Start] 阶段绝不发生任何网络 I/O，
//     只有在实际调用发信时才根据配置建立 SMTP 连接；
//   - 上下文严格受控：发信网络操作受调用方传入的 [context.Context] 严格管辖，
//     当上下文超时或取消时，插件通过内部底层监控立即强制关闭连接套接字，防止连接挂死；
//   - 强制退出联动：自动感知应用全局强制退出阶段（Force 预算），在关机超时时果断中断在途发送。
//
// # 停机排空与并发安全
//
// 当宿主触发应用停机调用 [Plugin.Stop] 时：
//
//   - 立即关闭发送门禁，后续所有提交均被拒绝并返回 [ErrStopped]；
//   - 阻塞等待当前已接纳的所有在途发送任务执行完毕（引用计数归零），不遗留孤儿协程；
//   - [Client] 的所有发送与模板方法均支持多 goroutine 并发安全调用；
//   - [Plugin.Stop] 支持多次重复调用，语义幂等且安全。
//
// # 错误处理
//
// 调用方应针对不同阶段的错误分支实施相应处理：
//
//   - 配置与启动阶段：配置缺失返回 [ErrMissingHost]、[ErrMissingUser]、[ErrMissingFrom] 等；
//   - 参数校验阶段：收件人为空返回 [ErrRecipientEmpty]，主题为空返回 [ErrSubjectEmpty]；
//   - 停机拒绝：停机后发送返回 [ErrStopped]，调用方可实施降级或转入离线队列；
//   - 网络与认证失败：返回底层 SMTP 协议或网络层包装错误。
package email

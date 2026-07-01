# plugins 知识库

## 概述

插件模块集合，提供可选的扩展能力。每个插件是独立的 Go module，按需引入。所有插件的 Runner 启动函数签名与 `xMain.Runner` 的附加协程参数兼容（`func(ctx context.Context, option ...any)`），可无缝挂载。

## 目录结构

```text
plugins/
├── grpc/                      # gRPC 框架插件（详见 grpc/AGENTS.md）
│   ├── runner/                #   gRPC 启动器
│   ├── interceptor/           #   一元/流式拦截器
│   ├── middleware/            #   服务级中间件注册表
│   ├── result/                #   gRPC 响应构建
│   ├── proto/                 #   proto 定义
│   ├── generate/              #   buf 生成的 Go 代码
│   ├── constant/              #   Metadata/Trailer 常量
│   └── utility/               #   gRPC 工具函数
├── cron/                      # 定时任务插件
│   ├── job.go                 #   Job 结构 + NewJob + AdaptJob（反射适配多种签名）
│   ├── runner/                #   Cron Runner（基于 robfig/cron/v3）
│   └── job_test.go / runner_test.go
├── async/                     # 异步任务插件
│   ├── async.go               #   Async() — 从父 context 分离独立上下文后异步执行
│   ├── task.go                #   Task 句柄（Cancel / Wait / IsDone）
│   ├── context.go             #   detachContext — 分离上下文（保留 values，断开 cancel 链）
│   └── option.go              #   WithName / WithDebug / WithLogger
└── email/                     # 邮件服务插件
    ├── client.go              #   EmailClient + InitClient 注册节点
    ├── message.go             #   邮件消息构建
    ├── template.go            #   TemplateManager（内置 embed 模板 + 外部目录覆盖）
    └── template/              #   内置 HTML 邮件模板
        ├── _base.html         #   基础布局模板
        ├── welcome.html       #   欢迎邮件
        ├── verification.html  #   验证码邮件
        └── reset_password.html#   重置密码邮件
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 启动 gRPC 服务 | `grpc/runner/runner.go` → `New()` | 详见 `grpc/AGENTS.md` |
| 启动定时任务 | `cron/runner/runner.go` → `New()` | 传入 `WithJobs()` Option |
| 定义定时任务 | `cron/job.go` → `NewJob(spec, fn)` | 支持 `func()` 和 `func(ctx)` 两种签名 |
| 异步执行函数 | `async/async.go` → `Async(parentCtx, fn, options...)` | 返回 `*Task` |
| 等待异步完成 | `async/task.go` → `Wait(task)` | 阻塞至任务完成 |
| 取消异步任务 | `async/task.go` → `Cancel(task)` | 非阻塞发送取消信号 |
| 发送邮件 | `email/client.go` → `EmailClient` | 通过 `xCtxUtil.GetEmailClient()` 获取实例 |
| 渲染邮件模板 | `email/template.go` → `TemplateManager.Render()` | 内置模板或外部目录覆盖 |
| 注册邮件节点 | `email/client.go` → `InitClient()` | 传入 `reg.Init.Use(xCtx.EmailClientKey, xEmail.InitClient)` |

## 约定

- **所有 Runner 统一签名**：`func(ctx context.Context, option ...any)`，可直接传入 `xMain.Runner` 的 `goroutineFunc...` 可变参数。
- **Option 模式统一**：grpc / cron 均使用 `WithXxx()` 函数配置，运行时通过 `option ...any` 支持动态覆盖。
- **异步任务禁止传 `*gin.Context`**：`Async()` 会 panic，必须用 `c.Request.Context()` 获取标准 `context.Context`。
- **异步上下文是分离的**：`detachContext` 保留父 context 的 values（DB、Redis 等），但断开 cancel 链，任务不受请求结束影响。
- **定时任务函数签名灵活**：通过反射适配，支持 `func()` 和 `func(context.Context)` 两种签名。
- **邮件模板支持覆盖**：内置 `embed` 模板，同名外部模板文件会覆盖内置版本（`EMAIL_TEMPLATE_DIR` 配置）。
- **邮件客户端通过注册节点初始化**：`InitClient` 是标准 `xRegNode.Node` 签名，注册到 `xCtx.EmailClientKey`。

## 反模式

- **禁止向 `Async()` 传 `*gin.Context`** — 会 panic，用 `c.Request.Context()` 代替。
- **禁止在异步任务中使用请求级 context** — 请求结束后 context 被取消，任务会中断。`Async()` 内部已处理分离，直接传入即可。
- **禁止跳过 `Async` 的 panic 恢复** — 内部 `defer recover()` 保证任务 panic 不会崩溃整个进程。
- **禁止手动创建 `EmailClient`** — 应通过注册节点 `InitClient` 初始化，从 context 获取。
- **禁止在 cron job 中执行长时间阻塞操作而不检查 ctx** — `Runner` 发出取消信号后，job 应及时退出。

## 调试路径

1. gRPC/cron Runner 不启动 — 检查是否作为 `goroutineFunc` 传给了 `xMain.Runner`。
2. 异步任务 panic 导致进程退出 — 不应该发生，`Async` 内部有 `recover`。检查是否绕过了 `Async` 直接 `go func()`。
3. 定时任务不执行 — 检查 cron 表达式语法（`WithSeconds` 默认 false，6 段式表达式需要开启）。
4. 邮件发送失败 — 检查 SMTP 配置（`EMAIL_HOST` / `EMAIL_PORT` / `EMAIL_TLS`），TLS 策略需匹配服务端。
5. 邮件模板渲染失败 — 确认模板名称不含 `.html` 后缀（`Render("welcome", data)` 而非 `Render("welcome.html", data)`）。

## 引用

- [grpc/](./grpc/AGENTS.md) — gRPC 框架插件（启动器/拦截器/中间件/proto）

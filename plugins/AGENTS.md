# plugins 知识库

**版本:** v2

## 概述

插件模块集合（`plugins/*`）基于 `application/v2` 的统一生命周期内核与 `foundation/v2` 基座构建，为下游应用提供可选扩展能力。仓库内共包含 7 个主插件模块与 5 个独立嵌套的数据库驱动适配器模块（均以 `/v2` 作为 module 路径），各主插件之间互相独立、零横向导入，全部通过 `application.Plugin` / `application.RunnablePlugin` 接口及 `ctxkey.Exposer` 接入拓扑装配与逆拓扑优雅停机。

## 目录结构

```text
plugins/
├── database/                  # 数据库核心插件 (plugins/database/v2)
│   ├── plugin.go              #   New() 受管 Tool 插件与连接池生命周期
│   ├── option.go              #   WithAdapter / FromEnv / 连接池与日志选项
│   ├── dialect/               #   方言注册表与 Adapter 接口契约
│   ├── models/                #   BaseEntity / BaseEntitySoftDelete 与泛型分页
│   ├── gormlog/               #   基于 logging.Logger 的 GORM 日志适配
│   └── adapter/               #   各驱动独立嵌套子模块（物理与依赖双隔离）
│       ├── mysql/             #     MySQL 驱动适配器 (adapter/mysql/v2)
│       ├── postgres/          #     PostgreSQL 驱动适配器 (adapter/postgres/v2)
│       ├── sqlite/            #     SQLite 纯 Go 驱动适配器 (adapter/sqlite/v2)
│       ├── sqlserver/         #     SQL Server 驱动适配器 (adapter/sqlserver/v2)
│       └── oracle/            #     Oracle 驱动适配器 (adapter/oracle/v2，需 CGO/Instant Client)
├── cache/                     # 泛型缓存插件 (plugins/cache/v2)
│   ├── plugin.go              #   New() 受管 Tool 插件（同时暴露 CacheManagerKey 与 RedisClientKey）
│   ├── config.go              #   WithMemory / WithRedis / FromEnv / Get / MustGet
│   ├── manager.go             #   KeyCacheOf / HashCacheOf / SetCacheOf / ListCacheOf 泛型门面
│   ├── driver/                #   编解码 Codec 与 Key/Hash/Set/List 接口契约
│   ├── memory/                #   分片内存后端与清理协程
│   └── redis/                 #   go-redis/v9 后端与错误归类
├── email/                     # 邮件服务插件 (plugins/email/v2)
│   ├── client.go              #   New() 受管 Tool 插件、Client.Send / SendHTML / SendTemplate
│   ├── message.go             #   NewMessage / Message / Attachment 构建
│   ├── template.go            #   TemplateManager（内置 embed 模板 + 外部目录/FS 覆盖）
│   └── template/              #   内置 HTML 邮件模板（_base / welcome / verification / reset_password）
├── web-gin/                   # Gin HTTP 服务插件 (plugins/web-gin/v2)
│   ├── plugin.go              #   New() 受管 Worker 插件（WithAddr / WithRoute / WithRequires / BridgeRequest）
│   ├── bind/                  #   泛型请求参数绑定与解码
│   ├── header/                #   标准 HTTP 请求头读写工具
│   ├── httpresult/            #   标准 envelope.Response 响应构建与状态码映射
│   ├── middleware/            #   追踪 ID、耗时记录、Recovery 与 CORS 等中间件
│   └── validator/             #   实例级参数校验引擎与多语言翻译（不污染全局 binding.Validator）
├── grpc/                      # gRPC 框架插件 (plugins/grpc/v2，详见 grpc/AGENTS.md)
│   ├── plugin.go              #   New() 受管 Worker 插件与拦截器桥接
│   ├── middleware/            #   服务级一元/流式中间件注册表
│   ├── grpcresult/            #   Success / SuccessWith[T] / Error / ToStatusError 与 R12 信封补齐
│   ├── constant/              #   Metadata / Trailer 常量
│   ├── proto/ & generate/     #   base.proto 与 buf 生成代码（严禁手改 .pb.go）
│   └── utility/               #   gRPC 元数据提取工具
├── cron/                      # 定时调度插件 (plugins/cron/v2)
│   └── plugin.go              #   New() 受管 Worker 插件（基于 gocron/v2，通过 Activator 激发 Component）
└── async/                     # 异步协程池插件 (plugins/async/v2)
    ├── plugin.go              #   New() 受管 Tool 插件、Pool 接口（PoolKey）与包级 Submit（基于 ants/v2）
    ├── task.go                #   Task 句柄（Ctx / Cancel / Done / Wait / WaitFor / Panic 观测）
    └── option.go              #   TaskOption 任务级选项
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 注册数据库 Tool | `database` → `database.New(database.WithAdapter(...))` | 通过 `ctxkey.DatabaseKey` 暴露 `*gorm.DB`；具体驱动由 `adapter/*` 独立引入 |
| 注册缓存 Tool | `cache` → `cache.New(cache.WithMemory() / WithRedis(...))` | 暴露 `ctxkey.CacheManagerKey`（Redis 模式额外暴露 `ctxkey.RedisClientKey`） |
| 注册邮件 Tool | `email` → `email.New(email.FromEnv() / WithConfig(...))` | 通过 `ctxkey.EmailClientKey` 暴露 `*email.Client` |
| 启动 Web HTTP Worker | `web-gin` → `webgin.New(webgin.WithRoute(...))` | 支持 `WithRequires(...)` 声明启动期依赖，`httpresult` 输出标准响应 |
| 启动 gRPC Worker | `grpc` → `grpc.New(grpc.WithServiceRegistrar(...))` | 详见 [grpc/AGENTS.md](./grpc/AGENTS.md) |
| 启动 Cron Worker | `cron` → `cron.New(cron.WithActivator(act), cron.WithCronJob(...))` | 到点经 `application.Activator` 触发目标 `Component` |
| 注册异步任务池 Tool | `async` → `async.New(async.WithCapacity(...))` | 通过 `async.PoolKey` 暴露 `async.Pool`；支持 `async.Submit(ctx, fn)` |

## 约定

- **统一插件返回工厂**：所有插件均通过 `New(opts ...Option)` 返回 `(application.PluginInfo, func() application.Plugin)` 或 `(application.PluginInfo, func() application.RunnablePlugin)`，交由 `app.RegisterTool` / `app.RegisterWorker` / `app.RegisterComponent` 统一装配。
- **强类型错误契约**：插件自有公开与内部失败出口统一返回 `*apperr.Error`（成功时必须返回真正的 `nil` 指针），仅 Go 标准库或第三方固定接口边界保留薄适配；受仓内 `make error-contract-check` 门禁守卫。
- **单向与零横向依赖**：所有插件仅依赖 `foundation/v2` 与 `application/v2`；`plugins/*` 之间禁止横向依赖；`plugins/database/adapter/*` 仅依赖 `foundation/v2` 与 `plugins/database/v2/dialect`。
- **类型安全上下文提取**：优先使用 `application.Get[T](ctx, key)`（或 `cache.Get(ctx)` 等强类型封装）提取共享组件，禁止裸 `ctx.Value` 类型断言或隐式全局单例。
- **显式日志注入**：需要日志能力时通过各插件 `WithLogger(logger)` 显式注入 `*logging.Logger`，禁止覆盖进程级 `slog.SetDefault`。
- **异步任务上下文隔离**：通过 `async.Pool.Submit` / `async.Submit` 提交后台任务时，内核会自动剥离请求级取消信号并保留容器与追踪信息，同时在 `App` 关机逆拓扑阶段先停入口流量再排空在途任务。
- **统一版本与发布**：全部 14 个模块清单由 `.github/scripts/manifest/v2-modules.txt` 统一维护，禁止手工修改单模块版本或直接编辑生成的 `.pb.go` 文件。

## 反模式

- **禁止在插件间横向 import** — 例如在 `web-gin` 中直接导入 `database` 或 `cache` 包；跨组件协作应通过 `ctxkey` 与 `application.Get[T]` 解耦。
- **禁止在 `plugins/database` 主模块引入具体数据库驱动** — 驱动依赖必须隔离在 `plugins/database/adapter/*` 各自独立的 `go.mod` 中。
- **禁止裸返回非强类型 `error` 作为插件自有 API 结果** — 会被 `make error-contract-check` 拦截；也禁止返回装入接口的 typed-nil。
- **禁止在 `cron` 作业或 `async` 任务中绕过 `Task.Ctx()` / `Component` 生命周期上下文** — 需响应停机取消信号以确保关机排空在预算内收敛。
- **禁止在 `web-gin` 中篡改全局 `binding.Validator`** — 自定义规则应通过 `webgin.WithValidatorRegistrar` 挂载到实例级校验器。

## 引用

- [根项目知识库](../AGENTS.md) — V2 总体架构、模块清单与常用门禁命令
- [grpc/](./grpc/AGENTS.md) — gRPC 框架插件详细说明

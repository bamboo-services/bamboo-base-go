# major 知识库

## 概述

核心层模块（`github.com/bamboo-services/bamboo-base-go/major`），构建于 `common` 和 `defined` 之上，提供应用启动框架（注册系统 + Runner）、HTTP 中间件链、统一响应处理、数据模型基类、泛型缓存系统、声明式选项配置与上下文提取工具。是下游业务应用直接交互的最高层 SDK。**major 层负责解耦框架依赖（gin），使 common 层保持干净、无 HTTP 框架依赖。**

## 目录结构

```text
major/
├── main/
│   ├── runner.go              # Runner() — HTTP 启动 + 信号 + 优雅关闭 + 附加协程
│   ├── web.go                 # WebServer 配置
│   ├── goroutine.go           # goroutineFunc 管理
│   └── option.go              # Runner 内部选项
├── register/                  # 节点化注册系统（详见 register/AGENTS.md）
│   ├── register.go            #   Register() 入口
│   ├── register_config.go     #   .env 加载
│   ├── register_logger.go     #   全局 slog 初始化 + GinLogExtractor 注入
│   ├── register_gin.go        #   Gin 引擎 + 中间件 + 验证器 + ContextExtractor 注入
│   ├── node/                  #   节点队列管理（Use / Exec / UseAfterExec）
│   └── init/                  #   内置初始化节点（雪花/缓存/数据库）
├── middleware/
│   ├── response.go            # ResponseMiddleware — 统一响应兜底中间件
│   ├── cors.go                # ReleaseAllCors — 全量 CORS 中间件
│   └── option_request.go      # OPTIONS 请求处理
├── result/
│   └── result.go              # Success / SuccessHasData / Error / AbortError
├── models/
│   ├── base_entity.go         # BaseEntity — 无软删除的实体基类
│   ├── base_entity_soft_delete.go # BaseEntityWithSoftDelete — 带软删除的实体基类
│   ├── page.go                # PageRequest / PageResponse / 分页规范化
│   └── provider.go            # GeneProvider 接口
├── helper/
│   ├── context.go             # RequestContext — 请求唯一 ID + 开始时间中间件
│   ├── http_logger.go         # HttpLogger — HTTP 请求日志中间件（含脱敏）
│   └── panic_recovery.go      # PanicRecovery — panic 恢复中间件
├── route/
│   ├── no_route.go            # NoRoute — 404 统一处理
│   └── no_method.go           # NoMethod — 405 统一处理
├── cache/                     # 泛型缓存系统（详见 cache/AGENTS.md）
│   ├── manager.go             #   Manager 门面 + 泛型工厂方法
│   ├── driver/                #   缓存接口（KeyCache / HashCache / SetCache / ListCache）
│   ├── memory/                #   内存后端（分片 + TTL + janitor）
│   └── redis/                 #   Redis 后端
├── option/                    # 声明式配置层（详见 option/AGENTS.md）
│   ├── option.go              #   Option 类型 + Config 聚合体 + Apply
│   ├── cache.go               #   薄桥接层：WithCache + 类型别名重导出
│   ├── database.go            #   薄桥接层：WithDatabase + 类型别名重导出
│   ├── router.go              #   RouteRegistrar + WithRoute
│   ├── cache/                 #   缓存配置子包（WithRedis/WithMemory/FromEnv）
│   └── database/              #   数据库配置子包（MySQL/Postgres/SQLite/FromEnv）
├── utility/                   # HTTP 请求绑定与上下文提取工具
│   ├── bind.go                #   Bind[T]() — 请求参数绑定入口
│   ├── binding.go             #   Binding[T].Data()/Query()/URI()/Header()
│   └── context/               #   上下文组件提取（GetDB / GetRDB / GetCacheManager 等）
│       ├── gin_extractor_impl.go # ContextExtractor 接口 + gin 实现
│       ├── custom.go          #     MustGet[T] / Get[T] — 通用泛型组件提取
│       ├── database.go        #     MustGetDB / GetDB
│       ├── nosql.go           #     MustGetRDB / GetRDB
│       ├── cache.go           #     MustGetCacheManager / GetCacheManager
│       ├── email.go           #     MustGetEmailClient / GetEmailClient
│       └── snowflake.go       #     GetSnowflakeNode / GenerateSnowflakeID
├── validator/                 # 验证错误处理（从 common 迁移来，解耦 gin 依赖）
│   ├── gin_validate_provider.go # ValidateProvider 实现（从 gin binding 获取引擎）
│   └── response.go            #   HandleValidationError — 友好中文验证错误响应
├── log/                       # 日志提取器（桥接 gin 与 common/log）
│   ├── gin_extractor.go       # GinLogExtractor — 从 gin.Context 提取 trace ID
│   └── gorm.go                # GORM 日志适配器（从 common/log 迁移来）
├── hook/
│   └── redis.go               # Redis 钩子
└── go.mod                     # 独立模块定义
```

> **Runner 装配链路**：`opts []xOption.Option` → `option.Apply()` → `DatabaseInit`/`CacheInit` 工厂 → `RegNode.UseAfterExec()` → context 注入。
> 业务侧原先在 `startup.Init()` 中手写的 db/redis 节点，现可由 `WithMySQL/WithPostgres/WithSQLite/WithRedis` 一行替代；AutoMigrate 表声明和建表后数据初始化（种子数据）已纳入 `DatabaseOption`，通过 `WithAutoMigrate` / `WithPrepare` 声明式装配，由 `DatabaseInit` 在 DB 建连后自动执行，不再需要手写迁移节点或借道 `WithRoute` 回调。

> **架构变更说明**：`HandleValidationError`、`Bind` 绑定工具、`GetDB/GetRDB` 等上下文提取函数已从 `common` 层迁移到 `major` 层，并通过 `ContextExtractor` 接口解耦 gin 依赖。common 层现在完全不依赖 gin，保持纯 Go 依赖。

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 启动应用 | `main/runner.go` → `Runner()` | `Runner(reg, logger, opts, ...goroutineFunc)` |
| 配置缓存后端 | `option/cache.go` → `WithCache(xOptionCache.WithRedis)` / `WithCache(xOptionCache.WithMemory)` | 声明式双层选择 Redis 或内存缓存 |
| 配置数据库 | `option/database.go` → `WithDatabase(xOptionDB.MySQL)` / `WithDatabase(xOptionDB.Postgres)` / `WithDatabase(xOptionDB.SQLite)` | 双层指定驱动，Runner 自动装配 |
| 从环境变量装配数据库 | `option/database.go` → `WithDatabase(xOptionDB.FromEnv)` | 自动读取 `DATABASE_DRIVER` + 分项配置拼装 DSN |
| 注册 HTTP 路由 | `option/router.go` → `WithRoute` / `WithRouteGroup` | 可叠加多个，支持插件自带路由 |
| 使用缓存 | 从 context 获取 `*xCache.Manager` → `xCache.KeyCacheOf(mgr)` 等 | 泛型缓存接口，自动按后端分发 |
| 返回成功响应 | `result/result.go` → `Success()` / `SuccessHasData()` | 自动注入 context / code / overhead |
| 返回错误响应 | `result/result.go` → `Error()` / `AbortError()` | 传入 `xError.ErrorCode` |
| 定义数据库实体 | `models/base_entity.go` → `BaseEntity` | 嵌入即可获得 ID + CreatedAt + UpdatedAt |
| 定义带软删除的实体 | `models/base_entity_soft_delete.go` | 嵌入 `BaseEntityWithSoftDelete` |
| 为实体指定基因 | `models/provider.go` → `GeneProvider` | 实现 `GetGene()` 方法 |
| 分页查询 | `models/page.go` → `PageRequest` / `PageResponse[T]` | 泛型分页，支持规范化 |
| 绑定请求参数 | `utility/bind.go` → `Bind(ctx, &req)` | `.Data()` / `.Query()` / `.URI()` / `.Header()` |
| 从 context 获取 DB | `utility/context/database.go` → `GetDB(ctx)` | 返回 `*gorm.DB, *xError.Error` |
| 从 context 获取 Redis | `utility/context/nosql.go` → `GetRDB(ctx)` | 返回 `*redis.Client, *xError.Error` |
| 从 context 获取缓存管理器 | `utility/context/cache.go` → `GetCacheManager(ctx)` | 返回 `*xCache.Manager, *xError.Error` |
| 处理 404 | `route/no_route.go` → `NoRoute()` | 绑定到 `router.NoRoute` |
| 添加 CORS | `middleware/cors.go` → `ReleaseAllCors()` | 全量放行，按需使用 |
| 响应兜底 | `middleware/response.go` → `ResponseMiddleware()` | 检查未写入响应的请求 |

## 约定

- **Runner 签名固定**：`Runner(reg *xReg.Reg, log *xLog.LogNamedLogger, opts []xOption.Option, goroutineFunc ...func(ctx context.Context, extra ...any))`。
- **路由通过 Option 注册**：使用 `xOption.WithRoute` / `xOption.WithRouteGroup` 声明路由，可叠加多个注册器并按调用顺序执行；插件可直接暴露 `RouteRegistrar` 供业务侧 `WithRoute` 导入，实现「插件自带路由、一行接入」。
- **Gin 中间件链顺序固定**：`RequestContext → PanicRecovery → HttpLogger → InjectContext`，由 `register_gin.go` 的 `engineInit()` 自动挂载，业务侧不需要手动添加。
- **响应必须通过 `xResult.*` 函数**：不要直接调用 `ctx.JSON()`，否则 `ResponseMiddleware` 兜底逻辑可能将其视为"开发者错误"。
- **ErrorCode.Code 前 3 位 = HTTP 状态码**：`xResult.Error()` 中 `int(errorCode.Code/100)` 决定 HTTP 响应码。
- **实体基类二选一**：`BaseEntity`（无软删除）或 `BaseEntityWithSoftDelete`（带软删除），不要自行定义 ID/CreatedAt/UpdatedAt 字段。
- **ID 自动生成**：`BeforeCreate` 钩子在实体 ID 为零值时自动调用 `xSnowflake.GenerateID(gene)`，不要手动设置 ID。
- **`CreatedAt` 序列化为 `created_at`**，`UpdatedAt` 序列化为 `updated_at`，均会输出到 JSON 响应中。
- **分页默认值**：页码从 1 开始，默认每页 20 条，最大 200 条。可通过覆盖 `DefaultPageConfig` 全局调整。
- **HttpLogger 自动脱敏**：password / token / secret / cookie 等敏感字段在调试日志中自动脱敏，不会泄露。
- **ContextExtractor 解耦**：`major/utility/context` 中的 `GetDB/GetRDB` 等函数通过 `ContextExtractor` 接口解耦 gin 依赖，在 `register_gin.go` 初始化时注入 gin 实现。common 层不再依赖 gin。

## 反模式

- **禁止在业务 handler 中直接 `ctx.JSON()`** — 绕过 `xResult` 会导致 `ResponseMiddleware` 兜底输出"开发者错误"。
- **禁止手动设置 `BaseEntity.ID`** — ID 由雪花算法钩子自动生成。
- **禁止在 Runner 外部创建 Gin 引擎** — 引擎由 `register_gin.go` 统一创建并挂载中间件链。
- **禁止跳过 `ResponseMiddleware`** — 它是未写入响应和 panic 恢复后的最后一道防线。
- **禁止用 `gin.Default()` 或 `gin.New()` 替代 SDK 引擎** — 会丢失所有中间件和验证器注册。
- **禁止在 `goroutineFunc` 中忽略 ctx 取消** — `Runner` 在收到信号后会 cancel ctx，附加协程应监听 `ctx.Done()` 及时退出。
- **禁止在 common 层直接依赖 gin** — 使用 major 层的 ContextExtractor 提取上下文，common 层保持纯 Go 依赖。
- **禁止直接使用 common/utility/context 中的 context 提取函数** — 这些函数已从 common 移出（或变为空壳），应使用 `major/utility/context` 中的函数。

## 调试路径

1. 所有请求返回"开发者错误" — handler 没有调用 `xResult.Success*()` 或 `xResult.Error()`，被 `ResponseMiddleware` 兜底。
2. 请求 ID 在响应中缺失 — 检查 `RequestContext()` 中间件是否在中间件链首位。
3. panic 后服务直接退出 — 检查 `PanicRecovery()` 中间件是否已挂载。
4. 分页参数不生效 — 确认调用了 `PageRequest.Normalize()` 或使用 `NewPageFromRequest()`。
5. 实体 ID 全为 0 — 确认嵌入的是 `BaseEntity` / `BaseEntityWithSoftDelete` 而非自定义字段，且 `BeforeCreate` 钩子未被覆盖。
6. 日志中敏感信息泄露 — 确认 HttpLogger 的脱敏字段列表覆盖了你的场景（检查 `sanitizeHeaders` / `sanitizeJSONBody`）。
7. `GetDB/GetRDB` 取不到实例 — 确认 `ContextExtractor` 已通过 `register_gin.go` 注入，且对应 ContextKey 在 init 阶段已注册。

## 引用

- [register/](./register/AGENTS.md) — 节点化注册系统与初始化流程
- [cache/](./cache/AGENTS.md) — 泛型缓存系统（Key/Hash/Set/List）
- [option/](./option/AGENTS.md) — 声明式配置层
- [register/init/](./register/init/AGENTS.md) — 内置初始化节点（雪花/缓存/数据库）
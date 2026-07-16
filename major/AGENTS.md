# major 知识库

## 概述

核心层模块（`github.com/bamboo-services/bamboo-base-go/major`），构建于 `common` 和 `defined` 之上，提供应用启动框架（注册系统 + Runner）、HTTP 中间件链、统一响应处理、数据模型基类与辅助工具。是下游业务应用直接交互的最高层 SDK。

## 目录结构

```text
major/
├── main/
│   └── runner.go              # Runner() — HTTP 服务启动 + 信号监听 + 优雅关闭 + 附加协程
├── register/                  # 节点化注册系统（详见 register/AGENTS.md）
│   ├── register.go            #   Register() 入口
│   ├── register_config.go     #   .env 加载
│   ├── register_logger.go     #   全局 slog 初始化
│   ├── register_gin.go        #   Gin 引擎 + 中间件 + 验证器注册
│   ├── node/                  #   节点队列管理
│   └── init/                  #   内置初始化节点
├── middleware/
│   ├── response.go            # ResponseMiddleware — 统一响应兜底中间件
│   ├── cors.go                # ReleaseAllCors — 全量 CORS 中间件
│   └── option.go              # 中间件选项
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
├── cache/
│   ├── interface.go           # KeyCache / HashCache 泛型缓存接口
│   └── struct.go              # 缓存结构体定义
├── hook/
│   └── redis.go               # Redis 钩子
└── go.mod                     # 独立模块定义
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 启动应用 | `main/runner.go` → `Runner()` | `Runner(reg, logger, routeFunc, ...goroutineFunc)` |
| 返回成功响应 | `result/result.go` → `Success()` / `SuccessHasData()` | 自动注入 context / code / overhead |
| 返回错误响应 | `result/result.go` → `Error()` / `AbortError()` | 传入 `xError.ErrorCode` |
| 定义数据库实体 | `models/base_entity.go` → `BaseEntity` | 嵌入即可获得 ID + CreatedAt + UpdatedAt |
| 定义带软删除的实体 | `models/base_entity_soft_delete.go` | 嵌入 `BaseEntityWithSoftDelete` |
| 为实体指定基因 | `models/provider.go` → `GeneProvider` | 实现 `GetGene()` 方法 |
| 分页查询 | `models/page.go` → `PageRequest` / `PageResponse[T]` | 泛型分页，支持规范化 |
| 绑定请求参数 | 通过 `xUtil.Bind(ctx, &req)` 调用 | 实现在 `common/utility/` 中 |
| 处理 404 | `route/no_route.go` → `NoRoute()` | 绑定到 `router.NoRoute` |
| 添加 CORS | `middleware/cors.go` → `ReleaseAllCors()` | 全量放行，按需使用 |
| 响应兜底 | `middleware/response.go` → `ResponseMiddleware()` | 检查未写入响应的请求 |

## 约定

- **Runner 签名固定**：`Runner(reg *xReg.Reg, log *xLog.LogNamedLogger, routeFunc func(reg *xReg.Reg), goroutineFunc ...func(ctx context.Context, option ...any))`。
- **Gin 中间件链顺序固定**：`RequestContext → PanicRecovery → HttpLogger → InjectContext`，由 `register_gin.go` 的 `engineInit()` 自动挂载，业务侧不需要手动添加。
- **响应必须通过 `xResult.*` 函数**：不要直接调用 `ctx.JSON()`，否则 `ResponseMiddleware` 兜底逻辑可能将其视为"开发者错误"。
- **ErrorCode.Code 前 3 位 = HTTP 状态码**：`xResult.Error()` 中 `int(errorCode.Code/100)` 决定 HTTP 响应码。
- **实体基类二选一**：`BaseEntity`（无软删除）或 `BaseEntityWithSoftDelete`（带软删除），不要自行定义 ID/CreatedAt/UpdatedAt 字段。
- **ID 自动生成**：`BeforeCreate` 钩子在实体 ID 为零值时自动调用 `xSnowflake.GenerateID(gene)`，不要手动设置 ID。
- **`CreatedAt` 序列化为 `created_at`**，`UpdatedAt` 序列化为 `updated_at`，均会输出到 JSON 响应中。
- **分页默认值**：页码从 1 开始，默认每页 20 条，最大 200 条。可通过覆盖 `DefaultPageConfig` 全局调整。
- **HttpLogger 自动脱敏**：password / token / secret / cookie 等敏感字段在调试日志中自动脱敏，不会泄露。

## 反模式

- **禁止在业务 handler 中直接 `ctx.JSON()`** — 绕过 `xResult` 会导致 `ResponseMiddleware` 兜底输出"开发者错误"。
- **禁止手动设置 `BaseEntity.ID`** — ID 由雪花算法钩子自动生成。
- **禁止在 Runner 外部创建 Gin 引擎** — 引擎由 `register_gin.go` 统一创建并挂载中间件链。
- **禁止跳过 `ResponseMiddleware`** — 它是未写入响应和 panic 恢复后的最后一道防线。
- **禁止用 `gin.Default()` 或 `gin.New()` 替代 SDK 引擎** — 会丢失所有中间件和验证器注册。
- **禁止在 `goroutineFunc` 中忽略 ctx 取消** — `Runner` 在收到信号后会 cancel ctx，附加协程应监听 `ctx.Done()` 及时退出。

## 调试路径

1. 所有请求返回"开发者错误" — handler 没有调用 `xResult.Success*()` 或 `xResult.Error()`，被 `ResponseMiddleware` 兜底。
2. 请求 ID 在响应中缺失 — 检查 `RequestContext()` 中间件是否在中间件链首位。
3. panic 后服务直接退出 — 检查 `PanicRecovery()` 中间件是否已挂载。
4. 分页参数不生效 — 确认调用了 `PageRequest.Normalize()` 或使用 `NewPageFromRequest()`。
5. 实体 ID 全为 0 — 确认嵌入的是 `BaseEntity` / `BaseEntityWithSoftDelete` 而非自定义字段，且 `BeforeCreate` 钩子未被覆盖。
6. 日志中敏感信息泄露 — 确认 HttpLogger 的脱敏字段列表覆盖了你的场景（检查 `sanitizeHeaders` / `sanitizeJSONBody`）。

## 引用

- [register/](./register/AGENTS.md) — 节点化注册系统与初始化流程

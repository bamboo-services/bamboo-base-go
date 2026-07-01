# register 知识库

## 概述

节点化注册系统，是整个应用的组件初始化与依赖注入核心。通过 `Register(ctx, nodeList)` 统一完成环境加载、日志器创建、雪花算法初始化、用户自定义节点注册及 Gin 引擎构建。

## 目录结构

```text
register/
├── register.go            # Register() 入口函数 + Reg 结构体定义
├── register_config.go     # .env 环境变量加载（godotenv）
├── register_logger.go     # 全局 slog 日志器初始化（控制台 + 文件切割）
├── register_gin.go        # Gin 引擎创建与中间件挂载
├── node/
│   └── node.go            # RegNode 节点队列：Use() / Exec() / InjectContext()
└── init/
    └── init_snowflake.go  # 内置节点：雪花算法初始化
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 启动应用 | `register.go` → `Register()` | 创建 Reg 实例，返回 `*Reg` |
| 注册自定义初始化节点 | `node/node.go` → `RegNode.Use()` | 传入 `xCtx.ContextKey` + `Node func` |
| 访问已初始化的组件 | `node/node.go` → `GetRegNodeList(ctx)` | 从 context 提取组件实例 |
| 添加 .env 配置项 | `register_config.go` → `configInit()` | godotenv 加载 .env |
| 调整日志输出 | `register_logger.go` → `loggerInit()` | 日志文件在 `.logs/` 下 |
| 修改 Gin 中间件链 | `register_gin.go` → `engineInit()` | RequestContext → PanicRecovery → HttpLogger → InjectContext |
| 修改内置初始化节点 | `init/init_snowflake.go` | 雪花算法节点为内置默认节点 |

## 约定

- **节点注册顺序即依赖顺序**：`Use()` 按调用顺序入队，`Exec()` 按入队顺序执行。被依赖的组件必须先注册（如 Config → Logger → Database → Business）。
- **ContextKey 全局唯一**：同一 `ContextKey` 重复注册会 panic。所有 Key 在 `defined/context/context.go` 统一定义。
- **Exec() 只能调用一次**：执行后 `list` 被置为 `nil`，再次调用 `Use()` 会 panic（"初始化外部禁止二次初始化"）。
- **节点函数签名固定**：`func(ctx context.Context) (any, error)`。返回值通过 `context.WithValue` 存入，后续节点用 `ctx.Value(key)` 提取。
- **内置节点不可绕过**：雪花算法节点（`SnowflakeNodeKey`）由 `Register()` 内部强制注册，无需用户传入。
- **`Reg` 结构体字段首字母大写**（`Serve`、`Init`）供外部直接访问，不使用 getter。

## 反模式

- **禁止在 `Exec()` 后注册新节点** — `list` 已被清空，调用 `Use()` 会 panic。
- **禁止传 nil 给 `Use()` 的 `registerFunc`** — 会直接 panic。
- **禁止传空 `ContextKey`**（`xCtx.Nil`）— `Use()` 内部会静默忽略。
- **禁止在节点函数中 panic 而不返回 error** — 虽然能中断启动，但会丢失错误上下文。应返回 `error`，由 `Exec()` 统一 panic 并携带索引/键名信息。

## 调试路径

1. 启动时 panic "重复注册 ContextKey: xxx" — 检查 `nodeList` 中是否有重复的 `xCtx.ContextKey`。
2. 启动时 panic "执行注册节点失败: index=N Key=xxx err=yyy" — 定位到第 N 个注册节点，检查其 `Node func` 的 error 返回值。
3. 请求上下文中取不到组件实例 — 确认 `InjectContext()` 中间件已挂载（`register_gin.go` 默认挂载），且 ContextKey 与注册时一致。
4. 日志文件未生成 — 检查工作目录是否有 `.logs/` 写入权限，`loggerInit()` 失败会 panic。

## 引用

- [init/](./init/AGENTS.md) — 内置初始化节点实现（雪花算法）

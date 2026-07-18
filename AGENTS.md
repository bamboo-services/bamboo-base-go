# 项目知识库

**生成日期:** 2026-07-01
**提交:** 558496c
**分支:** master

## 概述

Bamboo Base Go 是 Bamboo 服务的 Go 基础组件库，面向 Gin HTTP API 与 gRPC 服务的统一启动、错误处理、日志、配置与上下文注入。项目使用 Go Workspace（`go.work`）管理多个独立模块，以"节点化注册系统"为核心设计理念，提供从环境加载到优雅关闭的一站式应用框架。

## 目录结构

```text
bamboo-base/
├── go.work                       # Go 工作区配置（7 个模块）
├── go.mod                        # 根模块（仅占位，不包含代码）
├── Makefile                      # 构建/测试/发布命令
├── version                       # 根版本号（当前 v1）
├── .env.example                  # 环境变量模板
├── README.md                     # 项目说明文档
├── defined/                      # 定义层（最底层，无外部依赖）
│   ├── context/                  #   上下文键常量 + ContextNodeList
│   ├── env/                      #   环境变量键常量 + GetEnv* 读取函数
│   ├── http/                     #   HTTP 头部常量
│   └── go.mod
├── common/                       # 通用层（依赖 defined）
│   ├── error/                    #   错误码体系 + IError 接口
│   ├── log/                      #   slog 日志系统（双写 + 切割）
│   ├── snowflake/                #   基因雪花算法
│   ├── validator/                #   8 个自定义验证器 + 中文翻译
│   ├── utility/                  #   工具函数（绑定/加密/生成/密码/字符串...）
│   │   ├── context/              #     上下文组件提取
│   │   └── package/              #     工具实现
│   ├── base_response.go          #   BaseResponse 标准响应结构
│   └── go.mod
├── major/                        # 核心层（依赖 common + defined）
│   ├── main/                     #   Runner 应用启动器
│   ├── register/                 #   节点化注册系统
│   ├── middleware/               #   Gin 中间件（响应兜底/CORS）
│   ├── result/                   #   HTTP 响应处理
│   ├── models/                   #   实体基类 + 分页模型
│   ├── helper/                   #   请求上下文/日志/Panic 中间件
│   ├── route/                    #   404/405 统一处理
│   ├── cache/                    #   泛型缓存接口
│   ├── hook/                     #   Redis 钩子
│   └── go.mod
└── plugins/                      # 插件层（各自独立 module）
    ├── grpc/                     #   gRPC 框架插件
    ├── cron/                     #   定时任务插件
    ├── async/                    #   异步任务插件
    └── email/                    #   邮件服务插件
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 快速启动 HTTP 服务 | `major/register` → `Register()` + `major/main` → `Runner()` | 最少 3 行代码启动 |
| HTTP + gRPC 一体化启动 | `plugins/grpc/runner` → `New()` | 作为附加协程传入 Runner |
| 返回成功响应 | `major/result` → `Success()` / `SuccessHasData()` | |
| 返回错误响应 | `major/result` → `Error()` / `AbortError()` | |
| 绑定请求参数 | `common/utility` → `Bind(ctx, &req)` | `.Data()` / `.Query()` / `.URI()` / `.Header()` |
| 定义数据库实体 | `major/models` → `BaseEntity` / `BaseEntityWithSoftDelete` | |
| 分页查询 | `major/models` → `PageRequest` / `PageResponse[T]` | 泛型分页 |
| 生成唯一 ID | `common/snowflake` → `GenerateID(gene)` | 基因雪花算法 |
| 查找错误码 | `common/error` → `error_code.go` | 400xx-504xx 分段 |
| 创建命名日志器 | `common/log` → `WithName(xLog.Named*)` | |
| 添加自定义验证器 | `common/validator` → `RegisterCustomValidators()` | |
| 注册初始化组件 | `major/register/node` → `Use(key, func)` | |
| 从 context 取组件 | `common/utility/context` → `GetDatabase()` 等 | |
| 配置环境变量 | `defined/env` → `env.go` | 所有 Key 常量定义 |
| 添加定时任务 | `plugins/cron` → `NewJob(spec, fn)` | |
| 异步执行 | `plugins/async` → `Async(ctx, fn)` | |
| 发送邮件 | `plugins/email` → `EmailClient` | |

## 代码地图

| 符号 | 类型 | 位置 | 作用 |
|------|------|------|------|
| `xReg.Register` | 函数 | `major/register/register.go` | 应用入口：创建 Reg 实例 |
| `xMain.Runner` | 函数 | `major/main/runner.go` | HTTP 启动 + 信号 + 优雅关闭 |
| `xReg.Reg` | 结构体 | `major/register/register.go` | 核心注册结构（Serve + Init） |
| `xRegNode.RegNode` | 结构体 | `major/register/node/node.go` | 节点队列管理器 |
| `xResult.Success` | 函数 | `major/result/result.go` | 200 成功响应 |
| `xResult.Error` | 函数 | `major/result/result.go` | 错误响应（计算 HTTP 码） |
| `xError.ErrorCode` | 结构体 | `common/error/error_code.go` | 错误码定义 |
| `xError.NewError` | 函数 | `common/error/error_new.go` | 错误构造 |
| `xBase.BaseResponse` | 结构体 | `common/base_response.go` | 标准响应体 |
| `xLog.WithName` | 函数 | `common/log/command.go` | 创建命名日志器 |
| `xSnowflake.GenerateID` | 函数 | `common/snowflake/snowflake.go` | 雪花 ID 生成 |
| `xSnowflake.Gene` | 类型 | `common/snowflake/gene.go` | 基因类型（0-63） |
| `xModels.BaseEntity` | 结构体 | `major/models/base_entity.go` | 实体基类 |
| `xModels.PageRequest` | 结构体 | `major/models/page.go` | 分页请求 |
| `xModels.PageResponse[T]` | 泛型结构体 | `major/models/page.go` | 分页响应 |
| `xUtil.Bind` | 泛型函数 | `common/utility/utility.go` | 请求参数绑定入口 |
| `xCtxUtil.GetDatabase` | 函数 | `common/utility/context/database.go` | 从 context 获取 DB |
| `xEnv.GetEnvString` | 函数 | `defined/env/env.go` | 读环境变量 |
| `xCtx.ContextKey` | 类型 | `defined/context/context.go` | 上下文键类型 |
| `xGrpcRunner.New` | 函数 | `plugins/grpc/runner/runner.go` | gRPC 启动器 |
| `xCronRunner.New` | 函数 | `plugins/cron/runner/runner.go` | Cron 启动器 |
| `xAsync.Async` | 函数 | `plugins/async/async.go` | 异步任务执行 |
| `xEmail.InitClient` | 函数 | `plugins/email/client.go` | 邮件客户端注册节点 |

## 模块架构

```text
┌──────────────────────────────────────────────────────────┐
│                    业务应用 (下游)                         │
└──────────────┬──────────────────────────┬────────────────┘
               │                          │
    ┌──────────▼──────────┐   ┌──────────▼──────────┐
    │      major 层        │   │     plugins 层       │
    │  (register/main/     │   │  (grpc/cron/         │
    │   middleware/result/ │   │   async/email)       │
    │   models/helper)     │   │                      │
    └──────────┬──────────┘   └──────────┬──────────┘
               │                          │
               │     ┌────────────────────┘
               │     │
    ┌──────────▼─────▼──────────┐
    │       common 层            │
    │  (error/log/snowflake/     │
    │   validator/utility)       │
    └──────────┬────────────────┘
               │
    ┌──────────▼──────────┐
    │      defined 层      │
    │  (context/env/http)  │
    └─────────────────────┘
```

依赖方向（单向）：`defined ← common ← major`，`defined + common ← plugins/*`

各插件间的依赖关系：
- `plugins/grpc` 依赖 `defined` + `common`
- `plugins/cron` 依赖 `common`
- `plugins/async` 依赖 `common`
- `plugins/email` 依赖 `defined` + `common`

## 约定

- **Go Workspace 管理**：7 个独立 module 通过 `go.work` 关联，每个 module 有独立的 `go.mod` 和 `version` 文件。
- **导入别名统一**：使用 `x` 前缀的简短别名（`xReg` / `xMain` / `xError` / `xLog` / `xModels` / `xResult` / `xUtil` / `xEnv` / `xCtx` / `xVaild` / `xSnowflake` / `xMiddle` / `xHelper` / `xRoute` / `xCache`）。
- **中文注释**：所有公开类型和函数使用中文注释，遵循 Go doc 注释规范。
- **日志器命名体系**：使用 4 字母大写常量标识模块（`NamedMAIN` / `NamedINIT` / `NamedRESU` / `NamedHTTP` / `NamedGRPC` 等，定义在 `common/log/const.go`）。
- **错误码前 3 位 = HTTP 状态码**：`ErrorCode.Code/100` 决定 HTTP 响应状态码。
- **雪花 ID 基因体系**：ID 中嵌入业务基因类型（0-63），系统级 0-15、业务级 16-63。
- **节点化注册**：组件初始化通过 `Use(key, func)` → `Exec()` 链式完成，按注册顺序执行。
- **环境变量配置**：使用 `godotenv` 加载 `.env`，所有 Key 在 `defined/env/env.go` 统一定义为 `EnvKey` 类型常量。
- **`this.` 使用规则**：调用内部方法或继承方法时必须使用 `this.`；访问成员变量时禁止使用 `this.`。
- **Optional 判空**：在性能影响可忽略的情况下，优先使用 `Optional` 进行优雅判空。

## 反模式

- **禁止跨层依赖**：`defined` 不可依赖上层；`common` 不可依赖 `major`；`plugins` 间不可互相依赖（除非通过 common）。
- **禁止绕过注册系统创建 Gin 引擎**：必须通过 `xReg.Register()` 创建，否则丢失中间件和验证器。
- **禁止直接调用 `slog.*`**：使用 `xLog.WithName()` 创建命名日志器。
- **禁止直接 `ctx.JSON()`**：使用 `xResult.*` 系列函数返回响应。
- **禁止手动编辑 `generate/*.pb.go`**：由 `buf generate` 生成。
- **禁止跨模块使用相对路径导入**：必须使用完整的 `github.com/bamboo-services/bamboo-base-go/...` 路径。

## 独特风格

- **节点化注册系统**：非传统 DI 容器，通过 `context.Context` 传递已初始化的组件实例，`Use/Exec` 两阶段完成。
- **基因雪花算法**：ID 中嵌入业务类型基因（6 bit），支持 64 种业务类型分类，便于数据分片和类型识别。
- **ErrorCode 体系**：预定义了 150+ 个覆盖全 HTTP 状态码段的错误码常量，Output 字段使用大写下划线格式。
- **响应统一兜底**：`ResponseMiddleware` 确保所有未写入响应的请求得到标准错误输出（"开发者错误"）。
- **日志脱敏**：`HttpLogger` 自动对 password/token/secret/cookie 等字段脱敏。
- **多模块独立发版**：每个 module 有独立 `version` 文件，通过 `Makefile` 的 `release-all` 按依赖顺序打 tag 发布。

## 常用命令

```bash
# 开发
make test                    # 运行所有测试
make proto                   # 使用 buf 生成 gRPC 代码
make tidy                    # 整理 Go 模块依赖
make vet                     # go vet 检查所有模块

# 发布（统一版本号 vX.Y.Z）
make release VERSION=vX.Y.Z  # 创建 GitHub Release，触发 Action 自动给所有子模块打 tag
                             # 子 tag: defined/vX.Y.Z, common/vX.Y.Z, major/vX.Y.Z,
                             #         plugins/{cron,grpc,async,email}/vX.Y.Z
                             # Action 同时自动 bump 所有子模块 go.mod 依赖到 vX.Y.Z
```

> 旧的多版本号方案（每个模块独立 `version` 文件 + `make release PKG=` / `make release-all`）
> 已废弃。所有模块统一使用 release 的版本号，由 `.github/workflows/release.yml` 自动处理。

## 备注

- **Go 版本要求**：1.25.0+
- **根 `go.mod`**：仅占位，不包含实际代码，用于 `go get github.com/bamboo-services/bamboo-base-go` 的安装路径锚点。
- **版本号格式**：`v{major}-{YYYYMMDDHHMM}`，各模块独立版本号，通过 `./<module>/version` 文件管理子版本。
- **完整文档**：[https://doc.x-lf.com/docs/bamboo-base-go](https://doc.x-lf.com/docs/bamboo-base-go)

## 引用

- [major/](./major/AGENTS.md) — 核心层（启动框架/中间件/响应/模型）
- [common/](./common/AGENTS.md) — 通用层（错误/日志/雪花/验证器/工具）
- [plugins/](./plugins/AGENTS.md) — 插件层（grpc/cron/async/email）
  - [plugins/grpc/](./plugins/grpc/AGENTS.md) — gRPC 框架插件
- [major/register/](./major/register/AGENTS.md) — 节点化注册系统
  - [major/register/init/](./major/register/init/AGENTS.md) — 内置初始化节点

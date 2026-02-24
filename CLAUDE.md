# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

bamboo-base-go 是 Bamboo 服务的基础组件库，面向 Gin HTTP API 与 gRPC 服务的统一启动、错误处理、日志、配置与上下文注入。

完整文档: https://doc.x-lf.com/docs/bamboo-base-go

## 开发命令

```bash
make test                           # 运行所有测试 (go test -v ./...)
make tidy                           # 整理 Go 模块依赖
make proto                          # 使用 buf 生成 gRPC 代码
go test ./major/... -v              # 测试 major 模块
go test -run TestFuncName ./path    # 运行单个测试
go vet ./...                        # 静态检查
go fmt ./...                        # 格式化代码

# 发布命令
make release PKG=major              # 发布指定模块 (major|utility|defined)
make release-plugins PLG=grpc       # 发布指定插件 (cron|grpc)
make release-all                    # 发布全部模块和插件
```

版本格式: `{module}/v{ROOT_VERSION}.{SUB_VERSION}-{TIMESTAMP}`，ROOT_VERSION 来自 `./version`，SUB_VERSION 来自各模块的 `version` 文件。

## 多模块工作区架构

项目使用 Go Workspace (`go.work`) 管理 5 个独立模块，每个模块有独立的 `go.mod` 和 `version` 文件：

```
bamboo-base/
├── go.work                  # 工作区配置
├── major/                   # 核心层 - HTTP框架、注册系统、日志、错误处理
├── defined/                 # 定义层 - 上下文键常量、环境变量常量
├── utility/                 # 工具层 - 通用函数、上下文工具
├── plugins/cron/            # 插件 - 定时任务 (robfig/cron)
└── plugins/grpc/            # 插件 - gRPC框架 (拦截器、运行器、proto)
```

### 模块依赖链

```
defined → utility → major (基础依赖方向)
plugins/cron → major
plugins/grpc → defined + major + utility
```

### 各模块核心包

| 模块 | 包路径 | 别名 | 职责 |
|------|--------|------|------|
| major | `register/` | `xReg` | 节点化注册初始化系统 |
| major | `main/` | `xMain` | 应用运行器 (信号监听、优雅关闭) |
| major | `error/` | `xError` | 结构化错误码体系 |
| major | `result/` | `xResult` | HTTP 响应处理 |
| major | `log/` | `xLog` | slog 自定义 Handler (彩色控制台 + JSON 文件) |
| major | `middleware/` | `xMiddle` | Gin 中间件 (CORS、统一响应) |
| major | `validator/` | `xValidator` | 自定义验证器 (enum_int/enum_string/enum_float 等) |
| major | `snowflake/` | `xSnowflake` | 雪花算法 (标准 + 基因型) |
| major | `models/` | `xModels` | GORM 实体基类、分页模型 |
| major | `cache/` | `xCache` | 缓存泛型接口 |
| major | `http/` | `xHttp` | HTTP 常量 |
| major | `route/` | `xRoute` | 404/405 路由处理 |
| defined | `context/` | `xCtx` | 上下文键常量 (ContextKey 类型) |
| defined | `env/` | `xEnv` | 环境变量常量与类型安全获取函数 |
| utility | `package/` | `xUtil` | 通用工具 (Ptr/Val/字符串/时间/验证/密码/绑定) |
| utility | `context/` | `xCtxUtil` | 上下文工具 (DB/Redis/Snowflake 的 Must/Error 版本) |
| plugins/grpc | `runner/` | `xGrpcRunner` | gRPC 服务运行器 |
| plugins/grpc | `interceptor/` | - | 一元和流式 RPC 拦截器 |
| plugins/cron | `runner/` | - | 定时任务运行器 |

## 核心架构模式

### 节点化注册系统

应用启动通过 `xReg.Register(ctx, nodeList)` 完成，内部按顺序执行: 配置加载 → 日志初始化 → 雪花算法 → 用户自定义节点 → Gin 引擎初始化。

每个节点签名: `func(ctx context.Context) (any, error)`，可通过 `ctx.Value(key)` 访问已注册的依赖。使用 `xCtx.Exec` 键注册不需要存储结果的执行节点。

### 上下文注入

系统通过中间件自动将初始化上下文注入到每个 HTTP 请求。在 Handler 中使用 `c.Request.Context()` 获取标准 `context.Context`，然后通过 `xCtxUtil` 工具函数访问组件：

```go
// Must 版本 (失败 panic) vs Error 版本 (返回 error)
db := xCtxUtil.MustGetDB(c.Request.Context())
db, err := xCtxUtil.GetDB(c.Request.Context())
```

业务逻辑层必须使用 `context.Context` 而非 `gin.Context`，实现框架解耦。

### HTTP 应用启动范式

```go
reg := xReg.Register(context.Background(), nodeList)
logger := xLog.WithName(xLog.NamedMAIN)
xMain.Runner(reg, logger, func(r *xReg.Reg) {
    r.Serve.GET("/ping", handler)
}, optionalGrpcTask)
```

### 响应与错误模式

```go
xResult.Success(c, "操作成功")
xResult.SuccessHasData(c, "获取成功", data)
xResult.Error(c, xError.NotFound, "用户不存在", nil)
```

### 请求绑定

```go
data := xUtil.BindData(c, &CreateRequest{})  // JSON body
query := xUtil.BindQuery(c, &ListQuery{})     // query params
uri := xUtil.BindURI(c, &PathParams{})        // URI params
// 返回 nil 表示绑定失败，已自动写入错误响应
```

## 代码约定

### 包导入别名

所有包使用 `x` 前缀别名，这是强制约定: `xReg`、`xEnv`、`xCtx`、`xError`、`xResult`、`xMiddle`、`xLog`、`xUtil`、`xCtxUtil`、`xModels`、`xSnowflake`、`xValidator`、`xHttp`、`xRoute`、`xCache`、`xMain`。

### 环境变量

通过 `xEnv.GetEnvXxx()` 系列函数获取配置（`GetEnvString`、`GetEnvInt`、`GetEnvBool`、`GetEnvFloat`、`GetEnvDuration` 等），所有环境变量常量定义在 `defined/env/env.go` 中。复制 `.env.example` 为 `.env` 使用。

### 错误码

预定义错误码在 `major/error/error_code.go` 中，新增错误码需遵循现有编号规则。

### 验证器

自定义验证器在 `major/validator/` 中，包括 `strict_url`、`strict_uuid`、`alphanum_underscore`、`regexp`、`enum_int`、`enum_string`、`enum_float`。使用 `label` tag 提供中文字段名。

### GORM 实体

使用 `xModels.BaseEntity`（标准雪花 ID）或 `xModels.GeneBaseEntity`（基因雪花 ID，需实现 `GetGene()` 接口）。

## gRPC 开发

gRPC 插件在 `plugins/grpc/` 中，使用 buf 管理 proto 文件。Proto 定义在 `plugins/grpc/proto/`，生成代码在 `plugins/grpc/generate/`。

```bash
make proto                    # 生成 gRPC 代码
```

拦截器分为 `interceptor/unary/`（一元 RPC）和 `interceptor/stream/`（流式 RPC），自动处理上下文注入、追踪、恢复和中间件。

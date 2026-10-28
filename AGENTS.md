# 项目知识库

**版本:** v2
**分支:** feature-v2

## 概述

Bamboo Base Go 是 Bamboo 服务的 Go 基础组件库，面向 Web HTTP API 与 gRPC 服务的统一启动、生命周期编排、错误处理、日志、配置与上下文注入。项目使用 Go Workspace（`go.work`）管理 14 个独立模块，以"组件化与生命周期管理"为核心设计，提供从应用装配、启动看门狗到优雅平滑关机的一站式应用框架。

## 目录结构

```text
bamboo-base/
├── go.work                       # Go 工作区配置（14 个 v2 独立模块）
├── go.mod                        # 根模块（仅占位，不包含代码）
├── Makefile                      # 构建/测试/质量门禁命令
├── README.md                     # 项目说明文档
├── .github/
│   ├── scripts/
│   │   ├── ci/                   # CI 质量门禁（check-go, lint-go, generate-proto）
│   │   ├── release/              # 发布流程、Release Notes 与 Manifest 校验
│   │   ├── proxy/                # 本地模块 GOPROXY 代理基础设施 (with-local-modules, local-module-proxy)
│   │   ├── consumer/             # 独立消费端与示例模板编译校验 (check-v2-consumers, check-v2-examples)
│   │   ├── env/                  # 本地集成测试环境容器管控 (integration-env)
│   │   ├── manifest/             # 14 个 v2 模块权威清单 (v2-modules.txt)
│   │   └── tests/                # 脚本自测试套件
│   └── workflows/
│       ├── quality.yml           # CI 门禁检查 Workflow
│       └── release.yml           # 统一发布 Workflow
├── foundation/                   # 领域原语基座
│   ├── apperr/                   #   结构化错误码与业务错误构造
│   ├── ctxkey/                   #   强类型上下文键与元数据反射
│   ├── env/                      #   环境变量读取与转换
│   ├── envelope/                 #   标准协议响应信封 Response
│   ├── logging/                  #   slog 轮转双写日志系统
│   ├── snowflake/                #   64 位基因雪花算法
│   ├── toolkit/                  #   加解密、字符串、时间、验证工具库
│   └── typed/                    #   强类型 typed.String 原语与转换
├── application/                  # 应用运行时内核
│   ├── app.go                    #   App 状态机、RegisterTool/RegisterWorker/RegisterComponent
│   ├── activator.go              #   Activator 实例树与动态执行
│   ├── builtin/                  #   App 绑定的日志与雪花内置受管组件
│   ├── container.go              #   SharedContainer 共享组件容器与白名单视图
│   ├── graph.go                  #   Kahn 拓扑排序与依赖环检测
│   ├── supervisor.go             #   Worker 协程看门狗与 Panic 隔离
│   └── shutdown.go               #   关机协调器与预算超时控制
├── plugins/                      # 插件层（各自独立 module）
│   ├── database/                 #   数据库插件核心 (GORM 封装、模型基类、Dialector 适配)
│   │   └── adapter/              #     驱动适配器子模块
│   │       ├── mysql/            #       MySQL 驱动适配器
│   │       ├── postgres/         #       PostgreSQL 驱动适配器
│   │       ├── sqlite/           #       SQLite 纯 Go 驱动适配器
│   │       ├── sqlserver/        #       SQL Server 驱动适配器
│   │       └── oracle/           #       Oracle 驱动适配器
│   ├── cache/                    #   泛型缓存插件 (Memory / Redis 双后端)
│   ├── email/                    #   邮件服务插件 (SMTP / HTML 模板)
│   ├── web-gin/                  #   Gin HTTP API 插件 (中间件/绑定/响应/校验器)
│   ├── grpc/                     #   gRPC 框架插件 (拦截器/元数据透传/统一响应)
│   ├── cron/                     #   Cron 定时调度插件
│   └── async/                    #   异步协程池与任务调度插件
├── examples/                     # 独立示例模板
│   └── v2/                       #   v2 模板工程 (worker-tool, http-db-cache, grpc-cron-async)
└── test/                         # 验收与集成测试
    ├── consumer/                 #   候选模块无本地 replace 独立消费测试
    └── integration/              #   隔离集成测试环境
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 创建应用实例 | `application` → `application.New(opts...)` | 应用启动入口，支持函数式选项配置关机超时与日志 |
| 注册后台服务 Worker | `application` → `app.RegisterWorker(info, factory)` | 具有长驻事件循环或网络监听的服务（如 Web、gRPC、Worker） |
| 注册共享组件 Tool | `application` → `app.RegisterTool(info, factory)` | 为其它组件提供只读/共享能力的组件（如 DB、Cache、Logger） |
| 从 Context 获取共享组件 | `application` → `application.Get[T](ctx, key)` | 类型安全的泛型上下文提取，未注册或类型不符时返回明确错误 |
| 启动并等待退出 | `application` → `app.Run(ctx)` | 拓扑初始化、看门狗启动、信号捕获与逆拓扑平滑关机 |
| 构造 Web HTTP 插件 | `plugins/web-gin` → `webgin.New(opts...)` | 构造 Gin Worker，支持 `WithRoute`、`WithRequires` 等 |
| 返回成功/错误 HTTP 响应 | `plugins/web-gin/httpresult` → `Success()` / `Error()` | 输出标准结构化信封响应并计算状态码 |
| 绑定 HTTP 请求参数 | `plugins/web-gin/bind` → `Bind[T](c)` | 泛型参数绑定并自动触发实例级校验 |
| 构造 gRPC 插件 | `plugins/grpc` → `grpc.New(opts...)` | 构造 gRPC Worker，支持服务注册、拦截器链与元数据透传 |
| 返回 gRPC 响应 | `plugins/grpc/grpcresult` → `SuccessWith[T](ctx, msg)` | 为 Protobuf 响应注入标准 BaseResponse 信封 |
| 构造数据库插件 | `plugins/database` → `database.New(opts...)` | 支持 `WithAdapter(...)` 接入指定驱动适配器 |
| 构造缓存管理器 | `plugins/cache` → `cache.New(opts...)` | 支持 `WithMemory()` 与 `WithRedis(...)` 双后端切换 |
| 异步执行任务 | `plugins/async` → `async.New(opts...)` | 注册安全并发任务池，剥离请求上下文执行后台计算 |
| 定时调度作业 | `plugins/cron` → `cron.New(opts...)` | 注册 Cron 调度 Worker，触发 Activator 执行目标动作 |
| 构造业务错误 | `foundation/apperr` → `apperr.New(code, msg, errs...)` | 纯净错误构造，带 HTTP 映射错误码 |
| 上下文类型键 | `foundation/ctxkey` → `ctxkey.DatabaseKey` 等 | 强类型上下文键 |
| 强类型字符串原语 | `foundation/typed` → `typed.New(str)` | 提供 6 大格式通道无损编解码与类型转换 |

## 模块架构

```text
┌──────────────────────────────────────────────────────────┐
│                    业务应用 (下游)                         │
└──────────────┬──────────────────────────┬────────────────┘
               │                          │
    ┌──────────▼──────────┐   ┌──────────▼──────────┐
    │     application      │   │     plugins/*        │
    │  (App 运行时内核 /   │   │  (web-gin / grpc /   │
    │   容器 / 拓扑 /     │   │   database / cache / │
    │   看门狗 / 协调器)  │   │   cron / async 等)   │
    └──────────┬──────────┘   └──────────┬──────────┘
               │                          │
               │                          │
               └───────────┬──────────────┘
                           │
                ┌──────────▼──────────┐
                │     foundation       │
                │  (typed / ctxkey /   │
                │   apperr / envelope /│
                │   logging/snowflake) │
                └─────────────────────┘
```

依赖方向（单向，无循环）：`foundation ← application ← plugins/*`。`foundation` 位于最底层，没有任何上层依赖；`plugins` 之间互相独立，不发生跨插件横向依赖。

## 约定

- **Go Workspace 管理**：14 个独立 module 通过 `go.work` 关联，每个 module 有独立的 `go.mod`。
- **中文注释**：所有公开类型和函数使用中文注释，遵循 Go doc 注释规范。
- **模块路径规范**：所有子模块路径均采用小写 `/v2`（如 `.../foundation/v2`、`.../application/v2`）。
- **统一版本号**：所有模块共享同一版本标签（如 `foundation/v2.0.0`），由 CI 统一发布。
- **类型安全组件提取**：组件提取必须使用 `application.Get[T](ctx, key)`，禁止裸 `ctx.Value` 类型断言。
- **Fail-Fast 原则**：所有环境预检、清单校验与测试脚本均遵循严格白名单与 fail-fast 退出策略。

## 常用命令

```bash
# 开发与检查
make test                     # 运行全部 14 个 v2 模块单元测试
make race                     # 运行全部 14 个 v2 模块竞态检测
make vet                      # 运行 go vet 静态分析
make tidy                     # 整理全部 14 个 v2 模块依赖
make proto-check              # 校验 Protobuf 生成代码一致性
make lint                     # 文档门禁 + Go 规范检查
make lint-docs                # Prettier + Markdownlint 文档校验
make format-docs              # 格式化文档
make lint-go                  # 运行 golangci-lint 与代码格式检查
```

# grpc 知识库

## 概述

gRPC 框架插件（`plugins/grpc/v2`），提供符合 v2 架构的受管服务生命周期插件、一元与流式拦截器桥接、独立服务级中间件注册表、统一错误映射与 `grpcresult` 响应构建。通过 `New()` 创建受管 `application.Plugin`，在 App 生命周期管理下实现启动、请求上下文与追踪桥接、ACL 校验及优雅关闭。

## 目录结构

```text
grpc/
├── command.go                 # 命令行子命令与基础辅助
├── buf.yaml / buf.gen.yaml    # buf protobuf 代码生成配置
├── buf.lock                   # buf 依赖锁定
├── proto/
│   └── base.proto             # BaseResponse proto 定义（统一响应元信息）
├── generate/
│   └── base.pb.go             # buf 生成的 Go 代码（请勿手动编辑）
├── plugin.go                  # New() 受管插件、生命周期（Start/Stop）、内置拦截器与统一错误规整
├── constant/
│   ├── metadata.go            # gRPC Metadata 键常量（app-access-id 等）
│   └── trailer.go             # gRPC Trailer 键常量（x-request-uuid 等）
├── middleware/
│   ├── middleware.go          # Registry 中间件注册表（NewRegistry / UseUnary / UseStream）
│   └── middleware_test.go     # 中间件分发与洋葱模型单测
├── grpcresult/
│   ├── response.go            # Success / ErrorStatus / Error / ToStatusError 协议结果构建
│   ├── response_test.go       # 响应构建与状态映射单测
│   ├── generic.go             # SuccessWith[T] 强类型 Protobuf 响应泛型装配
│   ├── generic_test.go        # 泛型装配单测与反射校验
│   └── internaltest/          # 泛型单测所用 Proto 消息定义
└── utility/
    ├── grpc_util.go           # gRPC 通用元数据提取与工具函数
    └── grpc_util_test.go      # 工具函数单测
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 注册 gRPC 插件 | `plugin.go` → `New(opts ...Option)` | 返回受管 `application.Plugin`（Key: `grpc`） |
| 配置监听与端口 | `plugin.go` → `WithListener()` / `WithPort()` | 默认读取 `GRPC_PORT`（默认 1119） |
| 注册 gRPC 服务 | `plugin.go` → `WithServiceRegistrar()` | 传入业务注册函数 `func(server grpc.ServiceRegistrar)` |
| 注册服务级中间件 | `middleware/middleware.go` → `UseUnary()` / `UseStream()` | 通过 `Registry` 实例按服务名绑定洋葱模型链 |
| 绑定独立中间件注册表 | `plugin.go` → `WithMiddlewareRegistry()` | 显式注入已配置的中间件注册表 |
| 注入显式日志器 | `plugin.go` → `WithLogger()` | 显式传入 `*logging.Logger` 对象，不污染全局 Default |
| 构建成功响应 | `grpcresult/response.go` → `Success()` | 返回符合 `base.proto` 的 `*generate.BaseResponse` |
| 构建泛型成功响应 | `grpcresult/generic.go` → `SuccessWith[T]()` | 自动创建并注入包含唯一 `BaseResponse` 的 Protobuf 结构体指针 |
| 统一错误状态转换 | `grpcresult/response.go` → `Error()` / `ErrorStatus()` | 依据 `apperr.Error` 状态码映射到标准 gRPC `codes.Code` |
| 查看 Metadata/Trailer 键 | `constant/metadata.go` / `constant/trailer.go` | 元数据常量定义 |
| 重新生成 pb 代码 | `generate/base.pb.go` | `buf generate`（配置见 `buf.yaml`） |

## 约定

- **受管生命周期**：gRPC 服务作为 `application.Plugin` 接入，由框架统一在 `Start()` 启动并在 `Stop()` 执行 `GracefulStop()`（超时退化为 `Stop()`）。
- **拦截器桥接**：Unary 与 Stream 均内置统一的追踪 ID 提取（Header 与 Trailer 双写 `x-request-uuid`）、开始时间记录与 panic 恢复兜底。
- **中间件洋葱模型**：按服务名匹配中间件链，严格遵循 `A-enter → B-enter → handler → B-exit → A-exit` 执行次序。
- **强类型泛型注入**：`SuccessWith[T]` 要求类型参数 `T` 必须为具体的 Protobuf 指针类型，且顶层结构体中恰好包含一个类型为 `*generate.BaseResponse` 的字段。
- **显式日志依赖**：遵循整体架构原则，使用显式注入的 `*logging.Logger` 及其派生实例（如 `logger.WithName(...)`），禁止修改或依赖进程级 `slog.SetDefault`。
- **proto 代码生成**：`generate/` 下的 `.pb.go` 由 `buf generate` 产出，严禁手动修改。

## 反模式

- **禁止依赖进程级全局变量** — 不得预设全局雪花节点或全局日志器。
- **禁止在 `SuccessWith` 中传入非指针或不含唯一 `BaseResponse` 的类型** — 将被校验拦截并明确报错。
- **禁止手动向 context 强行写入未声明凭据** — 请求上下文由框架受管桥接（`BridgeRequest`）。

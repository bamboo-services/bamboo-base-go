# grpc 知识库

## 概述

gRPC 框架插件，提供 gRPC 服务的启动器、拦截器链路、服务级中间件分发、统一响应构建与错误转换。通过 `xGrpcRunner.New()` 创建启动函数，挂载到 `xMain.Runner` 的附加协程中，实现 HTTP + gRPC 一体化启动与优雅关闭。

## 目录结构

```text
grpc/
├── command.go                 # gRPC 命令行子命令（启动/停止等，如果存在）
├── buf.yaml / buf.gen.yaml    # buf protobuf 代码生成配置
├── buf.lock                   # buf 依赖锁定
├── proto/
│   └── base.proto             # BaseResponse proto 定义（统一响应元信息）
├── generate/
│   └── base.pb.go             # buf 生成的 Go 代码（请勿手动编辑）
├── runner/
│   └── runner.go              # New() 启动函数 + Config + Option 模式
├── interceptor/
│   ├── unary/                 # 一元拦截器实现
│   │   ├── middleware.go      #   服务级中间件分发（洋葱模型）
│   │   ├── response_builder.go#   响应构建（注入 trace ID + 耗时）
│   │   ├── init_context.go    #   上下文初始化
│   │   ├── trace.go           #   链路追踪
│   │   └── recover.go         #   panic 恢复
│   └── stream/                # 流式拦截器实现（结构同 unary）
├── middleware/
│   └── middleware.go          # 全局中间件注册表：UseUnary / UseStream
├── result/
│   └── result_success.go      # Success / SuccessWith 响应构建工具
├── constant/
│   ├── metadata.go            # gRPC Metadata 键常量（app-access-id 等）
│   └── trailer.go             # gRPC Trailer 键常量
└── utility/
    └── grpc_util.go           # gRPC 通用工具函数
```

## 导航指南

| 任务 | 位置 | 说明 |
|------|------|------|
| 启动 gRPC 服务 | `runner/runner.go` → `New(options...)` | 返回 `func(ctx, ...option)`，传入 `xMain.Runner` |
| 注册 gRPC 服务 | `runner/runner.go` → `WithRegisterService()` | Option 模式传入 `RegisterServiceFunc` |
| 添加服务级中间件 | `middleware/middleware.go` → `UseUnary()` | 按服务名绑定中间件链 |
| 添加内置拦截器 | `runner/runner.go` → `WithUnaryInterceptors()` | Option 模式传入拦截器 |
| 构建成功响应 | `result/result_success.go` → `Success()` / `SuccessWith()` | `SuccessWith` 用反射注入 `BaseResponse` |
| 查看 Metadata 键 | `constant/metadata.go` | `app-access-id` / `app-secret-key` / `x-request-uuid` |
| 修改 proto 定义 | `proto/base.proto` | 修改后执行 `make proto` 重新生成 |
| 重新生成 pb 代码 | `generate/base.pb.go` | `buf generate`（配置见 `buf.yaml`） |

## 约定

- **Option 模式配置**：所有 Runner 配置通过 `WithXxx()` 函数传入 `New()`，运行时可通过 `option ...any` 动态覆盖。
- **拦截器顺序**：`Middleware()`（中间件分发）应排在 `ResponseBuilder()` 之前，保证响应元信息最后注入。
- **中间件洋葱模型**：注册顺序 `[A, B, C]` 的执行顺序为 `A-enter → B-enter → C-enter → handler → C-exit → B-exit → A-exit`。
- **proto 生成代码不可手动编辑**：`generate/` 下的 `.pb.go` 文件由 buf 自动生成。修改 proto 后执行 `make proto`。
- **响应结构统一**：所有 gRPC 响应消息应嵌入 `BaseResponse`（proto 定义），由 `ResponseBuilder` 拦截器统一注入 trace ID 和耗时。
- **Metadata 键使用 `constant/` 常量**：不要在业务代码中硬编码 metadata 键名字符串。

## 反模式

- **禁止手动编辑 `generate/*.pb.go`** — 下次 `buf generate` 会覆盖。
- **禁止在 `SuccessWith` 泛型中传入非指针类型** — 会 panic（反射写入需要指针）。
- **禁止跳过 `recover.go` 拦截器** — 没有 panic 恢复会导致整个 gRPC 服务崩溃。
- **禁止在流式拦截器中使用一元拦截器签名** — unary 和 stream 的接口签名不同，混用会编译失败。

## 调试路径

1. gRPC 服务无法启动 — 检查 `GRPC_PORT` 是否被占用，`runner/runner.go` 的 `run()` 会从 `xEnv.GrpcPort` 读取端口。
2. 中间件未生效 — 确认 `Middleware()` 拦截器已注册到 Runner，且服务名匹配（`extractServiceName` 从 FullMethod 解析）。
3. 响应缺少 trace ID — 确认 `ResponseBuilder()` 拦截器已注册，且 `InitContext()` 在其之前执行。
4. proto 修改后编译失败 — 执行 `make proto` 重新生成 `generate/base.pb.go`。
5. 反射注入失败 panic — `SuccessWith` 要求 T 是指针类型且包含 `BaseResponse *xGrpcGenerate.BaseResponse` 嵌入字段。

# Bamboo Base Go

[![Go Version](https://img.shields.io/badge/Go-1.24.6+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Documentation](https://img.shields.io/badge/Docs-doc.x--lf.com-green?style=flat-square)](https://doc.x-lf.com/docs/bamboo-base-go)

Bamboo Base Go 是 Bamboo 服务的基础组件库，面向 Gin HTTP API 与 gRPC 服务的统一启动、错误处理、日志、配置与上下文注入。

## 文档

完整文档请访问：**[https://doc.x-lf.com/docs/bamboo-base-go](https://doc.x-lf.com/docs/bamboo-base-go)**

## 特性

- **节点化注册系统** - 基于 `xReg.Register(ctx, nodeList)` 的组件初始化与依赖注入
- **HTTP Runner** - `xMain.Runner` 支持信号监听、优雅关闭与附加后台协程
- **gRPC Runner** - 内置 gRPC 启动器、拦截器链路、错误转换与追踪元数据
- **请求绑定工具** - `BindData/BindQuery/BindURI/BindHeader` 统一绑定与校验失败处理
- **分页模型** - `PageRequest/PageResponse` 规范化分页参数与输出结构
- **错误与响应统一** - HTTP 与 gRPC 都可复用结构化错误码体系
- **日志与追踪** - 基于 `slog`，支持请求链路标识与结构化输出
- **通用工具与验证器** - 字符串、时间、类型解析、枚举校验等常用能力

## 安装

```bash
go get github.com/bamboo-services/bamboo-base-go
```

## 快速开始（HTTP）

```go
package main

import (
	"context"

	"github.com/gin-gonic/gin"

	xLog "github.com/bamboo-services/bamboo-base-go/log"
	xMain "github.com/bamboo-services/bamboo-base-go/main"
	xReg "github.com/bamboo-services/bamboo-base-go/register"
)

func main() {
	reg := xReg.Register(context.Background(), nil)
	logger := xLog.WithName(xLog.NamedMAIN)

	xMain.Runner(reg, logger, func(r *xReg.Reg) {
		r.Serve.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})
	})
}
```

## HTTP + gRPC 一体化启动（可选）

```go
grpcTask := xGrpcRunner.New(
	xGrpcRunner.WithLogger(xLog.WithName(xLog.NamedGRPC)),
	xGrpcRunner.WithRegisterService(func(ctx context.Context, server grpc.ServiceRegistrar) {
		// 在这里注册你的 gRPC 服务：
		// xGrpcGenerate.RegisterYourServiceServer(server, yourHandler)
	}),
)

xMain.Runner(reg, logger, routeFunc, grpcTask)
```

`xMain.Runner` 会在收到 `SIGINT/SIGTERM` 时统一触发 HTTP 与附加协程（例如 gRPC）的优雅退出。

## 请求绑定与分页

```go
type Query struct {
	Page int64 `form:"page" binding:"omitempty,min=1"`
	Size int64 `form:"size" binding:"omitempty,min=1,max=200"`
}

func ListHandler(c *gin.Context) {
	query := xUtil.BindQuery(c, &Query{})
	if query == nil {
		return
	}

	req := xModels.PageRequest{Page: query.Page, Size: query.Size}.Normalize()
	page := xModels.NewPage(req.Page, req.Size, 100, []string{"a", "b"})
	xResult.SuccessHasData(c, "ok", page)
}
```

## gRPC Proto 代码生成

仓库提供 `Makefile` 简化 proto 生成：

```bash
# 生成默认 proto（可通过 PROTO_FILE 覆盖）
make proto

# 指定 proto 文件
make proto PROTO_FILE=./proto/error.proto
```

## 配置

复制 `.env.example` 为 `.env` 并按需修改：

```bash
cp .env.example .env
```

常用配置项：

| 环境变量 | 说明 | 默认值（代码兜底） |
|----------|------|-------------------|
| `XLF_DEBUG` | 调试模式 | `false` |
| `XLF_HOST` | HTTP 监听地址 | `localhost` |
| `XLF_PORT` | HTTP 监听端口 | `1118` |
| `GRPC_PORT` | gRPC 监听端口 | `1119` |
| `GRPC_REFLECTION` | gRPC 反射开关 | `false` |
| `DATABASE_HOST` | 数据库主机 | `localhost` |
| `NOSQL_HOST` | Redis 主机 | `localhost` |
| `NOSQL_DATABASE` | Redis DB 索引 | `0` |

## 项目结构

```
bamboo-base/
├── cache/       # 缓存泛型接口定义 (xCache)
├── context/     # 上下文键常量与节点存储 (xCtx)
├── env/         # 环境变量管理 (xEnv)
├── error/       # 错误处理 (xError)
├── grpc/        # gRPC 运行时、拦截器、错误与响应 (xGrpc*)
├── http/        # HTTP 常量 (xHttp)
├── log/         # 日志系统 (xLog)
├── main/        # 应用运行器 (xMain)
├── middleware/  # Gin 中间件 (xMiddle)
├── models/      # 数据模型（含分页模型）(xModels)
├── proto/       # gRPC proto 定义
├── register/    # 注册初始化 (xReg)
├── result/      # HTTP 响应处理 (xResult)
├── route/       # 路由处理 (xRoute)
├── snowflake/   # 雪花算法 (xSnowflake)
├── utility/     # 工具函数与绑定函数 (xUtil)
└── validator/   # 验证器 (xValidator)
```

## 核心依赖

- [Gin](https://github.com/gin-gonic/gin) - HTTP Web 框架
- [gRPC](https://github.com/grpc/grpc-go) - gRPC 服务端框架
- [GORM](https://gorm.io/) - ORM
- [go-redis](https://github.com/redis/go-redis) - Redis 客户端
- [validator](https://github.com/go-playground/validator) - 请求验证
- [protobuf](https://github.com/protocolbuffers/protobuf-go) - Protobuf 消息与 Any 支持

## 参与贡献

欢迎参与项目维护！你可以通过以下方式贡献：

- **提交 Issue** - 报告 Bug 或提出新功能建议
- **提交 Pull Request** - 直接贡献代码改进

请访问 [GitHub 仓库](https://github.com/bamboo-services/bamboo-base-go) 参与贡献。

## 许可证

MIT License

## 链接

- [完整文档](https://doc.x-lf.com/docs/bamboo-base-go)
- [GitHub 仓库](https://github.com/bamboo-services/bamboo-base-go)
- [问题反馈](https://github.com/bamboo-services/bamboo-base-go/issues)
- [Pull Requests](https://github.com/bamboo-services/bamboo-base-go/pulls)

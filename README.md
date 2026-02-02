# Bamboo Base Go

[![Go Version](https://img.shields.io/badge/Go-1.24.6+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Documentation](https://img.shields.io/badge/Docs-doc.x--lf.com-green?style=flat-square)](https://doc.x-lf.com/docs/bamboo-base-go)

Bamboo Base Go 是一个为 Bamboo 服务提供基础组件的 Go 语言库。它被设计为一个可重用的基础库，用于构建基于 Gin 框架的 Web API，提供标准化的错误处理、日志记录、配置管理和响应格式化功能。

## 文档

完整文档请访问：**[https://doc.x-lf.com/docs/bamboo-base-go](https://doc.x-lf.com/docs/bamboo-base-go)**

## 特性

- **注册系统** - 集中式组件初始化，一键启动应用
- **环境变量管理** - 类型安全的配置获取 API
- **错误处理** - 结构化错误码和统一错误响应
- **响应格式化** - 标准化 API 响应结构
- **日志系统** - 基于 slog 的彩色控制台 + JSON 文件日志
- **雪花算法** - 标准雪花 ID 和基因雪花 ID 生成
- **请求验证** - 丰富的自定义验证器 (枚举、URL、UUID 等)
- **工具函数** - 字符串处理、时间操作、数据验证等

## 安装

```bash
go get github.com/bamboo-services/bamboo-base-go
```

## 快速开始

```go
package main

import (
    xReg "github.com/bamboo-services/bamboo-base-go/register"
)

func main() {
    // 初始化所有组件
    reg := xReg.Register()

    // 注册路由
    reg.Serve.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    // 启动服务
    reg.Serve.Run(":8080")
}
```

## 配置

复制 `.env.example` 为 `.env` 并根据需要修改：

```bash
cp .env.example .env
```

主要配置项：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `XLF_DEBUG` | 调试模式 | `false` |
| `XLF_HOST` | 监听地址 | `0.0.0.0` |
| `XLF_PORT` | 监听端口 | `8080` |
| `DATABASE_HOST` | 数据库主机 | `localhost` |
| `NOSQL_HOST` | Redis 主机 | `localhost` |

## 项目结构

```
bamboo-base/
├── context/     # 上下文键常量 (xCtx)
├── env/         # 环境变量管理 (xEnv)
├── error/       # 错误处理 (xError)
├── http/        # HTTP 常量 (xHttp)
├── log/         # 日志系统 (xLog)
├── middleware/  # Gin 中间件 (xMiddle)
├── models/      # 数据模型 (xModels)
├── register/    # 注册初始化 (xReg)
├── result/      # 响应处理 (xResult)
├── route/       # 路由处理 (xRoute)
├── snowflake/   # 雪花算法 (xSnowflake)
├── utility/     # 工具函数 (xUtil)
└── validator/   # 验证器 (xValidator)
```

## 核心依赖

- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [GORM](https://gorm.io/) - ORM
- [go-redis](https://github.com/redis/go-redis) - Redis 客户端
- [validator](https://github.com/go-playground/validator) - 请求验证

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

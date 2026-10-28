# Bamboo Base Go

[![Go Version](https://img.shields.io/badge/Go-1.25.0+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square)](LICENSE)
[![Documentation](https://img.shields.io/badge/Docs-doc.x--lf.com-green?style=flat-square)](https://doc.x-lf.com/docs/bamboo-base-go)

Bamboo Base Go 是 Bamboo 服务的基础组件库，面向 Web HTTP API 与 gRPC 服务的统一启动、生命周期编排、错误处理、日志、配置与上下文注入。

## 文档

完整文档请访问：**[https://doc.x-lf.com/docs/bamboo-base-go](https://doc.x-lf.com/docs/bamboo-base-go)**

## 特性

- **现代化生命周期内核 (`application`)** - 声明式 Plugin/Worker/Tool/Component 注册，Kahn 拓扑排序解析依赖，看门狗协作，优雅平滑关机
- **领域原语基座 (`foundation`)** - 强类型 `typed.String` 原语、类型化 `ctxkey` 上下文键、结构化 `apperr` 错误码体系与统一响应信封
- **独立日志与雪花算法** - 基于 `slog` 的自动轮转双写日志系统与 64-bit 基因雪花 ID 生成器，按 App 独立持有与管理
- **插件化协议入口** - `plugins/web-gin` (HTTP 绑定、错误兜底、验证器) 与 `plugins/grpc` (拦截器、元数据透传、统一响应)
- **通用缓存与持久化** - `plugins/cache` (Memory / Redis 双后端透明切换) 与 `plugins/database` (GORM 封装及 5 款独立适配器)
- **调度与异步任务** - `plugins/cron` (定时调度作业) 与 `plugins/async` (安全并发池与请求上下文分离)

## 模块清单（14 个 v2 独立模块）

项目使用 Go Workspace (`go.work`) 统一管理以下 14 个独立模块：

| 模块目录                             | 模块路径                                                        | 说明                                                                     |
| :----------------------------------- | :-------------------------------------------------------------- | :----------------------------------------------------------------------- |
| `foundation`                         | `github.com/bamboo-services/bamboo-base-go/foundation/v2`       | 基础原语（typed, ctxkey, apperr, envelope, logging, snowflake, toolkit） |
| `application`                        | `github.com/bamboo-services/bamboo-base-go/application/v2`      | 应用内核（App, Plugin, Worker, Tool, Component, Container, Supervisor）  |
| `plugins/database`                   | `github.com/bamboo-services/bamboo-base-go/plugins/database/v2` | 数据库核心接口与 GORM 模型基类                                           |
| `plugins/cache`                      | `github.com/bamboo-services/bamboo-base-go/plugins/cache/v2`    | 泛型缓存系统（Memory / Redis 双后端）                                    |
| `plugins/email`                      | `github.com/bamboo-services/bamboo-base-go/plugins/email/v2`    | 邮件客户端与模板引擎                                                     |
| `plugins/web-gin`                    | `github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2`  | Gin Web HTTP 服务插件与参数绑定                                          |
| `plugins/grpc`                       | `github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2`     | gRPC 服务插件与统一响应拦截器                                            |
| `plugins/cron`                       | `github.com/bamboo-services/bamboo-base-go/plugins/cron/v2`     | Cron 定时调度插件                                                        |
| `plugins/async`                      | `github.com/bamboo-services/bamboo-base-go/plugins/async/v2`    | 异步任务池插件                                                           |
| `plugins/database/adapter/mysql`     | `.../plugins/database/adapter/mysql/v2`                         | MySQL 数据库驱动适配器                                                   |
| `plugins/database/adapter/postgres`  | `.../plugins/database/adapter/postgres/v2`                      | PostgreSQL 数据库驱动适配器                                              |
| `plugins/database/adapter/sqlite`    | `.../plugins/database/adapter/sqlite/v2`                        | SQLite 纯 Go 数据库驱动适配器                                            |
| `plugins/database/adapter/sqlserver` | `.../plugins/database/adapter/sqlserver/v2`                     | SQL Server 数据库驱动适配器                                              |
| `plugins/database/adapter/oracle`    | `.../plugins/database/adapter/oracle/v2`                        | Oracle 数据库驱动适配器                                                  |

## 快速开始（HTTP + SQLite + 缓存）

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bamboo-services/bamboo-base-go/application/v2"
	"github.com/bamboo-services/bamboo-base-go/foundation/v2/ctxkey"
	cache "github.com/bamboo-services/bamboo-base-go/plugins/cache/v2"
	sqlite "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	database "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
	webgin "github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2"
	"github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2/httpresult"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Item struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"size:64;not null"`
}

func main() {
	app := application.New()

	// 注册 SQLite 数据库 Tool
	dbInfo, dbFac := database.New(
		database.WithAdapter(sqlite.New("file:app.db?cache=shared")),
	)
	if err := app.RegisterTool(dbInfo, dbFac); err != nil {
		fmt.Fprintf(os.Stderr, "注册数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 注册内存缓存 Tool
	cacheInfo, cacheFac := cache.New(cache.WithMemory())
	if err := app.RegisterTool(cacheInfo, cacheFac); err != nil {
		fmt.Fprintf(os.Stderr, "注册缓存失败: %v\n", err)
		os.Exit(1)
	}

	// 注册 Web Gin Worker
	webInfo, webFac := webgin.New(
		webgin.WithAddr("127.0.0.1:8080"),
		webgin.WithRequires(ctxkey.DatabaseKey, ctxkey.CacheManagerKey),
		webgin.WithRoute(func(startCtx context.Context, engine *gin.Engine) {
			db, err := application.Get[*gorm.DB](startCtx, ctxkey.DatabaseKey)
			if err != nil {
				return
			}
			_ = db.AutoMigrate(&Item{})

			engine.GET("/ping", func(c *gin.Context) {
				httpresult.SuccessWithData(c, "pong", gin.H{"status": "ok"})
			})
		}),
	)
	if err := app.RegisterWorker(webInfo, webFac); err != nil {
		fmt.Fprintf(os.Stderr, "注册 Web Worker 失败: %v\n", err)
		os.Exit(1)
	}

	// 运行应用（接管 SIGINT / SIGTERM 优雅退出）
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "应用退出异常: %v\n", err)
		os.Exit(1)
	}
}
```

更多独立模板可参考 `examples/v2/` 目录：

- `examples/v2/worker-tool`: Tool 与 Worker 编排
- `examples/v2/http-db-cache`: HTTP + SQLite + Memory 缓存组合服务
- `examples/v2/grpc-cron-async`: gRPC + Cron + Async 组合服务

## 常用开发命令

```bash
make test         # 运行全部 14 个 v2 模块单元测试
make race         # 运行全部 14 个 v2 模块竞态检测
make vet          # go vet 校验全部 14 个 v2 模块
make tidy         # 整理 14 个 v2 模块依赖
make lint         # 文档检查 + Go 模块 lint 与 format
make lint-docs    # Prettier + Markdownlint 检查文档
make format-docs  # 自动格式化文档
make proto-check  # 校验 Protobuf 生成代码一致性
make check-examples # 校验全部示例与 README 快速开始片段独立编译
```

## 许可证

[Apache License 2.0](LICENSE)

Copyright 2025-2026 Bamboo Services

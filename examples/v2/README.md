# Bamboo Base v2 示例

本目录包含 Bamboo Base v2 架构的独立模板示例。每个示例文件首行均带有 `//go:build ignore` 标记，专为下游参考设计。

## 示例用途

- **worker-tool**: 演示 Tool 与 Worker 依赖编排与执行。
- **http-db-cache**: 演示 HTTP + SQLite + Memory 缓存组合服务（默认 8080）。
- **grpc-cron-async**: 演示 gRPC 标准 health 检查 + Cron 定时任务 + Async 异步队列组合服务（默认 9090）。

## 验证与运行方式

当前 v2 尚未公网发布，请勿通过公网 `go get` 获取。测试无需本地 `replace`，亦不新增正式 module。

在仓库根目录下，通过以下命令分别在临时真实 consumer 环境（`GOWORK=off`）中使用候选 source fileproxy 运行 `go test`、`vet` 和 `build` 校验（仅证明模板可正常编译，非生产 E2E 全绿）：

```bash
bash .github/scripts/check-v2-examples.sh --example readme
bash .github/scripts/check-v2-examples.sh --example worker-tool
bash .github/scripts/check-v2-examples.sh --example http-db-cache
bash .github/scripts/check-v2-examples.sh --example grpc-cron-async
# 或通过 Makefile 一键回归全部示例：
make check-examples
```

注：`--fixture sqlite-async --race` 为真实 SQL 联动测试入口。

## 运行注意

App 自身已接管 `Ctrl+C` (SIGINT) 与 SIGTERM 信号关机，无需业务层手动捕获。在独立 consumer 模块中直接 `go run` 即可。

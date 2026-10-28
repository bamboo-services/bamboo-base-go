# v2-sqlite-async Consumer Template

本目录为待外部/父测试套件复制到真实临时消费工程（独立 module，`GOWORK=off`）中执行的 Go 测试模板。

## 契约与使用说明
1. **未编译前仅作为模板**：本目录不包含 `go.mod`，不参与当前仓库工作区（保持 14 个 v2 正式模块计数不变）。
2. **依赖隔离与消费方式**：真实候选消费时须配合 `fileproxy` / `local-module-proxy` 与 `GOWORK=off` 环境，独立引入：
   - `github.com/bamboo-services/bamboo-base-go/application/v2`
   - `github.com/bamboo-services/bamboo-base-go/foundation/v2`
   - `github.com/bamboo-services/bamboo-base-go/plugins/database/v2`
   - `github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2`
   - `github.com/bamboo-services/bamboo-base-go/plugins/async/v2`
3. **真实联动验证目标**：
   - 使用临时文件级真实 SQLite（非内存伪造）与真实 `async.Pool`。
   - 验证 `Async` 声明依赖 `DatabaseKey` 并在执行时持有底层连接时，App 优雅关闭与超时拦截（Incomplete）的强保障。

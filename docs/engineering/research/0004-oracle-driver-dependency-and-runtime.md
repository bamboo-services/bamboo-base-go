# v2.0.0 Oracle GORM 适配驱动依赖与运行环境调研

> 调研日期：2026-10-27 · Scope：plugin
> 说明：本篇记录 Oracle 驱动依赖源、GORM 适配器现状及运行时约束的客观证据。驱动选型已由用户基于调研裁决沿用 `github.com/oracle-samples/gorm-oracle` v1.1.3，不切换纯 Go 替代品；本记录注明事实与倾向，不替代用户裁决。

## 背景

在 v2.0.0 插件体系架构设计中，`plugins/database` 采用主模块配合 `adapter/*` 独立嵌套 `go.mod` 结构。其中 Oracle 适配器旧设计草案中引用了不可用的第三方仓库路径（`github.com/oesand/gorm-oracle`）。为确保 `plugins/database/adapter/oracle` 依赖可靠、编译环境与运行期契约清晰，需对公共依赖源、Go Proxy 存证、编译与运行时约束进行独立取证。

## 事实与证据

### 1. 现行选定依赖：`github.com/oracle-samples/gorm-oracle`

- **Go Proxy 元信息存证**：
  - 依赖源：`proxy.golang.org/github.com/oracle-samples/gorm-oracle/@v/v1.1.3.mod`
  - 版本与发布时间：v1.1.3 发布于 `2026-07-07T15:57:46Z`，提交哈希 `f3031b4354176c9b99f581a92c68acdd0755a873`（来源：`proxy.golang.org/github.com/oracle-samples/gorm-oracle/@v/v1.1.3.info`）。
  - 核心依赖版本：Go 语言基线 `go 1.25.11`，依赖底层驱动 `github.com/godror/godror v0.51.0` 与 GORM `gorm.io/gorm v1.31.2`。
- **环境约束（编译期与运行期）**：
  - 编译期：根据官方说明（`raw.githubusercontent.com/oracle-samples/gorm-oracle/v1.1.3/README.md`）及底层 godror 安装指南（`godror.github.io/godror/doc/installation.html`），该驱动基于 CGO 编译，编译环境需要具备 C 编译器（gcc/clang）与 CGO 支持；但在编译时不需要预先安装 Oracle 客户端库。
  - 运行期：必须依赖 Oracle Instant Client 动态链接库。macOS 与 Windows 可在 DSN 连接串中指定 `libDir` 参数加载客户端库路径；Linux 环境则依赖系统动态链接器路径（如 `LD_LIBRARY_PATH` 或 `/etc/ld.so.conf`）。
- **可用性与支持级别**：
  - 属于 Oracle 官方 GitHub 组织（oracle-samples）开源项目，但在开源协议和说明中不等于提供商业 SLA 承诺。
  - Oracle 官方 ARM64 下载页面在公网探测中存在 HTTP 403 限制，不能宣称本地已预装、必须 Rosetta 转译或已验证过具体版本。

### 2. 候选依赖与历史路径调查

- **旧引用路径 `github.com/oesand/gorm-oracle`**：
  - 公共 GitHub 仓库及 Go Proxy 均不可获取，公开路径不可用（注：HTTP 404 表明当前公开渠道无法拉取，但不证明该路径永久不存在或是否曾为私有仓库）。设计草案中的旧路径必须予以修正。
- **纯 Go 候选依赖 `github.com/godoes/gorm-oracle`（v1.6.20）**：
  - 基于纯 Go 驱动 `sijms/go-ora/v2` 实现，免除了 CGO 和 Instant Client 依赖。
  - 但其官方自述（`raw.githubusercontent.com/godoes/gorm-oracle/v1.6.20/README.md`）明确声明“not recommended for production”。
  - 仅作为技术候选记录，不采用作为生产实现，也不替换既有选型。

## 倾向与裁决

- **选型倾向**：在生产场景中优先保证驱动在复杂 SQL、类型映射与连接池上的稳定性；尽管 `oracle-samples/gorm-oracle` 引入了 CGO 与 Instant Client 运维成本，但稳定性预期仍优于实验性纯 Go 驱动。
- **用户裁决**：用户已基于前期调研明确裁决，**沿用 `github.com/oracle-samples/gorm-oracle` v1.1.3**，不切纯 Go 替代方案。
- **现状保留**：实际 Oracle 数据库服务与本地/CI 运行期客户端库尚未进行实机连通性验收。重构实施中保留 Oracle 驱动专项构建/运行环境阻塞标记，不以此阻碍其它无依赖支线的正常推进。

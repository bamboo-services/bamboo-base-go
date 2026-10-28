# Bamboo Base 集成测试隔离夹具说明

本文档描述用于 Bamboo Base 的本地集成测试容器夹具（MySQL、PostgreSQL、Redis、SMTP 及可选 SQLServer）的使用规范与边界定义。

## 1. 核心约束与安全原则

1. **强隔离与仅本地绑定**：
   - 宿主机所有服务均绑定在 `127.0.0.1::port` 随机端口，禁止绑定 `0.0.0.0`。
   - 绝不冲突开发者本地默认端口（如 3306、5432、6379）。
2. **专属参数命名空间**：
   - 测试连接参数统一导出为 `BAMBOO_TEST_*`（如 `BAMBOO_TEST_MYSQL_DSN`）。
   - 严格禁止读取 `.env`，严格禁止复用或覆盖生产/通用 `DATABASE_*` 环境变量。
3. **安全自包含与精准销毁**：
   - 容器均打上 `com.bamboo.test.fixture: "true"` 标签，数据目录采用 `tmpfs` 内存临时文件系统。
   - 清理命令 `down` 仅删除带有该标签的容器与临时卷，绝不执行全局 `docker system prune`，绝不触碰用户环境中的其他无关容器与数据卷。
4. **拒绝非本地 Docker 端点**：
   - 脚本严格核验 Docker endpoint，仅允许本地 unix socket 或 127.0.0.1/localhost。若检测到远程端点将直接阻断并报错退出。
5. **商业许可证与架构边界**：
   - 严禁静默同意商业协议。SQLServer 必须显式传递 `--with-sqlserver` 且提供 `BAMBOO_TEST_SQLSERVER_ACCEPT_EULA=Y`。在 aarch64/ARM 架构下，SQLServer 仍作为单独 profile 保持**隔离未验收**状态，不强行作为通过标准。
   - 绝不自动安装 Rosetta 转译环境。
   - Oracle 必须依赖真实的独立商业或企业测试环境，当前暂未具备，绝不使用 SQLite 或其他非对等容器替代。
6. **测试与运行解耦**：
   - 普通默认单元测试（`go test ./...`）不依赖也不自动拉起或连接这些外部服务。集成测试需显式通过环境变量驱动。

## 2. 夹具管理命令

通过 `.github/scripts/integration-env.sh` 管理：

```bash
# 1. 启动默认测试夹具 (MySQL, PostgreSQL, Redis, SMTP)
./.github/scripts/integration-env.sh up

# 2. 查看容器与健康检查状态
./.github/scripts/integration-env.sh status

# 3. 导出专用环境变量给后续 Go 测试进程
eval $(./.github/scripts/integration-env.sh env)

# 4. 停止并精准清理属于测试夹具的容器与卷
./.github/scripts/integration-env.sh down
```

### 可选组件与受限状态

- **SQLServer**：
  ```bash
  BAMBOO_TEST_SQLSERVER_ACCEPT_EULA=Y ./.github/scripts/integration-env.sh up --with-sqlserver
  ```
- **Oracle**：
  明确标注暂未具备真实测试环境。

## 3. 状态与验收边界

- `integration-env.sh status` 或 `docker compose ps` 显示健康，**仅证明本地容器测试夹具准备就绪**。
- 夹具就绪**不等于**相应插件驱动或集成用例已验收通过。各插件驱动的最终验收须通过真实 Go 测试连接与读写断言进行验证。

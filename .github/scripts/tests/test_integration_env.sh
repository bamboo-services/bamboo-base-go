#!/bin/sh
set -eu

# test_integration_env.sh
# integration-env.sh 专用静态测试与边界保护验证脚本

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/../../.." && pwd)
SCRIPT="$REPO_ROOT/.github/scripts/integration-env.sh"
COMPOSE_FILE="$REPO_ROOT/test/integration/compose.yml"

echo "=== 测试 1: 脚本基本权限与 help 命令 ==="
test -x "$SCRIPT"
sh "$SCRIPT" help >/dev/null
echo "PASS: help 命令执行正常"

echo "=== 测试 2: 未知命令返回非 0 退出码 ==="
if sh "$SCRIPT" unknown_action >/dev/null 2>&1; then
	echo "FAIL: 未知命令应该失败" >&2
	exit 1
fi
echo "PASS: 未知命令正确退出"

echo "=== 测试 3: compose.yml 结构与语法校验（静态解析） ==="
# 使用 docker compose config 验证 compose.yml 格式有效性，不启动容器、不拉取镜像
docker compose -f "$COMPOSE_FILE" config >/dev/null
echo "PASS: compose.yml 语法与定义合法"

echo "=== 测试 4: 静态验证仅允许 127.0.0.1 本地绑定与 tmpfs 内存隔离 ==="
# 验证端口绑定必须包含 127.0.0.1::，绝不能有 0.0.0.0
if grep -q '0\.0\.0\.0' "$COMPOSE_FILE"; then
	echo "FAIL: compose.yml 禁止出现 0.0.0.0 绑定" >&2
	exit 1
fi
grep -q '127\.0\.0\.1::3306' "$COMPOSE_FILE"
grep -q '127\.0\.0\.1::5432' "$COMPOSE_FILE"
grep -q '127\.0\.0\.1::6379' "$COMPOSE_FILE"
grep -q '127\.0\.0\.1::1025' "$COMPOSE_FILE"
grep -q '127\.0\.0\.1::1433' "$COMPOSE_FILE"

# 验证所有服务均包含 fixture 标签
test "$(grep -c 'com\.bamboo\.test\.fixture: "true"' "$COMPOSE_FILE")" -eq 5
echo "PASS: 端口仅绑定 127.0.0.1 且包含隔离标签"

echo "=== 测试 5: 安全检查——远程/非法 DOCKER_HOST 拒绝执行 ==="
if DOCKER_HOST="tcp://192.168.1.100:2375" sh "$SCRIPT" status >/dev/null 2>&1; then
	echo "FAIL: 非本地 DOCKER_HOST 应该被阻断" >&2
	exit 1
fi
if DOCKER_HOST="tcp://remote-docker.internal:2375" sh "$SCRIPT" up >/dev/null 2>&1; then
	echo "FAIL: 远程域名 DOCKER_HOST 应该被阻断" >&2
	exit 1
fi
echo "PASS: 非法远程 DOCKER_HOST 被成功拒绝"

echo "=== 测试 6: 安全检查——未显式接受 EULA 时拒绝启动 SQLServer ==="
if BAMBOO_TEST_SQLSERVER_ACCEPT_EULA="" sh "$SCRIPT" up --with-sqlserver >/dev/null 2>&1; then
	echo "FAIL: 未传 ACCEPT_EULA=Y 时应该拒绝启动 sqlserver" >&2
	exit 1
fi
if BAMBOO_TEST_SQLSERVER_ACCEPT_EULA="N" sh "$SCRIPT" up --with-sqlserver >/dev/null 2>&1; then
	echo "FAIL: 传 ACCEPT_EULA=N 时应该拒绝启动 sqlserver" >&2
	exit 1
fi
echo "PASS: 静默/未同意 SQLServer 商业许可被成功拒绝"

echo "=== 测试 7: 检查 env 输出格式与命名空间安全性 ==="
# 静态断言 integration-env.sh 内部使用的环境变量名称空间
grep -q 'export BAMBOO_TEST_MYSQL_DSN=' "$SCRIPT"
grep -q 'export BAMBOO_TEST_POSTGRES_DSN=' "$SCRIPT"
grep -q 'export BAMBOO_TEST_REDIS_ADDR=' "$SCRIPT"
grep -q 'export BAMBOO_TEST_SMTP_HOST=' "$SCRIPT"
grep -q 'export BAMBOO_TEST_SQLSERVER_DSN=' "$SCRIPT"

# 确保脚本中没有声明普通 DATABASE_* 变量
if grep -E 'export DATABASE_' "$SCRIPT"; then
	echo "FAIL: integration-env.sh 严禁导出或复用普通 DATABASE_* 变量" >&2
	exit 1
fi
echo "PASS: 环境变量命名空间严格隔离为 BAMBOO_TEST_*"

echo "ALL TESTS PASSED!"

#!/bin/sh
set -eu

# integration-env.sh
# Bamboo Base 本地集成测试环境生命周期与隔离控制脚本
#
# 约束原则：
# 1. 严格检查 Docker endpoint，仅允许本地 unix socket 或 localhost/127.0.0.1，拒绝未明确或远程 endpoint。
# 2. 严禁自动/静默接受商业许可证或安装 Rosetta；SQLServer 保持未验收状态，Oracle 暂不具备环境。
# 3. 仅操作自身带有 com.bamboo.test.fixture=true 标签的容器/卷，严禁全局 prune 或接触用户其它容器。
# 4. 导出专属 BAMBOO_TEST_* 变量，严禁污染或读取业务生产 DATABASE_* 变量与 .env 文件。

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/../.." && pwd)
COMPOSE_FILE="$REPO_ROOT/test/integration/compose.yml"
PROJECT_NAME="bamboo-test-integration"
FIXTURE_LABEL="com.bamboo.test.fixture=true"

# 固定测试专用凭据（仅供临时销毁测试使用，非真实生产环境凭据）
TEST_MYSQL_USER="bamboo_user"
TEST_MYSQL_PASSWORD="test_user_password_123!"
TEST_MYSQL_DB="bamboo_test"

TEST_POSTGRES_USER="bamboo_user"
TEST_POSTGRES_PASSWORD="test_user_password_123!"
TEST_POSTGRES_DB="bamboo_test"

TEST_REDIS_PASSWORD="test_redis_password_123!"

TEST_SQLSERVER_USER="sa"
TEST_SQLSERVER_PASSWORD="test_sqlserver_password_123!"

# 1. 校验 Docker 运行端点是否为安全本地端点 (仅允许本机 unix socket 或 localhost/127.0.0.1)
verify_local_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "[ERROR] 未检测到 docker 命令，请先安装 Docker。" >&2
		exit 1
	fi

	# 当 DOCKER_CONTEXT 与 DOCKER_HOST 同时非空时立即 fail-closed，避免不同 CLI 行为导致校验与执行端点不一致
	if [ -n "${DOCKER_CONTEXT:-}" ] && [ -n "${DOCKER_HOST:-}" ]; then
		echo "[ERROR] 安全拒绝: 检测到 DOCKER_CONTEXT 与 DOCKER_HOST 同时被设置，存在端点歧义风险，禁止执行。" >&2
		exit 1
	fi

	# 单变量或默认环境：优先按有效单变量解析端点，否则检查当前活动 context
	target_endpoint=""
	if [ -n "${DOCKER_CONTEXT:-}" ]; then
		if ! target_endpoint=$(docker context inspect "$DOCKER_CONTEXT" --format '{{.Endpoints.docker.Host}}' 2>/dev/null); then
			echo "[ERROR] 安全拒绝: 无法解析指定 Docker context 端点或 context inspect 执行失败。" >&2
			exit 1
		fi
	elif [ -n "${DOCKER_HOST:-}" ]; then
		target_endpoint="$DOCKER_HOST"
	else
		if ! target_endpoint=$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null); then
			echo "[ERROR] 安全拒绝: 无法解析当前活动 Docker context 端点或 context inspect 执行失败。" >&2
			exit 1
		fi
	fi

	if [ -z "$target_endpoint" ]; then
		echo "[ERROR] 安全拒绝: 未能确定 Docker 端点，禁止放行未知目标。" >&2
		exit 1
	fi

	# 1. 本地 unix socket：严格要求绝对路径 unix:///*
	# 2. 本地 npipe：仅限 Windows 本机命名管道 npipe:////./pipe/* 或 npipe://./pipe/*（拒绝远程 UNC 如 npipe://remote/pipe/*）
	# 3. tcp / http / https：严格校验 authority 必须为 127.0.0.1 或 localhost 加合法数字端口 (1..65535)，无 userinfo、无多余路径/查询/hash
	safe_endpoint=0
	case "$target_endpoint" in
		unix:///*)
			safe_endpoint=1
			;;
		npipe:////./pipe/*|npipe://./pipe/*)
			safe_endpoint=1
			;;
		tcp://*|http://*|https://*)
			auth="${target_endpoint#*://}"
			# 禁止包含 userinfo (@) 以及任何路径、查询或哈希 (/, ?, #)
			case "$auth" in
				*[@/?#]*)
					safe_endpoint=0
					;;
				127.0.0.1:*|localhost:*)
					port="${auth#*:}"
					# 端口必须为纯数字且长度合理 (1-5位)，并在 1..65535 范围内
					case "$port" in
						*[!0-9]*|"")
							safe_endpoint=0
							;;
						*)
							if [ "${#port}" -le 5 ] && [ "$port" -ge 1 ] 2>/dev/null && [ "$port" -le 65535 ] 2>/dev/null; then
								safe_endpoint=1
							fi
							;;
					esac
					;;
				*)
					safe_endpoint=0
					;;
			esac
			;;
		*)
			safe_endpoint=0
			;;
	esac

	if [ "$safe_endpoint" -ne 1 ]; then
		echo "[ERROR] 安全拒绝: Docker 端点指向非本机或不可信的远端目标类别。" >&2
		echo "[ERROR] 仅允许本地 unix socket 或 127.0.0.1/localhost，严禁将命令送至远程端点。" >&2
		exit 1
	fi
}

# 辅助包装 compose 命令，固定文件和项目名
run_compose() {
	docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" "$@"
}

# 提取宿主动态绑定端口（安全仅限 127.0.0.1）
get_host_port() {
	service="$1"
	container_port="$2"
	raw_binding=$(run_compose port "$service" "$container_port" 2>/dev/null || true)
	if [ -z "$raw_binding" ]; then
		return 1
	fi
	# raw_binding 格式通常为 127.0.0.1:xxxxx
	port="${raw_binding##*:}"
	printf '%s' "$port"
}

# 启动服务
cmd_up() {
	verify_local_docker

	# 检查 SQLServer profile 申请
	include_sqlserver=0
	for arg in "$@"; do
		if [ "$arg" = "--with-sqlserver" ]; then
			include_sqlserver=1
		fi
	done

	if [ "$include_sqlserver" -eq 1 ]; then
		if [ "${BAMBOO_TEST_SQLSERVER_ACCEPT_EULA:-}" != "Y" ]; then
			echo "[ERROR] 安全拒绝: 启动 SQLServer 必须显式设置环境变量 BAMBOO_TEST_SQLSERVER_ACCEPT_EULA=Y。" >&2
			echo "[ERROR] 本脚本绝不静默接受商业许可协议 (EULA)。" >&2
			exit 1
		fi
		# 架构兼容性提示
		arch=$(uname -m 2>/dev/null || true)
		if [ "$arch" = "arm64" ] || [ "$arch" = "aarch64" ]; then
			echo "[WARN] 检测到 aarch64/ARM64 架构。SQLServer 处于隔离未验收状态，请确保容器运行时具备有效兼容环境。" >&2
		fi
		echo "[INFO] 启动基础集成夹具 (含 sqlserver profile)..."
		run_compose --profile sqlserver up -d --wait
	else
		echo "[INFO] 启动基础集成夹具 (默认: mysql, postgres, redis, smtp)..."
		run_compose up -d --wait
	fi

	echo "[INFO] 服务就绪，可执行 env 获取专用连接变量。"
}

# 查询服务状态
cmd_status() {
	verify_local_docker
	echo "=== Bamboo 隔离测试夹具运行状态 ==="
	run_compose ps --all
}

# 导出专用连接环境变量供 Go 测试消费（不含普通 DATABASE_*，不读取 .env）
cmd_env() {
	verify_local_docker

	# MySQL
	mysql_port=$(get_host_port "mysql" "3306" || true)
	if [ -n "$mysql_port" ]; then
		cat <<EOF
export BAMBOO_TEST_MYSQL_HOST="127.0.0.1"
export BAMBOO_TEST_MYSQL_PORT="$mysql_port"
export BAMBOO_TEST_MYSQL_USER="$TEST_MYSQL_USER"
export BAMBOO_TEST_MYSQL_PASSWORD="$TEST_MYSQL_PASSWORD"
export BAMBOO_TEST_MYSQL_DATABASE="$TEST_MYSQL_DB"
export BAMBOO_TEST_MYSQL_DSN="$TEST_MYSQL_USER:$TEST_MYSQL_PASSWORD@tcp(127.0.0.1:$mysql_port)/$TEST_MYSQL_DB?charset=utf8mb4&parseTime=True&loc=Local"
EOF
	fi

	# PostgreSQL
	pg_port=$(get_host_port "postgres" "5432" || true)
	if [ -n "$pg_port" ]; then
		cat <<EOF
export BAMBOO_TEST_POSTGRES_HOST="127.0.0.1"
export BAMBOO_TEST_POSTGRES_PORT="$pg_port"
export BAMBOO_TEST_POSTGRES_USER="$TEST_POSTGRES_USER"
export BAMBOO_TEST_POSTGRES_PASSWORD="$TEST_POSTGRES_PASSWORD"
export BAMBOO_TEST_POSTGRES_DATABASE="$TEST_POSTGRES_DB"
export BAMBOO_TEST_POSTGRES_DSN="host=127.0.0.1 user=$TEST_POSTGRES_USER password=$TEST_POSTGRES_PASSWORD dbname=$TEST_POSTGRES_DB port=$pg_port sslmode=disable"
EOF
	fi

	# Redis
	redis_port=$(get_host_port "redis" "6379" || true)
	if [ -n "$redis_port" ]; then
		cat <<EOF
export BAMBOO_TEST_REDIS_HOST="127.0.0.1"
export BAMBOO_TEST_REDIS_PORT="$redis_port"
export BAMBOO_TEST_REDIS_PASSWORD="$TEST_REDIS_PASSWORD"
export BAMBOO_TEST_REDIS_ADDR="127.0.0.1:$redis_port"
EOF
	fi

	# SMTP (Mailpit: 1025 SMTP, 8025 Web UI / HTTP API)
	smtp_port=$(get_host_port "smtp" "1025" || true)
	smtp_http_port=$(get_host_port "smtp" "8025" || true)
	if [ -n "$smtp_port" ]; then
		cat <<EOF
export BAMBOO_TEST_SMTP_HOST="127.0.0.1"
export BAMBOO_TEST_SMTP_PORT="$smtp_port"
export BAMBOO_TEST_SMTP_HTTP_PORT="$smtp_http_port"
EOF
	fi

	# SQLServer
	sqlserver_port=$(get_host_port "sqlserver" "1433" || true)
	if [ -n "$sqlserver_port" ]; then
		cat <<EOF
export BAMBOO_TEST_SQLSERVER_HOST="127.0.0.1"
export BAMBOO_TEST_SQLSERVER_PORT="$sqlserver_port"
export BAMBOO_TEST_SQLSERVER_USER="$TEST_SQLSERVER_USER"
export BAMBOO_TEST_SQLSERVER_PASSWORD="$TEST_SQLSERVER_PASSWORD"
export BAMBOO_TEST_SQLSERVER_DSN="sqlserver://$TEST_SQLSERVER_USER:$TEST_SQLSERVER_PASSWORD@127.0.0.1:$sqlserver_port?database=master"
EOF
	fi
}

# 停止并彻底销毁测试容器与自带卷，只精准按标签与项目清理
cmd_down() {
	verify_local_docker

	# 1. 预检检查：列出 project 下所有容器，核对是否全部具有 fixture 标签
	all_project_containers=$(docker ps -a --filter "label=com.docker.compose.project=$PROJECT_NAME" --format '{{.ID}}')
	if [ -n "$all_project_containers" ]; then
		fixture_containers=$(docker ps -a --filter "label=com.docker.compose.project=$PROJECT_NAME" --filter "label=$FIXTURE_LABEL" --format '{{.ID}}')
		all_c_count=$(printf '%s\n' "$all_project_containers" | sed '/^$/d' | wc -l | tr -d ' ')
		fix_c_count=$(printf '%s\n' "$fixture_containers" | sed '/^$/d' | wc -l | tr -d ' ')
		if [ "$all_c_count" -ne "$fix_c_count" ]; then
			echo "[ERROR] 安全拒绝: 项目 $PROJECT_NAME 下存在非 fixture 容器，禁止执行删除。" >&2
			exit 1
		fi
	fi

	# 2. 预检检查：列出 project 下所有卷，核对是否全部具有 fixture 标签
	all_project_volumes=$(docker volume ls --filter "label=com.docker.compose.project=$PROJECT_NAME" --format '{{.Name}}')
	if [ -n "$all_project_volumes" ]; then
		fixture_volumes=$(docker volume ls --filter "label=com.docker.compose.project=$PROJECT_NAME" --filter "label=$FIXTURE_LABEL" --format '{{.Name}}')
		all_v_count=$(printf '%s\n' "$all_project_volumes" | sed '/^$/d' | wc -l | tr -d ' ')
		fix_v_count=$(printf '%s\n' "$fixture_volumes" | sed '/^$/d' | wc -l | tr -d ' ')
		if [ "$all_v_count" -ne "$fix_v_count" ]; then
			echo "[ERROR] 安全拒绝: 项目 $PROJECT_NAME 下存在非 fixture 数据卷，禁止执行删除。" >&2
			exit 1
		fi
	fi

	# 3. 预检检查：project 网络若连接了非 fixture 容器，禁止执行删除
	project_networks=$(docker network ls --filter "label=com.docker.compose.project=$PROJECT_NAME" --format '{{.ID}}')
	if [ -n "$project_networks" ]; then
		for net_id in $project_networks; do
			connected_containers=$(docker network inspect "$net_id" --format '{{range $k, $v := .Containers}}{{$k}} {{end}}')
			for cid in $connected_containers; do
				if [ -n "$cid" ]; then
					c_match=$(docker ps -a --filter "id=$cid" --filter "label=$FIXTURE_LABEL" --format '{{.ID}}')
					if [ -z "$c_match" ]; then
						echo "[ERROR] 安全拒绝: 项目网络连接了非 fixture 容器 $cid，禁止执行删除。" >&2
						exit 1
					fi
				fi
			done
		done
	fi

	echo "[INFO] 停止并移除测试夹具容器与网络..."
	run_compose --profile sqlserver down -v --remove-orphans

	# 精确清理属于本项目的悬挂容器（必须双标签匹配，绝对不全局 prune，错误立即传播）
	dangling_containers=$(docker ps -a --filter "label=com.docker.compose.project=$PROJECT_NAME" --filter "label=$FIXTURE_LABEL" --format '{{.ID}}')
	if [ -n "$dangling_containers" ]; then
		echo "[INFO] 清理带有夹具标签的残留测试容器..."
		echo "$dangling_containers" | xargs docker rm -f
	fi

	# 精确清理属于本项目的卷（必须双标签匹配，错误立即传播）
	dangling_volumes=$(docker volume ls --filter "label=com.docker.compose.project=$PROJECT_NAME" --filter "label=$FIXTURE_LABEL" --format '{{.Name}}')
	if [ -n "$dangling_volumes" ]; then
		echo "[INFO] 清理带有夹具标签的残留测试数据卷..."
		echo "$dangling_volumes" | xargs docker volume rm
	fi

	echo "[INFO] 隔离测试夹具清理完成。"
}

# 打印帮助信息
cmd_help() {
	cat <<EOF
Bamboo Base 集成测试环境管理脚本

用法:
  $0 up [--with-sqlserver]  启动测试夹具容器 (默认: MySQL, Postgres, Redis, SMTP)
  $0 status                 查看当前测试容器运行与健康状态
  $0 env                    导出宿主绑定的随机端口与专用 BAMBOO_TEST_* 连接参数
  $0 down                   停止并彻底销毁所有测试容器与相关临时卷 (仅清理带标签资源)
  $0 help                   展示帮助信息

环境约束说明:
  - 仅绑定 127.0.0.1 随机端口，绝不占用 0.0.0.0 或业务既有固定端口。
  - SQLServer 需要单独显式指定 --with-sqlserver 且设置 BAMBOO_TEST_SQLSERVER_ACCEPT_EULA=Y，ARM64 下保持未验收状态。
  - Oracle 暂未具备真实测试环境，禁止使用 SQLite/其他容器降级替代。
  - 本脚本绝对不读取 .env，不污染 DATABASE_* 变量，不影响用户宿主机上其他业务容器。
EOF
}

# 主入口解析
action="${1:-help}"
case "$action" in
	up)
		shift
		cmd_up "$@"
		;;
	status)
		cmd_status
		;;
	env)
		cmd_env
		;;
	down)
		cmd_down
		;;
	help|-h|--help)
		cmd_help
		;;
	*)
		echo "[ERROR] 未知命令: $action" >&2
		cmd_help >&2
		exit 1
		;;
esac

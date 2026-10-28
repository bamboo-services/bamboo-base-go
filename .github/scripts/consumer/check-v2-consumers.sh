#!/usr/bin/env bash
set -euo pipefail

# check-v2-consumers.sh
# 独立消费脚本集成与隔离检查工具
# 验证 v2 模块在脱离 go.work (GOWORK=off) 独立通过 local-module-proxy 消费时的完整性、依赖隔离性与编译/测试状态。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MANIFEST_FILE="${REPO_ROOT}/.github/scripts/manifest/v2-modules.txt"
if [ ! -f "${MANIFEST_FILE}" ] && [ -f "${SCRIPT_DIR}/v2-modules.txt" ]; then
	MANIFEST_FILE="${SCRIPT_DIR}/v2-modules.txt"
fi
WRAPPER="${REPO_ROOT}/.github/scripts/proxy/with-local-modules.sh"
if [ ! -f "${WRAPPER}" ] && [ -f "${SCRIPT_DIR}/with-local-modules.sh" ]; then
	WRAPPER="${SCRIPT_DIR}/with-local-modules.sh"
fi

USAGE="Usage: $0 [--manifest path] [--version vX.Y.Z] [--flags 'go test flags'] [--skip-tests] [--check-isolation] [--module dir]"

MANIFEST="${MANIFEST_FILE}"
VERSION="v2.0.0"
TEST_FLAGS="-mod=readonly"
SKIP_TESTS=0
CHECK_ISOLATION=1
FILTER_MODULE=""

while [ "$#" -gt 0 ]; do
	case "$1" in
	--manifest)
		if [ "$#" -lt 2 ]; then
			echo "error: --manifest requires a path argument" >&2
			exit 1
		fi
		MANIFEST="$2"
		shift 2
		;;
	--version)
		if [ "$#" -lt 2 ]; then
			echo "error: --version requires a version argument" >&2
			exit 1
		fi
		VERSION="$2"
		shift 2
		;;
	--flags)
		if [ "$#" -lt 2 ]; then
			echo "error: --flags requires an argument" >&2
			exit 1
		fi
		TEST_FLAGS="$2"
		shift 2
		;;
	--skip-tests)
		SKIP_TESTS=1
		shift
		;;
	--no-isolation)
		CHECK_ISOLATION=0
		shift
		;;
	--check-isolation)
		CHECK_ISOLATION=1
		shift
		;;
	--module)
		if [ "$#" -lt 2 ]; then
			echo "error: --module requires a directory argument" >&2
			exit 1
		fi
		FILTER_MODULE="$2"
		shift 2
		;;
	-h | --help)
		echo "${USAGE}"
		exit 0
		;;
	*)
		echo "error: unknown argument: $1" >&2
		echo "${USAGE}" >&2
		exit 1
		;;
	esac
done

if [ ! -f "${MANIFEST}" ]; then
	echo "error: manifest file not found: ${MANIFEST}" >&2
	exit 1
fi

# 严格移除对 WRAPPER 的 chmod 操作，改在调用时显式使用 bash "$WRAPPER"

# 1. 严格 14 个模块白名单定义与 manifest 校验
EXACT_14_ALLOWLIST=(
	"foundation"
	"application"
	"plugins/database"
	"plugins/cache"
	"plugins/email"
	"plugins/web-gin"
	"plugins/grpc"
	"plugins/cron"
	"plugins/async"
	"plugins/database/adapter/mysql"
	"plugins/database/adapter/postgres"
	"plugins/database/adapter/sqlite"
	"plugins/database/adapter/sqlserver"
	"plugins/database/adapter/oracle"
)

is_allowed_14() {
	local target="$1"
	for item in "${EXACT_14_ALLOWLIST[@]}"; do
		if [ "${item}" = "${target}" ]; then
			return 0
		fi
	done
	return 1
}

declare -a DIRS=()
declare -a MODULE_PATHS=()
ALL_MODULES_CSV=""

while IFS= read -r line || [ -n "${line}" ]; do
	line="$(echo "${line}" | xargs)"
	# 跳过空行和注释
	if [ -z "${line}" ] || [[ "${line}" =~ ^# ]]; then
		continue
	fi

	dir="$(echo "${line}" | awk '{print $1}')"
	mod="$(echo "${line}" | awk '{print $2}')"

	if [ -z "${dir}" ] || [ -z "${mod}" ]; then
		echo "error: malformed line in manifest: ${line}" >&2
		exit 1
	fi

	# 路径防穿越与防相对路径注入
	if [[ "${dir}" =~ \.\. ]] || [[ "${dir}" =~ ^/ ]]; then
		echo "error: invalid path or directory traversal detected: ${dir}" >&2
		exit 1
	fi

	# 必须严格存在于 14 白名单中
	if ! is_allowed_14 "${dir}"; then
		echo "error: directory '${dir}' is not in the allowed 14 v2 modules allowlist" >&2
		exit 1
	fi

	# 防重复项
	if [ "${#DIRS[@]}" -gt 0 ]; then
		for existing_d in "${DIRS[@]}"; do
			if [ "${existing_d}" = "${dir}" ]; then
				echo "error: duplicate directory entry in manifest: ${dir}" >&2
				exit 1
			fi
		done
	fi

	if [ "${#MODULE_PATHS[@]}" -gt 0 ]; then
		for existing_m in "${MODULE_PATHS[@]}"; do
			if [ "${existing_m}" = "${mod}" ]; then
				echo "error: duplicate module path entry in manifest: ${mod}" >&2
				exit 1
			fi
		done
	fi

	# 路径与 module 禁止包含 v1 关键字
	if [[ "${dir}" =~ ^(common|defined|major)$ ]] || [[ "${dir}" =~ ^plugins/database/(mysql|postgres|sqlite|oracle|sqlserver)$ ]]; then
		echo "error: manifest directory contains forbidden v1 path: ${dir}" >&2
		exit 1
	fi

	# 规范校验：module path 必须匹配约定 prefix/dir/v2
	EXPECTED_PREFIX="github.com/bamboo-services/bamboo-base-go/${dir}/v2"
	if [ "${mod}" != "${EXPECTED_PREFIX}" ]; then
		echo "error: module path '${mod}' does not match expected convention '${EXPECTED_PREFIX}'" >&2
		exit 1
	fi

	# 从实际真实 go.mod 校对
	ACTUAL_MOD_FILE="${REPO_ROOT}/${dir}/go.mod"
	if [ ! -f "${ACTUAL_MOD_FILE}" ]; then
		echo "error: candidate source directory missing go.mod: ${dir}" >&2
		exit 1
	fi

	ACTUAL_MOD_PATH="$(awk '$1 == "module" { print $2; exit }' "${ACTUAL_MOD_FILE}")"
	if [ "${ACTUAL_MOD_PATH}" != "${mod}" ]; then
		echo "error: go.mod in ${dir} declares '${ACTUAL_MOD_PATH}', manifest has '${mod}'" >&2
		exit 1
	fi

	DIRS+=("${dir}")
	MODULE_PATHS+=("${mod}")

	if [ -z "${ALL_MODULES_CSV}" ]; then
		ALL_MODULES_CSV="${dir}"
	else
		ALL_MODULES_CSV="${ALL_MODULES_CSV},${dir}"
	fi
done <"${MANIFEST}"

MANIFEST_COUNT="${#DIRS[@]}"
if [ "${MANIFEST_COUNT}" -ne 14 ]; then
	echo "error: manifest must contain exactly 14 modules, got ${MANIFEST_COUNT}" >&2
	exit 1
fi

# 确保白名单中每一个模块都在 manifest 中唯一定义
for required_dir in "${EXACT_14_ALLOWLIST[@]}"; do
	found=0
	for parsed_d in "${DIRS[@]}"; do
		if [ "${parsed_d}" = "${required_dir}" ]; then
			found=1
			break
		fi
	done
	if [ "${found}" -eq 0 ]; then
		echo "error: missing required v2 module from manifest: ${required_dir}" >&2
		exit 1
	fi
done

# 如果指定了 --module filter，必须严格命中 14 白名单之一
if [ -n "${FILTER_MODULE}" ]; then
	if ! is_allowed_14 "${FILTER_MODULE}"; then
		echo "error: filter --module '${FILTER_MODULE}' is not a valid v2 module in the 14-module allowlist" >&2
		exit 1
	fi
fi

echo "================================================================"
echo "Bamboo Base Go v2 消费者离线验证 (GOWORK=off)"
echo "Manifest: ${MANIFEST} (${MANIFEST_COUNT} modules)"
echo "Version:  ${VERSION}"
echo "Flags:    ${TEST_FLAGS}"
echo "================================================================"

# 2. 生成精确候选清单证据 (tagprefix, dir/vX, 当前源码 git sha / generator sha256)
echo ""
echo "--- [1/3] 生成候选模块清单与版本证据 ---"
TMP_PROXY_VERIFY="$(mktemp -d -t bamboo-verify-proxy.XXXXXX)"
cleanup_verify() {
	chmod -R u+w "${TMP_PROXY_VERIFY}" 2>/dev/null || true
	rm -rf "${TMP_PROXY_VERIFY}"
}
trap cleanup_verify EXIT INT TERM

# 运行 local-module-proxy 生成器收集 SHA256 证据，绝不吞错误
PROXY_LOG="${TMP_PROXY_VERIFY}/proxy_generator.log"
PROXY_GENERATOR="${REPO_ROOT}/.github/scripts/proxy/local-module-proxy.go"
if [ ! -f "${PROXY_GENERATOR}" ] && [ -f "${SCRIPT_DIR}/local-module-proxy.go" ]; then
	PROXY_GENERATOR="${SCRIPT_DIR}/local-module-proxy.go"
fi
if ! go run "${PROXY_GENERATOR}" \
	-repo-root "${REPO_ROOT}" \
	-out-dir "${TMP_PROXY_VERIFY}" \
	-modules "${ALL_MODULES_CSV}" \
	-version "${VERSION}" >"${PROXY_LOG}" 2>&1; then
	echo "error: local-module-proxy generator failed!" >&2
	cat "${PROXY_LOG}" >&2
	exit 1
fi
PROXY_OUTPUT="$(cat "${PROXY_LOG}")"

for i in "${!DIRS[@]}"; do
	d="${DIRS[$i]}"
	m="${MODULE_PATHS[$i]}"
	tag_prefix="${d}/${VERSION}"

	# 提取 generator 计算的 proxy sha256 (完整 64 位哈希)
	proxy_sha="$(echo "${PROXY_OUTPUT}" | grep "generated ${m}@${VERSION}" | sed -E 's/.*sha256: ([a-f0-9]{64}).*/\1/' || true)"
	if [ -z "${proxy_sha}" ]; then
		echo "error: could not extract proxy sha256 for ${m}@${VERSION}" >&2
		exit 1
	fi

	# 本地提交 tree/commit sha 仅作为辅助基准标明，以 zipSHA 为准
	git_sha="$(git -C "${REPO_ROOT}" log -n 1 --format="%h" -- "${d}" 2>/dev/null || echo "working-tree")"

	printf "Candidate: %-36s | Tag: %-38s | ZipSHA256: %s | (git-base: %s)\n" \
		"${m}" "${tag_prefix}" "${proxy_sha}" "${git_sha}"
done

# 3. 驱动与模块依赖隔离性检查
# 规则 A: plugins/database 主模块的 go list -m all 不能出现具体数据库驱动 concrete drivers
# 规则 B: consumer 仅 import sqlite 时，go list -deps 绝不能引入其他具体驱动 (mysql, postgres, oracle, sqlserver)
ISOLATION_FAILED=0

if [ "${CHECK_ISOLATION}" -eq 1 ]; then
	echo ""
	echo "--- [2/3] 主/驱动隔离与无泄漏依赖检查 ---"

	# 检查 A: plugins/database 主模块
	DB_MOD_DIR="${REPO_ROOT}/plugins/database"
	if [ -d "${DB_MOD_DIR}" ]; then
		echo "检查 plugins/database 依赖隔离性..."
		DB_LIST_LOG="${TMP_PROXY_VERIFY}/db_list.log"
		if ! (
			cd "${DB_MOD_DIR}"
			GOWORK=off bash "${WRAPPER}" --modules "${ALL_MODULES_CSV}" --version "${VERSION}" -- go list -m all >"${DB_LIST_LOG}" 2>&1
		); then
			echo "error: failed to inspect dependencies of plugins/database (go list -m all)" >&2
			cat "${DB_LIST_LOG}" >&2
			exit 1
		fi
		DB_MODULES="$(grep -v '^[[:space:]]*proxy:' "${DB_LIST_LOG}" | grep -v 'gorm.io/driver/sqlite' || true)"
		# 排除 GORM v1.31.2 自身 upstream go.mod 声明的 indirect 依赖 gorm.io/driver/sqlite；
		# Bamboo 的 SQLite 真实驱动实现为 libtnb/sqlite，任何对其的具体依赖均被严格禁止
		FORBIDDEN_CONCRETE_DRIVERS=("adapter/mysql" "adapter/postgres" "adapter/sqlite" "adapter/sqlserver" "adapter/oracle" "gorm.io/driver/mysql" "gorm.io/driver/postgres" "libtnb/sqlite" "gorm.io/driver/sqlserver")
		LEAKED=0
		for drv in "${FORBIDDEN_CONCRETE_DRIVERS[@]}"; do
			if echo "${DB_MODULES}" | grep -q "${drv}"; then
				echo "FAIL: plugins/database 泄漏了具体驱动依赖: ${drv}" >&2
				LEAKED=1
				ISOLATION_FAILED=1
			fi
		done
		if [ "${LEAKED}" -eq 0 ]; then
			echo "PASS: plugins/database 主模块无 concrete driver 依赖泄漏"
		fi
	fi

	# 检查 B: 仅引入 sqlite 适配器的独立 consumer 隔离性检查
	TMP_CONSUMER_DIR="${TMP_PROXY_VERIFY}/sqlite-consumer"
	mkdir -p "${TMP_CONSUMER_DIR}"
	cat <<EOF >"${TMP_CONSUMER_DIR}/go.mod"
module example.com/sqlite-consumer

go 1.25.0

require (
	github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2 ${VERSION}
	github.com/bamboo-services/bamboo-base-go/plugins/database/v2 ${VERSION}
)
EOF

	cat <<EOF >"${TMP_CONSUMER_DIR}/main.go"
package main

import (
	_ "github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2"
	_ "github.com/bamboo-services/bamboo-base-go/plugins/database/v2"
)

func main() {}
EOF

	echo "检查仅依赖 SQLite 消费者的依赖边界 (go list -deps)..."
	SQLITE_DEPS_LOG="${TMP_PROXY_VERIFY}/sqlite_deps.log"
	if ! (
		cd "${TMP_CONSUMER_DIR}"
		GOWORK=off bash "${WRAPPER}" --modules "${ALL_MODULES_CSV}" --version "${VERSION}" -- \
			bash -c "go mod tidy && go list -deps ./..." >"${SQLITE_DEPS_LOG}" 2>&1
	); then
		echo "error: failed to inspect dependencies of SQLite consumer (go list -deps)" >&2
		cat "${SQLITE_DEPS_LOG}" >&2
		exit 1
	fi
	CONSUMER_DEPS="$(grep -v '^[[:space:]]*proxy:' "${SQLITE_DEPS_LOG}" || true)"

	LEAKED_OTHER_DRIVERS=0
	for other in "adapter/mysql" "adapter/postgres" "adapter/sqlserver" "adapter/oracle"; do
		if echo "${CONSUMER_DEPS}" | grep -q "${other}"; then
			echo "FAIL: 仅依赖 SQLite 的消费者泄漏了其他驱动依赖: ${other}" >&2
			LEAKED_OTHER_DRIVERS=1
			ISOLATION_FAILED=1
		fi
	done
	if [ "${LEAKED_OTHER_DRIVERS}" -eq 0 ]; then
		echo "PASS: SQLite 消费者依赖完全隔离，无其它数据库驱动污染"
	fi
fi

if [ "${ISOLATION_FAILED}" -ne 0 ]; then
	echo "error: 驱动或依赖隔离性检查未通过！" >&2
	exit 1
fi

# 4. 逐 Module 真正离 Workspace (GOWORK=off) 编译/测试执行
echo ""
echo "--- [3/3] 逐模块真实消费执行 (GOWORK=off) ---"
if [ "${SKIP_TESTS}" -eq 1 ]; then
	echo "跳过详细测试执行 (--skip-tests)"
	if [ -n "${FILTER_MODULE}" ]; then
		echo "指定模块 [${FILTER_MODULE}] 清单与隔离性校验完成 (注意: 使用了 --module，未覆盖其余模块)。"
	else
		echo "所有 14 个模块清单与隔离性校验完成。"
	fi
	exit 0
fi

FAILED_MODULES=()
SUCCESS_MODULES=()

for d in "${DIRS[@]}"; do
	if [ -n "${FILTER_MODULE}" ] && [ "${FILTER_MODULE}" != "${d}" ]; then
		continue
	fi

	TARGET_DIR="${REPO_ROOT}/${d}"
	echo -n "Testing ${d} ... "

	# 运行编译/测试命令，使用用户指定的 TEST_FLAGS
	# 注意: 不使用全部跑多次的模式，只执行一次指定 flags
	SET_ERR=0
	TEST_LOG="${TMP_PROXY_VERIFY}/test_${d//\//_}.log"

	# 执行 GOWORK=off 模式命令
	(
		cd "${TARGET_DIR}"
		GOWORK=off bash "${WRAPPER}" --modules "${ALL_MODULES_CSV}" --version "${VERSION}" -- \
			go test ${TEST_FLAGS} ./... >"${TEST_LOG}" 2>&1
	) || SET_ERR=$?

	if [ "${SET_ERR}" -eq 0 ]; then
		echo "OK"
		SUCCESS_MODULES+=("${d}")
	else
		echo "FAIL (exit ${SET_ERR})"
		FAILED_MODULES+=("${d}")
		# 打印错误摘要最后几行
		tail -n 10 "${TEST_LOG}" | sed 's/^/    [err] /' || true
	fi
done

echo ""
echo "================================================================"
echo "执行总结:"
echo "成功模块 (${#SUCCESS_MODULES[@]}): ${SUCCESS_MODULES[*]:-none}"
echo "失败模块 (${#FAILED_MODULES[@]}): ${FAILED_MODULES[*]:-none}"
echo "================================================================"

if [ "${#FAILED_MODULES[@]}" -gt 0 ]; then
	echo "注意: 存在测试或编译失败模块，非静默截断，退出状态为 1" >&2
	exit 1
fi

if [ -n "${FILTER_MODULE}" ]; then
	echo "指定模块 [${FILTER_MODULE}] 检查完成 (注意: 使用了 --module，未覆盖其余模块)。"
else
	echo "所有 14 个模块检查成功完成。"
fi
exit 0

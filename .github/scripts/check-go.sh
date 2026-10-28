#!/usr/bin/env bash
set -euo pipefail

# check-go.sh - 遍历 v2 精确 14 个模块执行 test / race / vet / tidy
# 严格按照 .github/scripts/v2-modules.txt 清单逐 module 执行，fail-fast 退出。
# 根目录仅作为安装占位，不参与测试。
# 兼容 macOS 自带 Bash 3.2+。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MANIFEST="${SCRIPT_DIR}/v2-modules.txt"
WRAPPER="${SCRIPT_DIR}/with-local-modules.sh"

# 如果未处于局部代理环境且未处于 mock 测试（未设置 LOG_FILE），通过 with-local-modules.sh 装配候选代理透明重入
if [ "${BAMBOO_INSIDE_LOCAL_MODULES:-0}" -eq 0 ] && [ -z "${LOG_FILE:-}" ] && [ -f "${WRAPPER}" ]; then
	if [ ! -f "${MANIFEST}" ]; then
		echo "错误: 清单文件不存在: ${MANIFEST}" >&2
		exit 1
	fi
	MODULES_CSV="$(grep -v '^[[:space:]]*#' "${MANIFEST}" | grep -v '^[[:space:]]*$' | awk '{print $1}' | paste -sd, -)"
	exec env BAMBOO_INSIDE_LOCAL_MODULES=1 bash "${WRAPPER}" \
		--modules "${MODULES_CSV}" --version "v2.0.0" \
		-- bash "$0" "$@"
fi

usage() {
	echo "用法: $0 {test|race|vet|tidy}" >&2
	exit 2
}

if [ "$#" -ne 1 ]; then
	usage
fi

ACTION="$1"
case "${ACTION}" in
test | race | vet | tidy)
	;;
*)
	usage
	;;
esac

if [ ! -f "${MANIFEST}" ]; then
	echo "错误: 清单文件不存在: ${MANIFEST}" >&2
	exit 1
fi

export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.11}"

# ============================================================
# 严格 14 个 v2 模块白名单与顺序定义（Bash 3.2 兼容）
# ============================================================
EXACT_14_DIRS="
foundation
application
plugins/database
plugins/cache
plugins/email
plugins/web-gin
plugins/grpc
plugins/cron
plugins/async
plugins/database/adapter/mysql
plugins/database/adapter/postgres
plugins/database/adapter/sqlite
plugins/database/adapter/sqlserver
plugins/database/adapter/oracle
"

is_allowed_dir() {
	target="$1"
	for item in ${EXACT_14_DIRS}; do
		if [ "${item}" = "${target}" ]; then
			return 0
		fi
	done
	return 1
}

# ============================================================
# 阶段 1: 严格预检清单（结构、格式、唯一性、14 目录白名单）
# 任何非法格式或字段错直接 fail-closed，绝不在此时调用任何 go 命令
# ============================================================
seen_dirs=""
seen_mods=""
manifest_count=0

# 保存解析出的有序合法目录清单
PARSED_DIRS=""

while IFS= read -r raw_line || [ -n "${raw_line}" ]; do
	# 去除首尾空白
	line="$(echo "${raw_line}" | awk '{$1=$1};1')"
	case "${line}" in
	\#* | "") continue ;;
	esac

	# 检查列数：严格必须恰好两列 (dir module_path)
	col_count=$(echo "${line}" | awk '{print NF}')
	if [ "${col_count}" -ne 2 ]; then
		echo "错误: 清单行格式非法（列数必须恰好为 2）: '${raw_line}'" >&2
		exit 1
	fi

	dir="$(echo "${line}" | awk '{print $1}')"
	mod="$(echo "${line}" | awk '{print $2}')"

	# 路径防穿越与绝对路径注入
	case "${dir}" in
	*..* | /*)
		echo "错误: 清单路径非法（禁止相对逃逸或绝对路径）: ${dir}" >&2
		exit 1
		;;
	esac

	# 必须严格属于 14 个固定白名单目录
	if ! is_allowed_dir "${dir}"; then
		echo "错误: 目录 '${dir}' 不在允许的 14 个 v2 模块白名单中" >&2
		exit 1
	fi

	# 期望 module path 必须完全匹配 github.com/bamboo-services/bamboo-base-go/${dir}/v2
	expected_mod="github.com/bamboo-services/bamboo-base-go/${dir}/v2"
	if [ "${mod}" != "${expected_mod}" ]; then
		echo "错误: 模块路径不匹配规范: '${mod}'，预期: '${expected_mod}'" >&2
		exit 1
	fi

	# 检查清单内目录重复
	for seen in ${seen_dirs}; do
		if [ "${seen}" = "${dir}" ]; then
			echo "错误: 清单包含重复目录: ${dir}" >&2
			exit 1
		fi
	done

	# 检查清单内 module 重复
	for seen_m in ${seen_mods}; do
		if [ "${seen_m}" = "${mod}" ]; then
			echo "错误: 清单包含重复模块路径: ${mod}" >&2
			exit 1
		fi
	done

	seen_dirs="${seen_dirs} ${dir}"
	seen_mods="${seen_mods} ${mod}"
	manifest_count=$((manifest_count + 1))
	PARSED_DIRS="${PARSED_DIRS} ${dir}"
done <"${MANIFEST}"

# 必须恰好 14 个条目，拒绝重复或缺失
if [ "${manifest_count}" -ne 14 ]; then
	echo "错误: 清单必须恰好包含 14 个独立 v2 模块，实际有效条目: ${manifest_count}" >&2
	exit 1
fi

# 确保白名单中的每一个目录都在清单中出现
for req_dir in ${EXACT_14_DIRS}; do
	found=0
	for parsed in ${seen_dirs}; do
		if [ "${parsed}" = "${req_dir}" ]; then
			found=1
			break
		fi
	done
	if [ "${found}" -eq 0 ]; then
		echo "错误: 缺少必需的 v2 模块目录: ${req_dir}" >&2
		exit 1
	fi
done

# ============================================================
# 阶段 2: 预检所有 14 个模块的真实 go.mod 声明
# 在执行任何实际 go 命令前，全部校验通过才开始执行
# ============================================================
for dir in ${PARSED_DIRS}; do
	target_dir="${REPO_ROOT}/${dir}"
	mod_file="${target_dir}/go.mod"

	if [ ! -d "${target_dir}" ] || [ ! -f "${mod_file}" ]; then
		echo "错误: 模块路径缺失或缺少真实 go.mod: ${dir}" >&2
		exit 1
	fi

	expected_mod="github.com/bamboo-services/bamboo-base-go/${dir}/v2"
	actual_mod="$(awk '$1 == "module" { print $2; exit }' "${mod_file}")"
	if [ "${actual_mod}" != "${expected_mod}" ]; then
		echo "错误: ${dir}/go.mod 声明模块 '${actual_mod}' 与预期 '${expected_mod}' 不符" >&2
		exit 1
	fi
done

# ============================================================
# 阶段 3: 执行命令 (fail-fast)
# test/race/vet 显式加入 -mod=readonly；tidy 允许依赖整理
# ============================================================
idx=0
for dir in ${PARSED_DIRS}; do
	target_dir="${REPO_ROOT}/${dir}"
	expected_mod="github.com/bamboo-services/bamboo-base-go/${dir}/v2"
	idx=$((idx + 1))

	echo "==> [${idx}/14] ${ACTION}: ${dir} (${expected_mod})"

	case "${ACTION}" in
	test)
		(cd "${target_dir}" && go test -mod=readonly -v ./...)
		;;
	race)
		(cd "${target_dir}" && go test -mod=readonly -race -v ./...)
		;;
	vet)
		(cd "${target_dir}" && go vet -mod=readonly ./...)
		;;
	tidy)
		(cd "${target_dir}" && env GOFLAGS="" go mod tidy)
		;;
	esac
done

echo "==> 全部 14 个 v2 模块 ${ACTION} 完成。"

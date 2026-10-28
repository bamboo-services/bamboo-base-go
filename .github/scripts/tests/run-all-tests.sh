#!/usr/bin/env bash
# ==============================================================================
# BambooBase Scripts 自身自测试套件驱动脚本
#
# 功能:
#   一键执行 .github/scripts/ 体系下的所有单元测试与回归测试:
#   1. Go 脚本单元测试 (.github/scripts/proxy 及 .github/scripts/release，启用 -race)
#   2. .github/scripts/tests/ 下全部 Shell 自动化回归测试套件 (test_*.sh)
#
# 特性:
#   - 支持快速失败 (fail-fast)，遇错立即停止并保留错误现场与退出码
#   - 支持可选参数过滤单个或多个指定测试目标
#   - 友好、统一的分步进度与耗时统计
#
# 用法:
#   bash .github/scripts/tests/run-all-tests.sh [options] [target ...]
#   make test-scripts
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
cd "${REPO_ROOT}"

# 终端色彩适配
if [ -t 1 ]; then
	COLOR_RESET="\033[0m"
	COLOR_BOLD="\033[1m"
	COLOR_GREEN="\033[32m"
	COLOR_RED="\033[31m"
	COLOR_BLUE="\033[34m"
	COLOR_CYAN="\033[36m"
else
	COLOR_RESET=""
	COLOR_BOLD=""
	COLOR_GREEN=""
	COLOR_RED=""
	COLOR_BLUE=""
	COLOR_CYAN=""
fi

log_info() {
	printf "%b==>%b %s\n" "${COLOR_BLUE}${COLOR_BOLD}" "${COLOR_RESET}" "$*"
}

log_step() {
	printf "%b[%s]%b %s\n" "${COLOR_CYAN}${COLOR_BOLD}" "$1" "${COLOR_RESET}" "$2"
}

log_pass() {
	printf "%bPASS:%b %s\n" "${COLOR_GREEN}${COLOR_BOLD}" "${COLOR_RESET}" "$*"
}

log_fail() {
	printf "%bFAIL:%b %s\n" "${COLOR_RED}${COLOR_BOLD}" "${COLOR_RESET}" "$*" >&2
}

print_help() {
	cat <<EOF
BambooBase 脚本自测试套件驱动

用法:
  $(basename "$0") [选项] [目标...]

选项:
  -h, --help      显示帮助信息
  -l, --list      列出所有可用的自测试套件目标
  --no-race       禁用 Go 脚本测试的 -race 检测 (默认启用)

目标:
  go              运行 Go 模块代理与发布清单单测 (proxy + release)
  <script_name>   运行指定 test_*.sh 脚本 (支持完整文件名或前缀)

示例:
  $(basename "$0")                     # 运行全部自测试套件
  $(basename "$0") go                  # 仅运行 Go 脚本单测
  $(basename "$0") test_check_go.sh    # 仅运行 check-go 回归测试
EOF
}

# 发现所有可用 shell 测试
discover_shell_tests() {
	find "${SCRIPT_DIR}" -maxdepth 1 -name "test_*.sh" -type f | sort
}

list_targets() {
	echo "可用的自测试套件目标:"
	echo "  go (Go proxy & release manifest 单元测试)"
	discover_shell_tests | while read -r test_file; do
		printf "  %s\n" "$(basename "${test_file}")"
	done
}

ENABLE_RACE=1
TARGETS=()

while [ $# -gt 0 ]; do
	case "$1" in
	-h | --help)
		print_help
		exit 0
		;;
	-l | --list)
		list_targets
		exit 0
		;;
	--no-race)
		ENABLE_RACE=0
		shift
		;;
	*)
		TARGETS+=("$1")
		shift
		;;
	esac
done

START_TIME=$(date +%s)

echo "=================================================================="
echo "BambooBase Scripts Self-Test Suite"
echo "=================================================================="

# 确定待执行的测试项
RUN_GO_TEST=0
SHELL_TESTS=()

ALL_SHELL_TESTS=()
while IFS= read -r f; do
	ALL_SHELL_TESTS+=("$f")
done < <(discover_shell_tests)

if [ ${#TARGETS[@]} -eq 0 ]; then
	RUN_GO_TEST=1
	SHELL_TESTS=("${ALL_SHELL_TESTS[@]}")
else
	for target in "${TARGETS[@]}"; do
		if [ "$target" = "go" ]; then
			RUN_GO_TEST=1
			continue
		fi
		matched=0
		for st in "${ALL_SHELL_TESTS[@]}"; do
			base_st="$(basename "$st")"
			if [ "$base_st" = "$target" ] || [ "${base_st%.sh}" = "$target" ]; then
				SHELL_TESTS+=("$st")
				matched=1
				break
			fi
		done
		if [ $matched -eq 0 ]; then
			log_fail "未识别的测试目标: $target"
			echo "可运行 $(basename "$0") --list 查看全部可用测试。" >&2
			exit 1
		fi
	done
fi

TOTAL_TESTS=0
[ $RUN_GO_TEST -eq 1 ] && TOTAL_TESTS=$((TOTAL_TESTS + 1))
TOTAL_TESTS=$((TOTAL_TESTS + ${#SHELL_TESTS[@]}))

CURRENT_STEP=0
PASSED_COUNT=0

# 1. 运行 Go 脚本单测
if [ $RUN_GO_TEST -eq 1 ]; then
	CURRENT_STEP=$((CURRENT_STEP + 1))
	echo ""
	RACE_ARG=""
	if [ $ENABLE_RACE -eq 1 ]; then
		RACE_ARG="-race"
		log_step "${CURRENT_STEP}/${TOTAL_TESTS}" "运行 Go 脚本单元测试 (.github/scripts/proxy & release, -race)..."
	else
		log_step "${CURRENT_STEP}/${TOTAL_TESTS}" "运行 Go 脚本单元测试 (.github/scripts/proxy & release)..."
	fi

	# Go 脚本测试需在根工作区外使用 GOWORK=off 独立执行
	if GOWORK=off go test ${RACE_ARG} ./.github/scripts/proxy ./.github/scripts/release; then
		log_pass "Go 脚本单测全部通过"
		PASSED_COUNT=$((PASSED_COUNT + 1))
	else
		log_fail "Go 脚本单测失败"
		exit 1
	fi
fi

# 2. 依次运行各个 Shell 测试脚本
if [ ${#SHELL_TESTS[@]} -gt 0 ]; then
	for test_script in "${SHELL_TESTS[@]}"; do
		CURRENT_STEP=$((CURRENT_STEP + 1))
		test_name="$(basename "$test_script")"
		echo ""
		log_step "${CURRENT_STEP}/${TOTAL_TESTS}" "运行回归测试: ${test_name}..."

		if bash "$test_script"; then
			log_pass "${test_name} 执行通过"
			PASSED_COUNT=$((PASSED_COUNT + 1))
		else
			log_fail "${test_name} 执行失败"
			exit 1
		fi
	done
fi

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "=================================================================="
printf "%bSUCCESS:%b 全部 %d 个脚本自测试套件均已顺利通过！(耗时: %ds)\n" \
	"${COLOR_GREEN}${COLOR_BOLD}" "${COLOR_RESET}" "${PASSED_COUNT}" "${DURATION}"
echo "=================================================================="

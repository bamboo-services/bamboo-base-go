#!/usr/bin/env bash
# test_check_go.sh - check-go.sh 回归测试
# 在独立临时目录中模拟 repo、manifest 与 mock go，验证：
# 1. 严格 14 模块合法性校验
# 2. 重复条目、缺失条目、字段错误、相对逃逸、go.mod 不匹配时立即 fail-closed（0 次 go 调用）
# 3. test/race/vet 必须显式携带 -mod=readonly，tidy 允许写入
# 4. 子 go 命令失败时的 fail-fast 退出码传播
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REAL_REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/test-check-go.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' 0 1 2 15

MOCK_BIN="${TMP_DIR}/bin"
MOCK_REPO="${TMP_DIR}/repo"
LOG_FILE="${TMP_DIR}/go_invocations.log"

mkdir -p "${MOCK_BIN}" "${MOCK_REPO}/.github/scripts/ci" "${MOCK_REPO}/.github/scripts/manifest"

# 创建 mock go 二进制
cat <<'EOF' >"${MOCK_BIN}/go"
#!/usr/bin/env bash
if [ -n "${LOG_FILE:-}" ]; then
    echo "PWD=$(pwd) GO $*" >> "${LOG_FILE}"
fi
if [ "${MOCK_GO_FAIL:-0}" = "1" ]; then
    exit 42
fi
exit 0
EOF
chmod +x "${MOCK_BIN}/go"

export PATH="${MOCK_BIN}:${PATH}"
export LOG_FILE

# 复制待测 check-go.sh
cp "${REAL_REPO_ROOT}/.github/scripts/ci/check-go.sh" "${MOCK_REPO}/.github/scripts/ci/check-go.sh"
chmod +x "${MOCK_REPO}/.github/scripts/ci/check-go.sh"

V2_MODULES="
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

setup_valid_repo() {
	rm -rf "${LOG_FILE}"
	: >"${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt"
	for dir in ${V2_MODULES}; do
		mkdir -p "${MOCK_REPO}/${dir}"
		mod="github.com/bamboo-services/bamboo-base-go/${dir}/v2"
		echo "${dir} ${mod}" >>"${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt"
		cat <<EOF >"${MOCK_REPO}/${dir}/go.mod"
module ${mod}

go 1.25.11
EOF
	done
}

# -------------------------------------------------------------
# Case 1: 合法 14 个模块 test 执行
# -------------------------------------------------------------
setup_valid_repo
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh test >/dev/null)
count=$(wc -l <"${LOG_FILE}" | awk '{print $1}')
if [ "${count}" -ne 14 ]; then
	echo "FAIL Case 1: 期望 14 次 go 调用，实际: ${count}" >&2
	exit 1
fi
if ! grep -q "\-mod=readonly" "${LOG_FILE}"; then
	echo "FAIL Case 1: go test 未包含 -mod=readonly" >&2
	exit 1
fi
echo "PASS: Case 1 - 合法 14 模块 test 调用且包含 -mod=readonly"

# -------------------------------------------------------------
# Case 2: 合法 race / vet / tidy 执行参数检查
# -------------------------------------------------------------
# race
rm -f "${LOG_FILE}"
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh race >/dev/null)
if ! grep -q "\-mod=readonly \-race" "${LOG_FILE}"; then
	echo "FAIL Case 2: go test -race 未包含 -mod=readonly -race" >&2
	exit 1
fi

# vet
rm -f "${LOG_FILE}"
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh vet >/dev/null)
if ! grep -q "vet -mod=readonly" "${LOG_FILE}"; then
	echo "FAIL Case 2: go vet 未包含 -mod=readonly" >&2
	exit 1
fi

# tidy
rm -f "${LOG_FILE}"
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh tidy >/dev/null)
if grep -q "\-mod=readonly" "${LOG_FILE}"; then
	echo "FAIL Case 2: go mod tidy 不应包含 -mod=readonly" >&2
	exit 1
fi
echo "PASS: Case 2 - race/vet/tidy 参数验证正确"

# -------------------------------------------------------------
# Case 3: 清单存在重复条目 -> 必须 fail closed，0 次 go 调用
# -------------------------------------------------------------
setup_valid_repo
# 构造包含重复 foundation 的 14 行清单
head -n 13 "${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt" >"${TMP_DIR}/tmp_manifest"
echo "foundation github.com/bamboo-services/bamboo-base-go/foundation/v2" >>"${TMP_DIR}/tmp_manifest"
mv "${TMP_DIR}/tmp_manifest" "${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt"

rm -f "${LOG_FILE}"
set +e
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh test >/dev/null 2>&1)
exit_code=$?
set -e
if [ "${exit_code}" -eq 0 ]; then
	echo "FAIL Case 3: 重复模块清单应当失败，但返回了 0" >&2
	exit 1
fi
if [ -f "${LOG_FILE}" ] && [ -s "${LOG_FILE}" ]; then
	echo "FAIL Case 3: 重复模块必须 fail closed，不应调用 go，实际调用了:" >&2
	cat "${LOG_FILE}" >&2
	exit 1
fi
echo "PASS: Case 3 - 重复条目清单 fail closed（0 次 go 调用）"

# -------------------------------------------------------------
# Case 4: 缺少必需模块 -> 必须 fail closed，0 次 go 调用
# -------------------------------------------------------------
setup_valid_repo
# 删除最后一行 (oracle)
head -n 13 "${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt" >"${TMP_DIR}/tmp_manifest"
mv "${TMP_DIR}/tmp_manifest" "${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt"

rm -f "${LOG_FILE}"
set +e
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh test >/dev/null 2>&1)
exit_code=$?
set -e
if [ "${exit_code}" -eq 0 ]; then
	echo "FAIL Case 4: 缺失模块清单应当失败，但返回了 0" >&2
	exit 1
fi
if [ -f "${LOG_FILE}" ] && [ -s "${LOG_FILE}" ]; then
	echo "FAIL Case 4: 缺失模块必须 fail closed，不应调用 go" >&2
	exit 1
fi
echo "PASS: Case 4 - 缺失模块清单 fail closed（0 次 go 调用）"

# -------------------------------------------------------------
# Case 5: 相对逃逸路径或非法格式 -> fail closed，0 次 go 调用
# -------------------------------------------------------------
setup_valid_repo
echo "../escape github.com/bamboo-services/bamboo-base-go/escape/v2" >>"${MOCK_REPO}/.github/scripts/manifest/v2-modules.txt"

rm -f "${LOG_FILE}"
set +e
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh test >/dev/null 2>&1)
exit_code=$?
set -e
if [ "${exit_code}" -eq 0 ]; then
	echo "FAIL Case 5: 逃逸路径清单应当失败，但返回了 0" >&2
	exit 1
fi
if [ -f "${LOG_FILE}" ] && [ -s "${LOG_FILE}" ]; then
	echo "FAIL Case 5: 逃逸路径必须 fail closed，不应调用 go" >&2
	exit 1
fi
echo "PASS: Case 5 - 逃逸路径 fail closed（0 次 go 调用）"

# -------------------------------------------------------------
# Case 6: 真实 go.mod 声明模块不匹配 -> fail closed，0 次 go 调用
# -------------------------------------------------------------
setup_valid_repo
echo "module github.com/fake/wrong/module" >"${MOCK_REPO}/plugins/cache/go.mod"

rm -f "${LOG_FILE}"
set +e
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh test >/dev/null 2>&1)
exit_code=$?
set -e
if [ "${exit_code}" -eq 0 ]; then
	echo "FAIL Case 6: go.mod 不匹配应当失败，但返回了 0" >&2
	exit 1
fi
if [ -f "${LOG_FILE}" ] && [ -s "${LOG_FILE}" ]; then
	echo "FAIL Case 6: go.mod 不匹配必须 fail closed，不应调用 go" >&2
	exit 1
fi
echo "PASS: Case 6 - go.mod 不匹配 fail closed（0 次 go 调用）"

# -------------------------------------------------------------
# Case 7: 子 go 命令失败退出码传播
# -------------------------------------------------------------
setup_valid_repo
export MOCK_GO_FAIL=1
set +e
(cd "${MOCK_REPO}" && bash .github/scripts/ci/check-go.sh test >/dev/null 2>&1)
exit_code=$?
set -e
export MOCK_GO_FAIL=0
if [ "${exit_code}" -ne 42 ]; then
	echo "FAIL Case 7: 期望子命令退出码 42 传播，实际: ${exit_code}" >&2
	exit 1
fi
echo "PASS: Case 7 - 子 go 命令非 0 退出码正确传播"

echo "ALL 7 CASES PASSED SUCCESSFULLY."

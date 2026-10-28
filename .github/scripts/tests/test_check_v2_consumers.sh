#!/bin/sh
set -eu

# test_check_v2_consumers.sh
# 验证 check-v2-consumers.sh 的 manifest 校验、未知参数报错、快速冒烟

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/../../.." && pwd)
CHECK_SCRIPT="$REPO_ROOT/.github/scripts/consumer/check-v2-consumers.sh"

echo "=== 测试 1: bash 语法检测 (bash -n) ==="
bash -n "$CHECK_SCRIPT"
echo "PASS: check-v2-consumers.sh 语法正确"

echo "=== 测试 2: 参数检查与帮助信息 ==="
"$CHECK_SCRIPT" --help >/dev/null
if "$CHECK_SCRIPT" --unknown-param >/dev/null 2>&1; then
	echo "FAIL: 未知参数应该报错退出" >&2
	exit 1
fi
echo "PASS: 参数检查正常"

TMP_INVALID="$(mktemp -t invalid_manifest.XXXXXX)"
TMP_DIR="$(mktemp -d -t test_manifest.XXXXXX)"
cleanup() {
	rm -f "$TMP_INVALID"
	rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

# 构造 mock go 桩：若清单阶段意外泄露至 generator 则立即报错退出
MOCK_BIN="$TMP_DIR/mock_bin"
mkdir -p "$MOCK_BIN"
cat <<'EOF' >"$MOCK_BIN/go"
#!/bin/sh
echo "FAIL: mock go stub should never be called during manifest validation" >&2
exit 99
EOF
chmod +x "$MOCK_BIN/go"

assert_manifest_fail() {
	desc="$1"
	shift
	set +e
	out="$(PATH="$MOCK_BIN:$PATH" "$CHECK_SCRIPT" "$@" 2>&1)"
	status=$?
	set -e
	if [ "$status" -eq 0 ]; then
		echo "FAIL: $desc 应该报错退出但返回 0" >&2
		exit 1
	fi
	if echo "$out" | grep -q "mock go stub should never be called"; then
		echo "FAIL: $desc 校验泄露进入了 generator/go" >&2
		exit 1
	fi
}

echo "=== 测试 3: manifest 格式异常检测 ==="
echo "invalid/dir github.com/bamboo-services/bamboo-base-go/invalid/dir/v2" >"$TMP_INVALID"
assert_manifest_fail "不存在的目录" --manifest "$TMP_INVALID" --skip-tests --no-isolation
echo "PASS: 非法 manifest 校验拦截正常"

echo "=== 测试 4: 严格 14 模块白名单与缺项/重复/traversal 检查 ==="
# 提取有效模块行（跳过空行与注释）
FIRST_REAL_MODULE="$(awk 'NF && !/^[[:space:]]*#/ { print; exit }' "$REPO_ROOT/.github/scripts/manifest/v2-modules.txt")"

# 测试缺项（严格 14 模块白名单少 1，只有 13 个）
awk 'NF && !/^[[:space:]]*#/' "$REPO_ROOT/.github/scripts/manifest/v2-modules.txt" | head -n 13 >"$TMP_DIR/short_manifest.txt"
assert_manifest_fail "缺少模块(14少1)" --manifest "$TMP_DIR/short_manifest.txt" --skip-tests --no-isolation

# 测试重复项（务必追加第 1 个实际模块行形成重复）
(
	awk 'NF && !/^[[:space:]]*#/' "$REPO_ROOT/.github/scripts/manifest/v2-modules.txt"
	echo "$FIRST_REAL_MODULE"
) >"$TMP_DIR/dup_manifest.txt"
assert_manifest_fail "包含重复项" --manifest "$TMP_DIR/dup_manifest.txt" --skip-tests --no-isolation

# 测试路径穿越
(
	awk 'NF && !/^[[:space:]]*#/' "$REPO_ROOT/.github/scripts/manifest/v2-modules.txt" | head -n 13
	echo "../plugins/cache github.com/bamboo-services/bamboo-base-go/plugins/cache/v2"
) >"$TMP_DIR/traversal_manifest.txt"
assert_manifest_fail "包含目录穿越" --manifest "$TMP_DIR/traversal_manifest.txt" --skip-tests --no-isolation

# 测试未命中白名单的 --module
assert_manifest_fail "未命中白名单的 --module" --module "nonexistent" --skip-tests --no-isolation
echo "PASS: 严格 14 模块白名单与边界校验拦截正常"

echo "=== 测试 5: 单模块 filter 执行且无假绿误导文案 (--skip-tests) ==="
OUT="$("$CHECK_SCRIPT" --module "foundation" --skip-tests --no-isolation)"
if ! echo "$OUT" | grep -q "使用了 --module，未覆盖其余模块"; then
	echo "FAIL: 过滤模式应该显式声明未覆盖其余模块" >&2
	exit 1
fi
echo "PASS: 单模块覆盖提示正确"

echo "=== 测试 6: 候选清单真实 SHA256 提取 (非隔离跳过测试) ==="
OUT="$("$CHECK_SCRIPT" --skip-tests --no-isolation)"
if ! echo "$OUT" | grep -E "ZipSHA256: [a-f0-9]{64}"; then
	echo "FAIL: 未能生成 64 位完整 ZipSHA256" >&2
	exit 1
fi
echo "PASS: 64 位 ZipSHA256 完整证据输出正常"

echo "ALL V2 CONSUMER SCRIPT TESTS PASSED!"

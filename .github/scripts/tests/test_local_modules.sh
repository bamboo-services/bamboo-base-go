#!/bin/sh
set -eu

# test_local_modules.sh
# 验证 with-local-modules.sh 在 Workspace 模式、GOWORK=off 模式、参数安全校验、元文件哈希不变性

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/../../.." && pwd)
WRAPPER="$REPO_ROOT/.github/scripts/with-local-modules.sh"

echo "=== 测试 1: 参数解析与未知 flag / 缺值校验 ==="
if "$WRAPPER" --unknown-flag -- true >/dev/null 2>&1; then
	echo "FAIL: 未知 flag 应该退出并报错" >&2
	exit 1
fi

if "$WRAPPER" --modules >/dev/null 2>&1; then
	echo "FAIL: --modules 缺值应该报错" >&2
	exit 1
fi

if "$WRAPPER" --version >/dev/null 2>&1; then
	echo "FAIL: --version 缺值应该报错" >&2
	exit 1
fi

if "$WRAPPER" -- >/dev/null 2>&1; then
	echo "FAIL: 缺失 command 应该报错" >&2
	exit 1
fi
echo "PASS: 参数检查正常"

echo "=== 测试 2: 隔离性验证——元文件 hash 在执行前后保持绝对一致 ==="
calc_hashes() {
	(
		cd "$REPO_ROOT"
		shasum go.work go.work.sum plugins/database/go.mod plugins/database/go.sum 2>/dev/null || true
	)
}

BEFORE_HASHES="$(calc_hashes)"

# 运行 workspace 模式命令
"$WRAPPER" -- go env GOWORK >/dev/null

# 运行 GOWORK=off 模式命令
(
	cd "$REPO_ROOT/plugins/database"
	GOWORK=off "$WRAPPER" -- go env GOWORK >/dev/null
)

AFTER_HASHES="$(calc_hashes)"

if [ "$BEFORE_HASHES" != "$AFTER_HASHES" ]; then
	echo "FAIL: 源码或元文件哈希被修改!" >&2
	exit 1
fi
echo "PASS: 执行前后 go.work / go.mod / go.sum 哈希完全一致"

echo "=== 测试 3: GOWORK=off 模式下解决 missing sum 正常编译/测试 ==="
(
	cd "$REPO_ROOT/plugins/database"
	GOWORK=off "$WRAPPER" -- go test -run '^$' ./... >/dev/null 2>&1
)
echo "PASS: GOWORK=off 模式下模块正常编译且不报 missing sum"

echo "=== 测试 4: GOPRIVATE / GONOPROXY 继承保护（不绕过 file:// 代理） ==="
# 传入阻断性 GOPRIVATE/GONOPROXY，验证内部仍能解析
(
	cd "$REPO_ROOT/plugins/database"
	GOPRIVATE="github.com/bamboo-services/bamboo-base-go/*,other.corp.internal" \
		GONOPROXY="github.com/bamboo-services/bamboo-base-go/*,other.corp.internal" \
		GOWORK=off "$WRAPPER" -- go test -run '^$' ./... >/dev/null 2>&1
)
echo "PASS: GOPRIVATE/GONOPROXY 剥离保护生效"

echo "=== 测试 5: 广谱祖先模式安全拦截 (fail closed) ==="
if (
	cd "$REPO_ROOT/plugins/database"
	GONOPROXY="github.com/bamboo-services/*" \
		GOWORK=off "$WRAPPER" -- go test -run '^$' ./... >/dev/null 2>&1
); then
	echo "FAIL: 广谱匹配仓库根祖先的模式应当被 fail-closed 拦截" >&2
	exit 1
fi
echo "PASS: 广谱祖先模式被安全拦截"

echo "ALL LOCAL MODULE TESTS PASSED!"

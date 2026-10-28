#!/usr/bin/env bash
# ============================================================
# test_generate_notes.sh
#
# 冒烟测试：在无真实 API Key 的前提下验证
#   generate-release-notes.sh 的完整链路（mock server）。
#
# 覆盖：
#   1. jq 解析路径：mock 生成成功且内容断言
#   2. 缺 API Key 正确报错（不静默失败）
#   3. 无 jq 时降级解析（sed 回退）成功
#
# 用法: bash .github/scripts/tests/test_generate_notes.sh
# ============================================================
set -euo pipefail
# 从 tests/ 上三级回到仓库根（tests → scripts → .github → 根）
cd "$(dirname "$0")/../../.."

echo "🧪 编译并启动 mock server (Go 实现)..."
MOCK_BIN=$(mktemp)
go build -o "$MOCK_BIN" .github/scripts/tests/mock_openai.go
"$MOCK_BIN" 8099 &
MOCK_PID=$!
trap 'kill $MOCK_PID 2>/dev/null || true; rm -f "$MOCK_BIN"' EXIT
sleep 0.5

# 1. jq 解析路径：生成到文件并断言内容
echo "🧪 [1/3] mock 生成 Release Notes（jq 路径）..."
OUT=$(mktemp)
.github/scripts/release/generate-release-notes.sh \
	--base-url http://127.0.0.1:8099/v1 \
	--api-key test-key \
	--from v1.1.2 --to HEAD \
	--output "$OUT" --verbose
if grep -q "Breaking Changes" "$OUT"; then
	echo "✅ 内容含 Breaking Changes"
else
	echo "❌ 生成内容缺少 Breaking Changes"
	exit 1
fi

# 2. 缺 key 错误路径（应报错而非成功）
echo "🧪 [2/3] 缺 API Key 应报错..."
if .github/scripts/release/generate-release-notes.sh --from v1.1.2 >/dev/null 2>&1; then
	echo "❌ 缺 key 应报错却成功"
	exit 1
else
	echo "✅ 缺 key 正确报错"
fi

# 3. 降级解析路径（临时屏蔽 jq，走 sed 回退）
if command -v jq >/dev/null 2>&1; then
	echo "🧪 [3/3] 无 jq 降级解析（sed 回退）..."
	if PATH="/usr/bin:/bin" .github/scripts/release/generate-release-notes.sh \
		--base-url http://127.0.0.1:8099/v1 --api-key test-key \
		--from v1.1.2 --to HEAD | grep -q "Breaking Changes"; then
		echo "✅ 无 jq 回退解析成功"
	else
		echo "❌ 无 jq 回退解析失败"
		exit 1
	fi
else
	echo "⚠️ 本机未安装 jq，跳过降级路径测试"
fi

echo ""
echo "🎉 冒烟测试全部通过"

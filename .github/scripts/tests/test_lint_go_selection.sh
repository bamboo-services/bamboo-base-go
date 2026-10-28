#!/bin/sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH='' cd -- "$script_dir/../../.." && pwd)
script="$repo_root/.github/scripts/ci/lint-go.sh"

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/lint-go-test.XXXXXX")
trap 'rm -rf "$tmpdir"' 0

# 受控 mock linter：满足 version / config verify，并记录调用参数
fake_bin="$tmpdir/bin"
mkdir -p "$fake_bin"
expected_version=$(cat "$repo_root/.golangci-lint-version")
invocations="$tmpdir/invocations.log"
: >"$invocations"

cat <<EOF >"$fake_bin/mock-linter"
#!/bin/sh
set -eu
case "\${1:-}" in
version)
	printf 'golangci-lint has version %s built with go1.27.1\n' "$expected_version"
	exit 0
	;;
config)
	if [ "\${2:-}" = "verify" ]; then
		exit 0
	fi
	;;
run)
	shift
	printf 'RUN in %s args: %s\n' "\$PWD" "\$*" >>"$invocations"
	exit 0
	;;
fmt)
	shift
	printf 'FMT in %s args: %s\n' "\$PWD" "\$*" >>"$invocations"
	exit 0
	;;
esac
exit 0
EOF
chmod +x "$fake_bin/mock-linter"

run_lint() {
	GOLANGCI_LINT="$fake_bin/mock-linter" sh "$script" "$@"
}

echo "=== 测试 1: 显式选单个模块 foundation ==="
: >"$invocations"
run_lint foundation
grep -q "RUN in $repo_root/foundation" "$invocations"
grep -q "FMT in $repo_root/foundation" "$invocations"
test "$(wc -l <"$invocations")" -eq 2
echo "PASS: 单模块 foundation 选择正常"

echo "=== 测试 2: 显式选多个模块 ./foundation application ==="
: >"$invocations"
run_lint ./foundation application
grep -q "RUN in $repo_root/foundation" "$invocations"
grep -q "RUN in $repo_root/application" "$invocations"
test "$(wc -l <"$invocations")" -eq 4
echo "PASS: 多模块选择正常"

echo "=== 测试 3: 重复模块去重 ==="
: >"$invocations"
run_lint foundation ./foundation foundation/
test "$(grep -c "RUN in $repo_root/foundation" "$invocations")" -eq 1
echo "PASS: 模块去重正常"

echo "=== 测试 4: 根目录拒绝作为目标（无 Go 包） ==="
if run_lint . >/dev/null 2>&1; then
	echo "FAIL: 应该拒绝根目录" >&2
	exit 1
fi
echo "PASS: 根目录被拒绝"

echo "=== 测试 5: 不存在或无 go.mod 路径拒绝 ==="
if run_lint non-exist-dir >/dev/null 2>&1; then
	echo "FAIL: 应该拒绝不存在目录" >&2
	exit 1
fi
if run_lint .github >/dev/null 2>&1; then
	echo "FAIL: 应该拒绝无 go.mod 目录" >&2
	exit 1
fi
echo "PASS: 无效/无 go.mod 路径均被拒绝"

echo "=== 测试 6: 越出仓库路径拒绝 ==="
if run_lint ../ >/dev/null 2>&1; then
	echo "FAIL: 应该拒绝越界路径" >&2
	exit 1
fi
echo "PASS: 越界路径被拒绝"

echo "=== 测试 7: 旧 default 模式保持（跳过 oracle） ==="
: >"$invocations"
run_lint default >/dev/null
if grep -q "plugins/database/adapter/oracle" "$invocations"; then
	echo "FAIL: default 模式不应运行 oracle" >&2
	exit 1
fi
grep -q "RUN in $repo_root/foundation" "$invocations"
grep -q "RUN in $repo_root/application" "$invocations"
echo "PASS: default 模式行为一致"

echo "=== 测试 7.1: 无参数默认等同 default 行为 ==="
: >"$invocations"
run_lint >/dev/null
if grep -q "plugins/database/adapter/oracle" "$invocations"; then
	echo "FAIL: 无参数模式不应运行 oracle" >&2
	exit 1
fi
grep -q "RUN in $repo_root/foundation" "$invocations"
grep -q "RUN in $repo_root/application" "$invocations"
echo "PASS: 无参数默认模式行为正常"

echo "=== 测试 7.2: symlink 指向仓库外拒绝 ==="
symlink_out="$repo_root/.symlink_out_test_$$"
outside_dir=$(mktemp -d "${TMPDIR:-/tmp}/outside_mod.XXXXXX")
mkdir -p "$outside_dir"
echo "module outside" >"$outside_dir/go.mod"
ln -s "$outside_dir" "$symlink_out"
trap 'rm -rf "$tmpdir" "$outside_dir" "$symlink_out"' 0
if run_lint "./$(basename "$symlink_out")" >/dev/null 2>&1; then
	echo "FAIL: 指向仓库外的符号链接应该被拒绝" >&2
	rm -rf "$outside_dir" "$symlink_out"
	exit 1
fi
rm -rf "$outside_dir" "$symlink_out"
trap 'rm -rf "$tmpdir"' 0
echo "PASS: 指向仓库外的符号链接被拒绝"

echo "=== 测试 8: 失败传播验证 ==="
cat <<EOF >"$fake_bin/mock-fail-linter"
#!/bin/sh
set -eu
case "\${1:-}" in
version)
	printf 'golangci-lint has version %s built with go1.27.1\n' "$expected_version"
	exit 0
	;;
config) exit 0 ;;
run) exit 1 ;;
fmt) exit 0 ;;
esac
EOF
chmod +x "$fake_bin/mock-fail-linter"

if GOLANGCI_LINT="$fake_bin/mock-fail-linter" sh "$script" foundation >/dev/null 2>&1; then
	echo "FAIL: run 失败未传播" >&2
	exit 1
fi
echo "PASS: lint 失败传播正常"

echo "=== 测试 9: 验证临时 Workspace 正确注入本地精确版本 replace（覆盖块引用与单行引用） ==="
check_script="$tmpdir/check_replace.sh"
cat <<EOF >"$check_script"
#!/bin/sh
set -eu
root="$repo_root"
work_dir=\$(mktemp -d "\${TMPDIR:-/tmp}/bamboo-lint-test.XXXXXX")
trap 'rm -rf "\$work_dir"' 0

# 纯本地临时测试仓库环境，完全无网络：测试 require 块与 require 单行两种声明
test_repo="\$work_dir/repo"
mkdir -p "\$test_repo/mod_a" "\$test_repo/mod_b" "\$test_repo/mod_c"

cat <<'EOM' >"\$test_repo/mod_a/go.mod"
module example.com/moda/v2

go 1.25.0
EOM

# 块引用
cat <<'EOM' >"\$test_repo/mod_b/go.mod"
module example.com/modb

go 1.25.0

require (
	example.com/moda/v2 v2.0.0
)
EOM

# 单行引用
cat <<'EOM' >"\$test_repo/mod_c/go.mod"
module example.com/modc

go 1.25.0

require example.com/moda/v2 v2.0.0
EOM

find "\$test_repo" -name go.mod | LC_ALL=C sort >"\$work_dir/modules"

(cd "\$work_dir" && GOWORK=off go work init) >/dev/null
while IFS= read -r mf; do
	GOWORK="\$work_dir/go.work" go work use "\${mf%/go.mod}"
done <"\$work_dir/modules"

# 执行与 lint-go.sh 相同的 replace 注入逻辑
while IFS= read -r modfile; do
	module_dir="\${modfile%/go.mod}"
	mod_path=\$(go mod edit -json "\$module_dir/go.mod" 2>/dev/null | grep '"Path":' | head -n1 | sed -E 's/.*"Path": "([^"]+)".*/\1/' || true)
	if [ -n "\$mod_path" ]; then
		while IFS= read -r target_modfile; do
			awk -v target="\$mod_path" '
				BEGIN { in_block=0 }
				/^require[ \t]*\(/ { in_block=1; next }
				in_block && /^\)/ { in_block=0; next }
				in_block && \$1 == target && \$2 ~ /^v/ { print \$1 "@" \$2 }
				!in_block && \$1 == "require" && \$2 == target && \$3 ~ /^v/ { print \$2 "@" \$3 }
			' "\$target_modfile"
		done <"\$work_dir/modules" | sort -u | while IFS= read -r req_pair; do
			if [ -n "\$req_pair" ]; then
				go work edit -replace="\$req_pair=\$module_dir" "\$work_dir/go.work"
			fi
		done
	fi
done <"\$work_dir/modules"

# 验证生成的 go.work 包含了对 example.com/moda/v2@v2.0.0 的 replace（go.work 会格式化为模块与版本以空格分隔）
grep -E -q 'replace example.com/moda/v2(@|[[:space:]])v2.0.0 =>' "\$work_dir/go.work"

# 验证真实仓库应用中单行 require foundation/v2 v2.0.0 能被正确生成
(
	cd "\$root/application"
	# 确保在真实 application 模块下执行无网络解析成功
	GOWORK=off GOPROXY=off go mod edit -json go.mod >/dev/null
)
EOF
sh "$check_script"
echo "PASS: 临时 Workspace 单行/块版本精确覆盖与无网络解析验证通过"

echo "ALL TESTS PASSED!"

#!/bin/sh
# 从实际 go.mod 生成临时 Workspace，不依赖被忽略的开发者本地配置。
set -eu

root=$(CDPATH='' cd -- "$(dirname -- "$0")/../../.." && pwd -P)
cd "$root"

usage() {
	printf '用法: %s [default|oracle|all|<module-path>...]\n' "$0" >&2
	exit 2
}

case "${1:-default}" in
default | oracle | all)
	if [ "$#" -gt 1 ]; then
		usage
	fi
	mode=${1:-default}
	;;
*)
	mode=explicit
	;;
esac

linter=${GOLANGCI_LINT:-golangci-lint}
if ! command -v "$linter" >/dev/null 2>&1; then
	printf '未找到 golangci-lint，请安装 v%s（见 README）。\n' "$(cat .golangci-lint-version)" >&2
	exit 2
fi
expected=$(cat .golangci-lint-version)
actual=$("$linter" version)
case "$actual" in
*"version $expected "*) ;;
*)
	printf 'golangci-lint 版本不符：期望 %s，实际 %s\n' "$expected" "$actual" >&2
	exit 2
	;;
esac

"$linter" config verify --config "$root/.golangci.yml"
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/bamboo-lint.XXXXXX")
trap 'rm -rf "$work_dir"' 0
trap 'exit 1' 1 2 15

find . -type d -name .git -prune \
	-o -type d -name node_modules -prune \
	-o -type d -name .pi -prune \
	-o -type d -name .idea -prune \
	-o -type d -name .omo -prune \
	-o -type d -name vendor -prune \
	-o -type f -name go.mod -print | LC_ALL=C sort >"$work_dir/modules"

: >"$work_dir/selected"
if [ "$mode" = "explicit" ]; then
	for arg in "$@"; do
		target_path=${arg%/}
		case "$target_path" in
		/*)
			case "$target_path" in
			"$root"/*) target_path=.${target_path#"$root"} ;;
			*)
				printf '模块路径不能越出仓库: %s\n' "$arg" >&2
				exit 2
				;;
			esac
			;;
		./*) ;;
		*) target_path="./$target_path" ;;
		esac

		target_canon=$(CDPATH='' cd "$root" && cd "$target_path" 2>/dev/null && pwd -P || true)
		if [ -z "$target_canon" ]; then
			printf '指定的模块路径不存在: %s\n' "$arg" >&2
			exit 2
		fi
		case "$target_canon" in
		"$root"/*) ;;
		*)
			printf '模块路径不能越出仓库: %s\n' "$arg" >&2
			exit 2
			;;
		esac

		rel_target=.${target_canon#"$root"}
		if [ "$rel_target" = . ]; then
			printf '根目录仅为占位模块，不包含 Go 代码，不能作为目标: %s\n' "$arg" >&2
			exit 2
		fi

		if [ ! -f "$root/$rel_target/go.mod" ]; then
			printf '指定路径缺少真实 go.mod: %s\n' "$arg" >&2
			exit 2
		fi

		# 去重追加
		if ! grep -Fxq "$rel_target" "$work_dir/selected" 2>/dev/null; then
			printf '%s\n' "$rel_target" >>"$work_dir/selected"
		fi
	done
else
	while IFS= read -r modfile; do
		module=${modfile%/go.mod}
		# 根模块仅作安装锚点，不包含 Go 包，但仍参与内部依赖解析。
		if [ "$module" = . ]; then
			continue
		fi
		is_oracle=false
		case "$module" in
		./plugins/database/oracle | ./plugins/database/adapter/oracle) is_oracle=true ;;
		esac
		case "$mode:$is_oracle" in
		default:true) printf '跳过 Oracle 特殊环境模块：%s（另用 make lint-go-oracle）\n' "$module" ;;
		oracle:false) ;;
		*) printf '%s\n' "$module" >>"$work_dir/selected" ;;
		esac
	done <"$work_dir/modules"
fi

if [ ! -s "$work_dir/selected" ]; then
	printf '没有找到当前范围的 Go 模块，拒绝将空检查计为通过。\n' >&2
	exit 2
fi

# Workspace 包含全部本地模块，避免误用远程旧版内部依赖；只检查 selected 范围。
(
	cd "$work_dir"
	GOWORK=off go work init
)
while IFS= read -r modfile; do
	module=${modfile%/go.mod}
	GOWORK="$work_dir/go.work" go work use "$root/$module"
done <"$work_dir/modules"

# 为本地模块建立带版本的临时 replace，避免 MVS/typecheck 向远端查询未发布 tag（如 foundation/v2 v2.0.0）
while IFS= read -r modfile; do
	module_dir="$root/${modfile%/go.mod}"
	mod_path=$(go mod edit -json "$module_dir/go.mod" 2>/dev/null | grep '"Path":' | head -n1 | sed -E 's/.*"Path": "([^"]+)".*/\1/' || true)
	if [ -n "$mod_path" ]; then
		# 扫描所有模块中对该模块带有具体版本的 require（同时支持 require 块与 require 单行声明），补充临时精确版本 replace
		while IFS= read -r target_modfile; do
			awk -v target="$mod_path" '
				BEGIN { in_block=0 }
				/^require[ \t]*\(/ { in_block=1; next }
				in_block && /^\)/ { in_block=0; next }
				in_block && $1 == target && $2 ~ /^v/ { print $1 "@" $2 }
				!in_block && $1 == "require" && $2 == target && $3 ~ /^v/ { print $2 "@" $3 }
			' "$root/$target_modfile"
		done <"$work_dir/modules" | sort -u | while IFS= read -r req_pair; do
			if [ -n "$req_pair" ]; then
				go work edit -replace="$req_pair=$module_dir" "$work_dir/go.work"
			fi
		done
	fi
done <"$work_dir/modules"

status=0
while IFS= read -r module; do
	printf '\n==> Go lint: %s\n' "$module"
	if ! (cd "$root/$module" && GOWORK="$work_dir/go.work" "$linter" run --config "$root/.golangci.yml" ./...); then
		status=1
	fi
	printf '\n==> Go format check: %s\n' "$module"
	if format_diff=$(cd "$root/$module" && GOWORK="$work_dir/go.work" "$linter" fmt --config "$root/.golangci.yml" --diff ./...); then
		if [ -n "$format_diff" ]; then
			printf '%s\n' "$format_diff"
			status=1
		fi
	else
		status=1
	fi
done <"$work_dir/selected"

exit "$status"

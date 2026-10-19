#!/bin/sh
# 从实际 go.mod 生成临时 Workspace，不依赖被忽略的开发者本地配置。
set -eu

root=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

scope=${1:-default}
case "$scope" in
default | oracle | all) ;;
*)
	printf '用法: %s [default|oracle|all]\n' "$0" >&2
	exit 2
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
	case "$scope:$is_oracle" in
	default:true) printf '跳过 Oracle 特殊环境模块：%s（另用 make lint-go-oracle）\n' "$module" ;;
	oracle:false) ;;
	*) printf '%s\n' "$module" >>"$work_dir/selected" ;;
	esac
done <"$work_dir/modules"

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

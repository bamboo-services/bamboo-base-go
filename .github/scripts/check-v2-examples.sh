#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MANIFEST_FILE="${SCRIPT_DIR}/v2-modules.txt"
WRAPPER="${SCRIPT_DIR}/with-local-modules.sh"
SELF_SCRIPT="${BASH_SOURCE[0]}"

USAGE="Usage: $0 (--example worker-tool|http-db-cache|grpc-cron-async|readme|all | --fixture sqlite-async|http-db-cache) [--race] [--help]"

TARGET_TYPE=""
TARGET_NAME=""
RACE_FLAG=0
INSIDE_MODE=0

while [ "$#" -gt 0 ]; do
	case "$1" in
	--example)
		[ "$#" -lt 2 ] && {
			echo "error: --example requires an argument" >&2
			exit 1
		}
		[ -n "${TARGET_TYPE}" ] && {
			echo "error: multiple targets specified" >&2
			exit 1
		}
		TARGET_TYPE="example"
		TARGET_NAME="$2"
		shift 2
		;;
	--fixture)
		[ "$#" -lt 2 ] && {
			echo "error: --fixture requires an argument" >&2
			exit 1
		}
		[ -n "${TARGET_TYPE}" ] && {
			echo "error: multiple targets specified" >&2
			exit 1
		}
		TARGET_TYPE="fixture"
		TARGET_NAME="$2"
		shift 2
		;;
	--race)
		RACE_FLAG=1
		shift
		;;
	--inside)
		INSIDE_MODE=1
		TARGET_TYPE="$2"
		TARGET_NAME="$3"
		RACE_FLAG="$4"
		shift 4
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

[ -z "${TARGET_TYPE}" ] || [ -z "${TARGET_NAME}" ] && {
	echo "error: missing required --example or --fixture target" >&2
	echo "${USAGE}" >&2
	exit 1
}

if [ "${TARGET_TYPE}:${TARGET_NAME}" = "example:all" ]; then
	for ex in readme worker-tool http-db-cache grpc-cron-async; do
		if [ "${RACE_FLAG}" -eq 1 ]; then
			bash "${SELF_SCRIPT}" --example "${ex}" --race
		else
			bash "${SELF_SCRIPT}" --example "${ex}"
		fi
	done
	exit 0
fi

case "${TARGET_TYPE}:${TARGET_NAME}" in
example:readme)
	SRC_FILE="${REPO_ROOT}/README.md"
	MOD_NAME="example.com/bamboo-consumer-readme"
	;;
example:worker-tool)
	SRC_FILE="${REPO_ROOT}/examples/v2/worker-tool/main.go"
	MOD_NAME="example.com/bamboo-consumer-worker-tool"
	;;
example:http-db-cache)
	SRC_FILE="${REPO_ROOT}/examples/v2/http-db-cache/main.go"
	MOD_NAME="example.com/bamboo-consumer-http-db-cache"
	;;
example:grpc-cron-async)
	SRC_FILE="${REPO_ROOT}/examples/v2/grpc-cron-async/main.go"
	MOD_NAME="example.com/bamboo-consumer-grpc-cron-async"
	;;
fixture:sqlite-async)
	SRC_FILE="${REPO_ROOT}/test/consumer/v2-sqlite-async/lifecycle_test.go"
	MOD_NAME="example.com/bamboo-consumer-v2-sqlite-async"
	TEST_PATTERN='^TestSQLiteAsyncLifecycle'
	;;
fixture:http-db-cache)
	SRC_FILE="${REPO_ROOT}/test/consumer/v2-http-db-cache/lifecycle_test.go"
	MOD_NAME="example.com/bamboo-consumer-v2-http-db-cache"
	TEST_PATTERN='^TestHTTPDBCacheLifecycle'
	;;
example:*)
	echo "error: unknown example: ${TARGET_NAME}" >&2
	exit 1
	;;
fixture:*)
	echo "error: unknown fixture: ${TARGET_NAME}" >&2
	exit 1
	;;
esac

if [ ! -f "${SRC_FILE}" ]; then
	echo "error: target source file not found: ${SRC_FILE}" >&2
	echo "SELECTED ONLY: ${TARGET_TYPE}=${TARGET_NAME} (FAIL CLOSED: missing source file)" >&2
	exit 1
fi

if [ "${INSIDE_MODE}" -eq 0 ]; then
	[ ! -f "${MANIFEST_FILE}" ] && {
		echo "error: missing manifest ${MANIFEST_FILE}" >&2
		exit 1
	}
	MODULES_CSV="$(grep -v '^[[:space:]]*#' "${MANIFEST_FILE}" | grep -v '^[[:space:]]*$' | awk '{print $1}' | paste -sd, -)"
	exec env GOWORK="" bash "${WRAPPER}" \
		--modules "${MODULES_CSV}" --version "v2.0.0" \
		-- bash "${SELF_SCRIPT}" --inside "${TARGET_TYPE}" "${TARGET_NAME}" "${RACE_FLAG}"
fi

TMP_CONSUMER="$(mktemp -d -t bamboo-consumer.XXXXXX)"
cleanup() {
	chmod -R u+w "${TMP_CONSUMER}" 2>/dev/null || true
	rm -rf "${TMP_CONSUMER}"
}
trap cleanup EXIT INT TERM

echo "=== SELECTED ONLY: ${TARGET_TYPE}=${TARGET_NAME} ==="
echo "GOWORK verification: GOWORK=off strictly enforced"

export GOWORK=off
export GOFLAGS=""

mkdir -p "${TMP_CONSUMER}"
cd "${TMP_CONSUMER}"

cat <<EOF >go.mod
module ${MOD_NAME}

go 1.25.0

require (
	github.com/bamboo-services/bamboo-base-go/application/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/foundation/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/async/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/cache/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/cron/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/database/adapter/sqlite/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/database/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/email/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/grpc/v2 v2.0.0
	github.com/bamboo-services/bamboo-base-go/plugins/web-gin/v2 v2.0.0
)
EOF

TEST_FLAGS=("-mod=readonly" "-count=1" "-timeout=40s")
[ "${RACE_FLAG}" -eq 1 ] && TEST_FLAGS+=("-race")

if [ "${TARGET_TYPE}" = "example" ]; then
	if [ "${TARGET_NAME}" = "readme" ]; then
		awk '/```go/{flag=1;next}/```/{flag=0}flag' "${SRC_FILE}" >"${TMP_CONSUMER}/main.go"
		if [ ! -s "${TMP_CONSUMER}/main.go" ]; then
			echo "error: failed to extract Go code block from ${SRC_FILE}" >&2
			exit 1
		fi
	else
		sed '/^\/\/go:build ignore/d' "${SRC_FILE}" >"${TMP_CONSUMER}/main.go"
	fi
	go mod tidy
	go test "${TEST_FLAGS[@]}" ./...
	go vet ./...
	OUT_BIN="${TMP_CONSUMER}/example.bin"
	go build -mod=readonly -o "${OUT_BIN}" ./main.go
	echo "PASS: Successfully built consumer example ${TARGET_NAME} at ${OUT_BIN} (GOWORK=off verified)"
elif [ "${TARGET_TYPE}" = "fixture" ]; then
	cp "${SRC_FILE}" "${TMP_CONSUMER}/lifecycle_test.go"
	go mod tidy
	go test "${TEST_FLAGS[@]}" -run "${TEST_PATTERN}" ./...
	go vet ./...
	echo "PASS: Successfully executed consumer fixture ${TARGET_NAME} (GOWORK=off verified)"
fi

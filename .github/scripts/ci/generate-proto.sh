#!/usr/bin/env bash
set -euo pipefail

# Expected tool versions
REQUIRED_BUF_VERSION="1.73.0"
REQUIRED_PROTOC_GEN_GO_VERSION="v1.36.11"

MODE="generate"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--check)
		MODE="check"
		shift
		;;
	-h | --help)
		echo "Usage: $0 [--check]"
		exit 0
		;;
	*)
		echo "Error: Unknown argument '$1'" >&2
		echo "Usage: $0 [--check]" >&2
		exit 1
		;;
	esac
done

# Check buf binary and version
if ! command -v buf >/dev/null 2>&1; then
	echo "Error: 'buf' binary not found in PATH." >&2
	exit 1
fi

BUF_VER="$(buf --version 2>&1 | head -n 1 | tr -d '\r')"
if [[ "$BUF_VER" != "$REQUIRED_BUF_VERSION" ]]; then
	echo "Error: buf version mismatch. Expected '$REQUIRED_BUF_VERSION', got '$BUF_VER'." >&2
	exit 1
fi

# Check protoc-gen-go binary and version
if ! command -v protoc-gen-go >/dev/null 2>&1; then
	echo "Error: 'protoc-gen-go' binary not found in PATH." >&2
	exit 1
fi

PROTOC_GEN_GO_VER="$(protoc-gen-go --version 2>&1 | awk '{print $2}' | tr -d '\r')"
if [[ "$PROTOC_GEN_GO_VER" != "$REQUIRED_PROTOC_GEN_GO_VERSION" ]]; then
	echo "Error: protoc-gen-go version mismatch. Expected '$REQUIRED_PROTOC_GEN_GO_VERSION', got '$PROTOC_GEN_GO_VER'." >&2
	exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
GRPC_DIR="$REPO_ROOT/plugins/grpc"

if [[ "$MODE" == "check" ]]; then
	TMP_OUT="$(mktemp -d "${TMPDIR:-/tmp}/buf-generate-check.XXXXXX")"
	trap 'rm -rf "$TMP_OUT"' EXIT INT TERM

	(cd "$GRPC_DIR" && buf generate -o "$TMP_OUT")

	# Verify and compare the two generated files
	FILES=(
		"generate/base.pb.go"
		"grpcresult/internaltest/testdata.pb.go"
	)

	for file in "${FILES[@]}"; do
		if [[ ! -f "$GRPC_DIR/$file" ]]; then
			echo "Error: Target file '$GRPC_DIR/$file' does not exist in repository." >&2
			exit 1
		fi
		if [[ ! -f "$TMP_OUT/$file" ]]; then
			echo "Error: Expected generated file '$TMP_OUT/$file' was not produced." >&2
			exit 1
		fi
		if ! cmp -s "$GRPC_DIR/$file" "$TMP_OUT/$file"; then
			echo "Error: File '$file' differs from freshly generated protobuf output." >&2
			diff -u "$GRPC_DIR/$file" "$TMP_OUT/$file" || true
			exit 1
		fi
	done
	echo "Proto check passed: generated files are up to date."
else
	(cd "$GRPC_DIR" && buf generate)
	echo "Proto files generated successfully."
fi

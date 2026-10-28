#!/usr/bin/env bash
set -euo pipefail

# Find repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

USAGE="Usage: $0 [--modules m1,m2] [--version vX.Y.Z] -- <command> [args...]"

if [ "$#" -lt 1 ]; then
	echo "${USAGE}" >&2
	exit 1
fi

MODULES="foundation,application,plugins/database,plugins/cache"
VERSION="v2.0.0"

while [ "$#" -gt 0 ]; do
	case "$1" in
	--modules)
		if [ "$#" -lt 2 ]; then
			echo "error: --modules requires an argument" >&2
			exit 1
		fi
		MODULES="$2"
		shift 2
		;;
	--version)
		if [ "$#" -lt 2 ]; then
			echo "error: --version requires an argument" >&2
			exit 1
		fi
		VERSION="$2"
		shift 2
		;;
	--)
		shift
		break
		;;
	-*)
		echo "error: unknown flag: $1" >&2
		echo "${USAGE}" >&2
		exit 1
		;;
	*)
		break
		;;
	esac
done

if [ "$#" -eq 0 ]; then
	echo "error: no command specified after --" >&2
	exit 1
fi

TMP_DIR="$(mktemp -d -t bamboo-local-modules.XXXXXX)"
TMP_PROXY_DIR="${TMP_DIR}/proxy"
TMP_MODCACHE_DIR="${TMP_DIR}/modcache"
TMP_WORK_DIR="${TMP_DIR}/work"
TMP_OFF_DIR="${TMP_DIR}/gowork-off"

mkdir -p "${TMP_PROXY_DIR}" "${TMP_MODCACHE_DIR}" "${TMP_WORK_DIR}" "${TMP_OFF_DIR}"

cleanup() {
	chmod -R u+w "${TMP_MODCACHE_DIR}" 2>/dev/null || true
	chmod -R u+w "${TMP_DIR}" 2>/dev/null || true
	rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

# Run local-module-proxy generator using explicit Go file
go run "${SCRIPT_DIR}/local-module-proxy.go" \
	-repo-root "${REPO_ROOT}" \
	-out-dir "${TMP_PROXY_DIR}" \
	-modules "${MODULES}" \
	-version "${VERSION}" >&2

# Resolve GOPROXY with file:// URL to local proxy dir, fallback to existing or default public proxy
DETECTED_PUBLIC_PROXY="$(go env GOPROXY 2>/dev/null || true)"
PUBLIC_PROXY="${GOPROXY:-${DETECTED_PUBLIC_PROXY:-https://proxy.golang.org,direct}}"
RESOLVED_GOPROXY="file://${TMP_PROXY_DIR},${PUBLIC_PROXY}"

# GONOPROXY and GOPRIVATE: ensure our repository modules are routed to file:// proxy
# Strip exact repository root and submodule patterns only; do not strip other hosts/repos/orgs.
# If a wide user pattern matches both this repo and other private modules, fail closed to avoid leaking private paths.
strip_repo_prefixes() {
	local input="${1:-}"
	if [ -z "${input}" ]; then
		echo ""
		return
	fi
	local result=""
	local OLDIFS="${IFS}"
	IFS=','
	for part in ${input}; do
		part="$(echo "${part}" | xargs)"
		if [ -z "${part}" ]; then continue; fi
		case "${part}" in
		github.com/bamboo-services/bamboo-base-go | github.com/bamboo-services/bamboo-base-go/*)
			;;
		*)
			local matched_ancestor=0
			for target in "github.com" "github.com/bamboo-services" "github.com/bamboo-services/bamboo-base-go"; do
				case "${target}" in
				${part})
					matched_ancestor=1
					break
					;;
				esac
			done
			if [ "${matched_ancestor}" -eq 1 ]; then
				echo "error: unsafe pattern matches repository root or ancestor prefix; cannot safely split, failing closed" >&2
				exit 1
			fi
			if [ -n "${result}" ]; then
				result="${result},${part}"
			else
				result="${part}"
			fi
			;;
		esac
	done
	IFS="${OLDIFS}"
	echo "${result}"
}

SAFE_GOPRIVATE="$(strip_repo_prefixes "${GOPRIVATE:-}")"
SAFE_GONOPROXY="$(strip_repo_prefixes "${GONOPROXY:-}")"

# Append our prefix specifically to GONOSUMDB so local unreleased candidates do not hit sumdb, preserving user value
RESOLVED_GONOSUMDB="github.com/bamboo-services/bamboo-base-go/*${GONOSUMDB:+,${GONOSUMDB}}"

# Check whether user requested GOWORK=off explicitly in environment
DETECTED_GOWORK="${GOWORK:-}"

if [ "${DETECTED_GOWORK}" = "off" ]; then
	# Mode 2: GOWORK=off mode
	# Run inside an isolated temporary copy of the current module context with its own go.mod/go.sum
	CURRENT_DIR="$(pwd)"
	cp -R "${CURRENT_DIR}/." "${TMP_OFF_DIR}/"

	# Strip stale local candidate checksums from temporary copy so evolving local candidates do not mismatch
	if [ -f "${TMP_OFF_DIR}/go.sum" ]; then
		grep -v "github.com/bamboo-services/bamboo-base-go" "${TMP_OFF_DIR}/go.sum" >"${TMP_OFF_DIR}/go.sum.tmp" 2>/dev/null || true
		mv "${TMP_OFF_DIR}/go.sum.tmp" "${TMP_OFF_DIR}/go.sum"
	fi

	(
		cd "${TMP_OFF_DIR}"
		env \
			GOPROXY="${RESOLVED_GOPROXY}" \
			GOPRIVATE="${SAFE_GOPRIVATE}" \
			GONOPROXY="${SAFE_GONOPROXY}" \
			GONOSUMDB="${RESOLVED_GONOSUMDB}" \
			GOMODCACHE="${TMP_MODCACHE_DIR}" \
			GOWORK="off" \
			go mod tidy -e 2>/dev/null || go mod download 2>/dev/null || true
	)

	(
		cd "${TMP_OFF_DIR}"
		env \
			GOPROXY="${RESOLVED_GOPROXY}" \
			GOPRIVATE="${SAFE_GOPRIVATE}" \
			GONOPROXY="${SAFE_GONOPROXY}" \
			GONOSUMDB="${RESOLVED_GONOSUMDB}" \
			GOMODCACHE="${TMP_MODCACHE_DIR}" \
			GOWORK="off" \
			"$@"
	)
else
	# Mode 1: Workspace mode (default)
	TMP_GOWORK="${TMP_WORK_DIR}/go.work"

	GO_VERSION="1.25.0"
	if [ -f "${REPO_ROOT}/go.work" ]; then
		DETECTED_GO_VER="$(awk '$1 == "go" { print $2; exit }' "${REPO_ROOT}/go.work")"
		if [ -n "${DETECTED_GO_VER}" ]; then
			GO_VERSION="${DETECTED_GO_VER}"
		fi
	fi

	echo "go ${GO_VERSION}" >"${TMP_GOWORK}"
	echo "" >>"${TMP_GOWORK}"
	echo "use (" >>"${TMP_GOWORK}"

	RAW_MODULE_DIRS="${TMP_WORK_DIR}/raw_module_dirs.txt"
	: >"${RAW_MODULE_DIRS}"

	OLDIFS="${IFS}"
	IFS=','
	for m in ${MODULES}; do
		m="$(echo "${m}" | xargs)"
		if [ -n "${m}" ] && [ -f "${REPO_ROOT}/${m}/go.mod" ]; then
			(cd "${REPO_ROOT}/${m}" && pwd -P) >>"${RAW_MODULE_DIRS}"
		fi
	done
	IFS="${OLDIFS}"

	if [ -f "${REPO_ROOT}/go.work" ]; then
		while IFS= read -r line; do
			trimmed="$(echo "${line}" | xargs)"
			if [[ "${trimmed}" == ./* ]]; then
				rel_path="${trimmed#./}"
				if [ -n "${rel_path}" ] && [ -f "${REPO_ROOT}/${rel_path}/go.mod" ]; then
					(cd "${REPO_ROOT}/${rel_path}" && pwd -P) >>"${RAW_MODULE_DIRS}"
				fi
			fi
		done <"${REPO_ROOT}/go.work"
	fi

	if [ -s "${RAW_MODULE_DIRS}" ]; then
		sort -u "${RAW_MODULE_DIRS}" | while IFS= read -r dir_path; do
			if [ -n "${dir_path}" ]; then
				echo "	${dir_path}" >>"${TMP_GOWORK}"
			fi
		done
	fi
	echo ")" >>"${TMP_GOWORK}"

	export GOWORK="${TMP_GOWORK}"
	export GOFLAGS="${GOFLAGS:-} -mod=readonly"

	env \
		GOPROXY="${RESOLVED_GOPROXY}" \
		GOPRIVATE="${SAFE_GOPRIVATE}" \
		GONOPROXY="${SAFE_GONOPROXY}" \
		GONOSUMDB="${RESOLVED_GONOSUMDB}" \
		GOMODCACHE="${TMP_MODCACHE_DIR}" \
		GOWORK="${TMP_GOWORK}" \
		GOFLAGS="${GOFLAGS}" \
		"$@"
fi

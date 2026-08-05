#!/usr/bin/env bash
# ============================================================
# generate-release-notes.sh
#
# 用途：调用 OpenAI 兼容 chat/completions 接口，分析 git 变更
#       （完整 commit 列表 + 按变更量排序的关键 commit diff 摘要），
#       生成中文 GitHub Release Notes。
#
# 特性：
#   - 零第三方依赖：纯 bash + curl（响应解析优先 jq，缺失时 awk/sed 回退）
#   - 提供方可切换：--base-url 覆盖 OpenAI / DeepSeek / Ollama / OpenRouter 等
#   - 双入口复用：供 Makefile 本地发布 与 .github/workflows/release.yml 使用
#   - 兼容 bash 3.2（macOS 系统自带），不使用 4.0+ 特性
#
# 用法见 --help
# ============================================================
set -euo pipefail

# ---------------- 默认值 ----------------
DEFAULT_BASE_URL="https://api.openai.com/v1"
DEFAULT_MODEL="gpt-4o-mini"
DEFAULT_TEMPERATURE="0.3"
DEFAULT_TOP_FILES=20
DEFAULT_MAX_COMMITS=300

# ---------------- 参数变量（环境变量兜底，命令行优先） ----------------
BASE_URL="${RELEASE_AI_BASE_URL:-$DEFAULT_BASE_URL}"
MODEL="${RELEASE_AI_MODEL:-$DEFAULT_MODEL}"
API_KEY="${RELEASE_AI_API_KEY:-}"
FROM=""
TO="HEAD"
OUTPUT="${RELEASE_AI_OUTPUT:-}"
TEMPERATURE="${RELEASE_AI_TEMPERATURE:-$DEFAULT_TEMPERATURE}"
TOP_FILES="$DEFAULT_TOP_FILES"
MAX_COMMITS="$DEFAULT_MAX_COMMITS"
VERBOSE=0

# 提交解析正则（变量形式规避 bash 3.2 [[ =~ ]] 的解析坑）
# 匹配: type(scope)!: description 或 type: description 或非规范消息
CONVENTIONAL_RE='^([a-z]+)(\(([^)]*)\))?(!)?:(.*)$'

# ---------------- 工具函数 ----------------

# 日志输出到 stderr，不污染 stdout（stdout 仅用于 Release Notes 正文）
log()  { printf '\033[90m[generate-notes]\033[0m %s\n' "$*" >&2; }
debug(){ [ "$VERBOSE" = "1" ] && log "🔍 $*" || true; }
die()  { printf '❌ %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
用法: generate-release-notes.sh [选项]

选项 (均可通过同名环境变量提供, 命令行优先):
  --base-url <URL>   API Base URL   (默认 $RELEASE_AI_BASE_URL 或 https://api.openai.com/v1)
  --model <NAME>     模型名         (默认 $RELEASE_AI_MODEL 或 gpt-4o-mini)
  --api-key <KEY>    API Key        (默认 $RELEASE_AI_API_KEY, 必填)
  --from <REF>       起始 tag/ref   (默认自动探测最近 root tag)
  --to <REF>         结束 ref       (默认 HEAD)
  --output <FILE>    输出文件       (默认 stdout)
  --temperature <N>  采样温度       (默认 0.3)
  --top-files <N>    参与分析的 Top 变更 commit 数 (默认 20)
  --max-commits <N>  commit 数量上限 (默认 300)
  --verbose          打印调试信息
  --help             显示帮助

示例:
  # 本地发布（配合 make release）:
  make release VERSION=v1.3.0 AI_MODEL=deepseek-chat AI_API_KEY=sk-xxx

  # 直接调用脚本:
  .github/scripts/generate-release-notes.sh --from v1.2.0 --to HEAD \
      --base-url https://api.deepseek.com/v1 --model deepseek-chat \
      --api-key sk-xxx --output notes.md

  # Ollama 本地模型（--api-key 传任意占位值即可）:
  .github/scripts/generate-release-notes.sh --from v1.2.0 \
      --base-url http://localhost:11434/v1 --model llama3.1 --api-key ollama
EOF
}

# JSON 字符串转义（安全嵌入 prompt 与 payload）
json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  s="${s//$'\b'/\\b}"
  s="${s//$'\f'/\\f}"
  printf '%s' "$s"
}

# JSON 反转义（极简，用于无 jq 的降级路径）
json_unescape() {
  local s="$1"
  s="${s//\\\"/\"}"
  s="${s//\\n/$'\n'}"
  s="${s//\\r/$'\r'}"
  s="${s//\\t/$'\t'}"
  s="${s//\\\\/\\}"
  printf '%s' "$s"
}

# ---------------- 参数解析 ----------------
while [ $# -gt 0 ]; do
  case "$1" in
    --base-url=*)     BASE_URL="${1#*=}"; shift ;;
    --base-url)       [ $# -ge 2 ] || die "--base-url 需要一个值"; BASE_URL="$2"; shift 2 ;;
    --model=*)        MODEL="${1#*=}"; shift ;;
    --model)          [ $# -ge 2 ] || die "--model 需要一个值"; MODEL="$2"; shift 2 ;;
    --api-key=*)      API_KEY="${1#*=}"; shift ;;
    --api-key)        [ $# -ge 2 ] || die "--api-key 需要一个值"; API_KEY="$2"; shift 2 ;;
    --from=*)         FROM="${1#*=}"; shift ;;
    --from)           [ $# -ge 2 ] || die "--from 需要一个值"; FROM="$2"; shift 2 ;;
    --to=*)           TO="${1#*=}"; shift ;;
    --to)             [ $# -ge 2 ] || die "--to 需要一个值"; TO="$2"; shift 2 ;;
    --output=*)       OUTPUT="${1#*=}"; shift ;;
    --output)         [ $# -ge 2 ] || die "--output 需要一个值"; OUTPUT="$2"; shift 2 ;;
    --temperature=*)  TEMPERATURE="${1#*=}"; shift ;;
    --temperature)    [ $# -ge 2 ] || die "--temperature 需要一个值"; TEMPERATURE="$2"; shift 2 ;;
    --top-files=*)    TOP_FILES="${1#*=}"; shift ;;
    --top-files)      [ $# -ge 2 ] || die "--top-files 需要一个值"; TOP_FILES="$2"; shift 2 ;;
    --max-commits=*)  MAX_COMMITS="${1#*=}"; shift ;;
    --max-commits)    [ $# -ge 2 ] || die "--max-commits 需要一个值"; MAX_COMMITS="$2"; shift 2 ;;
    --verbose)        VERBOSE=1; shift ;;
    --help|-h)        usage; exit 0 ;;
    *) die "未知参数: $1 (使用 --help 查看用法)" ;;
  esac
done

# ---------------- 前置校验 ----------------
check_prereq() {
  command -v git  >/dev/null 2>&1 || die "未找到 git, 请先安装"
  command -v curl >/dev/null 2>&1 || die "未找到 curl, 请先安装"
  [ -n "$API_KEY" ] || die "缺少 API Key：请通过 --api-key 或环境变量 RELEASE_AI_API_KEY 提供。
  未配置时请回退使用 gh release create --generate-notes（见 Makefile 的自动回退逻辑）。"
  case "$TEMPERATURE" in
    ''|*[!0-9.]*) die "temperature 必须是数字, 当前: $TEMPERATURE" ;;
  esac
}

# ---------------- 分析逻辑 ----------------

# 收集 FROM..TO 之间符合 Conventional Commits 的提交
# 过滤：机器人 bump commit（chore(release): ...）
# 输出：type<TAB>scope<TAB>breaking<TAB>desc<TAB>hash
collect_commits() {
  local list hash subject
  list=$(git log --no-merges -n "$MAX_COMMITS" --format='%h%x09%s' "$FROM..$TO" 2>/dev/null || true)
  # 用 here-string 逐行读，避免管道 SIGPIPE 与子 shell 变量问题
  while IFS=$'\t' read -r hash subject; do
    [ -n "$hash" ] || continue
    case "$subject" in
      "chore(release):"*) continue ;;   # 过滤机器人 bump commit
    esac
    local type="" scope="" breaking="" desc="$subject"
    if [[ "$subject" =~ $CONVENTIONAL_RE ]]; then
      type="${BASH_REMATCH[1]}"
      scope="${BASH_REMATCH[3]}"
      breaking="${BASH_REMATCH[4]}"
      desc="${BASH_REMATCH[5]# }"
    else
      type="other"
    fi
    printf '%s\t%s\t%s\t%s\t%s\n' "$type" "$scope" "${breaking:-}" "$desc" "$hash"
  done <<<"$list"
}

# 单个 commit 的变更量（新增+删除行数）
commit_volume() {
  git show --numstat --format= "$1" 2>/dev/null | awk '{a+=$1; d+=$2} END{print a+d}' || true
}

# 汇总变更量 Top N 的 commit，并附其变更文件清单
collect_top_diffs() {
  local input="$1" ranked="" top v
  local type scope breaking desc hash
  while IFS=$'\t' read -r type scope breaking desc hash; do
    [ -n "$type" ] || continue
    v=$(commit_volume "$hash")
    [ -z "$v" ] && v=0
    ranked+="$v"$'\t'"$hash"$'\t'"$type${scope:+($scope)}${breaking:+"!"}: $desc"$'\n'
  done <<<"$input"

  # 按变更量降序取 Top N（|| true 规避 head 提前关闭导致的 SIGPIPE + pipefail）
  top=$(printf '%s' "$ranked" | sort -t$'\t' -k1,1 -rn | head -n "$TOP_FILES" || true)

  while IFS=$'\t' read -r vol hash subject; do
    [ -n "$vol" ] || continue
    printf '▶ 变更量 %s | %s | %s\n' "$vol" "$hash" "$subject"
    git show --name-only --format= "$hash" 2>/dev/null | sed '/^$/d' | head -n 15 \
      | sed 's/^/    - /' || true
  done <<<"$top"
}

# 构造 user 输入（完整 commit 列表 + 关键 commit 的 diff 摘要）
build_user_content() {
  local commits_block diffs_block
  commits_block=$(collect_commits)
  diffs_block=$(collect_top_diffs "$commits_block")
  # 用 printf 而非 heredoc，防止 commit subject 中 $/` 被展开
  printf '# 版本范围\n- 上一版本 (from): %s\n- 目标版本 (to):   %s\n\n## 提交列表 (Conventional Commits)\n%s\n\n## 关键变更详情 (按变更量排序 Top %s)\n%s\n' \
    "$FROM" "$TO" "$commits_block" "$TOP_FILES" "$diffs_block"
}

# ---------------- Prompt 设计 ----------------
# 引号定界 heredoc，避免变量展开
SYSTEM_PROMPT=$(cat <<'EOF'
你是 BambooBase (github.com/bamboo-services/bamboo-base-go) 开源基础组件库的发布说明撰写助手。
你擅长将 Conventional Commits 归纳为面向使用者的中文 Release Notes。

## 输出要求
1. 使用简体中文，语气专业、简洁，面向 Go 开发者。
2. 按模块分组：defined / common / major / plugins/*（可精确到 plugins/database/* 等）。
3. 每个模块内按类别组织：
   - 🚀 新功能 (feat)
   - 🛠️ 修复 (fix)
   - ♻️ 重构 (refactor / perf)
   - 📦 其他 (chore / docs / ci / build / test / 未知)
4. 若存在 Breaking Changes（commit 含 ! 或 diff 含破坏性变更），必须在文档开头用 "## ⚠️ Breaking Changes" 醒目列出，并给出简短迁移提示。
5. 只输出 Markdown Release Notes 正文，严禁输出解释性文字、代码块包裹或"以下是生成结果"之类的前言。
6. 每条变更用一句话概括，同一模块同一类别用无序列表，可附 commit hash（短号）便于追溯。
7. 不要编造不存在的变更；未涉及的模块不要强行输出。

## 模块识别
- 提交的 scope 与文件路径前缀是主要依据：defined/ → defined；common/ → common；major/ → major；plugins/** → 对应插件模块。
- 跨模块变更按影响最大的模块归类，无法归类的放 "📦 其他"。
EOF
)

# ---------------- API 调用 ----------------
call_api() {
  local user_content="$1"
  local payload response body http_code curl_rc

  # 用字符串拼接构造 JSON，避免 heredoc 对内容的二次展开
  payload='{"model":"'"$(json_escape "$MODEL")"'","temperature":'$TEMPERATURE',"messages":['
  payload+='{"role":"system","content":"'"$(json_escape "$SYSTEM_PROMPT")"'"},'
  payload+='{"role":"user","content":"'"$(json_escape "$user_content")"'"}]}'

  debug "POST $BASE_URL/chat/completions"
  [ "$VERBOSE" = "1" ] && { printf '==== payload(前 500 字符) ====\n' >&2; printf '%s' "$payload" | head -c 500 >&2; printf '\n' >&2; }

  # 分离 curl 错误与 http_code：stdout=body+code，stderr 丢弃，靠 exit code 判定连接失败
  set +e
  body=$(curl -sS -m 120 -w $'\n%{http_code}' \
      -X POST "$BASE_URL/chat/completions" \
      -H "Authorization: Bearer $API_KEY" \
      -H "Content-Type: application/json" \
      --data "$payload" 2>/dev/null)
  curl_rc=$?
  set -e

  if [ "$curl_rc" -ne 0 ]; then
    die "curl 请求失败 (exit $curl_rc)：$BASE_URL/chat/completions"
  fi
  http_code=$(printf '%s' "$body" | tail -n1)
  body=$(printf '%s' "$body" | sed '$d')

  if [ "$http_code" != "200" ]; then
    printf '❌ API 返回 HTTP %s\n%s\n' "$http_code" "$body" >&2
    exit 1
  fi
  [ "$VERBOSE" = "1" ] && { printf '==== response(前 800 字符) ====\n' >&2; printf '%s' "$body" | head -c 800 >&2; printf '\n' >&2; }
  printf '%s' "$body"
}

# ---------------- 响应解析 ----------------
# 优先 jq，缺失时用极简 sed 提取 choices[0].message.content
extract_content() {
  local json="$1" content=""
  if command -v jq >/dev/null 2>&1; then
    content=$(printf '%s' "$json" | jq -r '.choices[0].message.content // empty' 2>/dev/null || true)
    if [ -n "$content" ]; then
      printf '%s' "$content"
      return 0
    fi
  fi
  # 降级：匹配 message 对象内的第一个 "content":"..."
  content=$(printf '%s' "$json" \
    | sed -n 's/.*"message":{[^}]*"content":"\([^"]*\)".*/\1/p' \
    | head -n1 || true)
  if [ -n "$content" ]; then
    json_unescape "$content"
    return 0
  fi
  die "无法从响应中解析 content（响应前 400 字符）：$(printf '%s' "$json" | head -c 400)"
}

# ---------------- 主流程 ----------------
main() {
  check_prereq

  # 未指定 --from 时自动探测最近 root tag（排除子模块 tag 与 pseudo-version tag）
  if [ -z "$FROM" ]; then
    FROM=$(git describe --tags --abbrev=0 \
        --match 'v[0-9]*.[0-9]*.[0-9]*' \
        --exclude 'v*-*' \
        "${TO}~1" 2>/dev/null || true)
    [ -n "$FROM" ] || die "未找到上一版本 tag，请显式指定 --from"
    log "自动探测上一版本: $FROM"
  fi

  git rev-parse --verify "$FROM" >/dev/null 2>&1 || die "无法解析 --from: $FROM"
  git rev-parse --verify "$TO"   >/dev/null 2>&1 || die "无法解析 --to: $TO"

  if [ "$(git rev-list --count "$FROM..$TO" 2>/dev/null || echo 0)" = "0" ]; then
    log "⚠️ $FROM..$TO 之间没有提交"
    exit 0
  fi

  log "🤖 调用 AI 生成 Release Notes: $FROM → $TO (model=$MODEL)"
  local user_content resp content
  user_content=$(build_user_content)
  resp=$(call_api "$user_content")
  content=$(extract_content "$resp")
  [ -n "$content" ] || die "AI 返回内容为空"

  if [ -n "$OUTPUT" ]; then
    printf '%s\n' "$content" > "$OUTPUT"
    log "✅ 已写入 $OUTPUT"
  else
    printf '%s\n' "$content"
  fi
}

main "$@"

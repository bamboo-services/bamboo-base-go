.DEFAULT_GOAL := help

.PHONY: help proto proto-check check-generated tidy test race vet lint lint-docs format-docs lint-go lint-go-oracle lint-go-all release

# ============================================================
# 基础变量
# ============================================================

GOTOOLCHAIN ?= go1.25.11
export GOTOOLCHAIN

# ============================================================
# 帮助信息
# ============================================================

help:
	@echo "BambooBase - 可用命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make proto                    - 使用 buf 生成 gRPC / Protobuf 代码"
	@echo "  make proto-check              - 校验生成的 gRPC / Protobuf 代码一致性"
	@echo "  make check-generated          - 校验生成的 gRPC / Protobuf 代码一致性 (别名)"
	@echo "  make tidy                     - 整理 14 个 v2 模块依赖"
	@echo "  make test                     - 运行 14 个 v2 模块单元测试"
	@echo "  make race                     - 运行 14 个 v2 模块竞态检测测试"
	@echo "  make vet                      - go vet 检查 14 个 v2 模块"
	@echo "  make lint                     - 文档规范 + 常规 Go 模块检查"
	@echo "  make lint-docs                - Prettier + Markdownlint 检查文档和计划"
	@echo "  make format-docs              - 格式化 docs/（含计划）与根 README.md"
	@echo "  make lint-go                  - Go 正确性、命名/注释与 gofmt 检查"
	@echo "  make lint-go-oracle           - 单独检查 Oracle（需特殊构建环境）"
	@echo "  make lint-go-all              - 包含 Oracle 的全部模块检查"
	@echo ""
	@echo "发布命令:"
	@echo "  make release VERSION=<tag>    - 创建 GitHub Release (触发 Action 给子模块打 tag)"
	@echo "    可选 AI 生成 Release Notes:"
	@echo "      make release VERSION=vX.Y.Z AI_MODEL=<model> AI_API_KEY=<key>"
	@echo "      环境变量: RELEASE_AI_BASE_URL / RELEASE_AI_MODEL / RELEASE_AI_API_KEY"
	@echo "      未配置 AI 时回退 gh --generate-notes"
	@echo ""
	@echo "发布流程:"
	@echo "  1. make release VERSION=vX.Y.Z [AI_MODEL=... AI_API_KEY=...]"
	@echo "     → 有 AI 配置: 调用 .github/scripts/generate-release-notes.sh 生成中文 Notes"
	@echo "     → 无 AI 配置: gh release create --generate-notes"
	@echo "  2. GitHub Action 监听 release published 事件"
	@echo "     → 自动 bump 所有子模块 go.mod 依赖到 vX.Y.Z"
	@echo "     → 按 14 个 v2 独立子模块统一依赖并校验 Release Manifest"
	@echo "     → 为各子模块创建 <module-path>/vX.Y.Z 独立 tag"
	@echo "     → 推送 commit 和 tags 到默认分支"
	@echo ""
	@echo "示例:"
	@echo "  make release VERSION=v2.0.0"
	@echo "  make release VERSION=v2.0.0 AI_MODEL=deepseek-chat AI_API_KEY=sk-xxx"

# ============================================================
# 开发命令
# ============================================================

test:
	bash .github/scripts/check-go.sh test

race:
	bash .github/scripts/check-go.sh race

vet:
	bash .github/scripts/check-go.sh vet

tidy:
	bash .github/scripts/check-go.sh tidy

proto:
	bash .github/scripts/generate-proto.sh

proto-check:
	bash .github/scripts/generate-proto.sh --check

check-generated: proto-check

lint: lint-docs lint-go

lint-docs:
	pnpm run format:docs:check
	pnpm run lint:docs

format-docs:
	pnpm run format:docs

lint-go:
	sh .github/scripts/lint-go.sh default

lint-go-oracle:
	sh .github/scripts/lint-go.sh oracle

lint-go-all:
	sh .github/scripts/lint-go.sh all

# ============================================================
# 发布命令
# ============================================================

# 发布变量（命令行可覆盖，也读取 shell 环境变量）
# AI 生成 Release Notes 的可选配置；未配置 AI_API_KEY 时回退 gh --generate-notes
AI_BASE_URL ?= $(RELEASE_AI_BASE_URL)
AI_MODEL    ?= $(RELEASE_AI_MODEL)
AI_API_KEY  ?= $(RELEASE_AI_API_KEY)
# AI 生成的 Release Notes 临时文件（使用后删除）
RELEASE_NOTES_FILE ?= .release-notes.md

# --- make release VERSION=vX.Y.Z [AI_MODEL=...] [AI_API_KEY=...] ---
# 在 GitHub 上创建 Release，触发 .github/workflows/release.yml
# Action 会自动为每个子模块打 <path>/vX.Y.Z tag 并更新下游 go.mod 依赖
# 提供 AI_API_KEY 时：调用 .github/scripts/generate-release-notes.sh 生成中文 Release Notes
#                   （失败自动回退 --generate-notes）
# 未提供时：回退现有 gh release create --generate-notes
release:
ifndef VERSION
	$(error 请指定版本号: make release VERSION=vX.Y.Z)
endif
	@echo "🚀 创建 GitHub Release: $(VERSION)"
	@command -v gh >/dev/null 2>&1 || { echo "❌ 未找到 gh CLI,请先安装: https://cli.github.com/"; exit 1; }
	@set -e; \
	PREV_TAG=$$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' --exclude 'v*-*' HEAD^ 2>/dev/null || true); \
	NOTES_OPTS="--generate-notes"; \
	if [ -n "$$PREV_TAG" ]; then NOTES_OPTS="--generate-notes --notes-start-tag $$PREV_TAG"; fi; \
	NOTES_FILE="$(RELEASE_NOTES_FILE)"; \
	if [ -n "$(AI_API_KEY)" ]; then \
		echo "   🤖 AI 生成 Release Notes ($(AI_MODEL)): $${PREV_TAG:-<起始>}...$(VERSION)"; \
		if RELEASE_AI_API_KEY="$(AI_API_KEY)" \
		   RELEASE_AI_MODEL="$(AI_MODEL)" \
		   RELEASE_AI_BASE_URL="$(AI_BASE_URL)" \
		   .github/scripts/generate-release-notes.sh \
		     --from "$$PREV_TAG" --to HEAD --output "$$NOTES_FILE"; then \
			NOTES_OPTS="--notes-file $$NOTES_FILE"; \
			echo "   📝 已生成 AI Release Notes: $$NOTES_FILE"; \
		else \
			echo "   ⚠️ AI 生成失败, 回退 GitHub 自动生成"; \
		fi; \
	else \
		echo "   📝 未配置 AI (设置 AI_API_KEY 启用), 使用 GitHub 自动生成"; \
	fi; \
	gh release create "$(VERSION)" --title "$(VERSION)" $$NOTES_OPTS; \
	rm -f "$$NOTES_FILE"; \
	echo "✅ Release $(VERSION) 已创建"; \
	echo "   👀 查看: https://github.com/bamboo-services/bamboo-base-go/releases/tag/$(VERSION)"; \
	echo "   ⏳ GitHub Action 将自动为 14 个子模块打 tag (<module-path>/$(VERSION))"

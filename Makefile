.DEFAULT_GOAL := help

.PHONY: help proto tidy test vet release

# ============================================================
# 基础变量
# ============================================================

# 内部模块列表（非 plugins，用于 vet 遍历）
PACKAGES := major common defined

# 插件列表（用于 vet 遍历）
PLUGINS := cron grpc async email

# ============================================================
# 帮助信息
# ============================================================

help:
	@echo "BambooBase - 可用命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make proto                    - 使用 buf 生成 gRPC 代码"
	@echo "  make tidy                     - 整理 Go 模块依赖"
	@echo "  make test                     - 测试代码"
	@echo "  make vet                      - go vet 检查所有模块"
	@echo ""
	@echo "发布命令:"
	@echo "  make release VERSION=<tag>    - 创建 GitHub Release (触发 Action 给子模块打 tag)"
	@echo ""
	@echo "发布流程:"
	@echo "  1. make release VERSION=vX.Y.Z"
	@echo "     → 调用 gh release create --generate-notes 自动生成 What's Changed"
	@echo "  2. GitHub Action 监听 release published 事件"
	@echo "     → 自动 bump 所有子模块 go.mod 依赖到 vX.Y.Z"
	@echo "     → 按 defined → common → major → plugins/* 顺序打子 tag"
	@echo "     → 推送 commit 和 tags 到默认分支"
	@echo ""
	@echo "示例:"
	@echo "  make release VERSION=v1.2.0"

# ============================================================
# 开发命令
# ============================================================

test:
	go test -v ./...

proto:
	buf generate

tidy:
	go mod tidy

vet:
	@for pkg in $(PACKAGES); do \
		go vet ./$$pkg/...; \
	done
	@for plg in $(PLUGINS); do \
		go vet ./plugins/$$plg/...; \
	done

# ============================================================
# 发布命令
# ============================================================

# --- make release VERSION=vX.Y.Z ---
# 在 GitHub 上创建 Release，触发 .github/workflows/release.yml
# Action 会自动为每个子模块打 <path>/vX.Y.Z tag 并更新下游 go.mod 依赖
release:
ifndef VERSION
	$(error 请指定版本号: make release VERSION=vX.Y.Z)
endif
	@echo "🚀 创建 GitHub Release: $(VERSION)"
	@command -v gh >/dev/null 2>&1 || { echo "❌ 未找到 gh CLI,请先安装: https://cli.github.com/"; exit 1; }
	@# 获取上一个 tag 作为 notes 起始点（用于生成 What's Changed）
	@$(eval PREV_TAG := $(shell git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo ""))
	@if [ -n "$(PREV_TAG)" ]; then \
		echo "   📝 生成 Release Notes 范围: $(PREV_TAG)...$(VERSION)"; \
		gh release create "$(VERSION)" \
			--title "$(VERSION)" \
			--generate-notes \
			--notes-start-tag "$(PREV_TAG)"; \
	else \
		echo "   📝 首个 Release,生成全部 Release Notes"; \
		gh release create "$(VERSION)" \
			--title "$(VERSION)" \
			--generate-notes; \
	fi
	@echo "✅ Release $(VERSION) 已创建"
	@echo "   👀 查看: https://github.com/bamboo-services/bamboo-base-go/releases/tag/$(VERSION)"
	@echo "   ⏳ GitHub Action 将自动为子模块打 tag (defined/common/major/plugins/*/$(VERSION))"

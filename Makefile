.DEFAULT_GOAL := help

.PHONY: help proto tidy test release release-plugins release-all

# ============================================================
# 基础变量
# ============================================================

# 根版本号（去除 v 前缀，如 v1 → 1）
ROOT_VERSION := $(shell cat version | sed 's/^v//')

# 时间戳（格式：YYYYMMDDHHMM）
TIMESTAMP := $(shell date +"%Y%m%d%H%M")

# 内部模块列表（非 plugins）
PACKAGES := major utility defined

# 插件列表
PLUGINS := cron grpc

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
	@echo ""
	@echo "发布命令:"
	@echo "  make release PKG=<name>       - 发布指定模块 (major|utility|defined)"
	@echo "  make release-plugins PLG=<name> - 发布指定插件 (cron|grpc)"
	@echo "  make release-all              - 发布全部模块和插件"
	@echo ""
	@echo "版本格式: v{version}-{YYYYMMDDHHMM}"
	@echo "  根版本号:   $(ROOT_VERSION)  (来自 ./version)"
	@echo "  时间戳:     $(TIMESTAMP)"
	@echo ""
	@echo "示例:"
	@echo "  make release PKG=major        → major/v$(ROOT_VERSION).x.x-$(TIMESTAMP)"
	@echo "  make release-plugins PLG=grpc → plugins/grpc/v$(ROOT_VERSION).x.x-$(TIMESTAMP)"

# ============================================================
# 开发命令
# ============================================================

test:
	go test -v ./...

proto:
	buf generate

tidy:
	go mod tidy

# ============================================================
# 发布命令
# ============================================================

# 构建 tag 名称的函数
# $(1): 模块目录路径 (如 major, plugins/cron)
# 返回: <path>/v<ROOT_VERSION>.<SUB_VERSION>-<TIMESTAMP>
define build_tag
$(strip $(1))/v$(ROOT_VERSION).$(shell cat $(1)/version)-$(TIMESTAMP)
endef

# --- make release PKG=<name> ---
# 发布指定模块（major / utility / defined）
release:
ifndef PKG
	$(error 请指定模块名称: make release PKG=<major|utility|defined>)
endif
ifeq ($(filter $(PKG),$(PACKAGES)),)
	$(error 无效的模块名称 "$(PKG)",可选值: $(PACKAGES))
endif
	@$(eval TAG := $(call build_tag,$(PKG)))
	@echo "📦 发布模块: $(PKG)"
	@echo "   tag: $(TAG)"
	@git tag -a "$(TAG)" -m "Release $(TAG)"
	@git push origin "$(TAG)"
	@echo "✅ $(PKG) 发布完成: $(TAG)"

# --- make release-plugins PLG=<name> ---
# 发布指定插件（cron / grpc）
release-plugins:
ifndef PLG
	$(error 请指定插件名称: make release-plugins PLG=<cron|grpc>)
endif
ifeq ($(filter $(PLG),$(PLUGINS)),)
	$(error 无效的插件名称 "$(PLG)",可选值: $(PLUGINS))
endif
	@$(eval TAG := $(call build_tag,plugins/$(PLG)))
	@echo "🔌 发布插件: $(PLG)"
	@echo "   tag: $(TAG)"
	@git tag -a "$(TAG)" -m "Release $(TAG)"
	@git push origin "$(TAG)"
	@echo "✅ plugins/$(PLG) 发布完成: $(TAG)"

# --- make release-all ---
# 发布全部模块和插件
release-all:
	@echo "🚀 发布全部模块和插件"
	@echo "   时间戳: $(TIMESTAMP)"
	@echo ""
	@for pkg in $(PACKAGES); do \
		tag="$$pkg/v$(ROOT_VERSION).$$(cat $$pkg/version)-$(TIMESTAMP)"; \
		echo "📦 [$$pkg] → $$tag"; \
		git tag -a "$$tag" -m "Release $$tag"; \
		echo "   ✅ tag 创建成功"; \
	done
	@for plg in $(PLUGINS); do \
		tag="plugins/$$plg/v$(ROOT_VERSION).$$(cat plugins/$$plg/version)-$(TIMESTAMP)"; \
		echo "🔌 [plugins/$$plg] → $$tag"; \
		git tag -a "$$tag" -m "Release $$tag"; \
		echo "   ✅ tag 创建成功"; \
	done
	@echo ""
	@echo "推送所有 tag 到远程仓库..."
	@git push --tags
	@echo ""
	@echo "🎉 全部发布完成！"

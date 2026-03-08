.DEFAULT_GOAL := help

.PHONY: help proto tidy test release release-plugins release-all vet

# ============================================================
# 基础变量
# ============================================================

# 根版本号（去除 v 前缀，如 v1 → 1）
ROOT_VERSION := $(shell cat version | sed 's/^v//')

# 时间戳（格式：YYYYMMDDHHMM）
TIMESTAMP := $(shell date +"%Y%m%d%H%M")

# 内部模块列表（非 plugins）
PACKAGES := major common defined

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
	@echo "  make release PKG=<name>       - 发布指定模块 (major|common|defined)"
	@echo "  make release-plugins PLG=<name> - 发布指定插件 (cron|grpc)"
	@echo "  make release-all              - 按依赖顺序发布全部模块和插件"
	@echo ""
	@echo "依赖顺序: defined → common → major, plugins/cron, plugins/grpc"
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

vet:
	@for pkg in $(PACKAGES); do \
		tag="$$pkg"; \
		go vet ./$$tag/...; \
	done
	@for plg in $(PLUGINS); do \
		tag="plugins/$$plg"; \
		go vet ./$$tag/...; \
	done

# ============================================================
# 发布命令
# ============================================================

# 构建 tag 名称的函数
# $(1): 模块目录路径 (如 major, plugins/cron)
# 返回: <path>/v<ROOT_VERSION>.<SUB_VERSION>-<TIMESTAMP>
define build_tag
$(strip $(1))/v$(ROOT_VERSION).$(shell cat $(1)/version)-$(TIMESTAMP)
endef

# 更新下游模块依赖的函数
# $(1): 已发布的模块名 (如 defined, common)
# $(2): 新版本 tag (如 defined/v1.0.0-202603081200)
define update_dep
	@echo "   🔄 更新下游模块的 $(1) 依赖版本..."
	@if [ "$(1)" = "defined" ]; then \
		for mod in common major plugins/grpc; do \
			if [ -f "$$mod/go.mod" ]; then \
				sed -i '' 's|github.com/bamboo-services/bamboo-base-go/defined v[0-9].*|github.com/bamboo-services/bamboo-base-go/defined $(2)|g' $$mod/go.mod; \
				echo "      ✅ $$mod/go.mod"; \
			fi; \
		done; \
	elif [ "$(1)" = "common" ]; then \
		for mod in major plugins/cron plugins/grpc; do \
			if [ -f "$$mod/go.mod" ]; then \
				sed -i '' 's|github.com/bamboo-services/bamboo-base-go/common v[0-9].*|github.com/bamboo-services/bamboo-base-go/common $(2)|g' $$mod/go.mod; \
				echo "      ✅ $$mod/go.mod"; \
			fi; \
		done; \
	fi
endef

# --- make release PKG=<name> ---
# 发布指定模块（major / common / defined）
release:
ifndef PKG
	$(error 请指定模块名称: make release PKG=<major|common|defined>)
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
# 按依赖顺序发布全部模块和插件，并自动更新下游依赖
# 依赖链: defined → common → major, plugins/cron → common, plugins/grpc → defined + common
# 发布顺序: defined → common → major → plugins/cron → plugins/grpc
release-all:
	@echo "🚀 按依赖顺序发布全部模块和插件"
	@echo "   时间戳: $(TIMESTAMP)"
	@echo "   顺序: defined → common → major → plugins/cron → plugins/grpc"
	@echo ""
	@# 1. 发布 defined (无依赖，最底层)
	@$(eval TAG_DEFINED := $(call build_tag,defined))
	@echo "📦 [1/5] 发布 defined"
	@echo "   tag: $(TAG_DEFINED)"
	@git tag -a "$(TAG_DEFINED)" -m "Release $(TAG_DEFINED)"
	@git push origin "$(TAG_DEFINED)"
	@echo "   ✅ defined 发布完成"
	$(call update_dep,defined,$(TAG_DEFINED))
	@echo ""
	@# 2. 发布 common (依赖 defined)
	@$(eval TAG_COMMON := $(call build_tag,common))
	@echo "📦 [2/5] 发布 common"
	@echo "   tag: $(TAG_COMMON)"
	@git tag -a "$(TAG_COMMON)" -m "Release $(TAG_COMMON)"
	@git push origin "$(TAG_COMMON)"
	@echo "   ✅ common 发布完成"
	$(call update_dep,common,$(TAG_COMMON))
	@echo ""
	@# 3. 发布 major (依赖 common, defined)
	@$(eval TAG_MAJOR := $(call build_tag,major))
	@echo "📦 [3/5] 发布 major"
	@echo "   tag: $(TAG_MAJOR)"
	@git tag -a "$(TAG_MAJOR)" -m "Release $(TAG_MAJOR)"
	@git push origin "$(TAG_MAJOR)"
	@echo "   ✅ major 发布完成"
	@echo ""
	@# 4. 发布 plugins/cron (依赖 common)
	@$(eval TAG_CRON := $(call build_tag,plugins/cron))
	@echo "🔌 [4/5] 发布 plugins/cron"
	@echo "   tag: $(TAG_CRON)"
	@git tag -a "$(TAG_CRON)" -m "Release $(TAG_CRON)"
	@git push origin "$(TAG_CRON)"
	@echo "   ✅ plugins/cron 发布完成"
	@echo ""
	@# 5. 发布 plugins/grpc (依赖 defined + common)
	@$(eval TAG_GRPC := $(call build_tag,plugins/grpc))
	@echo "🔌 [5/5] 发布 plugins/grpc"
	@echo "   tag: $(TAG_GRPC)"
	@git tag -a "$(TAG_GRPC)" -m "Release $(TAG_GRPC)"
	@git push origin "$(TAG_GRPC)"
	@echo "   ✅ plugins/grpc 发布完成"
	@echo ""
	@echo "🎉 全部发布完成！"

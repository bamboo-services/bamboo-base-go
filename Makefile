.DEFAULT_GOAL := help

.PHONY: help proto tidy tag tag-upload release test update-plugins

# 版本文件路径
VERSION_FILE := version

# 获取版本号（去除 v 前缀）
VERSION := $(shell cat $(VERSION_FILE) | sed 's/^v//')

# 获取当前时间戳（格式：YYYYMMDDHHMM）
TIMESTAMP := $(shell date +"%Y%m%d%H%M")

# 完整 tag 名称
TAG_NAME := v$(VERSION)-$(TIMESTAMP)

# 主模块路径
MAIN_MODULE := github.com/bamboo-services/bamboo-base-go

# 插件目录列表
PLUGIN_DIRS := plugins/grpc plugins/cron

# 显示帮助信息
help:
	@echo "BambooBase - 可用命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make proto      	- 使用 buf 生成 gRPC 代码"
	@echo "  make tidy       	- 整理 Go 模块依赖"
	@echo "  make test       	- 测试代码"
	@echo ""
	@echo "发布命令:"
	@echo "  make tag        	- 创建带有时间戳的 tag（不推送）"
	@echo "                   	  格式: v{version}-{YYYYMMDDHHMM}"
	@echo "                   	  示例: v1.0.0-202602191755"
	@echo "  make tag-upload 	- 单独上传 tag"
	@echo "  make release    	- 更新插件版本、创建 tag 并推送到远程仓库"

test:
	go test -v ./...

proto:
	buf generate

tidy:
	go mod tidy

# 更新插件依赖的主模块版本号（不运行 go mod tidy，避免远程版本不存在问题）
update-plugins:
	@echo "更新插件依赖版本为: $(TAG_NAME)"
	@for dir in $(PLUGIN_DIRS); do \
		if [ -f "$$dir/go.mod" ]; then \
			sed -i '' 's|$(MAIN_MODULE) v[0-9].*|$(MAIN_MODULE) $(TAG_NAME)|g' $$dir/go.mod; \
			echo "  ✓ 更新 $$dir/go.mod"; \
		fi; \
	done
	@echo "✅ 插件版本更新完成"

# 创建 tag（仅本地）
tag:
	@echo "创建 tag: $(TAG_NAME)"
	git tag -a $(TAG_NAME) -m "Release $(TAG_NAME)"
	@echo "✅ Tag $(TAG_NAME) 创建成功"

tag-upload:
	@echo "推送 tag 到远程仓库..."
	git push --tags
	@echo "✅ Tag 推送成功！"

# 发布流程：更新插件版本 → 提交 → 创建 tag → 推送
release: test update-plugins
	@echo "提交插件版本更新..."
	git add plugins/*/go.mod
	git commit -m "chore: 更新插件依赖版本为 $(TAG_NAME)" || echo "没有需要提交的更改"
	$(MAKE) tag
	$(MAKE) tag-upload
	@echo ""
	@echo "🎉 发布完成！"
	@echo "   主模块: $(TAG_NAME)"
	@echo "   插件已更新依赖版本"

PROTO_FILE ?= ./proto/normal_upload.proto

.DEFAULT_GOAL := help

.PHONY: help swag proto run dev tag tag-upload release

# 版本文件路径
VERSION_FILE := version

# 获取版本号（去除 v 前缀）
VERSION := $(shell cat $(VERSION_FILE) | sed 's/^v//')

# 获取当前时间戳（格式：YYYYMMDDHHMM）
TIMESTAMP := $(shell date +"%Y%m%d%H%M")

# 完整 tag 名称
TAG_NAME := v$(VERSION)-$(TIMESTAMP)

# 显示帮助信息
help:
	@echo "BambooBase - 可用命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make proto      	- 生成指定 proto 的 gRPC 代码"
	@echo "                   	  示例: make proto PROTO_FILE=./proto/normal_upload.proto"
	@echo "  make tidy       	- 整理 Go 模块依赖"
	@echo ""
	@echo "发布命令:"
	@echo "  make tag        	- 创建带有时间戳的 tag（不推送）"
	@echo "                   	  格式: v{version}-{YYYYMMDDHHMM}"
	@echo "                   	  示例: v1.0.0-202602191755"
	@echo "  make tag-upload 	- 单独上传 tag"
	@echo "  make release    	- 创建 tag 并推送到远程仓库"

proto:
	protoc --go_out=. --go-grpc_out=. $(PROTO_FILE)

tidy:
	go mod tidy

# 创建 tag（仅本地）
tag:
	@echo "创建 tag: $(TAG_NAME)"
	git tag -a $(TAG_NAME) -m "Release $(TAG_NAME)"
	@echo "✅ Tag $(TAG_NAME) 创建成功"

tag-upload:
	@echo "推送 tag 到远程仓库..."
	git push --tags
	@echo "✅ Tag $(TAG_NAME) 推送成功！"

# 创建 tag 并推送
release: tag tag-upload

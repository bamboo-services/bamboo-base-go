PROTO_FILE ?= ./proto/normal_upload.proto

.DEFAULT_GOAL := help

.PHONY: help swag proto run dev

# 显示帮助信息
help:
	@echo "锋楪YggLeaf - 可用命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make proto      - 生成指定 proto 的 gRPC 代码"
	@echo "                   示例: make proto PROTO_FILE=./proto/normal_upload.proto"
	@echo "  make tidy	   - 整理 Go 模块依赖"

proto:
	protoc --go_out=. --go-grpc_out=. $(PROTO_FILE)

tidy:
	go mod tidy

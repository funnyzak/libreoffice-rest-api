BINARY_NAME=libreoffice-rest-api
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

CMD_DIR=./cmd/server
BUILD_DIR=build
DIST_DIR=dist

PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 linux/s390x linux/riscv64 linux/arm linux/ppc64le

.PHONY: help build build-all package release-prep test test-coverage coverage clean install deps lint fmt vet check run dev swagger race init-config mocks version

.DEFAULT_GOAL := help

help: ## 显示帮助信息
	@echo "libreoffice-rest-api 构建工具"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## 下载并整理依赖
	@echo "下载依赖..."
	go mod download
	go mod tidy

fmt: ## 格式化代码
	@echo "格式化代码..."
	go fmt ./...

vet: ## 运行 go vet 静态分析
	@echo "运行 go vet..."
	go vet ./...

lint: ## 运行 golangci-lint 代码检查
	@echo "运行代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过代码检查"; \
	fi

build: ## 构建当前平台二进制
	@echo "构建 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-all: ## 交叉编译所有平台
	@echo "交叉编译所有平台..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "构建 $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(DIST_DIR)/$$output_name $(CMD_DIR); \
	done

package: build-all ## 生成发布包
	@echo "生成发布包..."
	@mkdir -p $(DIST_DIR)/packages
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		package_name=$(BINARY_NAME)-$(VERSION)-$$os-$$arch.tar.gz; \
		echo "打包 $$package_name..."; \
		tar -czf $(DIST_DIR)/packages/$$package_name -C $(DIST_DIR) $$output_name; \
	done
	@echo "发布包输出路径: $(DIST_DIR)/packages/"

test: ## 运行测试
	@echo "运行测试..."
	go test -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "生成测试覆盖率..."
	@mkdir -p coverage
	go test -v -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "覆盖率报告路径: coverage/coverage.html"

coverage: test-coverage ## 覆盖率报告别名

install: ## 安装到 GOPATH/bin
	@echo "安装 $(BINARY_NAME)..."
	go install $(LDFLAGS) $(CMD_DIR)

run: build ## 构建并运行
	@echo "启动 $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) --config config.yaml

dev: ## 开发模式运行（支持热重载，需要安装 air）
	@if command -v air >/dev/null 2>&1; then \
		echo "使用 air 启动热重载开发模式..."; \
		air; \
	else \
		echo "air 未安装，使用普通模式运行（无热重载）"; \
		echo "安装 air: go install github.com/air-verse/air@latest"; \
		go run $(CMD_DIR) --config config.yaml; \
	fi

swagger: ## 生成 Swagger 文档（需要安装 swag）
	swag init -g cmd/server/main.go -o docs

race: ## 运行竞态条件检测测试
	go test -race ./...

clean: ## 清理构建产物和临时文件
	@echo "清理构建产物..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) coverage coverage.out

init-config: ## 初始化配置文件（从示例文件复制）
	cp config.yaml.example config.yaml

mocks: ## 生成 Mock 文件（暂无可生成的 Mock）
	@echo "暂无可生成的 Mock"

check: fmt lint test ## 运行格式化、检查与测试
	@echo "检查完成"

release-prep: clean deps check build-all ## 发布准备（清理、依赖、检查、全平台构建）
	@echo "发布准备完成"
	@echo "构建产物路径: $(DIST_DIR)/"

version: ## 显示版本信息
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_DATE)"

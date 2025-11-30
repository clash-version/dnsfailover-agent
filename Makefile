# DNS Failover Agent Makefile
# 支持跨平台编译

# 变量定义
APP_NAME = dnsfailover
VERSION = 0.1.1
BUILD_DIR = build
DIST_DIR = $(BUILD_DIR)/dist

# 检测操作系统
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    RM := cmd /C del /F /Q
    RMDIR := cmd /C rmdir /S /Q
    MKDIR := cmd /C if not exist
    MKDIR_END := mkdir
    COPY := cmd /C copy
    EXE_EXT := .exe
    PATH_SEP := \\
else
    DETECTED_OS := $(shell uname -s)
    RM := rm -f
    RMDIR := rm -rf
    MKDIR := mkdir -p
    MKDIR_END :=
    COPY := cp
    EXE_EXT :=
    PATH_SEP := /
endif

# 默认目标平台（可以通过 make GOOS=linux 覆盖）
GOOS ?= linux
GOARCH ?= amd64

# 根据目标平台设置二进制文件名
ifeq ($(GOOS),windows)
    BIN_NAME = $(APP_NAME).exe
else
    BIN_NAME = $(APP_NAME)
endif

# Go 编译参数
GO = go
LDFLAGS = -s -w -X main.Version=$(VERSION)

# 默认目标
.PHONY: all
all: clean build package

# 清理构建目录
.PHONY: clean
clean:
	@echo "清理构建目录..."
ifeq ($(DETECTED_OS),Windows)
	@if exist $(BUILD_DIR) $(RMDIR) $(BUILD_DIR)
	@if exist $(BIN_NAME) $(RM) $(BIN_NAME)
else
	@$(RMDIR) $(BUILD_DIR) $(BIN_NAME) 2>/dev/null || true
endif
	@echo "清理完成"

# 编译程序（Linux 版本）
.PHONY: build
build:
	@echo "开始编译 $(APP_NAME) v$(VERSION) for $(GOOS)/$(GOARCH)..."
ifeq ($(DETECTED_OS),Windows)
	@$(MKDIR) $(BUILD_DIR) $(MKDIR_END)
else
	@$(MKDIR) $(BUILD_DIR)
endif
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BIN_NAME) .
	@echo "编译完成: $(BUILD_DIR)/$(BIN_NAME)"

# 编译 Windows 版本
.PHONY: build-windows
build-windows:
	@echo "编译 Windows 版本..."
	@$(MAKE) build GOOS=windows GOARCH=amd64

# 编译 Linux 版本
.PHONY: build-linux
build-linux:
	@echo "编译 Linux 版本..."
	@$(MAKE) build GOOS=linux GOARCH=amd64

# 编译 ARM 版本
.PHONY: build-arm
build-arm:
	@echo "编译 ARM64 版本..."
	@$(MAKE) build GOOS=linux GOARCH=arm64

# 编译所有平台
.PHONY: build-all
build-all: clean
	@echo "编译所有平台版本..."
	@$(MAKE) build-linux
	@mv $(BUILD_DIR)/$(APP_NAME) $(BUILD_DIR)/$(APP_NAME)-linux-amd64
	@$(MAKE) build-arm
	@mv $(BUILD_DIR)/$(APP_NAME) $(BUILD_DIR)/$(APP_NAME)-linux-arm64
	@$(MAKE) build-windows
	@echo "所有平台编译完成"

# 打包发布文件（Linux 版本）
.PHONY: package
package:
	@echo "打包 Linux 发布文件..."
ifeq ($(DETECTED_OS),Windows)
	@$(MKDIR) $(DIST_DIR) $(MKDIR_END)
	@$(MKDIR) $(DIST_DIR)$(PATH_SEP)logs $(MKDIR_END)
else
	@$(MKDIR) $(DIST_DIR)
	@$(MKDIR) $(DIST_DIR)/logs
endif
	
	@echo "复制可执行文件..."
	@$(COPY) $(BUILD_DIR)$(PATH_SEP)$(BIN_NAME) $(DIST_DIR)$(PATH_SEP)$(BIN_NAME)
	
	@echo "复制配置文件..."
ifeq ($(DETECTED_OS),Windows)
	@if exist config.example.json $(COPY) config.example.json $(DIST_DIR)$(PATH_SEP)config.example.json
	@if exist build$(PATH_SEP)config.example.json $(COPY) build$(PATH_SEP)config.example.json $(DIST_DIR)$(PATH_SEP)config.example.json
else
	@[ -f config.example.json ] && $(COPY) config.example.json $(DIST_DIR)/ || true
	@[ -f build/config.example.json ] && $(COPY) build/config.example.json $(DIST_DIR)/ || true
endif
	
	@echo "复制脚本文件..."
ifeq ($(GOOS),linux)
ifeq ($(DETECTED_OS),Windows)
	@if exist scripts$(PATH_SEP)install.sh $(COPY) scripts$(PATH_SEP)install.sh $(DIST_DIR)$(PATH_SEP)install.sh
	@if exist scripts$(PATH_SEP)quick-install.sh $(COPY) scripts$(PATH_SEP)quick-install.sh $(DIST_DIR)$(PATH_SEP)quick-install.sh
	@if exist scripts$(PATH_SEP)start.sh $(COPY) scripts$(PATH_SEP)start.sh $(DIST_DIR)$(PATH_SEP)start.sh
	@if exist scripts$(PATH_SEP)stop.sh $(COPY) scripts$(PATH_SEP)stop.sh $(DIST_DIR)$(PATH_SEP)stop.sh
else
	@[ -f scripts/install.sh ] && $(COPY) scripts/install.sh $(DIST_DIR)/ && chmod +x $(DIST_DIR)/install.sh || true
	@[ -f scripts/quick-install.sh ] && $(COPY) scripts/quick-install.sh $(DIST_DIR)/ && chmod +x $(DIST_DIR)/quick-install.sh || true
	@[ -f scripts/start.sh ] && $(COPY) scripts/start.sh $(DIST_DIR)/ && chmod +x $(DIST_DIR)/start.sh || true
	@[ -f scripts/stop.sh ] && $(COPY) scripts/stop.sh $(DIST_DIR)/ && chmod +x $(DIST_DIR)/stop.sh || true
endif
endif
	
	@echo "复制文档..."
ifeq ($(DETECTED_OS),Windows)
	@if exist README.md $(COPY) README.md $(DIST_DIR)$(PATH_SEP)README.md
	@if exist build$(PATH_SEP)README.md $(COPY) build$(PATH_SEP)README.md $(DIST_DIR)$(PATH_SEP)README.md
	@if exist build$(PATH_SEP)INSTALL.md $(COPY) build$(PATH_SEP)INSTALL.md $(DIST_DIR)$(PATH_SEP)INSTALL.md
else
	@[ -f README.md ] && $(COPY) README.md $(DIST_DIR)/ || true
	@[ -f build/README.md ] && $(COPY) build/README.md $(DIST_DIR)/ || true
	@[ -f build/INSTALL.md ] && $(COPY) build/INSTALL.md $(DIST_DIR)/ || true
endif
	
	@echo "打包完成: $(DIST_DIR)"

# 打包 Windows 版本
.PHONY: package-windows
package-windows:
	@echo "打包 Windows 发布文件..."
	@$(MAKE) build-windows
	@$(MAKE) package GOOS=windows
	@echo "创建 Windows 启动脚本..."
ifeq ($(DETECTED_OS),Windows)
	@echo @echo off > $(DIST_DIR)$(PATH_SEP)start.bat
	@echo $(APP_NAME).exe monitor start >> $(DIST_DIR)$(PATH_SEP)start.bat
else
	@echo "@echo off" > $(DIST_DIR)/start.bat
	@echo "$(APP_NAME).exe monitor start" >> $(DIST_DIR)/start.bat
endif

# 打包 Linux 版本
.PHONY: package-linux
package-linux:
	@$(MAKE) build-linux
	@$(MAKE) package GOOS=linux

# 创建发布压缩包
.PHONY: release
release: package
	@echo "创建发布压缩包..."
ifeq ($(GOOS),windows)
	@cd $(BUILD_DIR) && tar -czf $(APP_NAME)-v$(VERSION)-windows-amd64.tar.gz dist || echo "请手动压缩 dist 目录"
else
	@cd $(BUILD_DIR) && tar -czf $(APP_NAME)-v$(VERSION)-$(GOOS)-$(GOARCH).tar.gz dist
endif
	@echo "发布包已创建: $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"

# 创建所有平台的发布包
.PHONY: release-all
release-all:
	@echo "创建所有平台的发布包..."
	@$(MAKE) clean
	@$(MAKE) release GOOS=linux GOARCH=amd64
	@$(MAKE) clean
	@$(MAKE) release GOOS=linux GOARCH=arm64
	@$(MAKE) clean
	@$(MAKE) release GOOS=windows GOARCH=amd64
	@echo "所有发布包创建完成"

# 快速构建（不清理）
.PHONY: quick
quick:
	@echo "快速编译 $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BIN_NAME) .
	@echo "完成: $(BUILD_DIR)/$(BIN_NAME)"

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 运行测试（带覆盖率）
.PHONY: test-coverage
test-coverage:
	@echo "运行测试（带覆盖率）..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告: coverage.html"

# 代码检查
.PHONY: lint
lint:
	@echo "运行代码检查..."
	@which golangci-lint > /dev/null || (echo "请先安装 golangci-lint" && exit 1)
	golangci-lint run ./...

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...
	@echo "代码格式化完成"

# 安装依赖
.PHONY: deps
deps:
	@echo "安装依赖..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "依赖安装完成"

# 本地运行（开发模式）
.PHONY: run
run:
	@echo "运行程序（开发模式）..."
	$(GO) run . monitor start

# 显示版本信息
.PHONY: version
version:
	@echo "DNS Failover Agent"
	@echo "版本: $(VERSION)"
	@echo "目标平台: $(GOOS)/$(GOARCH)"
	@echo "检测到的系统: $(DETECTED_OS)"

# 显示帮助信息
.PHONY: help
help:
	@echo "DNS Failover Agent 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  make                  - 完整构建（清理 + 编译 + 打包）"
	@echo "  make build            - 编译程序（默认 Linux/amd64）"
	@echo "  make build-linux      - 编译 Linux 版本"
	@echo "  make build-windows    - 编译 Windows 版本"
	@echo "  make build-arm        - 编译 ARM64 版本"
	@echo "  make build-all        - 编译所有平台"
	@echo "  make package          - 打包发布文件"
	@echo "  make package-linux    - 打包 Linux 版本"
	@echo "  make package-windows  - 打包 Windows 版本"
	@echo "  make release          - 创建发布压缩包"
	@echo "  make release-all      - 创建所有平台的发布包"
	@echo "  make clean            - 清理构建目录"
	@echo "  make quick            - 快速编译（不清理）"
	@echo "  make test             - 运行测试"
	@echo "  make test-coverage    - 运行测试（带覆盖率）"
	@echo "  make lint             - 代码检查"
	@echo "  make fmt              - 格式化代码"
	@echo "  make deps             - 安装依赖"
	@echo "  make run              - 本地运行（开发模式）"
	@echo "  make version          - 显示版本信息"
	@echo "  make help             - 显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make GOOS=linux GOARCH=amd64    - 编译 Linux x64 版本"
	@echo "  make GOOS=windows GOARCH=amd64  - 编译 Windows x64 版本"
	@echo "  make GOOS=linux GOARCH=arm64    - 编译 Linux ARM64 版本"


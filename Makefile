# DNS Failover Agent Makefile
# 适用于 Windows 环境

# 变量定义
APP_NAME = dnsfailover
VERSION = 0.1.1
BUILD_DIR = build
DIST_DIR = $(BUILD_DIR)/dist
BIN_NAME = $(APP_NAME).exe

# Go 编译参数
GO = go
GOOS = windows
GOARCH = amd64
LDFLAGS = -s -w -X main.Version=$(VERSION)

# 默认目标
.PHONY: all
all: clean build package

# 清理构建目录
.PHONY: clean
clean:
	@echo "清理构建目录..."
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)
	@if exist $(BIN_NAME) del /f /q $(BIN_NAME)
	@echo "清理完成"

# 编译程序
.PHONY: build
build:
	@echo "开始编译 $(APP_NAME) v$(VERSION)..."
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BIN_NAME) .
	@echo "编译完成: $(BUILD_DIR)/$(BIN_NAME)"

# 打包发布文件
.PHONY: package
package:
	@echo "打包发布文件..."
	@if not exist $(DIST_DIR) mkdir $(DIST_DIR)
	
	@REM 复制可执行文件
	@copy $(BUILD_DIR)\$(BIN_NAME) $(DIST_DIR)\$(BIN_NAME)
	
	@REM 生成默认配置文件
	@$(BUILD_DIR)\$(BIN_NAME) init -c $(DIST_DIR)\config.json
	
	@REM 创建日志目录
	@if not exist $(DIST_DIR)\logs mkdir $(DIST_DIR)\logs
	
	@REM 复制文档文件
	@copy README.md $(DIST_DIR)\README.md 2>nul || echo. > $(DIST_DIR)\README.md
	@copy CHANGELOG-v$(VERSION).md $(DIST_DIR)\CHANGELOG.md 2>nul || echo. > $(DIST_DIR)\CHANGELOG.md
	
	@REM 创建安装脚本
	@echo 创建安装脚本...
	@echo @echo off > $(DIST_DIR)\install.bat
	@echo echo ========================================== >> $(DIST_DIR)\install.bat
	@echo echo DNS Failover Agent v$(VERSION) 安装程序 >> $(DIST_DIR)\install.bat
	@echo echo ========================================== >> $(DIST_DIR)\install.bat
	@echo echo. >> $(DIST_DIR)\install.bat
	@echo if not exist config.json ( >> $(DIST_DIR)\install.bat
	@echo     echo 生成默认配置文件... >> $(DIST_DIR)\install.bat
	@echo     $(BIN_NAME) init >> $(DIST_DIR)\install.bat
	@echo     echo 请编辑 config.json 填入 Cloudflare API Token >> $(DIST_DIR)\install.bat
	@echo ) >> $(DIST_DIR)\install.bat
	@echo echo. >> $(DIST_DIR)\install.bat
	@echo echo 安装完成！ >> $(DIST_DIR)\install.bat
	@echo echo 使用方法： >> $(DIST_DIR)\install.bat
	@echo echo   1. 编辑 config.json 配置文件 >> $(DIST_DIR)\install.bat
	@echo echo   2. 运行: $(BIN_NAME) monitor start >> $(DIST_DIR)\install.bat
	@echo echo. >> $(DIST_DIR)\install.bat
	@echo pause >> $(DIST_DIR)\install.bat
	
	@REM 创建启动脚本
	@echo @echo off > $(DIST_DIR)\start.bat
	@echo $(BIN_NAME) monitor start >> $(DIST_DIR)\start.bat
	
	@REM 创建停止脚本
	@echo @echo off > $(DIST_DIR)\stop.bat
	@echo $(BIN_NAME) monitor stop >> $(DIST_DIR)\stop.bat
	
	@echo 打包完成: $(DIST_DIR)

# 创建发布压缩包
.PHONY: release
release: all
	@echo "创建发布压缩包..."
	@cd $(BUILD_DIR) && tar -czf $(APP_NAME)-v$(VERSION)-windows-amd64.tar.gz dist
	@echo "发布包已创建: $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-windows-amd64.tar.gz"

# 快速构建（不清理）
.PHONY: quick
quick:
	@echo "快速编译..."
	$(GO) build -o $(BUILD_DIR)/$(BIN_NAME) .
	@echo "完成: $(BUILD_DIR)/$(BIN_NAME)"

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 安装依赖
.PHONY: deps
deps:
	@echo "安装依赖..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "依赖安装完成"

# 显示帮助信息
.PHONY: help
help:
	@echo DNS Failover Agent 构建工具
	@echo.
	@echo 可用命令:
	@echo   make          - 完整构建（清理 + 编译 + 打包）
	@echo   make build    - 仅编译程序
	@echo   make package  - 打包发布文件
	@echo   make release  - 创建发布压缩包
	@echo   make clean    - 清理构建目录
	@echo   make quick    - 快速编译（不清理）
	@echo   make test     - 运行测试
	@echo   make deps     - 安装依赖
	@echo   make help     - 显示此帮助信息

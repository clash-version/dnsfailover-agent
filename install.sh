#!/bin/bash

# DNS Failover Agent 一键安装脚本
# 使用方法: curl -fsSL https://raw.githubusercontent.com/clash-version/n8n-agent/main/install.sh | bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 配置
REPO="clash-version/dnsfailover-agent"
BINARY_NAME="dnsfailover"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/dnsfailover"
LOG_DIR="/var/log/dnsfailover"
SERVICE_NAME="dnsfailover"

info() {
    echo -e "${GREEN}[INFO] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[WARN] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
    exit 1
}

# 检查 root 权限
if [ "$(id -u)" != "0" ]; then
    error "此脚本需要 root 权限运行。请使用 sudo su 或 sudo bash 运行。"
fi

# 检测系统架构
detect_arch() {
    local arch
    arch=$(uname -m)
    case $arch in
        x86_64) echo "amd64" ;;
        aarch64) echo "arm64" ;;
        i386|i686) echo "386" ;;
        *) error "不支持的架构: $arch" ;;
    esac
}

# 检测操作系统
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    if [ "$os" != "linux" ]; then
        error "此脚本仅支持 Linux 系统。当前系统: $os"
    fi
    echo "$os"
}

ARCH=$(detect_arch)
OS=$(detect_os)
PLATFORM="${OS}-${ARCH}"

info "检测到系统: $OS, 架构: $ARCH"

# 获取最新版本下载链接
get_download_url() {
    local url
    # 尝试从 GitHub API 获取最新 Release
    local api_url="https://api.github.com/repos/$REPO/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        url=$(curl -s $api_url | grep "browser_download_url" | grep "$PLATFORM.tar.gz" | head -n 1 | cut -d '"' -f 4)
    elif command -v wget >/dev/null 2>&1; then
        url=$(wget -qO- $api_url | grep "browser_download_url" | grep "$PLATFORM.tar.gz" | head -n 1 | cut -d '"' -f 4)
    fi

    if [ -z "$url" ]; then
        # 如果 API 失败（可能是 API 限制），尝试直接构造 URL
        # 注意：这里假设通过 tag 下载，如果 tag 不确定可以提示用户手动下载
        warn "无法通过 API 获取最新版本，尝试推断 URL..."
        # 既然没有版本号，无法构造确切 URL，这里只能报错提示
        error "无法获取最新版本下载链接。可能是 GitHub API 限制限制或网络问题。"
    fi
    echo "$url"
}

# 安装过程
install() {
    info "正在获取最新版本下载链接..."
    DOWNLOAD_URL=$(get_download_url)
    info "下载地址: $DOWNLOAD_URL"

    TMP_DIR=$(mktemp -d)
    FILE_NAME="dnsfailover.tar.gz"
    
    info "正在下载..."
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$TMP_DIR/$FILE_NAME" "$DOWNLOAD_URL"
    else
        wget -O "$TMP_DIR/$FILE_NAME" "$DOWNLOAD_URL"
    fi

    info "正在解压..."
    tar -xzf "$TMP_DIR/$FILE_NAME" -C "$TMP_DIR"
    
    # 查找二进制文件（解压后的文件名可能是 dnsfailover-linux-amd64）
    EXTRACTED_BIN=$(find "$TMP_DIR" -type f -name "dnsfailover-*" | head -n 1)
    if [ -z "$EXTRACTED_BIN" ]; then
        error "解压失败或未找到二进制文件"
    fi

    # 停止现有服务
    if systemctl is-active --quiet $SERVICE_NAME; then
        warn "停止现有服务..."
        systemctl stop $SERVICE_NAME
    fi

    # 移动二进制文件
    info "安装二进制文件到 $INSTALL_DIR/$BINARY_NAME..."
    mv "$EXTRACTED_BIN" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # 清理
    rm -rf "$TMP_DIR"

    # 创建目录
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"

    # 创建 Systemd 服务文件
    info "配置 Systemd 服务..."
    cat > /etc/systemd/system/$SERVICE_NAME.service <<EOF
[Unit]
Description=DNS Failover Agent Monitoring Service
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME monitor start --web --port 8080
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/service.log
StandardError=append:$LOG_DIR/service.log
Environment="DB_PATH=$CONFIG_DIR/probe.db"
Environment="LOG_PATH=$LOG_DIR/probe.log"
# Environment="WEBHOOK_URL="

[Install]
WantedBy=multi-user.target
EOF

    # 重新加载 Systemd
    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME

    info "安装完成！"
    info "服务状态:"
    systemctl status $SERVICE_NAME --no-pager
    
    echo
    echo -e "${GREEN}Web 管理面板已启动: http://<你的IP>:8080${NC}"
    echo -e "配置文件目录: $CONFIG_DIR"
    echo -e "日志文件目录: $LOG_DIR"
    echo -e "使用 systemctl status $SERVICE_NAME 查看运行状态"
}

# 执行安装
install

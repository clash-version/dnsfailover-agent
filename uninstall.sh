#!/usr/bin/env bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 检查权限
if [ "$(id -u)" -ne 0 ]; then
    printf "${RED}错误: 请使用 root 权限运行此脚本${NC}\n"
    # printf "使用: sudo bash $0\n"
    exit 1
fi

SERVICE_NAME="dnsfailover"
BINARY_PATH="/usr/local/bin/dnsfailover"
CONFIG_DIR="/etc/dnsfailover"
LOG_DIR="/var/log/dnsfailover"

printf "${YELLOW}===== DNS Failover Agent 卸载脚本 =====${NC}\n"

# 1. 停止服务
if systemctl is-active --quiet $SERVICE_NAME; then
    printf "${GREEN}停止服务...${NC}\n"
    systemctl stop $SERVICE_NAME
    printf "✓ 服务已停止\n"
fi

# 2. 禁用服务
if systemctl is-enabled --quiet $SERVICE_NAME 2>/dev/null; then
    printf "${GREEN}禁用开机自启...${NC}\n"
    systemctl disable $SERVICE_NAME
    printf "✓ 已禁用开机自启\n"
fi

# 3. 删除服务文件
if [ -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
    printf "${GREEN}删除服务文件...${NC}\n"
    rm -f "/etc/systemd/system/$SERVICE_NAME.service"
    printf "✓ 服务文件已删除\n"
fi

# 4. 重新加载 systemd
printf "${GREEN}重新加载 systemd...${NC}\n"
systemctl daemon-reload
systemctl reset-failed

# 5. 删除二进制文件
if [ -f "$BINARY_PATH" ]; then
    printf "${GREEN}删除二进制文件...${NC}\n"
    rm -f "$BINARY_PATH"
    printf "✓ 二进制文件已删除\n"
fi

# 6. 询问是否删除数据
printf "\n"
printf "是否删除配置文件和日志? (y/N): "
read -r REPLY
if [[ "$REPLY" =~ ^[Yy]$ ]]; then
    if [ -d "$CONFIG_DIR" ]; then
        printf "${GREEN}删除配置目录...${NC}\n"
        rm -rf "$CONFIG_DIR"
        printf "✓ 配置目录已删除\n"
    fi
    if [ -d "$LOG_DIR" ]; then
        printf "${GREEN}删除日志目录...${NC}\n"
        rm -rf "$LOG_DIR"
        printf "✓ 日志目录已删除\n"
    fi
else
    printf "${YELLOW}保留配置文件和日志${NC}\n"
    printf "配置目录: $CONFIG_DIR\n"
    printf "日志目录: $LOG_DIR\n"
fi

printf "\n"
printf "${GREEN}✓ 卸载完成！${NC}\n"

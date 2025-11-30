#!/usr/bin/env bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查是否以 root 身份运行
if [ "$(id -u)" -ne 0 ]; then
    printf "${RED}错误: 请使用 root 权限运行此脚本${NC}\n"
    printf "使用: sudo bash $0\n"
    exit 1
fi

printf "${YELLOW}===== DNS Failover Agent 卸载脚本 =====${NC}\n"

# 1. 停止服务
if systemctl is-active --quiet dnsfailover; then
    printf "${GREEN}停止服务...${NC}\n"
    systemctl stop dnsfailover
    printf "✓ 服务已停止\n"
fi

# 2. 禁用服务
if systemctl is-enabled --quiet dnsfailover 2>/dev/null; then
    printf "${GREEN}禁用开机自启...${NC}\n"
    systemctl disable dnsfailover
    printf "✓ 已禁用开机自启\n"
fi

# 3. 删除服务文件
if [ -f /etc/systemd/system/dnsfailover.service ]; then
    printf "${GREEN}删除服务文件...${NC}\n"
    rm -f /etc/systemd/system/dnsfailover.service
    printf "✓ 服务文件已删除\n"
fi

# 4. 重新加载 systemd
printf "${GREEN}重新加载 systemd...${NC}\n"
systemctl daemon-reload
systemctl reset-failed

# 5. 删除二进制文件
if [ -f /usr/local/bin/dnsfailover ]; then
    printf "${GREEN}删除二进制文件...${NC}\n"
    rm -f /usr/local/bin/dnsfailover
    printf "✓ 二进制文件已删除\n"
fi

# 6. 询问是否删除配置文件
printf "\n"
printf "是否删除配置文件和日志? (y/N): "
read -r REPLY
if [ "$REPLY" = "y" ] || [ "$REPLY" = "Y" ]; then
    if [ -d /etc/dnsfailover ]; then
        printf "${GREEN}删除配置目录...${NC}\n"
        rm -rf /etc/dnsfailover
        printf "✓ 配置目录已删除\n"
    fi
    if [ -d /var/log/dnsfailover ]; then
        printf "${GREEN}删除日志目录...${NC}\n"
        rm -rf /var/log/dnsfailover
        printf "✓ 日志目录已删除\n"
    fi
else
    printf "${YELLOW}保留配置文件和日志${NC}\n"
    printf "配置目录: /etc/dnsfailover\n"
    printf "日志目录: /var/log/dnsfailover\n"
fi

printf "\n"
printf "${GREEN}✓ DNS Failover Agent 已成功卸载！${NC}\n"

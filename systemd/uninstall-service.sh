#!/bin/bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查是否以 root 身份运行
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}错误: 请使用 root 权限运行此脚本${NC}"
    echo "使用: sudo $0"
    exit 1
fi

echo -e "${YELLOW}===== DNS Failover Agent 卸载脚本 =====${NC}"

# 1. 停止服务
if systemctl is-active --quiet dnsfailover; then
    echo -e "${GREEN}停止服务...${NC}"
    systemctl stop dnsfailover
    echo "✓ 服务已停止"
fi

# 2. 禁用服务
if systemctl is-enabled --quiet dnsfailover 2>/dev/null; then
    echo -e "${GREEN}禁用开机自启...${NC}"
    systemctl disable dnsfailover
    echo "✓ 已禁用开机自启"
fi

# 3. 删除服务文件
if [ -f /etc/systemd/system/dnsfailover.service ]; then
    echo -e "${GREEN}删除服务文件...${NC}"
    rm -f /etc/systemd/system/dnsfailover.service
    echo "✓ 服务文件已删除"
fi

# 4. 重新加载 systemd
echo -e "${GREEN}重新加载 systemd...${NC}"
systemctl daemon-reload
systemctl reset-failed

# 5. 删除二进制文件
if [ -f /usr/local/bin/dnsfailover ]; then
    echo -e "${GREEN}删除二进制文件...${NC}"
    rm -f /usr/local/bin/dnsfailover
    echo "✓ 二进制文件已删除"
fi

# 6. 询问是否删除配置文件
echo ""
read -p "是否删除配置文件和日志? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -d /etc/dnsfailover ]; then
        echo -e "${GREEN}删除配置目录...${NC}"
        rm -rf /etc/dnsfailover
        echo "✓ 配置目录已删除"
    fi
    if [ -d /var/log/dnsfailover ]; then
        echo -e "${GREEN}删除日志目录...${NC}"
        rm -rf /var/log/dnsfailover
        echo "✓ 日志目录已删除"
    fi
else
    echo -e "${YELLOW}保留配置文件和日志${NC}"
    echo "配置目录: /etc/dnsfailover"
    echo "日志目录: /var/log/dnsfailover"
fi

echo ""
echo -e "${GREEN}✓ DNS Failover Agent 已成功卸载！${NC}"

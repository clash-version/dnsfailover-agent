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

echo -e "${GREEN}===== DNS Failover Agent 安装脚本 =====${NC}"

# 获取当前脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. 停止现有服务（如果存在）
if systemctl is-active --quiet dnsfailover; then
    echo -e "${YELLOW}停止现有服务...${NC}"
    systemctl stop dnsfailover
fi

# 2. 复制二进制文件
echo -e "${GREEN}安装二进制文件...${NC}"
if [ -f "$SCRIPT_DIR/dnsfailover" ]; then
    cp "$SCRIPT_DIR/dnsfailover" /usr/local/bin/dnsfailover
    chmod +x /usr/local/bin/dnsfailover
    echo "✓ 二进制文件已安装到 /usr/local/bin/dnsfailover"
elif [ -f "$SCRIPT_DIR/dnsfailover-linux-amd64" ]; then
    cp "$SCRIPT_DIR/dnsfailover-linux-amd64" /usr/local/bin/dnsfailover
    chmod +x /usr/local/bin/dnsfailover
    echo "✓ 二进制文件已安装到 /usr/local/bin/dnsfailover"
else
    echo -e "${RED}错误: 未找到 dnsfailover 二进制文件${NC}"
    echo "请将编译好的二进制文件放在脚本同目录下"
    exit 1
fi

# 3. 创建配置目录
echo -e "${GREEN}创建配置目录...${NC}"
mkdir -p /etc/dnsfailover
mkdir -p /var/log/dnsfailover

# 4. 复制配置文件（如果存在）
if [ -f "$SCRIPT_DIR/config.json" ]; then
    if [ -f /etc/dnsfailover/config.json ]; then
        echo -e "${YELLOW}配置文件已存在，备份为 config.json.bak${NC}"
        cp /etc/dnsfailover/config.json /etc/dnsfailover/config.json.bak
    fi
    cp "$SCRIPT_DIR/config.json" /etc/dnsfailover/config.json
    echo "✓ 配置文件已复制到 /etc/dnsfailover/config.json"
else
    echo -e "${YELLOW}警告: 未找到 config.json，请手动创建配置文件${NC}"
    echo "配置文件位置: /etc/dnsfailover/config.json"
fi

# 5. 安装 systemd 服务
echo -e "${GREEN}安装 systemd 服务...${NC}"
if [ -f "$SCRIPT_DIR/dnsfailover.service" ]; then
    cp "$SCRIPT_DIR/dnsfailover.service" /etc/systemd/system/dnsfailover.service
    echo "✓ 服务文件已安装到 /etc/systemd/system/dnsfailover.service"
else
    echo -e "${RED}错误: 未找到 dnsfailover.service 文件${NC}"
    exit 1
fi

# 6. 重新加载 systemd
echo -e "${GREEN}重新加载 systemd...${NC}"
systemctl daemon-reload

# 7. 启用服务（开机自启）
echo -e "${GREEN}启用开机自启动...${NC}"
systemctl enable dnsfailover

# 8. 启动服务
echo -e "${GREEN}启动服务...${NC}"
systemctl start dnsfailover

# 9. 检查服务状态
sleep 2
if systemctl is-active --quiet dnsfailover; then
    echo -e "${GREEN}✓ 服务安装成功并已启动！${NC}"
    echo ""
    echo -e "${GREEN}===== 常用命令 =====${NC}"
    echo "查看服务状态: systemctl status dnsfailover"
    echo "查看日志:     journalctl -u dnsfailover -f"
    echo "停止服务:     systemctl stop dnsfailover"
    echo "启动服务:     systemctl start dnsfailover"
    echo "重启服务:     systemctl restart dnsfailover"
    echo "禁用开机自启: systemctl disable dnsfailover"
    echo ""
    echo -e "${GREEN}===== 当前服务状态 =====${NC}"
    systemctl status dnsfailover --no-pager
else
    echo -e "${RED}✗ 服务启动失败${NC}"
    echo "请检查日志: journalctl -u dnsfailover -n 50"
    exit 1
fi

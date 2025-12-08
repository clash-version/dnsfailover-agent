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

printf "${GREEN}===== DNS Failover Agent 安装脚本 =====${NC}\n"

# 获取当前脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 1. 停止现有服务（如果存在）
if systemctl is-active --quiet dnsfailover; then
    printf "${YELLOW}停止现有服务...${NC}\n"
    systemctl stop dnsfailover
fi

# 2. 复制二进制文件
printf "${GREEN}安装二进制文件...${NC}\n"
if [ -f "$SCRIPT_DIR/dnsfailover" ]; then
    cp "$SCRIPT_DIR/dnsfailover" /usr/local/bin/dnsfailover
    chmod +x /usr/local/bin/dnsfailover
    printf "✓ 二进制文件已安装到 /usr/local/bin/dnsfailover\n"
elif [ -f "$SCRIPT_DIR/dnsfailover-linux-amd64" ]; then
    cp "$SCRIPT_DIR/dnsfailover-linux-amd64" /usr/local/bin/dnsfailover
    chmod +x /usr/local/bin/dnsfailover
    printf "✓ 二进制文件已安装到 /usr/local/bin/dnsfailover\n"
else
    printf "${RED}错误: 未找到 dnsfailover 二进制文件${NC}\n"
    printf "请将编译好的二进制文件放在脚本同目录下\n"
    exit 1
fi

# 3. 创建配置目录
printf "${GREEN}创建配置目录...${NC}\n"
mkdir -p /etc/dnsfailover
mkdir -p /var/log/dnsfailover

# 4. 复制环境变量文件
if [ -f "$SCRIPT_DIR/.env" ]; then
    if [ -f /etc/dnsfailover/.env ]; then
        printf "${YELLOW}环境变量文件已存在，备份为 .env.bak${NC}\n"
        cp /etc/dnsfailover/.env /etc/dnsfailover/.env.bak
    fi
    cp "$SCRIPT_DIR/.env" /etc/dnsfailover/.env
    chmod 600 /etc/dnsfailover/.env  # 限制权限，保护敏感信息
    printf "✓ 环境变量文件已复制到 /etc/dnsfailover/.env\n"
else
    printf "${YELLOW}警告: 未找到 .env 文件，请手动创建${NC}\n"
    printf "文件位置: /etc/dnsfailover/.env\n"
fi

# 5. 安装 systemd 服务
printf "${GREEN}安装 systemd 服务...${NC}\n"
if [ -f "$SCRIPT_DIR/dnsfailover.service" ]; then
    cp "$SCRIPT_DIR/dnsfailover.service" /etc/systemd/system/dnsfailover.service
    printf "✓ 服务文件已安装到 /etc/systemd/system/dnsfailover.service\n"
else
    printf "${RED}错误: 未找到 dnsfailover.service 文件${NC}\n"
    exit 1
fi

# 6. 重新加载 systemd
printf "${GREEN}重新加载 systemd...${NC}\n"
systemctl daemon-reload

# 7. 启用服务（开机自启）
printf "${GREEN}启用开机自启动...${NC}\n"
systemctl enable dnsfailover

# 8. 启动服务
printf "${GREEN}启动服务...${NC}\n"
systemctl start dnsfailover

# 9. 检查服务状态
sleep 2
if systemctl is-active --quiet dnsfailover; then
    printf "${GREEN}✓ 服务安装成功并已启动！${NC}\n"
    printf "\n"
    printf "${GREEN}===== 常用命令 =====${NC}\n"
    printf "查看服务状态: systemctl status dnsfailover\n"
    printf "查看日志:     journalctl -u dnsfailover -f\n"
    printf "停止服务:     systemctl stop dnsfailover\n"
    printf "启动服务:     systemctl start dnsfailover\n"
    printf "重启服务:     systemctl restart dnsfailover\n"
    printf "禁用开机自启: systemctl disable dnsfailover\n"
    printf "\n"
    printf "${GREEN}===== 当前服务状态 =====${NC}\n"
    systemctl status dnsfailover --no-pager
else
    printf "${RED}✗ 服务启动失败${NC}\n"
    printf "请检查日志: journalctl -u dnsfailover -n 50\n"
    exit 1
fi

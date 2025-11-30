#!/bin/bash

# DNS Failover Agent 卸载脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置变量
INSTALL_DIR="/opt/dnsfailover"
SERVICE_NAME="dnsfailover"

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}DNS Failover Agent 卸载脚本${NC}"
echo -e "${YELLOW}========================================${NC}"

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 权限运行此脚本${NC}"
    echo "使用: sudo bash uninstall.sh"
    exit 1
fi

# 停止并禁用服务
echo -e "${YELLOW}停止服务...${NC}"
if systemctl is-active --quiet ${SERVICE_NAME}; then
    systemctl stop ${SERVICE_NAME}
    echo -e "${GREEN}✓ 服务已停止${NC}"
fi

if systemctl is-enabled --quiet ${SERVICE_NAME}; then
    systemctl disable ${SERVICE_NAME}
    echo -e "${GREEN}✓ 服务已禁用${NC}"
fi

# 删除 systemd 服务文件
echo -e "${YELLOW}删除服务文件...${NC}"
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    rm /etc/systemd/system/${SERVICE_NAME}.service
    systemctl daemon-reload
    echo -e "${GREEN}✓ 服务文件已删除${NC}"
fi

# 询问是否删除安装目录
echo ""
read -p "是否删除安装目录 ${INSTALL_DIR}? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -d "${INSTALL_DIR}" ]; then
        rm -rf ${INSTALL_DIR}
        echo -e "${GREEN}✓ 安装目录已删除${NC}"
    fi
else
    echo -e "${YELLOW}保留安装目录: ${INSTALL_DIR}${NC}"
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}卸载完成！${NC}"
echo -e "${GREEN}========================================${NC}"

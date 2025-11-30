#!/bin/bash

# DNS Failover Agent 安装脚本
# 适用于 Linux 系统

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置变量
INSTALL_DIR="/opt/dnsfailover"
SERVICE_NAME="dnsfailover"
BINARY_NAME="dnsfailover-agent"
CONFIG_FILE="config.json"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}DNS Failover Agent 安装脚本${NC}"
echo -e "${GREEN}========================================${NC}"

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 权限运行此脚本${NC}"
    echo "使用: sudo bash install.sh"
    exit 1
fi

# 检查 Go 环境
echo -e "${YELLOW}检查 Go 环境...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go 环境${NC}"
    echo "请先安装 Go 1.23 或更高版本"
    echo "访问: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✓ 找到 Go 环境: ${GO_VERSION}${NC}"

# 检查 Go 版本（推荐 1.23 或更高）
GO_VERSION_NUM=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
if [ ! -z "$GO_VERSION_NUM" ]; then
    MAJOR=$(echo $GO_VERSION_NUM | cut -d. -f1)
    MINOR=$(echo $GO_VERSION_NUM | cut -d. -f2)
    if [ "$MAJOR" -lt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -lt 23 ]); then
        echo -e "${YELLOW}警告: 建议使用 Go 1.23 或更高版本，当前版本: ${GO_VERSION_NUM}${NC}"
    fi
fi

# 创建安装目录
echo -e "${YELLOW}创建安装目录...${NC}"
mkdir -p ${INSTALL_DIR}
mkdir -p ${INSTALL_DIR}/logs
echo -e "${GREEN}✓ 目录创建完成${NC}"

# 编译应用程序
echo -e "${YELLOW}编译应用程序...${NC}"
go build -o ${INSTALL_DIR}/${BINARY_NAME} .
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 编译成功${NC}"
else
    echo -e "${RED}✗ 编译失败${NC}"
    exit 1
fi

# 设置可执行权限
chmod +x ${INSTALL_DIR}/${BINARY_NAME}

# 复制配置文件
echo -e "${YELLOW}配置文件设置...${NC}"
if [ -f "${CONFIG_FILE}" ]; then
    cp ${CONFIG_FILE} ${INSTALL_DIR}/${CONFIG_FILE}
    echo -e "${GREEN}✓ 配置文件已复制${NC}"
else
    echo -e "${YELLOW}! 未找到 config.json，将在首次运行时创建${NC}"
fi

# 如果存在 remote-config.json，也复制
if [ -f "remote-config.json" ]; then
    cp remote-config.json ${INSTALL_DIR}/remote-config.json
fi

# 创建 systemd 服务文件
echo -e "${YELLOW}创建 systemd 服务...${NC}"
cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=DNS Failover Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BINARY_NAME} monitor start -c ${INSTALL_DIR}/${CONFIG_FILE}
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}✓ systemd 服务文件创建完成${NC}"

# 重新加载 systemd
systemctl daemon-reload

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}安装完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "安装目录: ${INSTALL_DIR}"
echo "配置文件: ${INSTALL_DIR}/${CONFIG_FILE}"
echo "日志目录: ${INSTALL_DIR}/logs"
echo ""
echo "使用以下命令管理服务:"
echo -e "${YELLOW}启动服务:${NC} sudo systemctl start ${SERVICE_NAME}"
echo -e "${YELLOW}停止服务:${NC} sudo systemctl stop ${SERVICE_NAME}"
echo -e "${YELLOW}重启服务:${NC} sudo systemctl restart ${SERVICE_NAME}"
echo -e "${YELLOW}查看状态:${NC} sudo systemctl status ${SERVICE_NAME}"
echo -e "${YELLOW}开机自启:${NC} sudo systemctl enable ${SERVICE_NAME}"
echo -e "${YELLOW}查看日志:${NC} sudo journalctl -u ${SERVICE_NAME} -f"
echo ""
echo -e "${YELLOW}提示: 请先编辑配置文件 ${INSTALL_DIR}/${CONFIG_FILE}，然后启动服务${NC}"

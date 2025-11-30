#!/bin/bash

# ==============================================================================
# DNS Failover Agent - 快速安装向导
# 这是一个简化版的安装脚本，适用于快速部署
# ==============================================================================

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

clear
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}     DNS Failover Agent - 快速安装向导${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo ""
echo "本脚本将引导您完成 DNS Failover Agent 的安装配置"
echo ""

# 检查 root 权限
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}[提示]${NC} 需要 root 权限，请使用 sudo 运行"
    exit 1
fi

# 获取安装目录
INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${GREEN}安装目录:${NC} $INSTALL_DIR"
echo ""

# 步骤 1: 环境检查
echo -e "${BLUE}[步骤 1/4]${NC} 环境检查"
echo "-----------------------------------------------------------"

if [ -f "./install.sh" ]; then
    echo -e "${GREEN}✓${NC} 找到完整安装脚本"
    
    read -p "是否运行完整的环境检查? [Y/n] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        echo ""
        echo "执行完整安装脚本..."
        exec ./install.sh
        exit 0
    fi
else
    echo -e "${YELLOW}!${NC} 未找到完整安装脚本，使用快速模式"
fi

echo ""

# 步骤 2: 配置文件
echo -e "${BLUE}[步骤 2/4]${NC} 配置文件设置"
echo "-----------------------------------------------------------"

if [ ! -f "config.json" ]; then
    if [ -f "config.example.json" ]; then
        cp config.example.json config.json
        echo -e "${GREEN}✓${NC} 已创建配置文件"
    else
        echo -e "${YELLOW}!${NC} 未找到配置模板"
    fi
fi

echo ""
echo "请配置以下关键参数:"
echo "  1. Cloudflare API Token (必需)"
echo "  2. 监控域名列表"
echo "  3. 故障转移地址"
echo ""

read -p "是否现在编辑配置文件? [Y/n] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    if command -v vim >/dev/null 2>&1; then
        vim config.json
    elif command -v nano >/dev/null 2>&1; then
        nano config.json
    else
        echo "请手动编辑: $INSTALL_DIR/config.json"
    fi
fi

echo ""

# 步骤 3: 权限设置
echo -e "${BLUE}[步骤 3/4]${NC} 权限设置"
echo "-----------------------------------------------------------"

if [ -f "dnsfailover" ]; then
    chmod +x dnsfailover
    echo -e "${GREEN}✓${NC} 可执行权限设置完成"
    
    # 尝试设置网络权限
    if command -v setcap >/dev/null 2>&1; then
        setcap cap_net_raw+ep dnsfailover 2>/dev/null && \
            echo -e "${GREEN}✓${NC} 网络权限 (CAP_NET_RAW) 设置完成" || \
            echo -e "${YELLOW}!${NC} 网络权限设置失败，将以 root 运行"
    fi
else
    echo -e "${YELLOW}!${NC} 未找到可执行文件"
fi

# 创建日志目录
mkdir -p logs
echo -e "${GREEN}✓${NC} 日志目录创建完成"

echo ""

# 步骤 4: 服务配置
echo -e "${BLUE}[步骤 4/4]${NC} 服务配置"
echo "-----------------------------------------------------------"

if command -v systemctl >/dev/null 2>&1; then
    read -p "是否创建 systemd 服务? [Y/n] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
        cat > /etc/systemd/system/dnsfailover.service << EOF
[Unit]
Description=DNS Failover Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/dnsfailover monitor start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
        
        systemctl daemon-reload
        echo -e "${GREEN}✓${NC} systemd 服务创建完成"
        
        echo ""
        read -p "是否立即启动服务? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            systemctl start dnsfailover
            echo -e "${GREEN}✓${NC} 服务已启动"
            echo ""
            systemctl status dnsfailover --no-pager
        fi
        
        echo ""
        read -p "是否设置开机自启? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            systemctl enable dnsfailover
            echo -e "${GREEN}✓${NC} 开机自启已设置"
        fi
    fi
else
    echo -e "${YELLOW}!${NC} 未检测到 systemd，请手动管理服务"
fi

echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}     安装完成！${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo ""
echo "常用命令:"
echo "  systemctl start dnsfailover     # 启动服务"
echo "  systemctl status dnsfailover    # 查看状态"
echo "  journalctl -u dnsfailover -f    # 查看日志"
echo ""
echo "配置文件: $INSTALL_DIR/config.json"
echo "日志目录: $INSTALL_DIR/logs"
echo ""
echo "详细文档: 查看 INSTALL.md"
echo ""

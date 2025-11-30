#!/bin/bash

# ==============================================================================
# DNS Failover Agent 一键安装脚本
# 版本: 1.0.0
# 功能: 环境检查、依赖安装、服务配置
# ==============================================================================

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 全局变量
INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="dnsfailover"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
MIN_GO_VERSION="1.20"
LOG_FILE="${INSTALL_DIR}/install.log"

# ==============================================================================
# 辅助函数
# ==============================================================================

# 打印信息
info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

# 打印成功信息
success() {
    echo -e "${GREEN}[✓]${NC} $1" | tee -a "$LOG_FILE"
}

# 打印警告信息
warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# 打印错误信息并退出
error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
    exit 1
}

# 打印分隔线
separator() {
    echo "==========================================================================" | tee -a "$LOG_FILE"
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 版本比较
version_ge() {
    [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$2" ]
}

# ==============================================================================
# 环境检查函数
# ==============================================================================

# 检查操作系统
check_os() {
    info "检查操作系统..."
    
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
        success "操作系统: $PRETTY_NAME"
        
        case $OS in
            ubuntu|debian|centos|rhel|rocky|almalinux|fedora|amzn)
                info "支持的操作系统: $OS"
                ;;
            *)
                warn "未经测试的操作系统: $OS，继续安装可能存在问题"
                ;;
        esac
    else
        warn "无法检测操作系统版本"
    fi
}

# 检查是否以 root 运行
check_root() {
    info "检查运行权限..."
    if [ "$EUID" -ne 0 ]; then
        error "请以 root 权限运行此脚本 (使用 sudo ./install.sh)"
    fi
    success "运行权限检查通过"
}

# 检查系统架构
check_arch() {
    info "检查系统架构..."
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64|amd64)
            success "系统架构: x86_64"
            ;;
        aarch64|arm64)
            success "系统架构: ARM64"
            ;;
        armv7l)
            success "系统架构: ARMv7"
            ;;
        *)
            warn "未经测试的架构: $ARCH"
            ;;
    esac
}

# 检查 systemd
check_systemd() {
    info "检查 systemd..."
    if ! command_exists systemctl; then
        error "未找到 systemd，此安装脚本需要 systemd 支持"
    fi
    
    if ! systemctl --version >/dev/null 2>&1; then
        error "systemd 无法正常工作"
    fi
    
    local systemd_version=$(systemctl --version | head -n1 | awk '{print $2}')
    success "systemd 版本: $systemd_version"
}

# 检查网络连接
check_network() {
    info "检查网络连接..."
    
    if command_exists ping; then
        if ping -c 1 -W 3 8.8.8.8 >/dev/null 2>&1; then
            success "网络连接正常"
        else
            warn "无法连接到互联网，某些功能可能受限"
        fi
    else
        warn "未找到 ping 命令，跳过网络检查"
    fi
}

# 检查必要的系统工具
check_required_tools() {
    info "检查必要的系统工具..."
    
    local missing_tools=()
    local tools=("curl" "wget" "tar" "gzip")
    
    for tool in "${tools[@]}"; do
        if ! command_exists "$tool"; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        warn "缺少以下工具: ${missing_tools[*]}"
        install_basic_tools "${missing_tools[@]}"
    else
        success "所有必要工具已安装"
    fi
}

# 安装基础工具
install_basic_tools() {
    info "安装缺失的基础工具..."
    
    if command_exists apt-get; then
        apt-get update -qq
        apt-get install -y "$@"
    elif command_exists yum; then
        yum install -y "$@"
    elif command_exists dnf; then
        dnf install -y "$@"
    else
        error "无法识别包管理器，请手动安装: $*"
    fi
    
    success "基础工具安装完成"
}

# 检查 Go 环境（可选）
check_go_environment() {
    info "检查 Go 环境（用于源码编译）..."
    
    if command_exists go; then
        local go_version=$(go version | awk '{print $3}' | sed 's/go//')
        if version_ge "$go_version" "$MIN_GO_VERSION"; then
            success "Go 版本: $go_version (满足最低要求 $MIN_GO_VERSION)"
            GO_INSTALLED=true
        else
            warn "Go 版本 $go_version 过低，推荐 $MIN_GO_VERSION 及以上"
            GO_INSTALLED=false
        fi
    else
        info "未安装 Go 环境（使用预编译二进制文件不需要）"
        GO_INSTALLED=false
    fi
}

# 检查可执行文件
check_executable() {
    info "检查可执行文件..."
    
    if [ ! -f "$INSTALL_DIR/dnsfailover" ]; then
        error "未找到可执行文件: $INSTALL_DIR/dnsfailover"
    fi
    
    # 检查文件类型
    if command_exists file; then
        local file_type=$(file "$INSTALL_DIR/dnsfailover")
        info "文件类型: $file_type"
    fi
    
    success "可执行文件检查通过"
}

# 检查端口占用（如果需要）
check_ports() {
    info "检查端口占用..."
    
    # DNS Failover Agent 主要使用 ICMP，不需要监听端口
    # 这里可以根据实际需求添加端口检查
    
    success "端口检查通过"
}

# 检查磁盘空间
check_disk_space() {
    info "检查磁盘空间..."
    
    local available_space=$(df "$INSTALL_DIR" | tail -1 | awk '{print $4}')
    local required_space=102400  # 100MB in KB
    
    if [ "$available_space" -lt "$required_space" ]; then
        warn "可用磁盘空间不足 100MB，当前: $(($available_space / 1024))MB"
    else
        success "磁盘空间充足: $(($available_space / 1024))MB"
    fi
}

# 检查 setcap 支持
check_capabilities() {
    info "检查 Linux Capabilities 支持..."
    
    if command_exists setcap; then
        success "setcap 命令可用"
        SETCAP_AVAILABLE=true
    else
        warn "setcap 命令不可用，将以 root 权限运行"
        SETCAP_AVAILABLE=false
        
        # 尝试安装 libcap
        if command_exists apt-get; then
            apt-get install -y libcap2-bin
        elif command_exists yum; then
            yum install -y libcap
        fi
        
        # 再次检查
        if command_exists setcap; then
            success "setcap 安装成功"
            SETCAP_AVAILABLE=true
        fi
    fi
}

# ==============================================================================
# 安装函数
# ==============================================================================

# 设置可执行权限
setup_executable() {
    info "设置可执行权限..."
    chmod +x "$INSTALL_DIR/dnsfailover"
    success "已设置可执行权限"
}

# 创建配置文件
setup_config() {
    info "配置文件设置..."
    
    if [ -f "$INSTALL_DIR/config.json" ]; then
        warn "配置文件已存在，跳过创建"
        read -p "是否备份现有配置? [y/N] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            cp "$INSTALL_DIR/config.json" "$INSTALL_DIR/config.json.backup.$(date +%Y%m%d_%H%M%S)"
            success "配置文件已备份"
        fi
    else
        if [ -f "$INSTALL_DIR/config.example.json" ]; then
            cp "$INSTALL_DIR/config.example.json" "$INSTALL_DIR/config.json"
            success "已创建默认配置文件 config.json"
            warn "请编辑配置文件填入正确的 Cloudflare API Token 和监控域名"
        else
            warn "未找到配置文件模板，需要手动创建 config.json"
        fi
    fi
}

# 创建日志目录
setup_log_directory() {
    info "创建日志目录..."
    mkdir -p "$INSTALL_DIR/logs"
    chmod 755 "$INSTALL_DIR/logs"
    success "日志目录创建完成: $INSTALL_DIR/logs"
}

# 设置网络权限
setup_capabilities() {
    if [ "$SETCAP_AVAILABLE" = true ]; then
        info "设置网络权限 (CAP_NET_RAW)..."
        if setcap cap_net_raw+ep "$INSTALL_DIR/dnsfailover" 2>/dev/null; then
            success "已设置 CAP_NET_RAW 权限（允许发送 ICMP 包）"
        else
            warn "设置 CAP_NET_RAW 失败，服务将以 root 权限运行"
        fi
    fi
}

# 创建 systemd 服务
create_systemd_service() {
    info "创建 systemd 服务..."
    
    cat > "$SERVICE_FILE" << 'SERVICEFILE'
[Unit]
Description=DNS Failover Agent - Automatic DNS Failover Service
Documentation=https://github.com/clash-version/dnsfailover-agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=INSTALL_DIR_PLACEHOLDER
ExecStart=INSTALL_DIR_PLACEHOLDER/dnsfailover monitor start
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

# 安全设置
NoNewPrivileges=false
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=INSTALL_DIR_PLACEHOLDER/logs

# 允许发送 ICMP 包
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
SERVICEFILE

    # 替换安装目录占位符
    sed -i "s|INSTALL_DIR_PLACEHOLDER|$INSTALL_DIR|g" "$SERVICE_FILE"
    
    success "systemd 服务文件创建完成"
}

# 重载 systemd
reload_systemd() {
    info "重载 systemd 配置..."
    systemctl daemon-reload
    success "systemd 配置已重载"
}

# 测试配置文件
test_config() {
    info "测试配置文件..."
    
    if [ -f "$INSTALL_DIR/config.json" ]; then
        # 简单的 JSON 格式检查
        if command_exists python3; then
            if python3 -m json.tool "$INSTALL_DIR/config.json" >/dev/null 2>&1; then
                success "配置文件格式正确"
            else
                error "配置文件 JSON 格式错误"
            fi
        elif command_exists jq; then
            if jq empty "$INSTALL_DIR/config.json" >/dev/null 2>&1; then
                success "配置文件格式正确"
            else
                error "配置文件 JSON 格式错误"
            fi
        else
            warn "无法验证 JSON 格式（缺少 python3 或 jq）"
        fi
    else
        warn "配置文件不存在，请创建后再启动服务"
    fi
}

# 显示服务管理命令
show_usage() {
    separator
    echo -e "${GREEN}安装完成！${NC}"
    separator
    echo ""
    echo "服务管理命令:"
    echo "  启动服务:   systemctl start $SERVICE_NAME"
    echo "  停止服务:   systemctl stop $SERVICE_NAME"
    echo "  重启服务:   systemctl restart $SERVICE_NAME"
    echo "  查看状态:   systemctl status $SERVICE_NAME"
    echo "  查看日志:   journalctl -u $SERVICE_NAME -f"
    echo "  开机自启:   systemctl enable $SERVICE_NAME"
    echo "  禁用自启:   systemctl disable $SERVICE_NAME"
    echo ""
    echo "配置文件: $INSTALL_DIR/config.json"
    echo "日志目录: $INSTALL_DIR/logs"
    echo "安装日志: $LOG_FILE"
    echo ""
    separator
}

# 交互式启动服务
interactive_start() {
    echo ""
    read -p "是否立即启动服务? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        info "启动服务..."
        systemctl start $SERVICE_NAME
        sleep 2
        systemctl status $SERVICE_NAME --no-pager
        success "服务已启动"
    fi
}

# 交互式设置开机自启
interactive_enable() {
    echo ""
    read -p "是否设置开机自启? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl enable $SERVICE_NAME
        success "已设置开机自启"
    fi
}

# 卸载旧版本（如果存在）
uninstall_old_version() {
    if systemctl is-active --quiet $SERVICE_NAME; then
        warn "检测到服务正在运行，准备停止..."
        systemctl stop $SERVICE_NAME
        success "旧服务已停止"
    fi
    
    if [ -f "$SERVICE_FILE" ]; then
        info "检测到旧版本服务文件，将更新..."
    fi
}

# ==============================================================================
# 主安装流程
# ==============================================================================

main() {
    # 初始化日志
    echo "=== DNS Failover Agent Installation Log ===" > "$LOG_FILE"
    echo "Time: $(date)" >> "$LOG_FILE"
    echo "" >> "$LOG_FILE"
    
    separator
    echo -e "${BLUE}DNS Failover Agent 一键安装脚本${NC}"
    echo -e "版本: 1.0.0"
    echo -e "安装目录: ${GREEN}$INSTALL_DIR${NC}"
    separator
    echo ""
    
    # 环境检查阶段
    info "开始环境检查..."
    echo ""
    
    check_root
    check_os
    check_arch
    check_systemd
    check_network
    check_disk_space
    check_required_tools
    check_capabilities
    check_go_environment
    check_ports
    
    echo ""
    separator
    success "环境检查完成"
    separator
    echo ""
    
    # 安装阶段
    info "开始安装..."
    echo ""
    
    check_executable
    uninstall_old_version
    setup_executable
    setup_config
    setup_log_directory
    setup_capabilities
    create_systemd_service
    reload_systemd
    test_config
    
    echo ""
    separator
    success "安装阶段完成"
    separator
    
    # 显示使用说明
    show_usage
    
    # 交互式启动
    interactive_start
    interactive_enable
    
    echo ""
    separator
    success "所有操作完成！"
    separator
    echo ""
    
    # 提示信息
    if [ ! -f "$INSTALL_DIR/config.json" ] || grep -q "YOUR_CLOUDFLARE_API_TOKEN" "$INSTALL_DIR/config.json" 2>/dev/null; then
        warn "请编辑配置文件后再启动服务:"
        echo "  vim $INSTALL_DIR/config.json"
        echo ""
    fi
    
    info "如有问题，请查看日志: $LOG_FILE"
}

# 捕获错误
trap 'error "安装过程中发生错误，请查看日志: $LOG_FILE"' ERR

# 执行主函数
main "$@"

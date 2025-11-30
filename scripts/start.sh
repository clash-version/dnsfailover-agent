#!/bin/bash

# DNS Failover Agent 启动脚本

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 检查配置文件
if [ ! -f "config.json" ]; then
    echo "错误: 未找到 config.json 配置文件"
    echo "请先复制 config.example.json 为 config.json 并配置相关参数"
    exit 1
fi

# 创建日志目录
mkdir -p logs

# 检查是否以 root 运行
if [ "$EUID" -ne 0 ]; then
    echo "警告: 建议以 root 或 sudo 权限运行此程序（用于 ICMP ping）"
    echo "尝试启动..."
fi

echo "=========================================="
echo "DNS Failover Agent 启动中..."
echo "=========================================="
echo ""

# 启动程序
./dnsfailover monitor start

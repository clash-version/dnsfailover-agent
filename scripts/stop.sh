#!/bin/bash

# DNS Failover Agent 停止脚本

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "DNS Failover Agent 停止中..."
echo "=========================================="
echo ""

# 停止程序
./dnsfailover monitor stop

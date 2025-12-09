# DNS Failover Agent

DNS 故障转移代理，支持 Ping/TCP/HTTP 健康检测，自动切换 Cloudflare DNS 记录。

## 功能特性

- ✅ **多种健康检测方式**：Ping、TCP、HTTP
- ✅ **自动故障转移**：检测失败达到阈值后自动切换 DNS
- ✅ **远程配置**：支持从 S3 或 HTTP 加载配置
- ✅ **权重选择**：支持按权重选择备用地址
- ✅ **冷却机制**：防止频繁切换
- ✅ **Systemd 支持**：可作为系统服务运行

## 快速开始

### 1. 配置环境变量

复制 `.env.example` 为 `.env` 并填写配置：

```bash
cp .env.example .env
```

```properties
# Cloudflare API Token
CF_API_TOKEN=your-cloudflare-api-token

# AWS 配置 (用于从S3读取远程配置)
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key
AWS_REGION=ap-southeast-1

# 远程配置URL
REMOTE_CONFIG_URL=s3://your-bucket/dns/remote-config.json

# 日志配置
LOG_ENABLED=true
LOG_LEVEL=info
LOG_PATH=./logs/failover.log
LOG_MAX_DAYS=3
```

### 2. 远程配置文件格式

`remote-config.json` 示例：

```json
{
    "ping": {
        "frequency": 30,
        "failcount": 5,
        "timeout": 5,
        "retry": 3,
        "remote_update_freq": 60,
        "domains": [
            "example.com"
        ],
        "failover": [
            {"address": "backup1.example.com", "weight": 100},
            {"address": "backup2.example.com", "weight": 50}
        ]
    },
    "tcp": {
        "frequency": 30,
        "failcount": 5,
        "timeout": 5,
        "retry": 3,
        "domains": ["example.com:443"],
        "failover": []
    },
    "http": {
        "frequency": 30,
        "failcount": 5,
        "timeout": 5,
        "retry": 3,
        "domains": ["https://example.com/health"],
        "failover": []
    }
}
```

### 3. 运行

```bash
# 直接运行
./dnsfailover monitor start

# 或使用 systemd
sudo bash systemd/install-service.sh
```

## 命令

```bash
# 启动监控
./dnsfailover monitor start

# 查看帮助
./dnsfailover --help
```

## Systemd 部署

详见 [SYSTEMD.md](./SYSTEMD.md)

## 日志输出示例

```
[PING] ✓ example.com (延迟: 15ms)
[PING] ✗ example.com 失败 (2/5) - timeout
[PING] ⚠ example.com 触发故障转移
✓ 故障转移成功: example.com
```

## 编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o dnsfailover-linux-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o dnsfailover.exe .
```

## 许可证

MIT License

# Linux 环境安装部署指南

## 系统要求

- Linux 操作系统（Ubuntu、CentOS、Debian 等）
- Go 1.23 或更高版本（推荐 1.23.2+）
- systemd（用于服务管理）
- root 权限

## 快速安装

### 方法一：使用安装脚本（推荐）

1. **下载或克隆项目到 Linux 服务器**

```bash
# 使用 git 克隆
git clone <repository-url>
cd dnsfailover-agent

# 或者上传项目文件到服务器
```

2. **运行安装脚本**

```bash
sudo bash install.sh
```

安装脚本会自动完成以下操作：
- 检查 Go 环境
- 编译应用程序
- 创建安装目录 `/opt/dnsfailover`
- 复制配置文件和二进制文件
- 创建 systemd 服务
- 设置权限

3. **编辑配置文件**

```bash
sudo nano /opt/dnsfailover/config.json
```

根据你的需求修改配置，主要配置项：
- Cloudflare API 密钥和域名信息
- 监控目标和备用地址
- 检测间隔和超时设置

4. **启动服务**

```bash
# 启动服务
sudo systemctl start dnsfailover

# 查看状态
sudo systemctl status dnsfailover

# 设置开机自启
sudo systemctl enable dnsfailover

# 查看实时日志
sudo journalctl -u dnsfailover -f
```

### 方法二：手动安装

如果你想手动控制安装过程：

1. **安装 Go 环境**（如果未安装）

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# CentOS/RHEL
sudo yum install golang

# 或从官方下载指定版本（推荐 1.23.2 或更高）
wget https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

2. **编译应用程序**

```bash
cd dnsfailover-agent
go build -o dnsfailover-agent .
```

3. **创建安装目录**

```bash
sudo mkdir -p /opt/dnsfailover
sudo mkdir -p /opt/dnsfailover/logs
```

4. **复制文件**

```bash
sudo cp dnsfailover-agent /opt/dnsfailover/
sudo cp config.json /opt/dnsfailover/
sudo chmod +x /opt/dnsfailover/dnsfailover-agent
```

5. **创建 systemd 服务**

```bash
sudo nano /etc/systemd/system/dnsfailover.service
```

添加以下内容：

```ini
[Unit]
Description=DNS Failover Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/dnsfailover
ExecStart=/opt/dnsfailover/dnsfailover-agent monitor start -c /opt/dnsfailover/config.json
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

6. **启动服务**

```bash
sudo systemctl daemon-reload
sudo systemctl start dnsfailover
sudo systemctl enable dnsfailover
```

## 服务管理

### 常用命令

```bash
# 启动服务
sudo systemctl start dnsfailover

# 停止服务
sudo systemctl stop dnsfailover

# 重启服务
sudo systemctl restart dnsfailover

# 查看状态
sudo systemctl status dnsfailover

# 设置开机自启
sudo systemctl enable dnsfailover

# 取消开机自启
sudo systemctl disable dnsfailover

# 查看实时日志
sudo journalctl -u dnsfailover -f

# 查看最近日志
sudo journalctl -u dnsfailover -n 100

# 查看今天的日志
sudo journalctl -u dnsfailover --since today
```

### 查看应用日志

应用程序的详细日志存储在：
```bash
/opt/dnsfailover/logs/
```

查看日志：
```bash
# 查看最新日志
tail -f /opt/dnsfailover/logs/*.log

# 查看所有日志
ls -lh /opt/dnsfailover/logs/
```

## 配置文件

主配置文件位置：`/opt/dnsfailover/config.json`

配置示例：
```json
{
  "cloudflare": {
    "api_token": "your-cloudflare-api-token",
    "zone_id": "your-zone-id",
    "domain": "example.com",
    "record_name": "www.example.com"
  },
  "monitor": {
    "targets": ["8.8.8.8", "1.1.1.1"],
    "backup_address": "backup.example.com",
    "check_interval": 60,
    "timeout": 5,
    "failure_threshold": 3
  },
  "remote_config": {
    "enabled": true,
    "url": "https://example.com/remote-config.json",
    "update_interval": 300
  }
}
```

修改配置后重启服务：
```bash
sudo systemctl restart dnsfailover
```

## 卸载

使用卸载脚本：
```bash
sudo bash uninstall.sh
```

或手动卸载：
```bash
# 停止并禁用服务
sudo systemctl stop dnsfailover
sudo systemctl disable dnsfailover

# 删除服务文件
sudo rm /etc/systemd/system/dnsfailover.service
sudo systemctl daemon-reload

# 删除安装目录（可选）
sudo rm -rf /opt/dnsfailover
```

## 故障排查

### 服务无法启动

1. 查看服务状态和错误信息：
```bash
sudo systemctl status dnsfailover
sudo journalctl -u dnsfailover -n 50
```

2. 检查配置文件是否正确：
```bash
cat /opt/dnsfailover/config.json
```

3. 手动运行测试：
```bash
cd /opt/dnsfailover
./dnsfailover-agent monitor start -c config.json
```

### 权限问题

确保二进制文件有执行权限：
```bash
sudo chmod +x /opt/dnsfailover/dnsfailover-agent
```

### 网络连接问题

检查服务器能否访问 Cloudflare API：
```bash
curl -I https://api.cloudflare.com/client/v4/user
```

## 使用 Docker 部署（可选）

如果你更喜欢使用 Docker：

1. **创建 Dockerfile**（已包含在项目中）

2. **构建镜像**
```bash
docker build -t dnsfailover-agent .
```

3. **运行容器**
```bash
docker run -d \
  --name dnsfailover \
  --restart unless-stopped \
  -v $(pwd)/config.json:/app/config.json \
  -v $(pwd)/logs:/app/logs \
  dnsfailover-agent
```

4. **查看日志**
```bash
docker logs -f dnsfailover
```

## 安全建议

1. **保护配置文件**：配置文件包含敏感信息，设置适当权限
```bash
sudo chmod 600 /opt/dnsfailover/config.json
```

2. **使用非 root 用户**：修改 systemd 服务文件，使用专门的用户运行
```bash
sudo useradd -r -s /bin/false dnsfailover
sudo chown -R dnsfailover:dnsfailover /opt/dnsfailover
```

然后在服务文件中修改：
```ini
User=dnsfailover
Group=dnsfailover
```

3. **防火墙配置**：如果需要允许 ICMP ping，确保防火墙规则正确

4. **定期更新**：保持应用程序和依赖库的更新

## 监控和维护

### 设置日志轮转

日志文件会自动轮转（使用 lumberjack），默认配置：
- 最大文件大小：100MB
- 保留文件数：3 个
- 保留天数：28 天

### 性能监控

查看资源使用情况：
```bash
# CPU 和内存使用
ps aux | grep dnsfailover-agent

# 使用 top
top -p $(pgrep dnsfailover-agent)
```

## 技术支持

如有问题，请检查：
1. 系统日志：`sudo journalctl -u dnsfailover -f`
2. 应用日志：`/opt/dnsfailover/logs/`
3. 配置文件：`/opt/dnsfailover/config.json`

# Systemd 服务安装说明

本文档说明如何将 DNS Failover Agent 安装为 Linux 系统服务。

## 📦 文件说明

- `dnsfailover.service` - systemd 服务配置文件
- `install-service.sh` - 自动安装脚本
- `uninstall-service.sh` - 自动卸载脚本

## 🚀 快速安装

### 1. 下载并解压程序

```bash
# 下载 release 版本
wget https://github.com/clash-version/n8n-agent/releases/download/v1.0.0/dnsfailover-linux-amd64.tar.gz

# 解压
tar -xzf dnsfailover-linux-amd64.tar.gz

# 重命名为 dnsfailover
mv dnsfailover-linux-amd64 dnsfailover
```

### 2. 准备配置文件

确保 `config.json` 文件存在于同目录下。

### 3. 运行安装脚本

```bash
# 添加执行权限
chmod +x install-service.sh

# 以 root 权限运行安装脚本
sudo ./install-service.sh
```

安装脚本会自动完成以下操作：
- ✓ 复制二进制文件到 `/usr/local/bin/dnsfailover`
- ✓ 创建配置目录 `/etc/dnsfailover`
- ✓ 复制配置文件到 `/etc/dnsfailover/config.json`
- ✓ 安装 systemd 服务
- ✓ 启用开机自启动
- ✓ 启动服务

## 📝 手动安装（可选）

如果你想手动安装，可以按以下步骤操作：

```bash
# 1. 复制二进制文件
sudo cp dnsfailover /usr/local/bin/dnsfailover
sudo chmod +x /usr/local/bin/dnsfailover

# 2. 创建配置目录
sudo mkdir -p /etc/dnsfailover
sudo mkdir -p /var/log/dnsfailover

# 3. 复制配置文件
sudo cp config.json /etc/dnsfailover/config.json

# 4. 复制服务文件
sudo cp dnsfailover.service /etc/systemd/system/dnsfailover.service

# 5. 重新加载 systemd
sudo systemctl daemon-reload

# 6. 启用并启动服务
sudo systemctl enable dnsfailover
sudo systemctl start dnsfailover
```

## 🔧 常用命令

### 查看服务状态
```bash
sudo systemctl status dnsfailover
```

### 启动服务
```bash
sudo systemctl start dnsfailover
```

### 停止服务
```bash
sudo systemctl stop dnsfailover
```

### 重启服务
```bash
sudo systemctl restart dnsfailover
```

### 查看实时日志
```bash
sudo journalctl -u dnsfailover -f
```

### 查看最近 100 条日志
```bash
sudo journalctl -u dnsfailover -n 100
```

### 查看今天的日志
```bash
sudo journalctl -u dnsfailover --since today
```

### 启用开机自启动
```bash
sudo systemctl enable dnsfailover
```

### 禁用开机自启动
```bash
sudo systemctl disable dnsfailover
```

## 📂 文件位置

| 项目 | 路径 |
|------|------|
| 二进制文件 | `/usr/local/bin/dnsfailover` |
| 配置文件 | `/etc/dnsfailover/config.json` |
| 服务文件 | `/etc/systemd/system/dnsfailover.service` |
| 日志目录 | `/var/log/dnsfailover/` |
| 系统日志 | `journalctl -u dnsfailover` |

## 🗑️ 卸载服务

### 使用卸载脚本（推荐）
```bash
chmod +x uninstall-service.sh
sudo ./uninstall-service.sh
```

### 手动卸载
```bash
# 1. 停止并禁用服务
sudo systemctl stop dnsfailover
sudo systemctl disable dnsfailover

# 2. 删除服务文件
sudo rm /etc/systemd/system/dnsfailover.service

# 3. 重新加载 systemd
sudo systemctl daemon-reload

# 4. 删除二进制文件
sudo rm /usr/local/bin/dnsfailover

# 5. 删除配置和日志（可选）
sudo rm -rf /etc/dnsfailover
sudo rm -rf /var/log/dnsfailover
```

## 🔍 故障排查

### 服务启动失败
```bash
# 查看详细错误信息
sudo journalctl -u dnsfailover -n 50 --no-pager

# 检查配置文件是否正确
sudo cat /etc/dnsfailover/config.json

# 手动运行程序测试
/usr/local/bin/dnsfailover monitor
```

### 服务运行但无法工作
```bash
# 检查服务状态
sudo systemctl status dnsfailover

# 查看实时日志
sudo journalctl -u dnsfailover -f

# 检查权限
ls -la /etc/dnsfailover/
ls -la /var/log/dnsfailover/
```

### 配置修改后重启
```bash
# 修改配置文件
sudo nano /etc/dnsfailover/config.json

# 重启服务使配置生效
sudo systemctl restart dnsfailover

# 查看是否正常启动
sudo systemctl status dnsfailover
```

## ⚙️ 服务配置说明

服务配置文件 `/etc/systemd/system/dnsfailover.service` 的主要配置项：

- `User=root` - 运行用户（可根据需要修改）
- `ExecStart=/usr/local/bin/dnsfailover monitor` - 启动命令
- `Restart=always` - 自动重启策略
- `RestartSec=10` - 重启间隔 10 秒
- `WorkingDirectory=/etc/dnsfailover` - 工作目录

如需修改配置，编辑后执行：
```bash
sudo systemctl daemon-reload
sudo systemctl restart dnsfailover
```

## 📊 性能监控

### 查看资源使用
```bash
# 查看 CPU 和内存使用
systemctl status dnsfailover

# 使用 top 查看
top -p $(pidof dnsfailover)

# 使用 htop 查看（需安装 htop）
htop -p $(pidof dnsfailover)
```

### 查看进程信息
```bash
ps aux | grep dnsfailover
```

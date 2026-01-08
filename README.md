# DNS Failover Agent

一个轻量级的网络监控代理，支持 Ping、TCP、HTTP 检测，提供 Web 管理面板和灵活的 Webhook 告警通知。

![Dashboard Preview](https://via.placeholder.com/800x400?text=Web+Dashboard+Preview)

## ✨ 功能特性

- **多协议监控**：支持 ICMP Ping、TCP 端口连接、HTTP/HTTPS 请求状态检测。
- **可视化管理**：内置 Web 控制台，实时查看监控状态、日志和修改配置。
- **灵活告警**：
  - 支持自定义 Webhook（如钉钉、飞书、Slack、Telegram 等）。
  - 支持设置请求头、超时时间、重试次数。
  - **静默期机制**：告警触发后自动静默，防止消息轰炸。
- **定时任务**：支持 Crontab 表达式的定时检测或网络操作。
- **单文件部署**：Web 界面嵌入二进制文件，无需部署静态资源。

## 🚀 快速开始 (Linux)

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/clash-version/n8n-agent/main/install.sh | sudo bash
```

安装完成后，访问 Web 面板：`http://服务器IP:8080/`

### 一键卸载

下载并运行卸载脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/clash-version/n8n-agent/main/uninstall.sh | sudo bash
```

## 🛠️ 手动构建

如果你需要修改代码或在 Windows/macOS 上运行：

```bash
# 1. 克隆代码
git clone https://github.com/clash-version/n8n-agent.git
cd n8n-agent

# 2. 编译 (Web 资源会自动嵌入)
go build -o dnsfailover .

# 3. 运行
./dnsfailover monitor start --web --port 8080
```

## ⚙️ 配置说明

所有配置均可通过 Web 面板进行实时修改并持久化保存。

- **配置文件路径**: `/etc/dnsfailover/probe.db` (SQLite)
- **日志文件路径**: `/var/log/dnsfailover/`

### Webhook 数据格式

系统会向你的 Webhook URL 发送如下 JSON 数据：

```json
{
  "type": "down",                // 告警类型: down (故障) | recovery (恢复)
  "probe_type": "tcp",           // 检测类型: ping | tcp | http
  "target": "example.com:443",   // 目标地址
  "fail_count": 3,               // 当前连续失败次数
  "threshold": 3,                // 触发阈值
  "error": "i/o timeout",        // 具体的错误信息
  "timestamp": 1709880000,       // Unix 时间戳
  "message": "[tcp] example.com:443 连续失败 3 次..." // 可读消息
}
```

## 📝 License

MIT

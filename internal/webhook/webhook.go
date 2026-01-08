package webhook

import (
	"bytes"
	"dnsfailover/internal/config"
	"dnsfailover/internal/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeDown     AlertType = "down"     // 目标不可达
	AlertTypeRecovery AlertType = "recovery" // 目标恢复
)

// Alert 告警信息
type Alert struct {
	Type      AlertType `json:"type"`       // 告警类型
	ProbeType string    `json:"probe_type"` // 探针类型 (ping/tcp/http)
	Target    string    `json:"target"`     // 检测目标
	FailCount int       `json:"fail_count"` // 连续失败次数
	Threshold int       `json:"threshold"`  // 失败阈值
	Error     string    `json:"error"`      // 错误信息
	Timestamp int64     `json:"timestamp"`  // 时间戳
	Message   string    `json:"message"`    // 可读消息
}

// Client Webhook 客户端
type Client struct {
	cfg        *config.WebhookConfig
	httpClient *http.Client
}

// NewClient 创建 Webhook 客户端
func NewClient(cfg *config.WebhookConfig) *Client {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// SendAlert 发送告警
func (c *Client) SendAlert(alert *Alert) error {
	if c.cfg.URL == "" {
		logger.Warn("[WEBHOOK] URL 未配置，无法发送告警通知")
		return nil
	}

	// 设置时间戳
	alert.Timestamp = time.Now().Unix()

	// 生成可读消息
	if alert.Type == AlertTypeDown {
		alert.Message = fmt.Sprintf("[%s] %s 连续失败 %d 次（阈值: %d）: %s",
			alert.ProbeType, alert.Target, alert.FailCount, alert.Threshold, alert.Error)
	} else {
		alert.Message = fmt.Sprintf("[%s] %s 已恢复正常",
			alert.ProbeType, alert.Target)
	}

	// 序列化 JSON
	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("序列化告警失败: %w", err)
	}

	// 创建请求
	method := c.cfg.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, c.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 Headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range c.cfg.Headers {
		req.Header.Set(key, value)
	}

	// 打印详细日志
	logger.Infof("[WEBHOOK] ━━━━━━━━━━ 发送通知 ━━━━━━━━━━")
	logger.Infof("[WEBHOOK] URL: %s", c.cfg.URL)
	logger.Infof("[WEBHOOK] Method: %s", method)
	logger.Infof("[WEBHOOK] Type: %s", alert.Type)
	logger.Infof("[WEBHOOK] Target: %s", alert.Target)
	logger.Infof("[WEBHOOK] Message: %s", alert.Message)
	logger.Infof("[WEBHOOK] Body: %s", string(body))

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Errorf("[WEBHOOK] ✗ 发送失败: %v", err)
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warnf("[WEBHOOK] 响应状态码异常: %d", resp.StatusCode)
		return fmt.Errorf("响应状态码异常: %d", resp.StatusCode)
	}

	logger.Infof("[WEBHOOK] ✓ 告警发送成功: %s (状态码: %d)", alert.Target, resp.StatusCode)
	return nil
}

// SendDownAlert 发送故障告警
func (c *Client) SendDownAlert(probeType, target string, failCount, threshold int, errMsg string) error {
	return c.SendAlert(&Alert{
		Type:      AlertTypeDown,
		ProbeType: probeType,
		Target:    target,
		FailCount: failCount,
		Threshold: threshold,
		Error:     errMsg,
	})
}

// SendRecoveryAlert 发送恢复告警
func (c *Client) SendRecoveryAlert(probeType, target string) error {
	return c.SendAlert(&Alert{
		Type:      AlertTypeRecovery,
		ProbeType: probeType,
		Target:    target,
	})
}

// UpdateConfig 更新配置
func (c *Client) UpdateConfig(cfg *config.WebhookConfig) {
	c.cfg = cfg
	c.httpClient.Timeout = time.Duration(cfg.Timeout) * time.Second
}

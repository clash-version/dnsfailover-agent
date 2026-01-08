package schedule

import (
	"time"
)

// CheckType 检测类型
type CheckType string

const (
	CheckTypePing CheckType = "ping"
	CheckTypeTCP  CheckType = "tcp"
	CheckTypeHTTP CheckType = "http"
)

// Task 定时任务结构
type Task struct {
	ID          string            `json:"id"`           // 任务ID
	Name        string            `json:"name"`         // 任务名称
	Enabled     bool              `json:"enabled"`      // 是否启用
	Cron        string            `json:"cron"`         // Cron 表达式 (如: "0 18 * * *" 每天18点)
	CheckType   CheckType         `json:"check_type"`   // 检测类型: ping/tcp/http
	Target      string            `json:"target"`       // 检测目标 (域名/IP/URL)
	Port        int               `json:"port"`         // TCP检测端口 (仅 tcp 类型使用)
	Timeout     int               `json:"timeout"`      // 超时时间(秒)
	WebhookURL  string            `json:"webhook_url"`  // Webhook 回调地址
	WebhookData map[string]string `json:"webhook_data"` // Webhook 自定义数据 (会附加到通知中)
	CreatedAt   time.Time         `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time         `json:"updated_at"`   // 更新时间
	LastRunAt   *time.Time        `json:"last_run_at"`  // 上次执行时间
	LastResult  string            `json:"last_result"`  // 上次执行结果
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID      string    `json:"task_id"`
	TaskName    string    `json:"task_name"`
	Success     bool      `json:"success"`
	CheckType   CheckType `json:"check_type"`
	Target      string    `json:"target"`
	Message     string    `json:"message"`
	ExecutedAt  time.Time `json:"executed_at"`
	WebhookSent bool      `json:"webhook_sent"`
}

// WebhookPayload 定时任务 Webhook 通知内容
type WebhookPayload struct {
	Event      string            `json:"event"`       // 事件类型: schedule_check
	TaskID     string            `json:"task_id"`     // 任务ID
	TaskName   string            `json:"task_name"`   // 任务名称
	CheckType  CheckType         `json:"check_type"`  // 检测类型
	Target     string            `json:"target"`      // 检测目标
	Available  bool              `json:"available"`   // 目标是否可用
	Message    string            `json:"message"`     // 详细信息
	CustomData map[string]string `json:"custom_data"` // 用户自定义数据
	ExecutedAt time.Time         `json:"executed_at"` // 执行时间
	Timestamp  int64             `json:"timestamp"`   // 时间戳
}
